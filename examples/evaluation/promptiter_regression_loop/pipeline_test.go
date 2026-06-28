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
	"time"

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
	assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), report.Metadata.GeneratedAt)
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
	assert.Contains(t, string(jsonReport), "candidate_prompt")
	assert.Contains(t, string(jsonReport), "train_eval_result")
	assert.Contains(t, string(jsonReport), "validation_eval_result")
	assert.Contains(t, string(mdReport), "Optimization Report")
}

func TestRunRegressionLoopUsesOptimizerFakeModelQueue(t *testing.T) {
	configDir := copyConfigDir(t)
	outputDir := t.TempDir()
	customPrompt := "Queue supplied candidate prompt."
	queue := `{
  "optimizer": [
    [
      {
        "content": "{\"patches\":[{\"surface_id\":\"candidate#instruction\",\"value\":{\"text\":\"` + customPrompt + `\"},\"reason\":\"test queue\"}]}"
      }
    ]
  ]
}`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "fake_model_queue.json"), []byte(queue), 0644))

	report, err := RunRegressionLoop(RegressionLoopConfig{
		ConfigDir: configDir,
		OutputDir: outputDir,
	})

	require.NoError(t, err)
	require.Len(t, report.Rounds, 1)
	assert.Equal(t, customPrompt, report.Rounds[0].CandidatePrompt)
}

func copyConfigDir(t *testing.T) string {
	t.Helper()
	target := t.TempDir()
	entries, err := os.ReadDir("configs")
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join("configs", entry.Name()))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(target, entry.Name()), data, 0644))
	}
	return target
}
