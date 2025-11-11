package main

import (
	"flag"
	"fmt"
	"github.com/ericsuh/adapt/aptfile"
	"github.com/ericsuh/adapt/armor"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"slices"
	"strings"
)

var ensuredAddAptRepository bool = false
var needsUpdate bool = true

func main() {
	var dryRunFlag bool
	var shortDryRunFlag bool

	flag.BoolVar(&dryRunFlag, "dry-run", false, "show actions without making changes")
	flag.BoolVar(&shortDryRunFlag, "n", false, "alias for --dry-run")
	flag.Parse()

	dryRun := dryRunFlag || shortDryRunFlag

	if !dryRun {
		currentUser, err := user.Current()
		if err != nil {
			log.Fatal("Failed to get current user: ", err)
		}
		if currentUser.Uid != "0" {
			log.Fatal("This program must be run as root. Please use sudo or run as root user.")
		}
	}

	var aptfilePath string
	switch flag.NArg() {
	case 0:
		aptfilePath = "Aptfile"
	case 1:
		aptfilePath = flag.Arg(0)
	default:
		log.Fatal("Usage: adapt <Aptfile> or place Aptfile in current directory")
	}

	if _, err := os.Stat(aptfilePath); os.IsNotExist(err) {
		if aptfilePath == "Aptfile" {
			log.Fatal("Usage: adapt <Aptfile> or place Aptfile in current directory")
		}
		log.Fatalf("File %s not found", aptfilePath)
	}

	processAptfile(aptfilePath, dryRun)
}

func processAptfile(path string, dryRun bool) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("Failed to open Aptfile: %v", err)
	}
	defer func() {
		err2 := file.Close()
		if err2 != nil {
			log.Fatalf("Failed to close Aptfile: %v", err2)
		}
	}()
	dirs, err := aptfile.Parse(file)
	if err != nil {
		log.Fatalf("Failed to read Aptfile: %v", err)
	}

	pkgs := make([]aptfile.PackageDirective, 0)

	// First pass, skip package installation (except for .deb files,
	// which can be necessary for setting up repos or keyrings, etc.)
	for _, d := range dirs {
		switch dir := d.(type) {
		case aptfile.PpaDirective:
			if err := addPPA(dir.Name, dryRun); err != nil {
				log.Fatalf("Failed to add PPA %s: %v", dir.Name, err)
			} else {
				needsUpdate = true
			}
		case aptfile.RepoDirective:
			if err := addRepo(dir, dryRun); err != nil {
				log.Fatalf("Failed to add repository: %v", err)
			} else {
				needsUpdate = true
			}
		case aptfile.PackageDirective:
			pkgs = append(pkgs, dir)
			// Don't install in this phase
			continue
		case aptfile.DebFileDirective:
			if err := installDeb(dir.Path, dryRun); err != nil {
				log.Fatalf("Failed to install deb %s: %v", dir.Path, err)
			}
		case aptfile.PinDirective:
			if err := addPinPreference(dir, dryRun); err != nil {
				log.Fatalf("Failed to add pin: %v", err)
			}
		case aptfile.HoldDirective:
			if err := addHold(dir, dryRun); err != nil {
				log.Fatalf("Failed to add hold: %v", err)
			}
		default:
			log.Fatalf("Unknown directive: %v", d)
		}
	}

	err = installPackages(pkgs, dryRun)
	if err != nil {
		log.Fatalf("Failed to install packages: %v", err)
	}
}

