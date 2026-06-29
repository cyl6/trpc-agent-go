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
	"sort"

	"trpc.group/trpc-go/trpc-agent-go/evaluation/status"
)

// CaseDeltaType describes how a candidate changed one validation case.
type CaseDeltaType string

const (
	// CaseDeltaNewlyPassed means the candidate turned a non-passing case into passing.
	CaseDeltaNewlyPassed CaseDeltaType = "newly_passed"
	// CaseDeltaNewlyFailed means the candidate regressed a passing case into non-passing.
	CaseDeltaNewlyFailed CaseDeltaType = "newly_failed"
	// CaseDeltaScoreImproved means score improved without changing pass/fail status.
	CaseDeltaScoreImproved CaseDeltaType = "score_improved"
	// CaseDeltaScoreRegressed means score dropped without changing pass/fail status.
	CaseDeltaScoreRegressed CaseDeltaType = "score_regressed"
	// CaseDeltaUnchanged means neither status nor score changed.
	CaseDeltaUnchanged CaseDeltaType = "unchanged"
)

// CaseDelta stores the per-case validation difference between baseline and candidate.
type CaseDelta struct {
	EvalSetID       string            `json:"eval_set_id"`
	CaseID          string            `json:"case_id"`
	Type            CaseDeltaType     `json:"type"`
	BaselineScore   float64           `json:"baseline_score"`
	CandidateScore  float64           `json:"candidate_score"`
	ScoreDelta      float64           `json:"score_delta"`
	BaselineStatus  status.EvalStatus `json:"baseline_status"`
	CandidateStatus status.EvalStatus `json:"candidate_status"`
}

// CompareCaseDeltas compares validation results case by case.
func CompareCaseDeltas(baseline, candidate *EvaluationResult) []CaseDelta {
	baselineByCaseID := indexCasesByKey(baseline)
	candidateByCaseID := indexCasesByKey(candidate)
	caseKeys := make([]caseDeltaKey, 0, len(baselineByCaseID)+len(candidateByCaseID))
	seen := make(map[caseDeltaKey]struct{}, len(baselineByCaseID)+len(candidateByCaseID))
	for key := range baselineByCaseID {
		seen[key] = struct{}{}
		caseKeys = append(caseKeys, key)
	}
	for key := range candidateByCaseID {
		if _, ok := seen[key]; ok {
			continue
		}
		caseKeys = append(caseKeys, key)
	}
	sort.Slice(caseKeys, func(i, j int) bool {
		if caseKeys[i].evalSetID != caseKeys[j].evalSetID {
			return caseKeys[i].evalSetID < caseKeys[j].evalSetID
		}
		return caseKeys[i].evalCaseID < caseKeys[j].evalCaseID
	})

	deltas := make([]CaseDelta, 0, len(caseKeys))
	for _, key := range caseKeys {
		baseCase := baselineByCaseID[key]
		candCase := candidateByCaseID[key]
		baseScore := averageMetricScore(baseCase)
		candScore := averageMetricScore(candCase)
		baseStatus := caseStatusFromMetrics(baseCase.Metrics)
		candStatus := caseStatusFromMetrics(candCase.Metrics)
		delta := CaseDelta{
			EvalSetID:       key.evalSetID,
			CaseID:          key.evalCaseID,
			BaselineScore:   baseScore,
			CandidateScore:  candScore,
			ScoreDelta:      candScore - baseScore,
			BaselineStatus:  baseStatus,
			CandidateStatus: candStatus,
		}
		switch {
		case baseStatus != status.EvalStatusPassed && candStatus == status.EvalStatusPassed:
			delta.Type = CaseDeltaNewlyPassed
		case baseStatus == status.EvalStatusPassed && candStatus != status.EvalStatusPassed:
			delta.Type = CaseDeltaNewlyFailed
		case candScore > baseScore:
			delta.Type = CaseDeltaScoreImproved
		case candScore < baseScore:
			delta.Type = CaseDeltaScoreRegressed
		default:
			delta.Type = CaseDeltaUnchanged
		}
		deltas = append(deltas, delta)
	}
	return deltas
}

type caseDeltaKey struct {
	evalSetID  string
	evalCaseID string
}

func indexCasesByKey(result *EvaluationResult) map[caseDeltaKey]CaseResult {
	index := make(map[caseDeltaKey]CaseResult)
	if result == nil {
		return index
	}
	for _, evalSet := range result.EvalSets {
		for _, c := range evalSet.Cases {
			if c.EvalCaseID == "" {
				continue
			}
			evalSetID := c.EvalSetID
			if evalSetID == "" {
				evalSetID = evalSet.EvalSetID
			}
			index[caseDeltaKey{evalSetID: evalSetID, evalCaseID: c.EvalCaseID}] = c
		}
	}
	return index
}

func caseStatusFromMetrics(metrics []MetricResult) status.EvalStatus {
	hasPassed := false
	for _, metric := range metrics {
		switch metric.Status {
		case status.EvalStatusFailed:
			return status.EvalStatusFailed
		case status.EvalStatusPassed:
			hasPassed = true
		}
	}
	if hasPassed {
		return status.EvalStatusPassed
	}
	return status.EvalStatusNotEvaluated
}

func averageMetricScore(caseResult CaseResult) float64 {
	total, count := 0.0, 0
	for _, metric := range caseResult.Metrics {
		if metric.Status == status.EvalStatusNotEvaluated {
			continue
		}
		total += metric.Score
		count++
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}
