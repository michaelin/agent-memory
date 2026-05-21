package note

import (
	"regexp"
	"strings"
	"time"
)

var placeholderAngle = regexp.MustCompile(`^<.*>$`)

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
