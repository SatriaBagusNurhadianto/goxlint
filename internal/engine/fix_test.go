package engine

import (
	"testing"

	"github.com/egoisutolabs/goxlint"
)

func TestApplyFixesRejectsInvalidEdits(t *testing.T) {
	for _, tc := range []struct {
		name  string
		edits []Edit
	}{
		{name: "negative start", edits: []Edit{{Start: -1, End: 1}}},
		{name: "past end", edits: []Edit{{Start: 0, End: 7}}},
		{name: "reversed", edits: []Edit{{Start: 3, End: 1}}},
		{name: "overlapping replacements", edits: []Edit{{Start: 0, End: 3}, {Start: 2, End: 4}}},
		{name: "duplicate insertions", edits: []Edit{{Start: 2, End: 2}, {Start: 2, End: 2}}},
		{name: "insertion inside replacement", edits: []Edit{{Start: 0, End: 3}, {Start: 2, End: 2}}},
		{name: "invalid after valid", edits: []Edit{{Start: 0, End: 1}, {Start: 3, End: 2}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			diags := []Diagnostic{
				{RuleID: "invalid", Fix: &Fix{Kind: goxlint.FixSafe, Edits: tc.edits}},
				{RuleID: "valid", Fix: &Fix{Kind: goxlint.FixSafe, Edits: []Edit{{Start: 0, End: 1, NewText: "A"}}}},
			}
			out, applied := ApplyFixes([]byte("012345"), diags, goxlint.FixSafe)
			if string(out) != "A12345" || len(applied) != 1 || applied[0].RuleID != "valid" {
				t.Fatalf("want only valid fix applied, got %q, %+v", out, applied)
			}
		})
	}
}

func TestApplyFixesMultipleEdits(t *testing.T) {
	diags := []Diagnostic{{Fix: &Fix{Kind: goxlint.FixSafe, Edits: []Edit{
		{Start: 6, End: 6, NewText: "!"},
		{Start: 2, End: 4, NewText: "B"},
		{Start: 0, End: 2, NewText: "A"},
	}}}}
	out, applied := ApplyFixes([]byte("012345"), diags, goxlint.FixSafe)
	if string(out) != "AB45!" || len(applied) != 1 {
		t.Fatalf("want adjacent and out-of-order edits applied together, got %q, %+v", out, applied)
	}
}
