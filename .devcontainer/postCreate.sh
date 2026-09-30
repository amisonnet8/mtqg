#!/usr/bin/env bash
set -euo pipefail

# wget, gnupg,
# lsb-release:    Adding the Trivy apt repository below.
# gcc:            qsoku race (CGO_ENABLED=1 go test -race) needs a C compiler.
#                 The container itself runs with CGO_ENABLED=0 (.claude/rules/distribution.md).
# jq:             Inspecting journal.jsonl and --json output while debugging.
# ShellCheck:     Static analysis of tracked *.sh and *.bash files (qsoku shellcheck, .claude/rules/testing.md).
#                 Comment lines must not start with the lowercase directive word,
#                 or ShellCheck parses them as directives (SC1072/SC1073).
# fish:           One of the four shells that mtqg completion has a script for (the scripts
#                 are run by the real shells in e2e/completion_test.go), and one of the four
#                 shells qsoku's own shell integration supports (bash, zsh, fish, pwsh -- qsoku
#                 v0.2.0 added pwsh; qsokufile commands themselves still always run under sh
#                 regardless). The other three come from elsewhere: bash and zsh are in the
#                 base image, and PowerShell is the "powershell" feature in devcontainer.json.
# gh:             GitHub CLI (issues, pull requests, Actions runs) is the "github-cli" feature
#                 in devcontainer.json, not installed here. Both features are left unpinned
#                 (no "version" option): each rebuild installs the latest release, as the
#                 apt installs did before (decision, 2026-10-01).
sudo apt-get update
sudo apt-get install -y wget gnupg lsb-release gcc jq shellcheck fish

# Trivy: known vulnerabilities (CVE) and license compatibility of dependencies
# (qsoku trivy, .claude/rules/testing.md). Installed from the official apt repository.
wget -qO - https://aquasecurity.github.io/trivy-repo/deb/public.key | gpg --dearmor | sudo tee /usr/share/keyrings/trivy.gpg >/dev/null
echo "deb [signed-by=/usr/share/keyrings/trivy.gpg] https://aquasecurity.github.io/trivy-repo/deb $(lsb_release -sc) main" | sudo tee /etc/apt/sources.list.d/trivy.list >/dev/null
sudo apt-get update
sudo apt-get install -y trivy

# golangci-lint: lint (qsoku check, .golangci.yaml). The official install script
# puts the binary into GOPATH/bin. The version is pinned so that lint results
# do not change when the container is rebuilt; .golangci.yaml was verified with it.
wget -qO - https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b "$(go env GOPATH)/bin" v2.13.2

go install golang.org/x/tools/gopls@latest
go install golang.org/x/tools/cmd/goimports@latest

# goreleaser: builds each OS's binary from .goreleaser.yaml for a release
# (.claude/rules/distribution.md). @latest, not pinned, to match the CI
# workflow's own loose "~> v2" version constraint (.github/workflows/release.yml)
# rather than risking `goreleaser check` behaving differently here than there.
go install github.com/goreleaser/goreleaser/v2@latest

# qsoku: build/check/test entry points (qsokufile, replaces the former Makefile;
# see docs/design/history.md 2026-09-23). @latest, not pinned, while qsoku
# itself is still moving fast (decision, 2026-09-26): a regression is caught
# locally right away instead of silently, which is an acceptable trade while
# development is active on both sides. Re-pin to a specific release once
# mtqg's own development settles down (CI, below, is pinned the same way).
go install github.com/amisonnet8/qsoku/cmd/qsoku@latest

# mtqg: build from this repository's own source, so a fresh devcontainer has
# a working `mtqg` command (and its completion, below) without a manual step.
# This is the one place mtqg is installed from source rather than a tagged
# release -- see .claude/rules/mtqg-usage.md for why the *records* this
# container writes use a separately reinstalled, known-good build instead.
go install ./cmd/mtqg

# Wire up qsoku's shell integration (working-directory carry-back and
# completion) for bash, zsh, fish and pwsh. Idempotent: skipped if already
# present, so re-running postCreate.sh does not duplicate the line. The single
# quotes are intentional -- the line is meant to land in the rc file unexpanded.
# shellcheck disable=SC2016
grep -qF 'qsoku .shell bash' ~/.bashrc 2>/dev/null || echo 'eval "$(qsoku .shell bash)"' >>~/.bashrc
# shellcheck disable=SC2016
grep -qF 'qsoku .shell zsh' ~/.zshrc 2>/dev/null || echo 'eval "$(qsoku .shell zsh)"' >>~/.zshrc
mkdir -p ~/.config/fish
grep -qF 'qsoku .shell fish' ~/.config/fish/config.fish 2>/dev/null || echo 'qsoku .shell fish | source' >>~/.config/fish/config.fish
# pwsh has no $PROFILE of its own until asked; query it rather than hardcode
# the path (qsoku v0.2.0, cli.md "Shell integration"). The single quotes below
# are intentional too -- $PROFILE is pwsh's variable, not bash's.
# shellcheck disable=SC2016
pwsh_profile=$(pwsh -NoLogo -NoProfile -Command '$PROFILE')
mkdir -p "$(dirname "$pwsh_profile")"
grep -qF 'qsoku .shell pwsh' "$pwsh_profile" 2>/dev/null || echo 'Invoke-Expression (& qsoku .shell pwsh | Out-String)' >>"$pwsh_profile"

# mtqg's own completion, for interactive use in this container, for all four
# shells it has a script for (mtqg completion <shell>, docs/reference/cli.md
# "Shell completion"). Each install path is the one that section names.
mkdir -p ~/.local/share/bash-completion/completions
mtqg completion bash >~/.local/share/bash-completion/completions/mtqg

# zsh's fpath[1] in this image is a root-owned system directory, unlike
# bash's and fish's user-local completion directories above and below.
zsh_fpath=$(zsh -c 'echo $fpath[1]')
sudo mkdir -p "$zsh_fpath"
mtqg completion zsh | sudo tee "$zsh_fpath/_mtqg" >/dev/null

mkdir -p ~/.config/fish/completions
mtqg completion fish >~/.config/fish/completions/mtqg.fish

# pwsh has no separate completions directory: the script is meant to be
# appended to $PROFILE (docs/reference/cli.md), so re-running this idempotently
# needs the same marker-line check as the qsoku pwsh integration above.
grep -qF 'Register-ArgumentCompleter -Native -CommandName mtqg' "$pwsh_profile" 2>/dev/null || mtqg completion powershell >>"$pwsh_profile"
