//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-agent-go/evaluation/status"
)

func TestCompareCaseDeltasClassifiesStatusAndScoreChanges(t *testing.T) {
	baseline := &EvaluationResult{EvalSets: []EvalSetResult{{
		EvalSetID: "validation",
		Cases: []CaseResult{
			deltaCase("validation", "new_pass", metric("quality", 0.2, status.EvalStatusFailed)),
			deltaCase("validation", "new_fail", metric("quality", 1.0, status.EvalStatusPassed)),
			deltaCase("validation", "improved", metric("quality", 0.2, status.EvalStatusFailed)),
			deltaCase("validation", "regressed", metric("quality", 0.9, status.EvalStatusPassed)),
			deltaCase("validation", "unchanged", metric("quality", 0.7, status.EvalStatusPassed)),
		},
	}}}
	candidate := &EvaluationResult{EvalSets: []EvalSetResult{{
		EvalSetID: "validation",
		Cases: []CaseResult{
			deltaCase("validation", "new_pass", metric("quality", 1.0, status.EvalStatusPassed)),
			deltaCase("validation", "new_fail", metric("quality", 0.0, status.EvalStatusFailed)),
			deltaCase("validation", "improved", metric("quality", 0.6, status.EvalStatusFailed)),
			deltaCase("validation", "regressed", metric("quality", 0.8, status.EvalStatusPassed)),
			deltaCase("validation", "unchanged", metric("quality", 0.7, status.EvalStatusPassed)),
		},
	}}}

	deltas := CompareCaseDeltas(baseline, candidate)

	byID := map[string]CaseDelta{}
	for _, delta := range deltas {
		byID[delta.CaseID] = delta
	}
	assert.Equal(t, CaseDeltaNewlyPassed, byID["new_pass"].Type)
	assert.Equal(t, CaseDeltaNewlyFailed, byID["new_fail"].Type)
	assert.Equal(t, CaseDeltaScoreImproved, byID["improved"].Type)
	assert.Equal(t, CaseDeltaScoreRegressed, byID["regressed"].Type)
	assert.Equal(t, CaseDeltaUnchanged, byID["unchanged"].Type)
	assert.InDelta(t, 0.4, byID["improved"].ScoreDelta, 0.0001)
	assert.Equal(t, status.EvalStatusPassed, byID["new_pass"].CandidateStatus)
	assert.Equal(t, status.EvalStatusFailed, byID["new_fail"].CandidateStatus)
}

func deltaCase(evalSetID, evalCaseID string, metrics ...MetricResult) CaseResult {
	return CaseResult{
		EvalSetID:  evalSetID,
		EvalCaseID: evalCaseID,
		Metrics:    metrics,
	}
}

func metric(name string, score float64, metricStatus status.EvalStatus) MetricResult {
	return MetricResult{
		MetricName: name,
		Score:      score,
		Status:     metricStatus,
	}
}
