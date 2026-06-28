//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunRegressionLoopWritesOptimizationReports(t *testing.T) {
	outputDir := t.TempDir()

	report, err := RunRegressionLoop(RegressionLoopConfig{
		ConfigDir: "configs",
		OutputDir: outputDir,
	})

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.False(t, report.GateDecision.Accepted)
	assert.Contains(t, report.GateDecision.Reason, "critical")
	assert.NotEmpty(t, report.FailureAttributionStats.Attributions)
	assert.FileExists(t, filepath.Join(outputDir, "optimization_report.json"))
	assert.FileExists(t, filepath.Join(outputDir, "optimization_report.md"))

	jsonReport, err := os.ReadFile(filepath.Join(outputDir, "optimization_report.json"))
	require.NoError(t, err)
	mdReport, err := os.ReadFile(filepath.Join(outputDir, "optimization_report.md"))
	require.NoError(t, err)
	assert.Contains(t, string(jsonReport), "baseline")
	assert.Contains(t, string(jsonReport), "candidate")
	assert.Contains(t, string(jsonReport), "gate_decision")
	assert.Contains(t, string(mdReport), "Optimization Report")
}
