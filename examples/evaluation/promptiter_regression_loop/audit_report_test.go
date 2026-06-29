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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-agent-go/evaluation/status"
	promptiterengine "trpc.group/trpc-go/trpc-agent-go/evaluation/workflow/promptiter/engine"
)

func TestAuditReportBuilderOutputsRequiredJSONAndMarkdownFields(t *testing.T) {
	baseline := evalResult(0.4, promptiterengine.CaseResult{
		EvalSetID:  "validation",
		EvalCaseID: "case_1",
		Metrics: []promptiterengine.MetricResult{{
			MetricName: "final_response_exact_json",
			Score:      0.4,
			Status:     status.EvalStatusFailed,
			Reason:     "mismatch",
		}},
	})
	candidate := evalResult(0.7, promptiterengine.CaseResult{
		EvalSetID:  "validation",
		EvalCaseID: "case_1",
		Metrics: []promptiterengine.MetricResult{{
			MetricName: "final_response_exact_json",
			Score:      0.7,
			Status:     status.EvalStatusPassed,
		}},
	})
	deltas := promptiterengine.CompareCaseDeltas(baseline, candidate)
	decision := &promptiterengine.AcceptanceDecision{
		Accepted:   false,
		ScoreDelta: 0.3,
		Reason:     "critical case regressed",
		GateResults: []promptiterengine.GateResult{{
			GateName: "CriticalCasePreserve",
			Passed:   false,
			Reason:   "regressed critical cases: validation_overfit_guard",
		}},
	}
	attributions := []FailureAttribution{{
		CaseID:   "case_1",
		Category: FailureCategoryFinalResponseMismatch,
		Reason:   "mismatch",
		Evidence: "final_response_exact_json",
	}}
	report := OptimizationReport{
		Metadata: ReportMetadata{
			GeneratedAt:      time.Unix(1, 0).UTC(),
			RandomSeed:       7,
			ModelConfig:      "fake_engine",
			FakeEngineConfig: "configs/fake_model_queue.json",
			PipelineVersion:  "test",
		},
		Baseline:                SummarizeEvaluation("baseline", baseline),
		Candidate:               SummarizeEvaluation("candidate", candidate),
		Delta:                   SummarizeDeltas(deltas),
		GateDecision:            decision,
		FailureAttributionStats: SummarizeAttributions(attributions),
		CostLatency:             CostLatencySummary{TotalCost: 0, TotalAPICalls: 12, TotalLatencyMillis: 34},
	}

	jsonBytes, err := BuildJSONReport(report)
	require.NoError(t, err)
	md := BuildMarkdownReport(report)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(jsonBytes, &decoded))
	assert.Contains(t, string(jsonBytes), `"baseline"`)
	assert.Contains(t, string(jsonBytes), `"candidate"`)
	assert.Contains(t, string(jsonBytes), `"case_deltas"`)
	assert.Contains(t, string(jsonBytes), `"eval_set_id"`)
	assert.Contains(t, string(jsonBytes), `"case_id"`)
	assert.Contains(t, string(jsonBytes), `"gate_decision"`)
	assert.Contains(t, string(jsonBytes), `"accepted"`)
	assert.Contains(t, string(jsonBytes), `"failure_attribution_stats"`)
	assert.Contains(t, string(jsonBytes), `"cost_latency"`)
	assert.Contains(t, md, "Optimization Report")
	assert.Contains(t, md, "critical case regressed")
	assert.Contains(t, md, "Eval Set")
	assert.Contains(t, md, "case_1")
	assert.Contains(t, md, "final_response_mismatch")
}

func evalResult(score float64, cases ...promptiterengine.CaseResult) *promptiterengine.EvaluationResult {
	return &promptiterengine.EvaluationResult{
		OverallScore: score,
		EvalSets: []promptiterengine.EvalSetResult{{
			EvalSetID:    "validation",
			OverallScore: score,
			Cases:        cases,
		}},
	}
}
