//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package main

import (
	"strings"

	"trpc.group/trpc-go/trpc-agent-go/evaluation/status"
	promptiterengine "trpc.group/trpc-go/trpc-agent-go/evaluation/workflow/promptiter/engine"
)

// FailureCategory identifies a semantic failure class required by the challenge.
type FailureCategory string

const (
	FailureCategoryFinalResponseMismatch FailureCategory = "final_response_mismatch"
	FailureCategoryToolCallError         FailureCategory = "tool_call_error"
	FailureCategoryToolArgError          FailureCategory = "tool_arg_error"
	FailureCategoryRouteError            FailureCategory = "route_error"
	FailureCategoryFormatError           FailureCategory = "format_error"
	FailureCategoryKnowledgeRecall       FailureCategory = "knowledge_recall"
)

// FailureAttribution stores one explainable failure reason.
type FailureAttribution struct {
	EvalSetID string          `json:"eval_set_id"`
	CaseID    string          `json:"case_id"`
	Category  FailureCategory `json:"category"`
	Reason    string          `json:"reason"`
	Evidence  string          `json:"evidence"`
}

// FailureAttributor classifies failed metrics into challenge categories.
type FailureAttributor struct{}

// NewFailureAttributor creates a failure attributor.
func NewFailureAttributor() *FailureAttributor {
	return &FailureAttributor{}
}

// Attribute classifies failed cases in an engine evaluation result.
func (a *FailureAttributor) Attribute(result *promptiterengine.EvaluationResult) []FailureAttribution {
	if result == nil {
		return nil
	}
	attributions := make([]FailureAttribution, 0)
	for _, evalSet := range result.EvalSets {
		for _, caseResult := range evalSet.Cases {
			for _, metric := range caseResult.Metrics {
				if metric.Status != status.EvalStatusFailed {
					continue
				}
				attributions = append(attributions, FailureAttribution{
					EvalSetID: evalSetIDForCase(evalSet.EvalSetID, caseResult),
					CaseID:    caseResult.EvalCaseID,
					Category:  classifyMetric(metric),
					Reason:    metric.Reason,
					Evidence:  metric.MetricName,
				})
			}
		}
	}
	return attributions
}

func classifyMetric(metric promptiterengine.MetricResult) FailureCategory {
	name := strings.ToLower(metric.MetricName)
	reason := strings.ToLower(metric.Reason)
	switch {
	case strings.Contains(name, "route") || strings.Contains(reason, "transfer"):
		return FailureCategoryRouteError
	case strings.Contains(name, "tool"):
		if strings.Contains(reason, "argument") || strings.Contains(reason, "arguments") {
			return FailureCategoryToolArgError
		}
		return FailureCategoryToolCallError
	case strings.Contains(name, "final_response") || strings.Contains(name, "rouge"):
		return FailureCategoryFinalResponseMismatch
	case strings.Contains(reason, "format") || strings.Contains(reason, "schema") || strings.Contains(reason, "json"):
		return FailureCategoryFormatError
	case strings.Contains(reason, "knowledge") || strings.Contains(reason, "recall") || strings.Contains(reason, "fact"):
		return FailureCategoryKnowledgeRecall
	default:
		return FailureCategoryFinalResponseMismatch
	}
}

func evalSetIDForCase(evalSetID string, caseResult promptiterengine.CaseResult) string {
	if caseResult.EvalSetID != "" {
		return caseResult.EvalSetID
	}
	return evalSetID
}
