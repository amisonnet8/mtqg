package journal

import "testing"

func TestSchemaMarker(t *testing.T) {
	cases := []struct {
		name   string
		text   string
		want   string
		wantOK bool
	}{
		{"present", "some text\n<!-- schema as of mtqg 1.1.0 -->\nmore text", "1.1.0", true},
		{"absent", "some text with no marker", "", false},
		{"malformed: missing a component", "<!-- schema as of mtqg 1.1 -->", "", false},
		{"malformed: not numeric", "<!-- schema as of mtqg a.b.c -->", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := SchemaMarker(c.text)
			if got != c.want || ok != c.wantOK {
				t.Errorf("SchemaMarker(%q) = %q, %v, want %q, %v", c.text, got, ok, c.want, c.wantOK)
			}
		})
	}
}

func TestSchemaOlder(t *testing.T) {
	cases := []struct {
		name string
		a, b string
		want bool
	}{
		{"equal", "1.1.0", "1.1.0", false},
		{"older patch", "1.1.0", "1.1.1", true},
		{"newer patch", "1.1.1", "1.1.0", false},
		{"numeric, not lexical: 1.9.0 is older than 1.10.0", "1.9.0", "1.10.0", true},
		{"numeric, not lexical: 1.10.0 is not older than 1.9.0", "1.10.0", "1.9.0", false},
		{"older major", "1.9.0", "2.0.0", true},
		{"unparseable a is treated as oldest", "not-a-version", "1.0.0", true},
		{"unparseable b: a is not older", "1.0.0", "not-a-version", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := schemaOlder(c.a, c.b); got != c.want {
				t.Errorf("schemaOlder(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
			}
		})
	}
}

func TestSchemaStale(t *testing.T) {
	cases := []struct {
		name    string
		current []byte
		want    bool
	}{
		{"nil (no existing SCHEMA.md)", nil, true},
		{"no marker", []byte("plain text"), true},
		{"current marker", []byte("text\n<!-- schema as of mtqg " + SchemaVersion + " -->\n"), false},
		{"older marker", []byte("<!-- schema as of mtqg 0.0.1 -->"), true},
		{"newer marker", []byte("<!-- schema as of mtqg 9.9.9 -->"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := schemaStale(c.current); got != c.want {
				t.Errorf("schemaStale(%q) = %v, want %v", c.current, got, c.want)
			}
		})
	}
}
