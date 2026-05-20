//go:build integration

package integration_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"

	"github.com/michaelin/agent-memory/internal/testutil"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "agent-memory Integration Suite")
}

var _ = ReportAfterSuite("tree reporter", func(report types.Report) {
	testutil.PrintTreeReport(report)
})
