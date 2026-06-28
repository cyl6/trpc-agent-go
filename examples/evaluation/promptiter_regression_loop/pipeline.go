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
	"os"
	"path/filepath"
	"strings"
	"time"

	"trpc.group/trpc-go/trpc-agent-go/evaluation/status"
	promptiterengine "trpc.group/trpc-go/trpc-agent-go/evaluation/workflow/promptiter/engine"
)

const pipelineVersion = "promptiter-regression-loop/v1"

// RegressionLoopConfig configures the deterministic regression-loop example.
type RegressionLoopConfig struct {
	ConfigDir string
	OutputDir string
}

type promptIterConfig struct {
	MaxRounds       int      `json:"max_rounds"`
	MinScoreGain    float64  `json:"min_score_gain"`
	TargetScore     float64  `json:"target_score"`
	NoNewHardFail   bool     `json:"no_new_hard_fail"`
	CriticalCaseIDs []string `json:"critical_case_ids"`
	MaxAPICalls     int      `json:"max_api_calls"`
}

// RunRegressionLoop runs the deterministic Evaluation + PromptIter regression loop.
func RunRegressionLoop(cfg RegressionLoopConfig) (*OptimizationReport, error) {
	if cfg.ConfigDir == "" {
		cfg.ConfigDir = "configs"
	}
	if cfg.OutputDir == "" {
		cfg.OutputDir = "report"
	}
	promptiterCfg, err := loadPromptIterConfig(filepath.Join(cfg.ConfigDir, "promptiter.json"))
	if err != nil {
		return nil, err
	}
	baseline := sampleBaselineValidation()
	candidate := sampleCandidateValidation()
	deltas := promptiterengine.CompareCaseDeltas(baseline, candidate)
	attributions := NewFailureAttributor().Attribute(baseline)
	decision := evaluateSampleGate(promptiterCfg, baseline.OverallScore, candidate.OverallScore, deltas)
	report := OptimizationReport{
		Metadata: ReportMetadata{
			GeneratedAt:      time.Now().UTC(),
			RandomSeed:       7,
			ModelConfig:      "fake_engine",
			FakeEngineConfig: filepath.Join(cfg.ConfigDir, "fake_model_queue.json"),
			PipelineVersion:  pipelineVersion,
		},
		Baseline:                SummarizeEvaluation("baseline", baseline),
		Candidate:               SummarizeEvaluation("candidate", candidate),
		Delta:                   SummarizeDeltas(deltas),
		GateDecision:            decision,
		FailureAttributionStats: SummarizeAttributions(attributions),
		CostLatency:             CostLatencySummary{TotalCost: 0, TotalAPICalls: 12, TotalLatencyMillis: 1},
		Rounds: []RoundSnapshot{{
			RoundID:    1,
			CaseDeltas: deltas,
			Acceptance: decision,
		}},
	}
	if err := writeReports(cfg.OutputDir, report); err != nil {
		return nil, err
	}
	return &report, nil
}

func loadPromptIterConfig(path string) (promptIterConfig, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return promptIterConfig{}, fmt.Errorf("read promptiter config: %w", err)
	}
	var cfg promptIterConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return promptIterConfig{}, fmt.Errorf("decode promptiter config: %w", err)
	}
	return cfg, nil
}

func writeReports(outputDir string, report OptimizationReport) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	jsonReport, err := BuildJSONReport(report)
	if err != nil {
		return fmt.Errorf("build json report: %w", err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "optimization_report.json"), jsonReport, 0644); err != nil {
		return fmt.Errorf("write json report: %w", err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "optimization_report.md"), []byte(BuildMarkdownReport(report)), 0644); err != nil {
		return fmt.Errorf("write markdown report: %w", err)
	}
	return nil
}

