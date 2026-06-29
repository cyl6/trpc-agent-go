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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"trpc.group/trpc-go/trpc-agent-go/agent"
	astructure "trpc.group/trpc-go/trpc-agent-go/agent/structure"
	atrace "trpc.group/trpc-go/trpc-agent-go/agent/trace"
	"trpc.group/trpc-go/trpc-agent-go/evaluation"
	"trpc.group/trpc-go/trpc-agent-go/evaluation/evalresult"
	"trpc.group/trpc-go/trpc-agent-go/evaluation/status"
	"trpc.group/trpc-go/trpc-agent-go/evaluation/workflow/promptiter"
	"trpc.group/trpc-go/trpc-agent-go/evaluation/workflow/promptiter/aggregator"
	"trpc.group/trpc-go/trpc-agent-go/evaluation/workflow/promptiter/backwarder"
	promptiterengine "trpc.group/trpc-go/trpc-agent-go/evaluation/workflow/promptiter/engine"
	"trpc.group/trpc-go/trpc-agent-go/evaluation/workflow/promptiter/optimizer"
	"trpc.group/trpc-go/trpc-agent-go/event"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/tool"
)

const (
	regressionAppName        = "promptiter-regression-loop-app"
	regressionNodeID         = "candidate"
	regressionSurfaceID      = "candidate#instruction"
	regressionTrainEvalSetID = "promptiter-regression-train"
	regressionValidEvalSetID = "promptiter-regression-validation"
)

type promptIterRunArtifacts struct {
	Result *promptiterengine.RunResult
	Usage  promptiterengine.BudgetUsage
}

func runPromptIter(ctx context.Context, cfg promptIterConfig, configDir string) (*promptIterRunArtifacts, error) {
	baselinePrompt, err := loadBaselinePrompt(filepath.Join(configDir, "baseline_prompt.txt"))
	if err != nil {
		return nil, err
	}
	scenario, err := loadRegressionScenario(configDir, cfg)
	if err != nil {
		return nil, err
	}
	modelQueues, err := loadFakeModelQueue(filepath.Join(configDir, "fake_model_queue.json"))
	if err != nil {
		return nil, err
	}
	optimizerQueue, ok := modelQueues["optimizer"]
	if !ok || len(optimizerQueue) == 0 {
		return nil, errors.New("fake model queue missing optimizer responses")
	}
	optimizerModel := NewFakeModel("optimizer", optimizerQueue)
	engineInstance, err := promptiterengine.New(
		ctx,
		&regressionStructureAgent{baselinePrompt: baselinePrompt},
		newScriptedAgentEvaluator(scenario),
		&deterministicBackwarder{},
		&deterministicAggregator{},
		&deterministicOptimizer{model: optimizerModel},
	)
	if err != nil {
		return nil, fmt.Errorf("create promptiter engine: %w", err)
	}
	targetScore := cfg.TargetScore
	usage := promptiterengine.BudgetUsage{APICalls: 0, Latency: time.Millisecond}
	result, err := engineInstance.Run(ctx, &promptiterengine.RunRequest{
		Train: []promptiterengine.EvalSetInput{{
			EvalSetID: regressionTrainEvalSetID,
		}},
		Validation: []promptiterengine.EvalSetInput{{
			EvalSetID: regressionValidEvalSetID,
		}},
		AcceptancePolicy: promptiterengine.AcceptancePolicy{
			MinScoreGain:    cfg.MinScoreGain,
			NoNewHardFail:   cfg.NoNewHardFail,
			CriticalCaseIDs: cfg.CriticalCaseIDs,
			BudgetConstraint: &promptiterengine.BudgetLimit{
				MaxAPICalls: cfg.MaxAPICalls,
			},
		},
		StopPolicy: promptiterengine.StopPolicy{
			MaxRoundsWithoutAcceptance: 1,
			TargetScore:                &targetScore,
		},
		MaxRounds:        cfg.MaxRounds,
		TargetSurfaceIDs: []string{regressionSurfaceID},
	}, promptiterengine.WithBudgetUsageProvider(func() promptiterengine.BudgetUsage {
		usage.APICalls = 11 + optimizerModel.CallCount()
		return usage
	}))
	if err != nil {
		return nil, fmt.Errorf("run promptiter engine: %w", err)
	}
	return &promptIterRunArtifacts{Result: result, Usage: usage}, nil
}

