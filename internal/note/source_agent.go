package note

import "os"

// DetectSourceAgent returns a source-agent identifier based on known harness
// environment variables. Returns empty string when no harness is detected.
//
// Detection order:
//  1. AGENT_MEMORY_SOURCE_AGENT → value as-is
//  2. OPENCODE_RUN_ID → "opencode:{value}"
//  3. Fallback → ""
func DetectSourceAgent() string {
	if v := os.Getenv("AGENT_MEMORY_SOURCE_AGENT"); v != "" {
		return v
	}
	if v := os.Getenv("OPENCODE_RUN_ID"); v != "" {
		return "opencode:" + v
	}
	return ""
}
