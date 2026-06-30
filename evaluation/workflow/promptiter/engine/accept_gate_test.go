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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcceptRejectsNewHardFail(t *testing.T) {
	decision := (&engine{}).accept(AcceptancePolicy{
		MinScoreGain:  0.01,
		NoNewHardFail: true,
	}, 0.5, 0.7, []CaseDelta{{
		CaseID: "critical_tool_case",
		Type:   CaseDeltaNewlyFailed,
	}}, BudgetUsage{})

	require.NotNil(t, decision)
	assert.False(t, decision.Accepted)
	assert.Contains(t, decision.Reason, "newly failed")
	assertGateFailed(t, decision, "NoNewHardFail")
}

func TestAcceptRejectsInsufficientValidationScoreGain(t *testing.T) {
	decision := (&engine{}).accept(AcceptancePolicy{
		MinScoreGain: 0.01,
	}, 0.7, 0.6, []CaseDelta{{
		CaseID: "regressed_but_not_new_fail",
		Type:   CaseDeltaScoreRegressed,
	}}, BudgetUsage{})

	require.NotNil(t, decision)
	assert.False(t, decision.Accepted)
	assert.Contains(t, decision.Reason, "validation score gain insufficient")
	assert.InDelta(t, -0.1, decision.ScoreDelta, 0.0001)
	assertGateFailed(t, decision, "ValidationScoreGain")
}

func TestAcceptRejectsCriticalCaseRegression(t *testing.T) {
	decision := (&engine{}).accept(AcceptancePolicy{
		MinScoreGain:    0.01,
		CriticalCaseIDs: []string{"must_keep"},
	}, 0.5, 0.7, []CaseDelta{{
		CaseID: "must_keep",
		Type:   CaseDeltaScoreRegressed,
	}}, BudgetUsage{})

	require.NotNil(t, decision)
	assert.False(t, decision.Accepted)
	assert.Contains(t, decision.Reason, "critical")
	assertGateFailed(t, decision, "CriticalCasePreserve")
}

func TestAcceptRejectsBudgetOverrun(t *testing.T) {
	decision := (&engine{}).accept(AcceptancePolicy{
		MinScoreGain: 0.01,
		BudgetConstraint: &BudgetLimit{
			MaxCost:     1,
			MaxAPICalls: 10,
			MaxLatency:  2 * time.Second,
		},
	}, 0.5, 0.7, nil, BudgetUsage{
		Cost:     0.5,
		APICalls: 11,
		Latency:  time.Second,
	})

	require.NotNil(t, decision)
	assert.False(t, decision.Accepted)
	assert.Contains(t, decision.Reason, "budget")
	assertGateFailed(t, decision, "BudgetConstraint")
}

func TestAcceptRecordsAllPassingGateResults(t *testing.T) {
	decision := (&engine{}).accept(AcceptancePolicy{
		MinScoreGain:    0.01,
		NoNewHardFail:   true,
		CriticalCaseIDs: []string{"must_keep"},
		BudgetConstraint: &BudgetLimit{
			MaxCost:     1,
			MaxAPICalls: 10,
			MaxLatency:  time.Second,
		},
	}, 0.5, 0.7, []CaseDelta{{
		CaseID: "must_keep",
		Type:   CaseDeltaScoreImproved,
	}}, BudgetUsage{
		Cost:     0.1,
		APICalls: 3,
		Latency:  10 * time.Millisecond,
	})

	require.NotNil(t, decision)
	assert.True(t, decision.Accepted)
	assert.InDelta(t, 0.2, decision.ScoreDelta, 0.0001)
	assert.Len(t, decision.GateResults, 4)
	for _, gate := range decision.GateResults {
		assert.True(t, gate.Passed, gate.GateName)
	}
}

func assertGateFailed(t *testing.T, decision *AcceptanceDecision, gateName string) {
	t.Helper()
	for _, gate := range decision.GateResults {
		if gate.GateName == gateName {
			assert.False(t, gate.Passed)
			return
		}
	}
	assert.Failf(t, "missing gate", "gate %s not found in %+v", gateName, decision.GateResults)
}
