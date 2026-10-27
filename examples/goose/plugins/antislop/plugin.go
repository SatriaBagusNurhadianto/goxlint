// Package antislop rejects low-evidence and low-signal Go patterns. It is a Go
// port of the ideas in https://github.com/dmmulroy/anti-slop.
package antislop

import (
	"go/types"

	"github.com/egoisutolabs/goose-lint/goose"
)

// Plugin registers every anti-slop rule.
var Plugin = goose.Plugin{
	Name: "anti-slop",
	Rules: []*goose.Rule{
		NoAnyMapValues,
		NoAnyParameters,
		NoAnyReturns,
		NoBlankError,
		NoDetachedContext,
		NoErrorStringMatching,
		NoGenericPackageNames,
		NoLogAndReturnError,
		NoPanicInLibrary,
		NoPlaceholderComments,
		NoPrintInLibrary,
		NoRedundantErrorPrefix,
		NoReflectDynamicAccess,
		NoRestatingComments,
		NoSleepInTests,
		NoSwallowedError,
		RequireSafetyCommentForTypeAssertion,
	},
}

// isEmptyInterface reports whether t is any, interface{}, or a named type or
// alias with an empty interface underneath. Type parameters are not, even when
// their constraint is any.
func isEmptyInterface(t types.Type) bool {
	if t == nil {
		return false
	}
	t = types.Unalias(t)
	if _, ok := t.(*types.TypeParam); ok {
		return false
	}
	iface, ok := t.Underlying().(*types.Interface)
	return ok && iface.Empty()
}
