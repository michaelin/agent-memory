package cli

import "github.com/michaelin/agent-memory/internal/note"

// validateSlug delegates to note.ValidateSlug for backward compatibility
// with any internal callers. The canonical implementation lives in the
// note package to enforce slug safety at the library boundary.
func validateSlug(slug string) error {
	return note.ValidateSlug(slug)
}
