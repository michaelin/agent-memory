package testutil

import (
	"fmt"
	"strings"

	"github.com/onsi/ginkgo/v2/types"
)

// PrintTreeReport prints a nested spec tree from a ginkgo report.
// Call this from a ReportAfterSuite block.
func PrintTreeReport(report types.Report) {
	type leafEntry struct {
		containers []string
		leaf       string
		state      types.SpecState
	}

	var entries []leafEntry
	for _, spec := range report.SpecReports {
		if spec.LeafNodeType != types.NodeTypeIt {
			continue
		}
		entries = append(entries, leafEntry{
			containers: spec.ContainerHierarchyTexts,
			leaf:       spec.LeafNodeText,
			state:      spec.State,
		})
	}

	printed := make(map[string]bool)

	fmt.Println() // separate from ginkgo's default output
	for _, e := range entries {
		for depth, container := range e.containers {
			key := strings.Join(e.containers[:depth+1], "\x00")
			if !printed[key] {
				fmt.Printf("%s%s\n", strings.Repeat("  ", depth), container)
				printed[key] = true
			}
		}
		indent := strings.Repeat("  ", len(e.containers))
		fmt.Printf("%s%s %s\n", indent, e.leaf, stateGlyph(e.state))
	}

	var passed, failed int
	for _, e := range entries {
		switch e.state {
		case types.SpecStatePassed:
			passed++
		default:
			failed++
		}
	}
	fmt.Printf("\n%d passed, %d failed\n", passed, failed)
}

func stateGlyph(s types.SpecState) string {
	switch s {
	case types.SpecStatePassed:
		return "✓"
	case types.SpecStateFailed, types.SpecStatePanicked, types.SpecStateTimedout:
		return "✗"
	case types.SpecStatePending:
		return "P"
	case types.SpecStateSkipped:
		return "S"
	default:
		return "?"
	}
}