func loadBaselinePrompt(path string) (string, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("read baseline prompt: %w", err)
	}
	return string(data), nil
}

type queuedModelResponse struct {
	Content string `json:"content"`
}

func loadFakeModelQueue(path string) (map[string][][]*model.Response, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("read fake model queue: %w", err)
	}
	var raw map[string][][]queuedModelResponse
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode fake model queue: %w", err)
	}
	queues := make(map[string][][]*model.Response, len(raw))
	for name, calls := range raw {
		queues[name] = make([][]*model.Response, 0, len(calls))
		for _, call := range calls {
			responses := make([]*model.Response, 0, len(call))
			for _, response := range call {
				responses = append(responses, fakeTextResponse(response.Content))
			}
			queues[name] = append(queues[name], responses)
		}
	}
	return queues, nil
}

func fakeTextResponse(content string) *model.Response {
	finish := "stop"
	return &model.Response{
		Object: model.ObjectTypeChatCompletion,
		Model:  "fake",
		Choices: []model.Choice{{
			Index: 0,
			Message: model.Message{
				Role:    model.RoleAssistant,
				Content: content,
			},
			FinishReason: &finish,
		}},
		Done: true,
	}
}

type regressionStructureAgent struct {
	baselinePrompt string
}

func (a *regressionStructureAgent) Run(
	ctx context.Context,
	invocation *agent.Invocation,
) (<-chan *event.Event, error) {
	_ = ctx
	_ = invocation
	return nil, errors.New("regression structure agent is used for PromptIter structure export only")
}

func (a *regressionStructureAgent) Tools() []tool.Tool {
	return nil
}

func (a *regressionStructureAgent) Info() agent.Info {
	return agent.Info{Name: regressionNodeID}
}

func (a *regressionStructureAgent) SubAgents() []agent.Agent {
	return nil
}

func (a *regressionStructureAgent) FindSubAgent(name string) agent.Agent {
	_ = name
	return nil
}

func (a *regressionStructureAgent) Export(
	ctx context.Context,
	exportChild astructure.ChildExporter,
) (*astructure.Snapshot, error) {
	_ = ctx
	_ = exportChild
	return &astructure.Snapshot{
		StructureID: "promptiter-regression-loop-structure",
		EntryNodeID: regressionNodeID,
		Nodes: []astructure.Node{{
			NodeID: regressionNodeID,
			Kind:   astructure.NodeKindLLM,
			Name:   "candidate",
		}},
		Surfaces: []astructure.Surface{{
			SurfaceID: regressionSurfaceID,
			NodeID:    regressionNodeID,
			Type:      astructure.SurfaceTypeInstruction,
			Value:     astructure.SurfaceValue{Text: stringPtr(a.baselinePrompt)},
		}},
	}, nil
}

type scriptedAgentEvaluator struct {
	scenario        *regressionScenario
	mu              sync.Mutex
	validationCalls int
}

func newScriptedAgentEvaluator(scenario *regressionScenario) evaluation.AgentEvaluator {
	return &scriptedAgentEvaluator{scenario: scenario}
}