func installPackages(pkgs []aptfile.PackageDirective, dryRun bool) error {
	names := make([]string, len(pkgs))
	for i, p := range pkgs {
		if p.Version != "" {
			names[i] = fmt.Sprintf("%s=%s", p.Name, p.Version)
		} else if p.Release != "" {
			names[i] = fmt.Sprintf("%s/%s", p.Name, p.Release)
		} else {
			names[i] = p.Name
		}
	}
	if dryRun {
		if needsUpdate {
			fmt.Println("[dry-run] Would update package lists")
		}
		fmt.Printf("[dry-run] Would install packagess: %s\n", strings.Join(names, ", "))
		return nil
	}
	if needsUpdate {
		fmt.Printf("Updating package lists...\n")
		cmd := exec.Command("apt-get", "update", "--yes")
		cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("error updating package lists: %w", err)
		}
		needsUpdate = false
	}
	fmt.Printf("Installing package: %s\n", strings.Join(names, ", "))
	cmd := exec.Command("apt-get", slices.Concat([]string{"install", "--yes", "--no-install-recommends"}, names)...)
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func addPPA(ppa string, dryRun bool) error {
	if !ensuredAddAptRepository {
		if _, err := exec.LookPath("add-apt-repository"); err == nil {
			ensuredAddAptRepository = true
		} else if dryRun {
			fmt.Println("[dry-run] Would install utility add-apt-repository (package software-properties-common)")
			ensuredAddAptRepository = true
		} else {
			fmt.Println("Installing required utility add-apt-repository (package software-properties-common)")
			err := installPackages([]aptfile.PackageDirective{{Name: "software-properties-common"}}, dryRun)
			if err != nil {
				return err
			}
			ensuredAddAptRepository = true
		}
	}

	if dryRun {
		fmt.Printf("[dry-run] Would add PPA: %s\n", ppa)
		return nil
	}

	fmt.Printf("Adding PPA: %s\n", ppa)
	cmd := exec.Command("add-apt-repository", "--yes", fmt.Sprintf("ppa:%s", ppa))
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func installDeb(path string, dryRun bool) error {
	var debFile string

	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		if dryRun {
			fmt.Printf("[dry-run] Would download .deb from: %s\n", path)
			debFile = path
		} else {
			fmt.Printf("Downloading .deb from: %s\n", path)
			tempFile, err := downloadFile(path)
			if err != nil {
				return err
			}
			defer func() {
				err2 := os.Remove(tempFile)
				if err2 != nil {
					log.Printf("Error cleaning up temp file: %v", err2)
				}
			}()
			debFile = tempFile
		}
	} else {
		debFile = path
	}

	if dryRun {
		fmt.Printf("[dry-run] Would install .deb: %s\n", debFile)
		return nil
	}

	fmt.Printf("Installing .deb: %s\n", debFile)
	cmd := exec.Command("dpkg", "-i", debFile)
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Try with apt-get instead
	if err := cmd.Run(); err != nil {
		fixCmd := exec.Command("apt-get", "install", "-f", "-y")
		fixCmd.Stdout = os.Stdout
		fixCmd.Stderr = os.Stderr
		err := fixCmd.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func addRepo(d aptfile.RepoDirective, dryRun bool) error {
	repoType := "deb"
	if d.IsSrc {
		repoType = "deb-src"
	}

	keyringPath := ""
	if d.SignedBy != "" {
		keyringPath = fmt.Sprintf("/usr/share/keyrings/%s.gpg", sanitizeFilename(d.URL))
		if dryRun {
			fmt.Printf("[dry-run] Would download GPG key from: %s\n", d.SignedBy)
		} else {
			fmt.Printf("Downloading GPG key from: %s\n", d.SignedBy)
			if err := downloadGPGKey(d.SignedBy, keyringPath); err != nil {
				return err
			}
		}
	}

	listFile := fmt.Sprintf("/etc/apt/sources.list.d/%s.list", sanitizeFilename(d.URL))
	var sourceLine string
	opts := make([]string, 0)
	if d.Arch != "" {
		opts = append(opts, fmt.Sprintf("arch=%s", d.Arch))
	}
	if keyringPath != "" {
		opts = append(opts, fmt.Sprintf("signed-by=%s", keyringPath))
	}
	if len(opts) > 0 {
		sourceLine = fmt.Sprintf("%s [%s] %s %s %s", repoType, strings.Join(opts, " "), d.URL, d.Suite, d.Component)
	} else {
		sourceLine = fmt.Sprintf("%s %s %s %s", repoType, d.URL, d.Suite, d.Component)
	}
	if dryRun {
		fmt.Printf("[dry-run] Would add repository: %s\n", sourceLine)
		return nil
	}
	return os.WriteFile(listFile, []byte(sourceLine), 0644)
}

func downloadFile(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer func() {
		err2 := resp.Body.Close()
		if err2 != nil {
			log.Printf("Error closing HTTP request: %v", err2)
		}
	}()

	tempFile, err := os.CreateTemp("", "*.deb")
	if err != nil {
		return "", err
	}
	defer func() {
		err2 := tempFile.Close()
		if err2 != nil {
			log.Printf("Error closing temp file: %v", err2)
		}
	}()

	_, err = tempFile.ReadFrom(resp.Body)
	if err != nil {
		err2 := os.Remove(tempFile.Name())
		if err2 != nil {
			log.Printf("Error cleaning up temp file: %v", err2)
		}
		return "", err
	}

	return tempFile.Name(), nil
}

func downloadGPGKey(url, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		if err2 := resp.Body.Close(); err2 != nil {
			log.Printf("Error closing response body: %v", err2)
		}
	}()
	dearm, err := armor.Parse(resp.Body)
	if err != nil {
		return err
	}
	err = os.WriteFile(destPath, dearm, 0644)
	return err
}

var okFileCharsRegex = regexp.MustCompile(`[^a-zA-Z0-9-_]+`)

func sanitizeFilename(s string) string {
	s = okFileCharsRegex.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if len(s) > 50 {
		s = s[:50]
	}
	return strings.ToLower(s)
}

func addPinPreference(pin aptfile.PinDirective, dryRun bool) error {
	var pinValue string
	if pin.Version != "" {
		pinValue = fmt.Sprintf("version %s", pin.Version)
	} else if pin.Release != "" {
		pinValue = fmt.Sprintf("release %s", pin.Release)
	} else if pin.Origin != "" {
		pinValue = fmt.Sprintf("origin %s", pin.Origin)
	}
	basename := sanitizeFilename(pin.PackageName)
	if len(basename) == 0 {
		basename = sanitizeFilename(pinValue)
	}
	pinFile := fmt.Sprintf("/etc/apt/preferences.d/%s.pin", basename)
	content := fmt.Sprintf("Package: %s\nPin-Priority: %d\nPin: %s\n", pin.PackageName, pin.Priority, pinValue)
	if dryRun {
		fmt.Printf("[dry-run] Would write pin file \"%s\"\n", pinFile)
		return nil
	} else {
		return os.WriteFile(pinFile, []byte(content), 0644)
	}
}

func addHold(hold aptfile.HoldDirective, dryRun bool) error {
	if dryRun {
		fmt.Printf("[dry-run] Would run `apt-mark hold %s`\n", hold.PackageName)
		return nil
	} else {
		fixCmd := exec.Command("apt-mark", "hold", hold.PackageName)
		fixCmd.Stdout = os.Stdout
		fixCmd.Stderr = os.Stderr
		return fixCmd.Run()
	}
}
