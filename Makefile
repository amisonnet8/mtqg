# Entry points for building and checking mtqg (.claude/rules/testing.md).
# The recipes assume a POSIX shell (Git Bash on Windows).

.PHONY: build fmt vet lint unit check test race trivy shellcheck

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

# Also reports files that the formatters would change.
lint:
	golangci-lint run

unit:
	go test ./...

check: vet lint unit

# End-to-end tests. There are none yet (PLAN.md, Step 8).
test:
	@echo "make test: no end-to-end tests yet (PLAN.md, Step 8)" >&2

# -race needs cgo, which the container turns off (.claude/rules/testing.md).
race:
	CGO_ENABLED=1 go test -race -count=1 ./...

trivy:
	trivy fs --config trivy.yaml .

shellcheck:
	git ls-files '*.sh' | xargs -r shellcheck