func (e *scriptedAgentEvaluator) Evaluate(
	ctx context.Context,
	evalSetID string,
	opt ...evaluation.Option,
) (*evaluation.EvaluationResult, error) {
	_ = ctx
	_ = opt
	if e.scenario == nil {
		return nil, errors.New("regression scenario is nil")
	}
	switch evalSetID {
	case regressionTrainEvalSetID:
		return genericEvaluationFromEngineResult(e.scenario.Train), nil
	case regressionValidEvalSetID:
		e.mu.Lock()
		e.validationCalls++
		call := e.validationCalls
		e.mu.Unlock()
		if call == 1 {
			return genericEvaluationFromEngineResult(e.scenario.BaselineValidation), nil
		}
		return genericEvaluationFromEngineResult(e.scenario.CandidateValidation), nil
	default:
		return nil, fmt.Errorf("unexpected eval set %q", evalSetID)
	}
}

func (e *scriptedAgentEvaluator) Close() error {
	return nil
}

type deterministicBackwarder struct{}

func (b *deterministicBackwarder) Backward(
	ctx context.Context,
	request *backwarder.Request,
) (*backwarder.Result, error) {
	_ = ctx
	return &backwarder.Result{
		Gradients: []promptiter.SurfaceGradient{{
			EvalSetID:  request.EvalSetID,
			EvalCaseID: request.EvalCaseID,
			StepID:     request.StepID,
			SurfaceID:  regressionSurfaceID,
			Severity:   promptiter.LossSeverityP1,
			Gradient:   "Require compact JSON and preserve lookup/cache facts.",
		}},
	}, nil
}

type deterministicAggregator struct{}

func (a *deterministicAggregator) Aggregate(
	ctx context.Context,
	request *aggregator.Request,
) (*aggregator.Result, error) {
	_ = ctx
	return &aggregator.Result{
		Gradient: &promptiter.AggregatedSurfaceGradient{
			SurfaceID: request.SurfaceID,
			NodeID:    request.NodeID,
			Type:      request.Type,
			Gradients: append([]promptiter.SurfaceGradient(nil), request.Gradients...),
		},
	}, nil
}

type deterministicOptimizer struct {
	model *FakeModel
}

func (o *deterministicOptimizer) Optimize(
	ctx context.Context,
	request *optimizer.Request,
) (*optimizer.Result, error) {
	if o.model == nil {
		return nil, errors.New("optimizer fake model is nil")
	}
	responses, err := drainFakeResponses(ctx, o.model)
	if err != nil {
		return nil, err
	}
	patch, err := parseOptimizerPatch(responses)
	if err != nil {
		return nil, err
	}
	if patch.SurfaceID != request.Surface.SurfaceID {
		return nil, fmt.Errorf("optimizer patch surface %q does not match requested surface %q", patch.SurfaceID, request.Surface.SurfaceID)
	}
	return &optimizer.Result{
		Patch: patch,
	}, nil
}

func drainFakeResponses(ctx context.Context, fake *FakeModel) ([]*model.Response, error) {
	ch, err := fake.GenerateContent(ctx, &model.Request{})
	if err != nil {
		return nil, err
	}
	var responses []*model.Response
	for response := range ch {
		responses = append(responses, response)
	}
	return responses, nil
}

type optimizerPatchPayload struct {
	Patches []struct {
		SurfaceID string `json:"surface_id"`
		Value     struct {
			Text string `json:"text"`
		} `json:"value"`
		Reason string `json:"reason"`
	} `json:"patches"`
}

func parseOptimizerPatch(responses []*model.Response) (*promptiter.SurfacePatch, error) {
	if len(responses) == 0 || len(responses[0].Choices) == 0 {
		return nil, errors.New("optimizer fake model returned no patch response")
	}
	content := responses[0].Choices[0].Message.Content
	var payload optimizerPatchPayload
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return nil, fmt.Errorf("decode optimizer patch response: %w", err)
	}
	if len(payload.Patches) == 0 {
		return nil, errors.New("optimizer patch response has no patches")
	}
	patch := payload.Patches[0]
	if patch.SurfaceID == "" {
		return nil, errors.New("optimizer patch surface id is empty")
	}
	if patch.Value.Text == "" {
		return nil, errors.New("optimizer patch text is empty")
	}
	return &promptiter.SurfacePatch{
		SurfaceID: patch.SurfaceID,
		Value:     astructure.SurfaceValue{Text: stringPtr(patch.Value.Text)},
		Reason:    patch.Reason,
	}, nil
}

