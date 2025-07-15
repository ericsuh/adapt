package aptfile

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type DirectiveLine struct {
	Type    string
	Args    []string
	Options map[string]string
}

type PackageDirective struct {
	Name    string
	Version string
	Release string
}

type PinDirective struct {
	Priority    int32
	PackageName string
	// A specific version or pattern
	Version string

	// A specific origin or pattern. "" is localhost
	// like origin *ubuntu.com*
	Origin string

	// Release specifiers
	// These look like "a=Debian l=Nvidia"
	Release string
}

type PpaDirective struct {
	Name string
}

type RepoDirective struct {
	IsSrc     bool
	Arch      string
	SignedBy  string
	Suite     string
	Component string
	URL       string
}

type DebFileDirective struct {
	Path string
}

type HoldDirective struct {
	PackageName string
}

var (
	ErrNoDirective = errors.New("no directive found")
	ErrParsing     = errors.New("error parsing aptfile")
)

func Parse(r io.Reader) ([]any, error) {
	s := bufio.NewScanner(r)
	result := make([]any, 0)
	lineNum := 0
	for s.Scan() {
		dir, err := ParseLine(lineNum, s.Text())
		lineNum += 1
		if err == ErrNoDirective {
			continue
		} else if err != nil {
			return []any{}, err
		}
		result = append(result, dir)
	}
	if err := s.Err(); err != nil {
		return []any{}, err
	}
	return result, nil
}

// Parse a generic directive of the form `command arg1 arg2 "arg3", key1: "val1", key2: "val2"`
func ParseLine(lineNum int, line string) (any, error) {
	toks, err := lexLine(FileCoord{Line: line, LineNum: lineNum})
	if err != nil {
		return nil, err
	}
	if len(toks) == 0 {
		return nil, ErrNoDirective
	}
	if toks[0].Type != StringToken {
		return nil, ParseError{
			Message: "expected string value",
			Coord:   toks[0].Coord,
		}
	}
	cmd := toks[0].Text()
	args := make([]string, 0)
	opts := make(map[string]string, 0)
	optsPhase := false
	curr := 1
	for curr < len(toks) {
		if optsPhase {
			if curr+3 >= len(toks) {
				return DirectiveLine{}, ParseError{
					Message: "expected option, got end of line",
					Coord: FileCoord{
						Line:     line,
						LineNum:  lineNum,
						ColStart: toks[curr].Coord.ColStart,
						ColEnd:   toks[len(toks)-1].Coord.ColEnd,
					},
				}
			}
			if toks[curr].Type == CommaToken &&
				toks[curr+1].Type == StringToken &&
				toks[curr+2].Type == ColonToken &&
				toks[curr+3].Type == StringToken {
				opts[toks[curr+1].Text()] = toks[curr+3].Text()
				curr += 4
			} else {
				return DirectiveLine{}, ParseError{
					Message: "expected key-value pair",
					Coord: FileCoord{
						Line:     line,
						LineNum:  lineNum,
						ColStart: toks[curr].Coord.ColStart,
						ColEnd:   toks[min(curr+3, len(toks)-1)].Coord.ColEnd,
					},
				}
			}
		} else if toks[curr].Type == CommaToken {
			optsPhase = true
			// Don't advance
		} else if toks[curr].Type == ColonToken {
			return DirectiveLine{}, ParseError{
				Message: "unexpected colon",
				Coord:   toks[curr].Coord,
			}
		} else {
			args = append(args, toks[curr].Text())
			curr += 1
		}
	}
	switch cmd {
	case "repo", "repo-src":
		return parseRepoDirective(cmd, args, opts)
	case "package":
		return parsePackageDirective(cmd, args, opts)
	case "deb":
		return parseDebFileDirective(cmd, args, opts)
	case "ppa":
		return parsePpaDirective(cmd, args, opts)
	case "pin":
		return parsePinDirective(cmd, args, opts)
	case "hold":
		return parseHoldDirective(cmd, args, opts)
	default:
		return nil, fmt.Errorf(`unexpected directive "%s"`, cmd)
	}
}

