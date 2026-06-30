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
	"fmt"
	"strings"
	"time"

	"trpc.group/trpc-go/trpc-agent-go/evaluation/status"
	promptiterengine "trpc.group/trpc-go/trpc-agent-go/evaluation/workflow/promptiter/engine"
)

// OptimizationReport is the challenge audit artifact.
type OptimizationReport struct {
	Metadata                ReportMetadata                       `json:"metadata"`
	Baseline                EvaluationSummary                    `json:"baseline"`
	Candidate               EvaluationSummary                    `json:"candidate"`
	Delta                   DeltaSummary                         `json:"delta"`
	GateDecision            *promptiterengine.AcceptanceDecision `json:"gate_decision"`
	FailureAttributionStats FailureAttributionStats              `json:"failure_attribution_stats"`
	CostLatency             CostLatencySummary                   `json:"cost_latency"`
	Rounds                  []RoundSnapshot                      `json:"rounds,omitempty"`
}

// ReportMetadata stores reproducibility metadata.
type ReportMetadata struct {
	GeneratedAt      time.Time `json:"generated_at"`
	RandomSeed       int64     `json:"random_seed"`
	ModelConfig      string    `json:"model_config"`
	FakeEngineConfig string    `json:"fake_engine_config"`
	PipelineVersion  string    `json:"pipeline_version"`
}

// EvaluationSummary summarizes one evaluation result.
type EvaluationSummary struct {
	Name         string  `json:"name"`
	OverallScore float64 `json:"overall_score"`
	PassedCases  int     `json:"passed_cases"`
	FailedCases  int     `json:"failed_cases"`
	TotalCases   int     `json:"total_cases"`
}

// DeltaSummary summarizes per-case validation changes.
type DeltaSummary struct {
	CaseDeltas []promptiterengine.CaseDelta `json:"case_deltas"`
	Counts     map[string]int               `json:"counts"`
}

// FailureAttributionStats summarizes failure attribution counts.
type FailureAttributionStats struct {
	Counts       map[FailureCategory]int `json:"counts"`
	Attributions []FailureAttribution    `json:"attributions"`
}

// CostLatencySummary stores deterministic run budget data.
type CostLatencySummary struct {
	TotalCost          float64 `json:"total_cost"`
	TotalAPICalls      int     `json:"total_api_calls"`
	TotalLatencyMillis int64   `json:"total_latency_ms"`
}

// RoundSnapshot stores one round audit snapshot.
type RoundSnapshot struct {
	RoundID              int                                  `json:"round_id"`
	CandidatePrompt      string                               `json:"candidate_prompt"`
	TrainEvalResult      *promptiterengine.EvaluationResult   `json:"train_eval_result"`
	ValidationEvalResult *promptiterengine.EvaluationResult   `json:"validation_eval_result"`
	CaseDeltas           []promptiterengine.CaseDelta         `json:"case_deltas"`
	Acceptance           *promptiterengine.AcceptanceDecision `json:"acceptance_decision"`
}

// SummarizeEvaluation summarizes an engine evaluation result.
func SummarizeEvaluation(name string, result *promptiterengine.EvaluationResult) EvaluationSummary {
	summary := EvaluationSummary{Name: name}
	if result == nil {
		return summary
	}
	summary.OverallScore = result.OverallScore
	for _, evalSet := range result.EvalSets {
		for _, caseResult := range evalSet.Cases {
			summary.TotalCases++
			if summarizeCaseStatus(caseResult.Metrics) == status.EvalStatusPassed {
				summary.PassedCases++
			} else {
				summary.FailedCases++
			}
		}
	}
	return summary
}

// SummarizeDeltas summarizes case delta counts.
func SummarizeDeltas(deltas []promptiterengine.CaseDelta) DeltaSummary {
	counts := make(map[string]int)
	for _, delta := range deltas {
		counts[string(delta.Type)]++
	}
	return DeltaSummary{CaseDeltas: deltas, Counts: counts}
}

// SummarizeAttributions summarizes failure attribution counts.
func SummarizeAttributions(attributions []FailureAttribution) FailureAttributionStats {
	counts := make(map[FailureCategory]int)
	for _, attribution := range attributions {
		counts[attribution.Category]++
	}
	return FailureAttributionStats{Counts: counts, Attributions: attributions}
}

// BuildJSONReport serializes an optimization report.
func BuildJSONReport(report OptimizationReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

// BuildMarkdownReport renders an optimization report for humans.
func BuildMarkdownReport(report OptimizationReport) string {
	var md strings.Builder
	md.WriteString("# Optimization Report\n\n")
	accepted := false
	reason := ""
	if report.GateDecision != nil {
		accepted = report.GateDecision.Accepted
		reason = report.GateDecision.Reason
	}
	md.WriteString(fmt.Sprintf("## Decision\n\nAccepted: %v\n\nReason: %s\n\n", accepted, reason))
	md.WriteString("## Baseline vs Candidate\n\n")
	md.WriteString("| Name | Score | Passed | Failed | Total |\n")
	md.WriteString("| --- | ---: | ---: | ---: | ---: |\n")
	writeEvalSummary(&md, report.Baseline)
	writeEvalSummary(&md, report.Candidate)
	md.WriteString("\n## Case Delta\n\n")
	md.WriteString("| Eval Set | Case | Type | Baseline | Candidate | Delta |\n")
	md.WriteString("| --- | --- | --- | ---: | ---: | ---: |\n")
	for _, delta := range report.Delta.CaseDeltas {
		md.WriteString(fmt.Sprintf("| %s | %s | %s | %.4f | %.4f | %.4f |\n",
			delta.EvalSetID, delta.CaseID, delta.Type, delta.BaselineScore, delta.CandidateScore, delta.ScoreDelta))
	}
	md.WriteString("\n## Failure Attribution\n\n")
	md.WriteString("| Eval Set | Case | Category | Reason |\n")
	md.WriteString("| --- | --- | --- | --- |\n")
	for _, attribution := range report.FailureAttributionStats.Attributions {
		md.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
			attribution.EvalSetID, attribution.CaseID, attribution.Category, attribution.Reason))
	}
	md.WriteString("\n## Cost / Latency\n\n")
	md.WriteString(fmt.Sprintf("- API calls: %d\n", report.CostLatency.TotalAPICalls))
	md.WriteString(fmt.Sprintf("- Cost: %.4f\n", report.CostLatency.TotalCost))
	md.WriteString(fmt.Sprintf("- Latency: %d ms\n", report.CostLatency.TotalLatencyMillis))
	return md.String()
}

func writeEvalSummary(md *strings.Builder, summary EvaluationSummary) {
	md.WriteString(fmt.Sprintf("| %s | %.4f | %d | %d | %d |\n",
		summary.Name, summary.OverallScore, summary.PassedCases, summary.FailedCases, summary.TotalCases))
}

func summarizeCaseStatus(metrics []promptiterengine.MetricResult) status.EvalStatus {
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
