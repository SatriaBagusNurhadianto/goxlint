package engine

import (
	"sort"

	"github.com/egoisutolabs/goose-lint/goose"
)

// ApplyFixes applies the fixes in diags whose kind is at most maxKind to src.
// A fix is skipped in its entirety if any edit is out of bounds, overlaps another
// edit in the same fix, or overlaps an already-applied fix. It returns the new
// source and the diagnostics whose fixes were applied.
func ApplyFixes(src []byte, diags []Diagnostic, maxKind goose.FixKind) ([]byte, []Diagnostic) {
	var candidates []Diagnostic
	for _, d := range diags {
		if d.Fix != nil && d.Fix.Kind >= goose.FixSafe && d.Fix.Kind <= goose.FixDangerous && d.Fix.Kind <= maxKind && len(d.Fix.Edits) > 0 {
			candidates = append(candidates, d)
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return fixStart(candidates[i].Fix) < fixStart(candidates[j].Fix)
	})

	type span struct{ start, end int }
	var taken []span
	overlaps := func(s span, spans []span) bool {
		for _, t := range spans {
			if s.start < t.end && t.start < s.end || s.start == t.start {
				return true
			}
		}
		return false
	}

	var edits []Edit
	var applied []Diagnostic
	for _, d := range candidates {
		var spans []span
		ok := true
		for _, e := range d.Fix.Edits {
			s := span{e.Start, e.End}
			if e.Start < 0 || e.End < e.Start || e.End > len(src) || overlaps(s, taken) || overlaps(s, spans) {
				ok = false
				break
			}
			spans = append(spans, s)
		}
		if !ok {
			continue
		}
		taken = append(taken, spans...)
		edits = append(edits, d.Fix.Edits...)
		applied = append(applied, d)
	}
	if len(edits) == 0 {
		return src, nil
	}

	sort.SliceStable(edits, func(i, j int) bool { return edits[i].Start < edits[j].Start })
	out := make([]byte, 0, len(src))
	last := 0
	for _, e := range edits {
		out = append(out, src[last:e.Start]...)
		out = append(out, e.NewText...)
		last = e.End
	}
	out = append(out, src[last:]...)
	return out, applied
}

func fixStart(f *Fix) int {
	start := f.Edits[0].Start
	for _, e := range f.Edits[1:] {
		start = min(start, e.Start)
	}
	return start
}
