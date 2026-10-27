package engine

import (
	"fmt"
	"go/token"
	"strings"
)

// Suppression directives:
//
//	//goose:ignore <rules> [reason]       suppresses its own line and the next line
//	//goose:ignore-file <rules> [reason]  suppresses the whole file
//
// <rules> is a comma-separated list of "plugin/rule", "plugin", or "*".
const (
	ignoreDirective     = "//goose:ignore "
	ignoreFileDirective = "//goose:ignore-file "
)

// UnusedSuppressionRuleID identifies engine findings for unused suppression targets.
const UnusedSuppressionRuleID = "goose/unused-suppression"

type suppression struct {
	line  int // 0 for file-wide
	rules []suppressionTarget
}

type suppressionTarget struct {
	name     string
	pos, end token.Pos
	used     bool
}

func applySuppressions(f File, diags []Diagnostic, unusedSeverity Severity) []Diagnostic {
	var sups []suppression
	for _, cg := range f.AST.Comments {
		for _, c := range cg.List {
			text, line := c.Text, f.Fset.PositionFor(c.Slash, false).Line
			var prefix string
			switch {
			case strings.HasPrefix(text, ignoreFileDirective):
				prefix, line = ignoreFileDirective, 0
			case strings.HasPrefix(text, ignoreDirective):
				prefix = ignoreDirective
			default:
				continue
			}
			rest := text[len(prefix):]
			fields := strings.Fields(rest)
			if len(fields) == 0 {
				continue
			}
			pos := c.Slash + token.Pos(len(prefix)+strings.Index(rest, fields[0]))
			s := suppression{line: line}
			for _, name := range strings.Split(fields[0], ",") {
				s.rules = append(s.rules, suppressionTarget{name: name, pos: pos, end: pos + token.Pos(len(name))})
				pos += token.Pos(len(name) + 1)
			}
			sups = append(sups, s)
		}
	}
	if len(sups) == 0 {
		return diags
	}
	tf := f.Fset.File(f.AST.FileStart)
	out := diags[:0]
	for _, d := range diags {
		line := tf.Line(tf.Pos(d.Start.Offset))
		if !suppressed(sups, d.RuleID, line) {
			out = append(out, d)
		}
	}
	if unusedSeverity == Off {
		return out
	}
	// Add these after filtering so directives cannot hide their own unused targets.
	for _, s := range sups {
		for _, target := range s.rules {
			if target.used {
				continue
			}
			start, end := f.Fset.PositionFor(target.pos, false), f.Fset.PositionFor(target.end, false)
			start.Filename, end.Filename = f.Filename, f.Filename
			out = append(out, Diagnostic{
				RuleID: UnusedSuppressionRuleID, Severity: unusedSeverity,
				Filename: f.Filename, Start: start, End: end,
				Message: fmt.Sprintf("Unused suppression for %q: no diagnostic was suppressed.", target.name),
				Help:    "Remove this target or correct its rule name. Suppressions are checked against the rules enabled for this file.",
			})
		}
	}
	return out
}

// The first matching directive and target receives credit. Later duplicates
// are redundant, and removing a reported unused target leaves suppression intact.
func suppressed(sups []suppression, ruleID string, line int) bool {
	for _, s := range sups {
		if s.line != 0 && line != s.line && line != s.line+1 {
			continue
		}
		for i := range s.rules {
			target := &s.rules[i]
			if target.name == "*" || target.name == ruleID || strings.HasPrefix(ruleID, target.name+"/") {
				target.used = true
				return true
			}
		}
	}
	return false
}
