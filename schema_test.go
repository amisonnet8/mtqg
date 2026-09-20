package mtqg_test

import (
	"strings"
	"testing"

	"github.com/amisonnet8/mtqg"
)

func TestSchemaIsEmbedded(t *testing.T) {
	if !strings.HasPrefix(mtqg.Schema, "# mtqg journal format") {
		t.Errorf("Schema does not start with the title of schema.md: %.40q", mtqg.Schema)
	}
}
