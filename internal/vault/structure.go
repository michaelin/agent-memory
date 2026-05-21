package vault

import "embed"

//go:embed templates/*
var templateFS embed.FS

// VaultEntry defines a required path in the vault structure.
type VaultEntry struct {
	Path     string
	IsDir    bool
	Template string // template filename in embedded FS (empty for dirs)
}

// VaultStructure returns the ordered list of entries that define a complete vault.
// Directories always come before their child files.
func VaultStructure() []VaultEntry {
	return []VaultEntry{
		{Path: "_meta", IsDir: true},
		{Path: "_inbox", IsDir: true},
		{Path: "_contested", IsDir: true},
		{Path: "notes", IsDir: true},
		{Path: "_deprecated", IsDir: true},
		{Path: "_meta/templates", IsDir: true},
		{Path: "_meta/writing-protocol.md", IsDir: false, Template: "templates/writing-protocol.md"},
		{Path: "_meta/tag-taxonomy.md", IsDir: false, Template: "templates/tag-taxonomy.md"},
		{Path: "_meta/constraints-summary.md", IsDir: false, Template: "templates/constraints-summary.md"},
		{Path: "_meta/status-lifecycle.md", IsDir: false, Template: "templates/status-lifecycle.md"},
		{Path: "_meta/log.md", IsDir: false, Template: "templates/log.md"},
		{Path: "_meta/templates/librarian-agent.md", IsDir: false, Template: "templates/librarian-agent.md"},
		{Path: "_meta/templates/librarian-skill.md", IsDir: false, Template: "templates/librarian-skill.md"},
	}
}