func parsePackageDirective(_ string, args []string, opts map[string]string) (PackageDirective, error) {
	if len(args) != 1 {
		return PackageDirective{}, fmt.Errorf("expected only one argument, got %v", args)
	}
	name := args[0]
	for k, v := range opts {
		switch k {
		case "version":
			return PackageDirective{
				Name:    name,
				Version: v,
			}, nil
		case "release":
			return PackageDirective{
				Name:    name,
				Release: v,
			}, nil
		default:
			return PackageDirective{}, fmt.Errorf("unknown package option: %s", k)
		}
	}
	pieces := strings.SplitN(name, "=", 2)
	if len(pieces) == 2 {
		return PackageDirective{
			Name:    pieces[0],
			Version: pieces[1],
		}, nil
	}
	pieces = strings.SplitN(name, "/", 2)
	if len(pieces) == 2 {
		return PackageDirective{
			Name:    pieces[0],
			Release: pieces[1],
		}, nil
	}
	return PackageDirective{Name: name}, nil
}

// pin directives are formatted like, `pin "package1" 333, version: "1.2.3"`
func parsePinDirective(_ string, args []string, opts map[string]string) (PinDirective, error) {
	if len(args) != 2 {
		return PinDirective{}, errors.New("expected two positional arguments")
	}
	pri, err := strconv.ParseInt(args[1], 10, 32)
	if err != nil {
		return PinDirective{}, err
	}
	dir := PinDirective{
		PackageName: args[0],
		Priority:    int32(pri),
	}
	for k, v := range opts {
		switch k {
		case "version":
			dir.Version = v
			return dir, nil
		case "origin":
			dir.Origin = v
			return dir, nil
		case "release":
			dir.Release = v
			return dir, nil
		default:
			return PinDirective{}, fmt.Errorf("unknown pin option %s", k)
		}
	}
	return dir, nil
}

// ppa directives are formatted like, `ppa "fish-shell/fish-3"`
func parsePpaDirective(_ string, args []string, opts map[string]string) (PpaDirective, error) {
	if len(args) != 1 {
		return PpaDirective{}, fmt.Errorf("expected one argument, got %v", args)
	}
	if len(opts) > 0 {
		return PpaDirective{}, fmt.Errorf("unexpected options %v", opts)
	}
	return PpaDirective{
		Name: args[0],
	}, nil
}

// repo directives are formatted like, `repo "http://repo/url" "suite" "component", arch: "amd64", signed-by: "https://url/to/key.gpg`
func parseRepoDirective(cmd string, args []string, opts map[string]string) (RepoDirective, error) {
	if len(args) == 0 {
		return RepoDirective{}, errors.New("expected at least one argument")
	}
	dir := RepoDirective{
		IsSrc: cmd == "repo-src",
		URL:   args[0],
	}
	if len(args) >= 2 {
		dir.Suite = args[1]
	}
	if len(args) >= 3 {
		dir.Component = args[2]
	}
	for k, v := range opts {
		switch k {
		case "arch":
			dir.Arch = v
		case "signed-by":
			dir.SignedBy = v
		default:
			return RepoDirective{}, fmt.Errorf(`unexpected option "%s"`, k)
		}
	}
	return dir, nil
}

// deb file directives are formatted like, `deb "http://url/to/file.deb"`
func parseDebFileDirective(_ string, args []string, opts map[string]string) (DebFileDirective, error) {
	if len(args) != 1 {
		return DebFileDirective{}, fmt.Errorf("expected one argument, got %v", args)
	}
	if len(opts) > 0 {
		return DebFileDirective{}, fmt.Errorf("unexpected options %v", opts)
	}
	return DebFileDirective{Path: args[0]}, nil
}

// hold directives are formatted like, `hold "curl"`
func parseHoldDirective(_ string, args []string, opts map[string]string) (HoldDirective, error) {
	if len(args) != 1 {
		return HoldDirective{}, fmt.Errorf("expected one argument, got %v", args)
	}
	if len(opts) > 0 {
		return HoldDirective{}, fmt.Errorf("unexpected options %v", opts)
	}
	return HoldDirective{
		PackageName: args[0],
	}, nil
}
