//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

// Package engine implements PromptIter orchestration and runtime flow for a generation round.
package engine

import (
	"fmt"
	"strings"
	"time"
)

// AcceptancePolicy controls whether a generated profile is accepted into next round input.
type AcceptancePolicy struct {
	// MinScoreGain is the minimum score increase required to accept a round patch.
	MinScoreGain float64
	// NoNewHardFail rejects candidates that introduce newly failed cases.
	NoNewHardFail bool
	// CriticalCaseIDs identifies validation cases that must not regress.
	CriticalCaseIDs []string
	// BudgetConstraint rejects candidates that exceed usage limits.
	BudgetConstraint *BudgetLimit
}

// BudgetLimit configures optional acceptance limits for cost, calls, and latency.
type BudgetLimit struct {
	// MaxCost is the maximum allowed cost. Zero disables this limit.
	MaxCost float64
	// MaxAPICalls is the maximum allowed API calls. Zero disables this limit.
	MaxAPICalls int
	// MaxLatency is the maximum allowed latency. Zero disables this limit.
	MaxLatency time.Duration
}

// BudgetUsage records the current usage snapshot for acceptance checks.
type BudgetUsage struct {
	// Cost is the candidate cost.
	Cost float64
	// APICalls is the candidate API call count.
	APICalls int
	// Latency is the candidate latency.
	Latency time.Duration
}

// GateResult records one acceptance gate outcome.
type GateResult struct {
	// GateName identifies the gate.
	GateName string `json:"gate_name"`
	// Passed is true if this gate accepted the candidate.
	Passed bool `json:"passed"`
	// Reason explains this gate outcome.
	Reason string `json:"reason"`
}

// AcceptanceDecision records round-level pass/fail outcome and score delta.
type AcceptanceDecision struct {
	// Accepted is true if validation gains satisfy acceptance criteria.
	Accepted bool `json:"accepted"`
	// ScoreDelta is the metric difference compared with previous accepted baseline.
	ScoreDelta float64 `json:"score_delta"`
	// Reason explains why acceptance succeeded or failed.
	Reason string `json:"reason"`
	// GateResults records detailed multi-dimensional gate outcomes.
	GateResults []GateResult `json:"gate_results,omitempty"`
}

func (e *engine) accept(
	policy AcceptancePolicy,
	baselineScore float64,
	candidateScore float64,
	caseDeltas []CaseDelta,
	usage BudgetUsage,
) *AcceptanceDecision {
	scoreDelta := candidateScore - baselineScore
	gateResults := []GateResult{scoreGainGate(scoreDelta, policy.MinScoreGain)}
	reasons := make([]string, 0, 4)
	if !gateResults[0].Passed {
		reasons = append(reasons, "validation score gain insufficient")
	}
	if policy.NoNewHardFail {
		result := noNewHardFailGate(caseDeltas)
		gateResults = append(gateResults, result)
		if !result.Passed {
			reasons = append(reasons, result.Reason)
		}
	}
	if len(policy.CriticalCaseIDs) > 0 {
		result := criticalCasePreserveGate(caseDeltas, policy.CriticalCaseIDs)
		gateResults = append(gateResults, result)
		if !result.Passed {
			reasons = append(reasons, result.Reason)
		}
	}
	if policy.BudgetConstraint != nil {
		result := budgetGate(usage, *policy.BudgetConstraint)
		gateResults = append(gateResults, result)
		if !result.Passed {
			reasons = append(reasons, result.Reason)
		}
	}
	accepted := len(reasons) == 0
	reason := "all gates passed"
	if !accepted {
		reason = strings.Join(reasons, "; ")
	}
	return &AcceptanceDecision{
		Accepted:    accepted,
		ScoreDelta:  scoreDelta,
		Reason:      reason,
		GateResults: gateResults,
	}
}

func scoreGainGate(scoreDelta, threshold float64) GateResult {
	return GateResult{
		GateName: "ValidationScoreGain",
		Passed:   scoreDelta >= threshold,
		Reason:   fmt.Sprintf("scoreDelta=%.4f, threshold=%.4f", scoreDelta, threshold),
	}
}

func noNewHardFailGate(caseDeltas []CaseDelta) GateResult {
	newFails := make([]string, 0)
	for _, delta := range caseDeltas {
		if delta.Type == CaseDeltaNewlyFailed {
			newFails = append(newFails, delta.CaseID)
		}
	}
	return GateResult{
		GateName: "NoNewHardFail",
		Passed:   len(newFails) == 0,
		Reason:   fmt.Sprintf("newly failed cases: %s", strings.Join(newFails, ",")),
	}
}

func criticalCasePreserveGate(caseDeltas []CaseDelta, criticalCaseIDs []string) GateResult {
	critical := make(map[string]struct{}, len(criticalCaseIDs))
	for _, caseID := range criticalCaseIDs {
		critical[caseID] = struct{}{}
	}
	regressed := make([]string, 0)
	for _, delta := range caseDeltas {
		if _, ok := critical[delta.CaseID]; !ok {
			continue
		}
		if delta.Type == CaseDeltaNewlyFailed || delta.Type == CaseDeltaScoreRegressed {
			regressed = append(regressed, delta.CaseID)
		}
	}
	return GateResult{
		GateName: "CriticalCasePreserve",
		Passed:   len(regressed) == 0,
		Reason:   fmt.Sprintf("regressed critical cases: %s", strings.Join(regressed, ",")),
	}
}

func budgetGate(usage BudgetUsage, limit BudgetLimit) GateResult {
	passed := true
	reasons := make([]string, 0, 3)
	if limit.MaxCost > 0 && usage.Cost > limit.MaxCost {
		passed = false
		reasons = append(reasons, fmt.Sprintf("cost=%.4f/%.4f", usage.Cost, limit.MaxCost))
	}
	if limit.MaxAPICalls > 0 && usage.APICalls > limit.MaxAPICalls {
		passed = false
		reasons = append(reasons, fmt.Sprintf("apiCalls=%d/%d", usage.APICalls, limit.MaxAPICalls))
	}
	if limit.MaxLatency > 0 && usage.Latency > limit.MaxLatency {
		passed = false
		reasons = append(reasons, fmt.Sprintf("latency=%s/%s", usage.Latency, limit.MaxLatency))
	}
	reason := "within budget"
	if len(reasons) > 0 {
		reason = "budget exceeded: " + strings.Join(reasons, ", ")
	}
	return GateResult{
		GateName: "BudgetConstraint",
		Passed:   passed,
		Reason:   reason,
	}
}