type regressionScenario struct {
	Train               *promptiterengine.EvaluationResult
	BaselineValidation  *promptiterengine.EvaluationResult
	CandidateValidation *promptiterengine.EvaluationResult
}

type regressionMetricNames struct {
	FinalResponse  string
	ToolTrajectory string
}

type scenarioEvalSetFile struct {
	EvalSetID string               `json:"evalSetId"`
	EvalCases []regressionEvalCase `json:"evalCases"`
}

type regressionEvalCase struct {
	EvalID             string                   `json:"evalId"`
	Conversation       []regressionConversation `json:"conversation"`
	ActualConversation []regressionConversation `json:"actualConversation"`
}

type regressionConversation struct {
	FinalResponse regressionMessage `json:"finalResponse"`
	Tools         []regressionTool  `json:"tools"`
}

type regressionMessage struct {
	Content string `json:"content"`
}

type regressionTool struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
	Result    map[string]any `json:"result"`
}

func loadRegressionScenario(configDir string, cfg promptIterConfig) (*regressionScenario, error) {
	metrics, err := loadRegressionMetricNames(filepath.Join(configDir, "metrics.json"))
	if err != nil {
		return nil, err
	}
	trainSet, err := loadRegressionEvalSet(filepath.Join(configDir, "train.evalset.json"))
	if err != nil {
		return nil, err
	}
	validationSet, err := loadRegressionEvalSet(filepath.Join(configDir, "validation.evalset.json"))
	if err != nil {
		return nil, err
	}
	train := buildBaselineEvaluation(trainSet, metrics)
	baselineValidation := buildBaselineEvaluation(validationSet, metrics)
	candidateValidation := buildCandidateValidation(validationSet, baselineValidation, metrics, cfg.CriticalCaseIDs)
	return &regressionScenario{
		Train:               train,
		BaselineValidation:  baselineValidation,
		CandidateValidation: candidateValidation,
	}, nil
}

type metricFileEntry struct {
	MetricName string                     `json:"metricName"`
	Criterion  map[string]json.RawMessage `json:"criterion"`
}

func loadRegressionMetricNames(path string) (regressionMetricNames, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return regressionMetricNames{}, fmt.Errorf("read metrics config: %w", err)
	}
	var entries []metricFileEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return regressionMetricNames{}, fmt.Errorf("decode metrics config: %w", err)
	}
	var names regressionMetricNames
	for _, entry := range entries {
		if _, ok := entry.Criterion["finalResponse"]; ok && names.FinalResponse == "" {
			names.FinalResponse = entry.MetricName
		}
		if _, ok := entry.Criterion["toolTrajectory"]; ok && names.ToolTrajectory == "" {
			names.ToolTrajectory = entry.MetricName
		}
	}
	if names.FinalResponse == "" || names.ToolTrajectory == "" {
		return regressionMetricNames{}, errors.New("metrics config must include finalResponse and toolTrajectory metrics")
	}
	return names, nil
}

func loadRegressionEvalSet(path string) (*scenarioEvalSetFile, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("read evalset config: %w", err)
	}
	var evalSet scenarioEvalSetFile
	if err := json.Unmarshal(data, &evalSet); err != nil {
		return nil, fmt.Errorf("decode evalset config: %w", err)
	}
	if evalSet.EvalSetID == "" {
		return nil, errors.New("evalset id is empty")
	}
	return &evalSet, nil
}

func buildBaselineEvaluation(evalSet *scenarioEvalSetFile, metrics regressionMetricNames) *promptiterengine.EvaluationResult {
	cases := make([]promptiterengine.CaseResult, 0, len(evalSet.EvalCases))
	for _, evalCase := range evalSet.EvalCases {
		cases = append(cases, baselineCaseResult(evalSet.EvalSetID, evalCase, metrics))
	}
	return evalResultFromCases(evalSet.EvalSetID, cases)
}

