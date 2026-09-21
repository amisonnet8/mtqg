package cli

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// inputText gives the text of a record: the words joined with spaces, or all of
// standard input when the only word is "-", or what is written in $EDITOR when
// there are no words. Trailing line breaks are dropped. A text that is empty, or
// only white space, is refused.
func (c *ctx) inputText(words []string) (string, error) {
	var text string
	switch {
	case len(words) == 1 && words[0] == "-":
		data, err := io.ReadAll(c.env.Stdin)
		if err != nil {
			return "", &failure{msgStdinFailed(err)}
		}
		text = string(data)
	case len(words) == 0:
		edited, err := c.editText()
		if err != nil {
			return "", err
		}
		text = edited
	default:
		text = strings.Join(words, " ")
	}
	text = strings.TrimRight(text, "\r\n")
	if strings.TrimSpace(text) == "" {
		return "", &failure{msgEmptyText()}
	}
	return text, nil
}

// editText opens $EDITOR on an empty file and returns what was saved.
func (c *ctx) editText() (string, error) {
	editor := strings.TrimSpace(c.env.Getenv("EDITOR"))
	if editor == "" {
		return "", &failure{msgNoEditor()}
	}
	argv, err := splitCommand(editor)
	if err != nil {
		return "", &failure{msgBadEditorCommand(err.Error())}
	}

	file, err := os.CreateTemp("", "mtqg-*.txt")
	if err != nil {
		return "", err
	}
	name := filepath.Clean(file.Name())
	if err := file.Close(); err != nil {
		return "", err
	}
	defer func() { _ = os.Remove(name) }()

	if err := c.env.RunEditor(append(argv, name)); err != nil {
		return "", &failure{msgEditorFailed(err)}
	}
	// The file is read again by name: many editors save by writing a new file.
	data, err := os.ReadFile(name)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// splitCommand splits a command line, such as the value of $EDITOR, into the
// command and its arguments. White space separates them; a part in single or
// double quotes is one word, quotes gone. Nothing else is special, so a Windows
// path such as "C:\Program Files\Editor\ed.exe" needs quotes and no escaping. It
// is not run through a shell.
func splitCommand(line string) ([]string, error) {
	var (
		words  []string
		word   strings.Builder
		inWord bool
		quote  rune
	)
	for _, r := range line {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				word.WriteRune(r)
			}
		case r == '"' || r == '\'':
			quote, inWord = r, true
		case unicode.IsSpace(r):
			if inWord {
				words = append(words, word.String())
				word.Reset()
				inWord = false
			}
		default:
			word.WriteRune(r)
			inWord = true
		}
	}
	if quote != 0 {
		return nil, errors.New("a quote is not closed")
	}
	if inWord {
		words = append(words, word.String())
	}
	if len(words) == 0 {
		return nil, errors.New("it is empty")
	}
	return words, nil
}
