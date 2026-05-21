package cli

import "fmt"

// validateSlug checks that slug is a non-empty, lowercase alphanumeric string
// with hyphens, suitable for use as a vault note identifier. It rejects path
// traversal characters and other unsafe inputs at the CLI boundary.
func validateSlug(slug string) error {
	if slug == "" {
		return fmt.Errorf("invalid slug %q: slug must not be empty", slug)
	}

	runes := []rune(slug)

	if runes[0] == '-' {
		return fmt.Errorf("invalid slug %q: slug must not start with '-'", slug)
	}
	if runes[len(runes)-1] == '-' {
		return fmt.Errorf("invalid slug %q: slug must not end with '-'", slug)
	}

	for _, r := range runes {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return fmt.Errorf("invalid slug %q: slug must contain only lowercase letters, digits, and hyphens", slug)
		}
	}

	return nil
}
