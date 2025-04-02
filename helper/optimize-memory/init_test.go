package optimizememory_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitOptimizeMemory(t *testing.T) {
	suite := spec.New("cmd/optimize-memory/internal", spec.Report(report.Terminal{}))
	suite("Run", testRun)
	suite.Run(t)
}
