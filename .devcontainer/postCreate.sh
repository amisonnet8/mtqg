#!/usr/bin/env bash
set -euo pipefail

# wget, gnupg,
# lsb-release:    Adding the Trivy and GitHub CLI apt repositories below.
# gcc:            qsoku race (CGO_ENABLED=1 go test -race) needs a C compiler.
#                 The container itself runs with CGO_ENABLED=0 (.claude/rules/distribution.md).
# jq:             Inspecting journal.jsonl and --json output while debugging.
# ShellCheck:     Static analysis of tracked *.sh and *.bash files (qsoku shellcheck, .claude/rules/testing.md).
#                 Comment lines must not start with the lowercase directive word,
#                 or ShellCheck parses them as directives (SC1072/SC1073).
# zsh, fish:      Two of the four shells that mtqg completion has a script for (the scripts
#                 are run by the real shells in e2e/completion_test.go; bash is there already,
#                 and PowerShell is installed below), and two of the three shells qsoku's own
#                 shell integration supports (bash, zsh, fish -- not PowerShell, since qsokufile
#                 commands always run under sh).
sudo apt-get update
sudo apt-get install -y wget gnupg lsb-release gcc jq shellcheck zsh fish

# Trivy: known vulnerabilities (CVE) and license compatibility of dependencies
# (qsoku trivy, .claude/rules/testing.md). Installed from the official apt repository.
wget -qO - https://aquasecurity.github.io/trivy-repo/deb/public.key | gpg --dearmor | sudo tee /usr/share/keyrings/trivy.gpg >/dev/null
echo "deb [signed-by=/usr/share/keyrings/trivy.gpg] https://aquasecurity.github.io/trivy-repo/deb $(lsb_release -sc) main" | sudo tee /etc/apt/sources.list.d/trivy.list >/dev/null
sudo apt-get update
sudo apt-get install -y trivy

# PowerShell (pwsh): the fourth shell of the completion scripts (e2e/completion_test.go).
# Installed from the Microsoft apt repository (same pattern as Trivy).
wget -qO - https://packages.microsoft.com/keys/microsoft.asc | gpg --dearmor | sudo tee /usr/share/keyrings/microsoft-prod.gpg >/dev/null
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/microsoft-prod.gpg] https://packages.microsoft.com/debian/$(lsb_release -rs)/prod $(lsb_release -sc) main" | sudo tee /etc/apt/sources.list.d/microsoft-prod.list >/dev/null
sudo apt-get update
sudo apt-get install -y powershell

# gh: GitHub CLI, for checking issues, pull requests and Actions runs.
# Installed from the official apt repository (same pattern as Trivy).
sudo mkdir -p -m 755 /etc/apt/keyrings
wget -qO - https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo tee /etc/apt/keyrings/githubcli-archive-keyring.gpg >/dev/null
sudo chmod go+r /etc/apt/keyrings/githubcli-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | sudo tee /etc/apt/sources.list.d/github-cli.list >/dev/null
sudo apt-get update
sudo apt-get install -y gh

# golangci-lint: lint (qsoku check, .golangci.yaml). The official install script
# puts the binary into GOPATH/bin. The version is pinned so that lint results
# do not change when the container is rebuilt; .golangci.yaml was verified with it.
wget -qO - https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b "$(go env GOPATH)/bin" v2.13.2

go install golang.org/x/tools/gopls@latest
go install golang.org/x/tools/cmd/goimports@latest

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
# completion) for bash, zsh and fish. Idempotent: skipped if already present,
# so re-running postCreate.sh does not duplicate the line. The single quotes
# are intentional -- the line is meant to land in the rc file unexpanded.
# shellcheck disable=SC2016
grep -qF 'qsoku .shell bash' ~/.bashrc 2>/dev/null || echo 'eval "$(qsoku .shell bash)"' >>~/.bashrc
# shellcheck disable=SC2016
grep -qF 'qsoku .shell zsh' ~/.zshrc 2>/dev/null || echo 'eval "$(qsoku .shell zsh)"' >>~/.zshrc
mkdir -p ~/.config/fish
grep -qF 'qsoku .shell fish' ~/.config/fish/config.fish 2>/dev/null || echo 'qsoku .shell fish | source' >>~/.config/fish/config.fish

# mtqg's own bash completion, for interactive use in this container (mtqg
# completion <shell>, docs/reference/cli.md).
mkdir -p ~/.local/share/bash-completion/completions
mtqg completion bash >~/.local/share/bash-completion/completions/mtqg