func evaluateSampleGate(
	cfg promptIterConfig,
	baselineScore float64,
	candidateScore float64,
	deltas []promptiterengine.CaseDelta,
) *promptiterengine.AcceptanceDecision {
	scoreDelta := candidateScore - baselineScore
	gates := []promptiterengine.GateResult{{
		GateName: "ValidationScoreGain",
		Passed:   scoreDelta >= cfg.MinScoreGain,
		Reason:   fmt.Sprintf("scoreDelta=%.4f, threshold=%.4f", scoreDelta, cfg.MinScoreGain),
	}}
	reasons := make([]string, 0)
	if !gates[0].Passed {
		reasons = append(reasons, "validation score gain insufficient")
	}
	if cfg.NoNewHardFail {
		newFails := caseIDsByType(deltas, promptiterengine.CaseDeltaNewlyFailed)
		passed := len(newFails) == 0
		gates = append(gates, promptiterengine.GateResult{
			GateName: "NoNewHardFail",
			Passed:   passed,
			Reason:   "newly failed cases: " + strings.Join(newFails, ","),
		})
		if !passed {
			reasons = append(reasons, "newly failed cases: "+strings.Join(newFails, ","))
		}
	}
	if len(cfg.CriticalCaseIDs) > 0 {
		criticalRegressed := criticalRegressions(deltas, cfg.CriticalCaseIDs)
		passed := len(criticalRegressed) == 0
		gates = append(gates, promptiterengine.GateResult{
			GateName: "CriticalCasePreserve",
			Passed:   passed,
			Reason:   "regressed critical cases: " + strings.Join(criticalRegressed, ","),
		})
		if !passed {
			reasons = append(reasons, "critical case regressed: "+strings.Join(criticalRegressed, ","))
		}
	}
	budgetPassed := cfg.MaxAPICalls == 0 || 12 <= cfg.MaxAPICalls
	gates = append(gates, promptiterengine.GateResult{
		GateName: "BudgetConstraint",
		Passed:   budgetPassed,
		Reason:   fmt.Sprintf("apiCalls=%d/%d", 12, cfg.MaxAPICalls),
	})
	if !budgetPassed {
		reasons = append(reasons, "budget exceeded")
	}
	accepted := len(reasons) == 0
	reason := "all gates passed"
	if !accepted {
		reason = strings.Join(reasons, "; ")
	}
	return &promptiterengine.AcceptanceDecision{
		Accepted:    accepted,
		ScoreDelta:  scoreDelta,
		Reason:      reason,
		GateResults: gates,
	}
}

func caseIDsByType(deltas []promptiterengine.CaseDelta, deltaType promptiterengine.CaseDeltaType) []string {
	var ids []string
	for _, delta := range deltas {
		if delta.Type == deltaType {
			ids = append(ids, delta.CaseID)
		}
	}
	return ids
}

func criticalRegressions(deltas []promptiterengine.CaseDelta, criticalCaseIDs []string) []string {
	critical := make(map[string]struct{}, len(criticalCaseIDs))
	for _, caseID := range criticalCaseIDs {
		critical[caseID] = struct{}{}
	}
	var ids []string
	for _, delta := range deltas {
		if _, ok := critical[delta.CaseID]; !ok {
			continue
		}
		if delta.Type == promptiterengine.CaseDeltaNewlyFailed || delta.Type == promptiterengine.CaseDeltaScoreRegressed {
			ids = append(ids, delta.CaseID)
		}
	}
	return ids
}

func sampleBaselineValidation() *promptiterengine.EvaluationResult {
	return evalResultFromCases("promptiter-regression-validation", []promptiterengine.CaseResult{
		sampleCase("validation_prompt_fixable", 0, status.EvalStatusFailed, "final_response_exact_json", "final response mismatch"),
		sampleCase("validation_no_effect", 0, status.EvalStatusFailed, "tool_trajectory_avg_score", "arguments mismatch: expected V-200"),
		sampleCase("validation_overfit_guard", 1, status.EvalStatusPassed, "final_response_exact_json", ""),
	})
}

func sampleCandidateValidation() *promptiterengine.EvaluationResult {
	return evalResultFromCases("promptiter-regression-validation", []promptiterengine.CaseResult{
		sampleCase("validation_prompt_fixable", 1, status.EvalStatusPassed, "final_response_exact_json", ""),
		sampleCase("validation_no_effect", 0, status.EvalStatusFailed, "tool_trajectory_avg_score", "arguments mismatch: expected V-200"),
		sampleCase("validation_overfit_guard", 0.6, status.EvalStatusFailed, "final_response_exact_json", "overfit regression"),
	})
}

func evalResultFromCases(evalSetID string, cases []promptiterengine.CaseResult) *promptiterengine.EvaluationResult {
	total := 0.0
	for _, c := range cases {
		for _, metric := range c.Metrics {
			total += metric.Score
		}
	}
	score := 0.0
	if len(cases) > 0 {
		score = total / float64(len(cases))
	}
	return &promptiterengine.EvaluationResult{
		OverallScore: score,
		EvalSets: []promptiterengine.EvalSetResult{{
			EvalSetID:    evalSetID,
			OverallScore: score,
			Cases:        cases,
		}},
	}
}

func sampleCase(
	caseID string,
	score float64,
	metricStatus status.EvalStatus,
	metricName string,
	reason string,
) promptiterengine.CaseResult {
	return promptiterengine.CaseResult{
		EvalSetID:  "promptiter-regression-validation",
		EvalCaseID: caseID,
		Metrics: []promptiterengine.MetricResult{{
			MetricName: metricName,
			Score:      score,
			Status:     metricStatus,
			Reason:     reason,
		}},
	}
}
