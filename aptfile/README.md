# adapt - a declarative apt package management tool

This is a small wrapper around standard `apt` tools like `apt-get` and `add-apt-repository` to
make repository and package installation more declarative. This makes things like installing
apt packages into VM or Docker images much simpler, more readable, and less error prone. In
addition, it has minimal dependencies, so you do not need to install curl, gpg, or ca-certificates
before using this tool, unlike with shell scripts.

## Installation

Download the appropriate release binary from the Github releases page. If one is not available
for your platform, you can build the binary easily from this repo. This repo has minimal external
dependencies.

## Usage

To use, create an `Aptfile` with directives, one per line:

```
# Use '#' for comments
package "curl"
# Use = and / syntax for installing specific versions or releases
package "ffmpeg=7:6.1.1-3ubuntu5"
package "ffmpeg/noble"

# Add new apt repo sources
repo "https://cli.github.com/packages" "stable" "main", arch: "amd64", signed-by: "https://cli.github.com/packages/githubcli-archive-keyring.gpg"
package "gh"

# You can also use Ubuntu PPAs sources
ppa "fish-shell/release-3"
package "fish"

# Mark a package to hold to the current version and prevent upgrades
hold "ffmpeg"

# Use pins to control package source selection
pin "*" 600, release: "l=NVIDIA CUDA"
```

