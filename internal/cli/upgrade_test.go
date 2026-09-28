package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/amisonnet8/mtqg/internal/journal"
)

func (h *harness) schemaPath() string {
	return filepath.Join(h.root, ".mtqg", "SCHEMA.md")
}

func (h *harness) writeSchema(content string) {
	h.t.Helper()
	if err := os.WriteFile(h.schemaPath(), []byte(content), 0o600); err != nil {
		h.t.Fatal(err)
	}
}

func readOSFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestUpgradeUnchanged(t *testing.T) {
	h := initialized(t)
	code, out, errOut := h.run("upgrade")
	wantExit(t, code, 0, out, errOut)
	if want := "Unchanged: format 1\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

func TestUpgradeRaisesTheFormat(t *testing.T) {
	h := initialized(t)
	if err := os.WriteFile(filepath.Join(h.root, ".mtqg", "version"), []byte("0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	h.writeSchema("old schema, no marker")

	code, out, errOut := h.run("upgrade")
	wantExit(t, code, 0, out, errOut)
	if want := "Upgraded: format 0 -> 1\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
	if got := readOSFile(t, filepath.Join(h.root, ".mtqg", "version")); got != "1\n" {
		t.Errorf("version = %q, want \"1\\n\"", got)
	}
}

func TestUpgradeRewritesSchemaWithoutRaisingTheFormat(t *testing.T) {
	h := initialized(t)
	h.writeSchema("stale content\n<!-- schema as of mtqg 0.9.0 -->\n")

	code, out, errOut := h.run("upgrade")
	wantExit(t, code, 0, out, errOut)
	if want := "Updated: SCHEMA.md (format 1 unchanged)\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
	if data, err := os.ReadFile(h.schemaPath()); err != nil || string(data) == "stale content\n<!-- schema as of mtqg 0.9.0 -->\n" {
		t.Errorf("SCHEMA.md was not rewritten: %q, %v", data, err)
	}
	if got := readOSFile(t, filepath.Join(h.root, ".mtqg", "version")); got != "1\n" {
		t.Errorf("version changed: %q", got)
	}
}

func TestUpgradeDryRunOfSchemaOnlyChange(t *testing.T) {
	h := initialized(t)
	stale := "stale content\n<!-- schema as of mtqg 0.9.0 -->\n"
	h.writeSchema(stale)

	code, out, errOut := h.run("upgrade", "-n")
	wantExit(t, code, 0, out, errOut)
	if want := "Updated (dry run): SCHEMA.md (format 1 unchanged)\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
	if data, err := os.ReadFile(h.schemaPath()); err != nil || string(data) != stale {
		t.Errorf("SCHEMA.md was written despite -n: %q, %v", data, err)
	}
}

func TestUpgradeJSON(t *testing.T) {
	t.Run("nothing to do", func(t *testing.T) {
		h := initialized(t)
		obj := jsonObject(t, mustRun(h, "--json", "upgrade"))
		if obj["command"] != "upgrade" || field(t, obj, "from") != float64(1) || field(t, obj, "to") != float64(1) ||
			field(t, obj, "schema_updated") != false || field(t, obj, "dry_run") != false {
			t.Errorf("%v", obj)
		}
	})

	t.Run("schema only", func(t *testing.T) {
		h := initialized(t)
		h.writeSchema("stale\n<!-- schema as of mtqg 0.9.0 -->\n")
		obj := jsonObject(t, mustRun(h, "--json", "upgrade"))
		if field(t, obj, "from") != float64(1) || field(t, obj, "to") != float64(1) ||
			field(t, obj, "schema_updated") != true {
			t.Errorf("%v", obj)
		}
	})

	t.Run("format raised", func(t *testing.T) {
		h := initialized(t)
		if err := os.WriteFile(filepath.Join(h.root, ".mtqg", "version"), []byte("0\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		obj := jsonObject(t, mustRun(h, "--json", "upgrade"))
		if field(t, obj, "from") != float64(0) || field(t, obj, "to") != float64(journal.SupportedVersion) ||
			field(t, obj, "schema_updated") != true {
			t.Errorf("%v", obj)
		}
	})
}
