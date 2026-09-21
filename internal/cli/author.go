package cli

import (
	"errors"
	"strings"

	"github.com/amisonnet8/mtqg/internal/journal"
)

// author says who is writing: MTQG_AUTHOR_KIND (human or ai; human when it is
// not set) and MTQG_AUTHOR_NAME, and if there is no name, the user.name that git
// has for the repository at root. An AI is never recorded under the name from
// git: with MTQG_AUTHOR_KIND=ai the name must be given.
func (c *ctx) author(root string) (journal.Author, error) {
	kind := strings.ToLower(strings.TrimSpace(c.env.Getenv("MTQG_AUTHOR_KIND")))
	if kind == "" {
		kind = journal.AuthorHuman
	}
	if kind != journal.AuthorHuman && kind != journal.AuthorAI {
		return journal.Author{}, &failure{msgBadAuthorKind(c.env.Getenv("MTQG_AUTHOR_KIND"))}
	}

	name := strings.TrimSpace(c.env.Getenv("MTQG_AUTHOR_NAME"))
	if name == "" && kind == journal.AuthorAI {
		return journal.Author{}, &failure{msgAIneedsName()}
	}
	if name == "" {
		fromGit, err := journal.GitUserName(root)
		switch {
		case errors.Is(err, journal.ErrNoUserName):
			return journal.Author{}, &failure{msgNoUserName()}
		case err != nil:
			return journal.Author{}, err
		}
		name = fromGit
	}
	return journal.Author{Kind: kind, Name: name}, nil
}