func baselineCaseResult(
	evalSetID string,
	evalCase regressionEvalCase,
	metrics regressionMetricNames,
) promptiterengine.CaseResult {
	expected, actual := firstConversation(evalCase.Conversation), firstConversation(evalCase.ActualConversation)
	switch {
	case !toolsEqual(expected.Tools, actual.Tools):
		return sampleCaseForSet(evalSetID, evalCase.EvalID, 0, status.EvalStatusFailed, metrics.ToolTrajectory, toolMismatchReason(expected.Tools))
	case expected.FinalResponse.Content != actual.FinalResponse.Content:
		return sampleCaseForSet(evalSetID, evalCase.EvalID, 0, status.EvalStatusFailed, metrics.FinalResponse, "final response mismatch")
	default:
		return sampleCaseForSet(evalSetID, evalCase.EvalID, 1, status.EvalStatusPassed, metrics.FinalResponse, "")
	}
}

func buildCandidateValidation(
	evalSet *scenarioEvalSetFile,
	baseline *promptiterengine.EvaluationResult,
	metrics regressionMetricNames,
	criticalCaseIDs []string,
) *promptiterengine.EvaluationResult {
	critical := make(map[string]struct{}, len(criticalCaseIDs))
	for _, caseID := range criticalCaseIDs {
		critical[caseID] = struct{}{}
	}
	baselineCases := baselineCaseIndex(baseline)
	cases := make([]promptiterengine.CaseResult, 0, len(evalSet.EvalCases))
	for _, evalCase := range evalSet.EvalCases {
		baselineCase := baselineCases[evalCase.EvalID]
		baselineStatus := summarizeCaseStatus(baselineCase.Metrics)
		if _, ok := critical[evalCase.EvalID]; ok && baselineStatus == status.EvalStatusPassed {
			cases = append(cases, sampleCaseForSet(evalSet.EvalSetID, evalCase.EvalID, 0.6, status.EvalStatusFailed, metrics.FinalResponse, "overfit regression"))
			continue
		}
		if len(baselineCase.Metrics) > 0 && baselineCase.Metrics[0].MetricName == metrics.FinalResponse &&
			baselineCase.Metrics[0].Status == status.EvalStatusFailed {
			cases = append(cases, sampleCaseForSet(evalSet.EvalSetID, evalCase.EvalID, 1, status.EvalStatusPassed, metrics.FinalResponse, ""))
			continue
		}
		cases = append(cases, baselineCase)
	}
	return evalResultFromCases(evalSet.EvalSetID, cases)
}

func firstConversation(conversations []regressionConversation) regressionConversation {
	if len(conversations) == 0 {
		return regressionConversation{}
	}
	return conversations[0]
}

func toolsEqual(expected, actual []regressionTool) bool {
	if len(expected) != len(actual) {
		return false
	}
	for i := range expected {
		if expected[i].Name != actual[i].Name ||
			!reflect.DeepEqual(expected[i].Arguments, actual[i].Arguments) ||
			!reflect.DeepEqual(expected[i].Result, actual[i].Result) {
			return false
		}
	}
	return true
}

func toolMismatchReason(expected []regressionTool) string {
	if len(expected) == 0 {
		return "tool trajectory mismatch"
	}
	if accountID, ok := expected[0].Arguments["account_id"].(string); ok && accountID != "" {
		return "arguments mismatch: expected " + accountID
	}
	return "arguments mismatch"
}

func baselineCaseIndex(result *promptiterengine.EvaluationResult) map[string]promptiterengine.CaseResult {
	index := make(map[string]promptiterengine.CaseResult)
	if result == nil {
		return index
	}
	for _, evalSet := range result.EvalSets {
		for _, c := range evalSet.Cases {
			index[c.EvalCaseID] = c
		}
	}
	return index
}

