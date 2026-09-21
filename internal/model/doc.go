// Package model gives the events of the journal their meaning.
//
// It builds the state of records from events, checks what may be done to a
// record (a memo cannot be done), resolves the IDs people type, and puts
// together the events that create a record or change its state. It never
// touches a file: it is given events, and hands events back to be written by
// the journal layer (.claude/rules/directory-structure.md). It also knows
// nothing about how anything is shown; the entry points do that.
package model
