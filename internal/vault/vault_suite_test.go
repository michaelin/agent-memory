package vault

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"

	"github.com/michaelin/agent-memory/internal/testutil"
)

func TestVault(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Vault Suite")
}

var _ = ReportAfterSuite("tree reporter", func(report types.Report) {
	testutil.PrintTreeReport(report)
})
