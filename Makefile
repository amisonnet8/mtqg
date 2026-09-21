# Entry points for building and checking mtqg (.claude/rules/testing.md).
# The recipes assume a POSIX shell (Git Bash on Windows).

.PHONY: build fmt vet lint unit check test docs-examples race trivy shellcheck

# Compile every package first: a package that cmd/mtqg does not import yet
# would otherwise be skipped, and so would its build errors.
build:
	go build ./...
	go build ./cmd/mtqg

# Rewrite files with the formatters enabled in .golangci.yaml.
fmt:
	golangci-lint fmt

vet:
	go vet ./...
	go vet -tags e2e ./e2e/...

# Also reports files that the formatters would change.
lint:
	golangci-lint run

unit:
	go test ./...

check: vet lint unit

# End-to-end tests: the real mtqg binary against real git repositories (e2e/,
# built with the tag e2e, so make check does not run them).
test:
	go test -tags e2e -count=1 ./e2e/...

# Write what mtqg prints into the examples of docs/reference/ (e2e/examples_test.go).
# Read the diff: an example that changed is a document that went stale, or a bug.
docs-examples:
	go test -tags e2e -count=1 -run '^TestDocExamples$$' ./e2e/... -update

# -race needs cgo, which the container turns off (.claude/rules/testing.md).
race:
	CGO_ENABLED=1 go test -race -count=1 ./...

trivy:
	trivy fs --config trivy.yaml .

shellcheck:
	git ls-files '*.sh' | xargs -r shellcheck
