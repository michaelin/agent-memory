package cli

import (
	"testing"
)

func TestValidateSlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		slug    string
		wantErr bool
	}{
		{name: "empty string", slug: "", wantErr: true},
		{name: "path traversal with dots", slug: "../foo", wantErr: true},
		{name: "path with slash", slug: "foo/bar", wantErr: true},
		{name: "starts with hyphen", slug: "-foo", wantErr: true},
		{name: "ends with hyphen", slug: "foo-", wantErr: true},
		{name: "simple valid slug", slug: "foo", wantErr: false},
		{name: "hyphenated slug", slug: "foo-bar", wantErr: false},
		{name: "slug with digits", slug: "foo-bar-123", wantErr: false},
		{name: "uppercase not allowed", slug: "FOO", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateSlug(tc.slug)
			if tc.wantErr && err == nil {
				t.Errorf("validateSlug(%q) = nil, want error", tc.slug)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("validateSlug(%q) = %v, want nil", tc.slug, err)
			}
		})
	}
}
