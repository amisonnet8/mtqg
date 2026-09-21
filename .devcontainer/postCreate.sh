#!/usr/bin/env bash
set -euo pipefail

# make:           Build and test entry points (Makefile).
# wget, gnupg,
# lsb-release:    Adding the Trivy and GitHub CLI apt repositories below.
# gcc:            make race (CGO_ENABLED=1 go test -race) needs a C compiler.
#                 The container itself runs with CGO_ENABLED=0 (.claude/rules/distribution.md).
# jq:             Inspecting journal.jsonl and --json output while debugging.
# ShellCheck:     Static analysis of tracked *.sh and *.bash files (make shellcheck, .claude/rules/testing.md).
#                 Comment lines must not start with the lowercase directive word,
#                 or ShellCheck parses them as directives (SC1072/SC1073).
# zsh, fish:      Two of the four shells that mtqg completion has a script for. The scripts
#                 are run by the real shells in e2e/completion_test.go (bash is there already,
#                 and PowerShell is installed below).
sudo apt-get update
sudo apt-get install -y make wget gnupg lsb-release gcc jq shellcheck zsh fish

# Trivy: known vulnerabilities (CVE) and license compatibility of dependencies
# (make trivy, .claude/rules/testing.md). Installed from the official apt repository.
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

# golangci-lint: lint (make check, .golangci.yaml). The official install script
# puts the binary into GOPATH/bin. The version is pinned so that lint results
# do not change when the container is rebuilt; .golangci.yaml was verified with it.
wget -qO - https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b "$(go env GOPATH)/bin" v2.13.2

go install golang.org/x/tools/gopls@latest
go install golang.org/x/tools/cmd/goimports@latest
