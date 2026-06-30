//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-agent-go/evaluation/status"
	promptiterengine "trpc.group/trpc-go/trpc-agent-go/evaluation/workflow/promptiter/engine"
)

func TestFailureAttributorClassifiesSixRequiredCategories(t *testing.T) {
	result := &promptiterengine.EvaluationResult{EvalSets: []promptiterengine.EvalSetResult{{
		EvalSetID: "validation",
		Cases: []promptiterengine.CaseResult{
			failedCase("final", "final_response_exact_json", "final response does not match expected JSON"),
			failedCase("tool_name", "tool_trajectory_avg_score", "name mismatch: expected account_lookup got cache_read"),
			failedCase("tool_args", "tool_trajectory_avg_score", "arguments mismatch: expected account_id A-100"),
			failedCase("route", "route_transfer", "unexpected transfer to fallback_agent"),
			failedCase("format", "llm_rubric_json_grounding", "format_json failed: answer is prose"),
			failedCase("knowledge", "llm_rubric_json_grounding", "knowledge_recall failed: changed account id"),
		},
	}}}

	attributions := NewFailureAttributor().Attribute(result)

	byCase := map[string]FailureCategory{}
	for _, attribution := range attributions {
		byCase[attribution.CaseID] = attribution.Category
		assert.Equal(t, "validation", attribution.EvalSetID)
		assert.NotEmpty(t, attribution.Reason)
		assert.NotEmpty(t, attribution.Evidence)
	}
	assert.Equal(t, FailureCategoryFinalResponseMismatch, byCase["final"])
	assert.Equal(t, FailureCategoryToolCallError, byCase["tool_name"])
	assert.Equal(t, FailureCategoryToolArgError, byCase["tool_args"])
	assert.Equal(t, FailureCategoryRouteError, byCase["route"])
	assert.Equal(t, FailureCategoryFormatError, byCase["format"])
	assert.Equal(t, FailureCategoryKnowledgeRecall, byCase["knowledge"])
}

func failedCase(caseID, metricName, reason string) promptiterengine.CaseResult {
	return promptiterengine.CaseResult{
		EvalSetID:  "validation",
		EvalCaseID: caseID,
		Metrics: []promptiterengine.MetricResult{{
			MetricName: metricName,
			Score:      0,
			Status:     status.EvalStatusFailed,
			Reason:     reason,
		}},
	}
}
