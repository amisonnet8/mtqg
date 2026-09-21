package model

import "strings"

// Search returns the records in view whose text contains query, oldest first. A
// glossary entry is found by its word as well. Case is ignored and nothing else
// is: no patterns, no word boundaries. Only what is in view is searched, so a
// deleted record is never found.
func (s *State) Search(query string) []*Record {
	q := strings.ToLower(query)
	return s.pick(func(r *Record) bool {
		return strings.Contains(strings.ToLower(r.Text), q) ||
			(r.Word != "" && strings.Contains(strings.ToLower(r.Word), q))
	})
}