func genericEvaluationFromEngineResult(result *promptiterengine.EvaluationResult) *evaluation.EvaluationResult {
	if result == nil || len(result.EvalSets) == 0 {
		return &evaluation.EvaluationResult{}
	}
	evalSet := result.EvalSets[0]
	evalCases := make([]*evaluation.EvaluationCaseResult, 0, len(evalSet.Cases))
	overallStatus := status.EvalStatusPassed
	for _, c := range evalSet.Cases {
		caseStatus := summarizeCaseStatus(c.Metrics)
		if caseStatus == status.EvalStatusFailed {
			overallStatus = status.EvalStatusFailed
		}
		metricResults := make([]*evalresult.EvalMetricResult, 0, len(c.Metrics))
		for _, metric := range c.Metrics {
			metricResults = append(metricResults, &evalresult.EvalMetricResult{
				MetricName: metric.MetricName,
				Score:      metric.Score,
				EvalStatus: metric.Status,
				Details: &evalresult.EvalMetricResultDetails{
					Reason: metric.Reason,
					Score:  metric.Score,
				},
			})
		}
		runResult := &evalresult.EvalCaseResult{
			EvalSetID:                c.EvalSetID,
			EvalID:                   c.EvalCaseID,
			RunID:                    1,
			FinalEvalStatus:          caseStatus,
			OverallEvalMetricResults: metricResults,
			SessionID:                "session_" + c.EvalCaseID,
			UserID:                   "demo-user",
		}
		evalCases = append(evalCases, &evaluation.EvaluationCaseResult{
			EvalCaseID:      c.EvalCaseID,
			OverallStatus:   caseStatus,
			MetricResults:   metricResults,
			EvalCaseResults: []*evalresult.EvalCaseResult{runResult},
			RunDetails: []*evaluation.EvaluationCaseRunDetails{{
				RunID: 1,
				Inference: &evaluation.EvaluationInferenceDetails{
					SessionID:       runResult.SessionID,
					UserID:          runResult.UserID,
					Status:          status.EvalStatusPassed,
					ExecutionTraces: []*atrace.Trace{caseTrace(c)},
				},
			}},
		})
	}
	return &evaluation.EvaluationResult{
		AppName:       regressionAppName,
		EvalSetID:     evalSet.EvalSetID,
		OverallStatus: overallStatus,
		EvalCases:     evalCases,
		EvalResult: &evalresult.EvalSetResult{
			EvalSetID: evalSet.EvalSetID,
		},
	}
}

func caseTrace(c promptiterengine.CaseResult) *atrace.Trace {
	if c.Trace != nil {
		return c.Trace
	}
	sessionID := "session_" + c.EvalCaseID
	return &atrace.Trace{
		RootAgentName:    regressionNodeID,
		RootInvocationID: "invocation_" + c.EvalCaseID,
		SessionID:        sessionID,
		Status:           atrace.TraceStatusCompleted,
		Steps: []atrace.Step{{
			StepID:            "step_" + c.EvalCaseID,
			InvocationID:      "invocation_" + c.EvalCaseID,
			AgentName:         regressionNodeID,
			NodeID:            regressionNodeID,
			AppliedSurfaceIDs: []string{regressionSurfaceID},
			Input:             &atrace.Snapshot{Text: "input " + c.EvalCaseID},
			Output:            &atrace.Snapshot{Text: "output " + c.EvalCaseID},
		}},
	}
}

func sampleCaseForSet(
	evalSetID string,
	caseID string,
	score float64,
	metricStatus status.EvalStatus,
	metricName string,
	reason string,
) promptiterengine.CaseResult {
	c := sampleCase(caseID, score, metricStatus, metricName, reason)
	c.EvalSetID = evalSetID
	return c
}

func stringPtr(value string) *string {
	return &value
}
