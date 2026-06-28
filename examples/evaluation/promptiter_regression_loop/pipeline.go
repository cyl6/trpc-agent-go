//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"trpc.group/trpc-go/trpc-agent-go/evaluation/status"
	"trpc.group/trpc-go/trpc-agent-go/evaluation/workflow/promptiter"
	promptiterengine "trpc.group/trpc-go/trpc-agent-go/evaluation/workflow/promptiter/engine"
)

const pipelineVersion = "promptiter-regression-loop/v1"

var deterministicGeneratedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

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
	artifacts, err := runPromptIter(context.Background(), promptiterCfg, cfg.ConfigDir)
	if err != nil {
		return nil, err
	}
	if len(artifacts.Result.Rounds) == 0 {
		return nil, fmt.Errorf("promptiter run produced no rounds")
	}
	lastRound := artifacts.Result.Rounds[len(artifacts.Result.Rounds)-1]
	baseline := artifacts.Result.BaselineValidation
	candidate := lastRound.Validation
	deltas := promptiterengine.CompareCaseDeltas(baseline, candidate)
	attributions := NewFailureAttributor().Attribute(baseline)
	decision := lastRound.Acceptance
	report := OptimizationReport{
		Metadata: ReportMetadata{
			GeneratedAt:      deterministicGeneratedAt,
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
		CostLatency: CostLatencySummary{
			TotalCost:          artifacts.Usage.Cost,
			TotalAPICalls:      artifacts.Usage.APICalls,
			TotalLatencyMillis: artifacts.Usage.Latency.Milliseconds(),
		},
		Rounds: buildRoundSnapshots(baseline, artifacts.Result.Rounds),
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

func buildRoundSnapshots(
	baseline *promptiterengine.EvaluationResult,
	rounds []promptiterengine.RoundResult,
) []RoundSnapshot {
	snapshots := make([]RoundSnapshot, 0, len(rounds))
	acceptedValidation := baseline
	for _, round := range rounds {
		deltas := promptiterengine.CompareCaseDeltas(acceptedValidation, round.Validation)
		snapshots = append(snapshots, RoundSnapshot{
			RoundID:              round.Round,
			CandidatePrompt:      profilePrompt(round.OutputProfile),
			TrainEvalResult:      round.Train,
			ValidationEvalResult: round.Validation,
			CaseDeltas:           deltas,
			Acceptance:           round.Acceptance,
		})
		if round.Acceptance != nil && round.Acceptance.Accepted {
			acceptedValidation = round.Validation
		}
	}
	return snapshots
}

func profilePrompt(profile *promptiter.Profile) string {
	if profile == nil {
		return ""
	}
	for _, override := range profile.Overrides {
		if override.SurfaceID == regressionSurfaceID && override.Value.Text != nil {
			return *override.Value.Text
		}
	}
	return ""
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
