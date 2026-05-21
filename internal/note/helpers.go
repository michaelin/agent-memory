package note

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var placeholderAngle = regexp.MustCompile(`^<[^>]+>$`)

// IsPlaceholder reports whether value is a placeholder that has not been
// filled in. It returns true for:
//   - empty string
//   - angle-bracket patterns such as "<your-title-here>"
//   - case-insensitive matches for the keywords TODO, TBD, FIXME, xxx, placeholder
func IsPlaceholder(value string) bool {
	if value == "" {
		return true
	}
	if placeholderAngle.MatchString(value) {
		return true
	}
	switch strings.ToLower(value) {
	case "todo", "tbd", "fixme", "xxx", "placeholder":
		return true
	}
	return false
}

// IsValidDate reports whether value is a valid date in YYYY-MM-DD format.
func IsValidDate(value string) bool {
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}

// ValidateSlug checks that slug is a safe, non-empty identifier containing
// only lowercase letters, digits, and hyphens. It rejects path traversal
// characters and other unsafe inputs at the package boundary.
func ValidateSlug(slug string) error {
	if slug == "" {
		return fmt.Errorf("invalid slug: must not be empty")
	}
	if slug[0] == '-' {
		return fmt.Errorf("invalid slug %q: must not start with '-'", slug)
	}
	if slug[len(slug)-1] == '-' {
		return fmt.Errorf("invalid slug %q: must not end with '-'", slug)
	}
	for _, r := range slug {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return fmt.Errorf("invalid slug %q: must contain only lowercase letters, digits, and hyphens", slug)
		}
	}
	return nil
}
