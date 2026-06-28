//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigFilesContainRequiredRegressionLoopInputs(t *testing.T) {
	train := readEvalSet(t, "configs/train.evalset.json")
	validation := readEvalSet(t, "configs/validation.evalset.json")
	metricsData, err := os.ReadFile(filepath.Clean("configs/metrics.json"))
	require.NoError(t, err)
	promptiterData, err := os.ReadFile(filepath.Clean("configs/promptiter.json"))
	require.NoError(t, err)
	promptData, err := os.ReadFile(filepath.Clean("configs/baseline_prompt.txt"))
	require.NoError(t, err)
	queueData, err := os.ReadFile(filepath.Clean("configs/fake_model_queue.json"))
	require.NoError(t, err)

	assert.Equal(t, "promptiter-regression-train", train.EvalSetID)
	assert.Len(t, train.EvalCases, 3)
	assert.Equal(t, "promptiter-regression-validation", validation.EvalSetID)
	assert.Len(t, validation.EvalCases, 3)
	assert.Contains(t, string(metricsData), "final_response")
	assert.Contains(t, string(metricsData), "toolTrajectory")
	assert.Contains(t, string(metricsData), "llmJudge")
	assert.Contains(t, string(promptData), "baseline")
	assert.JSONEq(t, `{"max_rounds":2,"min_score_gain":0.01,"target_score":0.95,"no_new_hard_fail":true,"critical_case_ids":["validation_overfit_guard"],"max_api_calls":200}`, string(promptiterData))
	assert.Contains(t, string(queueData), "candidate")
	assert.Contains(t, string(queueData), "judge")
	assert.Contains(t, string(queueData), "backwarder")
	assert.Contains(t, string(queueData), "aggregator")
	assert.Contains(t, string(queueData), "optimizer")
}

type regressionEvalSetFile struct {
	EvalSetID string `json:"evalSetId"`
	EvalCases []struct {
		EvalID string `json:"evalId"`
	} `json:"evalCases"`
}

func readEvalSet(t *testing.T, path string) regressionEvalSetFile {
	t.Helper()
	data, err := os.ReadFile(filepath.Clean(path))
	require.NoError(t, err)
	var evalSet regressionEvalSetFile
	require.NoError(t, json.Unmarshal(data, &evalSet))
	return evalSet
}
