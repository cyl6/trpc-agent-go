# trpc-agent-go 项目文档续写计划

## 一、任务摘要

承接上一阶段工作，继续撰写 [`PROJECT_DEEP_DIVE.zh.md`](file:///home/cyl/agent/trpc-agent-go/PROJECT_DEEP_DIVE.zh.md) 的剩余部分（第 7-10 部分 + 附录），完成一份深度中文项目文档，覆盖 Evaluation 子系统、高并发实现、赛题方案设计三大板块，并补全文件超链接索引与构建测试速查。**本任务只写文档，不修改任何业务代码。**

## 二、当前状态分析

### 已完成（第 1-6 部分，第 1-16 章，约 1256 行）

| 部分 | 章节 | 主题 |
|------|------|------|
| 一 | 1 | 项目总览与 monorepo 结构 |
| 二 | 2-6 | Agent 运行时（Agent Loop、Invocation、Event、Callbacks、Plugins、ExecutionTrace、StreamHub、LLMAgent） |
| 三 | 7-10 | Skills & Tools（SkillRepository、Tool 框架、MCP、内置工具） |
| 四 | 11 | Sandbox / CodeExecutor |
| 五 | 12-15 | Memory & 数据库（Memory 接口、SQLite/MySQL/Postgres/Redis/向量、Session 层、记忆↔会话耦合） |
| 六 | 16 | Prompt 与 Surface 抽象 |

文档末尾位于 **第 1256 行**，以 HTML 注释标记 `<!-- 文档正文将在后续章节继续 -->` 作为续写锚点。

### 关键代码细节已校验（通过两个 search subagent 确认）

1. **Evaluation 子系统**：
   - `evaluation/evaluation.go` L42-47 `AgentEvaluator` 接口；L186-214 `Evaluate` 主流程；L529-588 `aggregateCaseRuns` 多 run 聚合
   - `evaluation/pass.go` L97-139 `PassAtK`（log-space + `-math.Expm1`）；L217-242 `PassHatK`；L245-258 `ParsePassNC`
   - `evaluation/evalset/evalcase.go` L19-26 `EvalModeTrace` 常量（**澄清：不在 agent/trace/**）
   - `evaluation/metric/criterion/tooltrajectory/tooltrajectory.go` L128-148 `unorderedMatch` 使用 `kuhn.New` + `FullLeftMatch`（**二部图最大匹配，非加权 Kuhn-Munkres**）
   - `internal/kuhn/kuhn.go` L18-108 Kuhn 算法实现
   - `evaluation/metric/criterion/llm/llm.go` L22-29 `LLMCriterion`；L45-48 `JudgeRunnerOptions`（运行时注入，JSON 标签 `-`）
   - `evaluation/service/local/local.go` L48-67 `local` 结构体；L215-258 `Evaluate`；ants 协程池并行
   - `evaluation/workflow/promptiter/engine/engine.go` L208-329 `run` 主循环；L372-495 `executeRound`
   - **`evaluation/workflow/promptiter/engine/accept.go` L35 `Accepted: scoreDelta >= policy.MinScoreGain`**（单维 gate，关键 gap 点）
   - `evaluation/workflow/promptiter/engine/loss.go` L30-82 `loss`；L207-254 `traceTerminalStepIDs`
   - `evaluation/workflow/promptiter/manager/observer.go` L23-215 Observer 持久化
   - `test/mock_model.go` L23-53 `QueueModel`（每次回放整个队列，非 FIFO）
   - `examples/evaluation/promptiter/{syncrun,asyncrun,server,multinode}/main.go` 四种部署形态

2. **高并发实现**：
   - `graph/executor.go` L2584-2633 `executeStep` worker pool + `sync.WaitGroup`；L2678-2719 `executeStepTask` 内置 `recover()` panic 隔离
   - `graph/executor_dag.go` L28-56 `dagLoop` 信号量 `sem chan struct{}` + `done chan dagTaskResult`
   - `graph/graph.go` L487-535 `ExecutionContext` 含 5 把锁 + `seq atomic.Int64` 用于确定性重放
   - `graph/checkpoint.go` L458-471 `Fork`；L762-852 `BranchToNewLineage`
   - `graph/internal/channel/channel.go` 4 种 Behavior（LastValue/Topic/Ephemeral/Barrier）
   - `runner/runner.go` L300-323 `runner` 结构体；L1286-1370 `runEventLoop` 三路 select
   - `runner/ralph_loop.go` L231-282 `runLoop` 外层循环；三种停止条件
   - `runner/candidate_selector.go` L225-265 `runAttemptsParallel` 用 `errgroup.Group` + `SetLimit`
   - `runner/bestofn/evaluation.go` pointwise/pairwise 两种选择模式
   - `team/team.go` L32-49 `Team`；两种 Mode（Coordinator/Swarm）
   - `team/runtime.go` L42-69 `OnTransfer` 安全限制（MaxHandoffs + 滑动窗口循环检测）
   - `model/hedge/hedge.go` L257-293 `handleEvent` first-meaningful-response；L381-388 `cancelLosers`

### 待完成工作

| 部分 | 章节 | 主题 | 预估行数 |
|------|------|------|---------|
| 七 | 17-21 | Evaluation 子系统 | ~500 |
| 八 | 22-26 | 高并发实现 | ~450 |
| 九 | 27-31 | 赛题方案设计（不修改代码，仅方案） | ~600 |
| 十 | 附录 A/B/C | 文件超链接索引、数据流图、构建测试速查 | ~250 |

## 三、拟议变更（续写内容详细方案）

### 第 7 部分：Evaluation 子系统（第 17-21 章）

#### 第 17 章 Evaluation 服务核心
- **17.1 AgentEvaluator 接口与构造**：引用 [evaluation/evaluation.go](file:///home/cyl/agent/trpc-agent-go/evaluation/evaluation.go) L42-47 接口、L50-117 `New` 构造、L120-143 结构体字段
- **17.2 Evaluate 主流程**：L186-214 `Evaluate` → `collectCaseResults` → `summarizeOverallStatus`；`runEvaluation` L330-393 多 run 并行/串行
- **17.3 Option 系统**：[evaluation/options.go](file:///home/cyl/agent/trpc-agent-go/evaluation/options.go) L33-55 `options` 结构体、函数式选项
- **17.4 pass@k 与 pass^k**：[evaluation/pass.go](file:///home/cyl/agent/trpc-agent-go/evaluation/pass.go) L97-139 pass@k（log-space 防溢出）、L217-242 pass^k（可靠性）、L245-258 `ParsePassNC` 数据流
- **17.5 多 run 聚合**：[evaluation/evaluation.go](file:///home/cyl/agent/trpc-agent-go/evaluation/evaluation.go) L529-588 `aggregateCaseRuns`；`evaluation/internal/multirun/multirun.go` L24-52 `SummarizeMultiRun`

#### 第 18 章 评测数据结构与 Metric 框架
- **18.1 EvalSet / EvalCase / Invocation / Tool**：[evaluation/evalset/evalset.go](file:///home/cyl/agent/trpc-agent-go/evaluation/evalset/evalset.go) L20-31；[evalcase.go](file:///home/cyl/agent/trpc-agent-go/evaluation/evalset/evalcase.go) L19-26 `EvalMode`、L29-50 `EvalCase`、L97-114 `Invocation`、L117-122 `Tool`
- **18.2 EvalMetric 与 Criterion 容器**：[evaluation/metric/metric.go](file:///home/cyl/agent/trpc-agent-go/evaluation/metric/metric.go) L20-25 `EvalMetric`；[criterion/criterion.go](file:///home/cyl/agent/trpc-agent-go/evaluation/metric/criterion/criterion.go) L20-27 三类子准则组合
- **18.3 EvalResult 与 multi-run summary**：[evaluation/evalresult/evalresult.go](file:///home/cyl/agent/trpc-agent-go/evaluation/evalresult/evalresult.go) L23-36 `EvalSetResult`；[summary.go](file:///home/cyl/agent/trpc-agent-go/evaluation/evalresult/summary.go) L15-26 `EvalSetResultSummary`
- **18.4 本地 evaluator 服务**：[evaluation/service/local/local.go](file:///home/cyl/agent/trpc-agent-go/evaluation/service/local/local.go) L48-67 结构体、L215-258 `Evaluate`、ants 协程池、L382-515 `evaluatePerCase`

#### 第 19 章 Metric 实现详解
- **19.1 ROUGE**：[evaluation/metric/criterion/rouge/rouge.go](file:///home/cyl/agent/trpc-agent-go/evaluation/metric/criterion/rouge/rouge.go) L23-40 配置、L100-163 `Match`；底层 `evaluation/internal/rouge`
- **19.2 LLM Rubric**：[evaluation/metric/criterion/llm/llm.go](file:///home/cyl/agent/trpc-agent-go/evaluation/metric/criterion/llm/llm.go) L22-29 `LLMCriterion`、L45-48 `JudgeRunnerOptions`（运行时注入）、L112-133 `MarshalJSON` 清空 APIKey + `os.ExpandEnv`；模板变量绑定 actual/expected × userContent/finalResponse
- **19.3 Tool Trajectory**：[evaluation/metric/criterion/tooltrajectory/tooltrajectory.go](file:///home/cyl/agent/trpc-agent-go/evaluation/metric/criterion/tooltrajectory/tooltrajectory.go) L40-53 配置、L63-86 `Match`、L106-125 `orderedMatch`、L128-148 `unorderedMatch`；[internal/kuhn/kuhn.go](file:///home/cyl/agent/trpc-agent-go/internal/kuhn/kuhn.go) L18-108 二部图最大匹配（澄清：非加权 Kuhn-Munkres）

#### 第 20 章 PromptIter 优化引擎
- **20.1 Engine 接口与 RunRequest**：[evaluation/workflow/promptiter/engine/engine.go](file:///home/cyl/agent/trpc-agent-go/evaluation/workflow/promptiter/engine/engine.go) L32-37 `Engine`、L40-67 `RunRequest`
- **20.2 run 主循环**：L213-329 `run`（baseline validation → 循环 executeRound → 更新 acceptedProfile → stop 判定）
- **20.3 executeRound 七步**：L372-495（evaluate train → loss → backward → aggregate → optimize → applyPatchSet → evaluate validation → accept）
- **20.4 接受门禁（单维 gate）**：[accept.go](file:///home/cyl/agent/trpc-agent-go/evaluation/workflow/promptiter/engine/accept.go) **L33-37 `scoreDelta := candidateScore - baselineScore; Accepted: scoreDelta >= policy.MinScoreGain`** —— 标注为赛题 gap 点
- **20.5 StopPolicy**：[stop.go](file:///home/cyl/agent/trpc-agent-go/evaluation/workflow/promptiter/engine/stop.go) L13-52 三种停止条件
- **20.6 Loss 提取**：[loss.go](file:///home/cyl/agent/trpc-agent-go/evaluation/workflow/promptiter/engine/loss.go) L30-82 `loss`；L207-254 `traceTerminalStepIDs` 通过拓扑找终态 step
- **20.7 Observer 事件流**：[event.go](file:///home/cyl/agent/trpc-agent-go/evaluation/workflow/promptiter/engine/event.go) L17-84 12 种 EventKind
- **20.8 异步 Manager**：[evaluation/workflow/promptiter/manager/manager.go](file:///home/cyl/agent/trpc-agent-go/evaluation/workflow/promptiter/manager/manager.go) L29-38 接口、L144-190 `run`；[observer.go](file:///home/cyl/agent/trpc-agent-go/evaluation/workflow/promptiter/manager/observer.go) L23-215 增量持久化
- **20.9 四种部署形态示例**：[syncrun/main.go](file:///home/cyl/agent/trpc-agent-go/examples/evaluation/promptiter/syncrun/main.go)、[asyncrun/main.go](file:///home/cyl/agent/trpc-agent-go/examples/evaluation/promptiter/asyncrun/main.go)、[server/main.go](file:///home/cyl/agent/trpc-agent-go/examples/evaluation/promptiter/server/main.go)、[multinode/main.go](file:///home/cyl/agent/trpc-agent-go/examples/evaluation/promptiter/multinode/main.go)

#### 第 21 章 Fake Model / Trace Mode / 确定性运行
- **21.1 QueueModel**：[test/mock_model.go](file:///home/cyl/agent/trpc-agent-go/test/mock_model.go) L23-53（回放整个队列语义，非 FIFO）
- **21.2 EvalModeTrace**：[evaluation/evalset/evalcase.go](file:///home/cyl/agent/trpc-agent-go/evaluation/evalset/evalcase.go) L19-26（澄清：不在 agent/trace/）；[recorder/options.go](file:///home/cyl/agent/trpc-agent-go/evaluation/evalset/recorder/options.go) L74-75 `WithTraceModeEnabled`
- **21.3 Execution Trace**：[agent/execution_trace.go](file:///home/cyl/agent/trpc-agent-go/agent/execution_trace.go) L21-25 `WithExecutionTraceEnabled`、L112-147 `StartExecutionTraceStep`、L204-218 `BuildExecutionTrace`；[agent/trace/trace.go](file:///home/cyl/agent/trpc-agent-go/agent/trace/trace.go) L27-35 `Trace`、L38-52 `Step`（PredecessorStepIDs 用于终态定位）

### 第 8 部分：高并发实现（第 22-26 章）

#### 第 22 章 Graph 执行引擎
- **22.1 Graph 不可变结构**：[graph/graph.go](file:///home/cyl/agent/trpc-agent-go/graph/graph.go) L229-246 `Graph` + `sync.RWMutex`；L487-535 `ExecutionContext`（5 把锁 + `seq atomic.Int64`）
- **22.2 BSP 执行器**：[graph/executor.go](file:///home/cyl/agent/trpc-agent-go/graph/executor.go) L1020-1050 `runBspLoop`、L1052-1143 `runBspStep`、L2584-2633 `executeStep`（worker pool + WaitGroup）、L2678-2719 `executeStepTask`（recover panic 隔离）、L2659-2670 `workerCount`
- **22.3 DAG 执行器**：[graph/executor_dag.go](file:///home/cyl/agent/trpc-agent-go/graph/executor_dag.go) L28-56 `dagLoop`（`sem chan struct{}` 信号量 + `done chan dagTaskResult`）、L123-135 五阶段主循环、L283-305 `launchTask`
- **22.4 Checkpoint 与分支**：[graph/checkpoint.go](file:///home/cyl/agent/trpc-agent-go/graph/checkpoint.go) L458-471 `Fork`、L762-852 `BranchToNewLineage`、L1096-1117 `deepCopy`（JSON 防竞争）
- **22.5 Time Travel**：[graph/time_travel.go](file:///home/cyl/agent/trpc-agent-go/graph/time_travel.go) L209-293 `EditState`、L363-380 `isProtectedTimeTravelKey`
- **22.6 Pregel Channel**：[graph/internal/channel/channel.go](file:///home/cyl/agent/trpc-agent-go/graph/internal/channel/channel.go) L19-30 4 种 Behavior、L233-248 `ConsumeIfAvailable` 原子化
- **22.7 Cache 与 Retry**：[graph/cache.go](file:///home/cyl/agent/trpc-agent-go/graph/cache.go) L73-86 `InMemoryCache`（深拷贝隔离）；[graph/retry.go](file:///home/cyl/agent/trpc-agent-go/graph/retry.go) L54-87 指数退避 + crypto/rand jitter

#### 第 23 章 Runner 与 Best-of-N
- **23.1 Runner 接口体系**：[runner/runner.go](file:///home/cyl/agent/trpc-agent-go/runner/runner.go) L204-217 `Runner`、L225-235 `ManagedRunner`、L242-247 `SteerableRunner`
- **23.2 runHandle 与 run 注册表**：L300-323 `runner` 结构体（`runsMu sync.RWMutex` + `runs map[string]*runHandle`）；L325-331 `runHandle`（cancel + steer.Queue）
- **23.3 事件循环**：L1286-1370 `runEventLoop` 三路 select + recover
- **23.4 Ralph Loop 外层循环**：[runner/ralph_loop.go](file:///home/cyl/agent/trpc-agent-go/runner/ralph_loop.go) L231-282 `runLoop`；三种停止条件（CompletionPromise / VerifyCommand / Verifiers）；L284-318 `newInnerInvocation` 跨迭代借用 steer queue
- **23.5 CandidateSelector 并行尝试**：[runner/candidate_selector.go](file:///home/cyl/agent/trpc-agent-go/runner/candidate_selector.go) L225-265 `runAttemptsParallel`（`errgroup.Group` + `SetLimit`）；L286 `attemptSession.Clone()` 隔离副作用；L345-359 bypass 场景
- **23.6 Best-of-N 评估选择器**：[runner/bestofn/evaluation.go](file:///home/cyl/agent/trpc-agent-go/runner/bestofn/evaluation.go) L112-139 pointwise、L141-188 pairwise；[options.go](file:///home/cyl/agent/trpc-agent-go/runner/bestofn/options.go) L27-35 SelectionMode

#### 第 24 章 Team 多 Agent 协作
- **24.1 Team 结构与两种模式**：[team/team.go](file:///home/cyl/agent/trpc-agent-go/team/team.go) L32-49 `Team` + `sync.RWMutex`；L52-62 `Mode`（Coordinator/Swarm）；L197-220 `runCoordinator`、L241-327 `runSwarm`
- **24.2 Swarm 转移控制器**：[team/runtime.go](file:///home/cyl/agent/trpc-agent-go/team/runtime.go) L29-40 `swarmRuntime` + `sync.Mutex`；L42-69 `OnTransfer`（MaxHandoffs + 滑动窗口循环检测）；L106-124 `isolateTargetSession`
- **24.3 Swarm 配置**：[team/swarm.go](file:///home/cyl/agent/trpc-agent-go/team/swarm.go) L24-40 `SwarmConfig`；L88-94 `DefaultSwarmConfig`（MaxHandoffs=20 等保守默认）
- **24.4 动态花名册**：[team/swarm_members.go](file:///home/cyl/agent/trpc-agent-go/team/swarm_members.go) L30-45 `UpdateSwarmMembers`、L72-107 `RemoveSwarmMember`（禁止移除 entry）
- **24.5 HistoryScope 与选项**：[team/options.go](file:///home/cyl/agent/trpc-agent-go/team/options.go) L30-44 `HistoryScope`（Default/Isolated/ParentBranch）

#### 第 25 章 Hedge 请求
- **25.1 Hedge 模型架构**：[model/hedge/hedge.go](file:///home/cyl/agent/trpc-agent-go/model/hedge/hedge.go) L27-32 `hedgeModel`、L47-62 `hedgeRun`（`eventChan chan attemptEvent` fan-in）
- **25.2 错峰启动**：L461-492 `resolveLaunchOffsets`；L354 `updateLaunchTimer`
- **25.3 first-meaningful-response**：L257-293 `handleEvent`；L563-567 `isWinningResponse`；L381-388 `cancelLosers`
- **25.4 失败聚合**：[model/hedge/failure.go](file:///home/cyl/agent/trpc-agent-go/model/hedge/failure.go) L41-58 `buildFailureMessage`、L83-92 `buildFailureResponse`
- **25.5 选项**：[model/hedge/options.go](file:///home/cyl/agent/trpc-agent-go/model/hedge/options.go) L43-47 `WithCandidates`（累加语义）、L68-79 `WithDelay`/`WithDelays`

#### 第 26 章 并发原语汇总
- 表格汇总：信号量、Worker Pool + WaitGroup、errgroup + SetLimit、Fan-in channel、sync.RWMutex + map、sync.Once、atomic.Int64、context.CancelFunc、recover panic 隔离

### 第 9 部分：赛题方案设计（第 27-31 章，仅方案不修改代码）

#### 第 27 章 赛题解读与 Gap 分析
- **27.1 赛题要求六阶段**：Baseline 评测 → 失败归因 → PromptIter 优化 → 候选验证 → 接受策略 → 审计落盘（引用 [赛题.md](file:///home/cyl/agent/trpc-agent-go/赛题.md)）
- **27.2 现有能力映射**：
  - Baseline 评测 ✅ [evaluation/evaluation.go](file:///home/cyl/agent/trpc-agent-go/evaluation/evaluation.go) `Evaluate`
  - PromptIter 优化 ✅ [engine/engine.go](file:///home/cyl/agent/trpc-agent-go/evaluation/workflow/promptiter/engine/engine.go) `Run`
  - 候选验证（部分）✅ engine `executeRound` 已含 validation 评估
  - fake model ✅ [test/mock_model.go](file:///home/cyl/agent/trpc-agent-go/test/mock_model.go) `QueueModel`
  - trace mode ✅ [evaluation/evalset/evalcase.go](file:///home/cyl/agent/trpc-agent-go/evaluation/evalset/evalcase.go) `EvalModeTrace`
- **27.3 关键 Gap**：
  - **失败归因**：现有 `loss.go` 只提取 TerminalLoss，未做语义归类（工具调用错误/参数错误/route 错误/格式错误/知识召回不足）
  - **逐 case delta**：现有 `accept.go` **L35** 只比较 validation 总分增量，未做 case 级 delta（新增通过/新增失败/分数提升/分数下降）
  - **多维 gate**：现有 `AcceptancePolicy` 只有 `MinScoreGain` 单字段，缺 hard-fail/critical-case/budget gate
  - **审计报告**：现有 manager 只存 run 状态，未生成 `optimization_report.json` / `.md`
  - **确定性 runner**：QueueModel 语义是回放整个队列，需包装为按调用次数消费的 fake engine
- **27.4 evolution 澄清**：`evolution/` 目录是 skill 库管理（从 session 抽取 SKILL.md），**不是** prompt 优化，赛题接受门禁应对应 PromptIter `AcceptancePolicy`

#### 第 28 章 总体架构设计
- **28.1 分层架构图**（文字版）：
  - 复用层：evaluation service + PromptIter engine + QueueModel + EvalModeTrace
  - 新增层：FailureAttributor / CaseDeltaCalculator / MultiDimensionGate / AuditReportBuilder / FakeEngine
  - 编排层：RegressionLoop（pipeline 主入口）
- **28.2 数据流**：baseline_eval → attributor → engine.Run（多轮）→ case_delta → gate → audit
- **28.3 目录建议**：`examples/evaluation/promptiter_regression_loop/`（main.go + configs/ + report/ + README）

#### 第 29 章 详细实现阶段

**阶段 0：环境与样例**
- 准备 train.evalset.json（3 条）、validation.evalset.json（3 条，含可优化成功/优化无效/验证集退化三类）
- 准备 metrics.json（ROUGE + ToolTrajectory + LLM Rubric）、promptiter.json、baseline prompt
- 配置 fake model 队列

**阶段 1：Baseline 评测**
- 调 `evaluation.New` + `Evaluate` 跑 train 与 validation
- 记录每 case 的 metric 分、pass/fail、trace、tool trajectory
- 复用 [evaluation/service/local/local.go](file:///home/cyl/agent/trpc-agent-go/evaluation/service/local/local.go) `evaluatePerCase`

**阶段 2：失败归因（FailureAttributor）**
- 输入：baseline EvaluationResult + ExecutionTrace + tool trajectory
- 规则归类（基于 [engine/loss.go](file:///home/cyl/agent/trpc-agent-go/evaluation/workflow/promptiter/engine/loss.go) `TerminalLoss` 扩展）：
  - 最终回复不匹配：ROUGE/FinalResponse fail
  - 工具调用错误：ToolTrajectory `matchTool` Name 不匹配
  - 工具参数错误：Arguments 子准则 fail
  - route 错误：Swarm/Team transfer 与预期不符
  - 格式错误：结构化输出 schema 校验 fail
  - 知识召回不足：LLM Rubric + 知识类 rubric fail
- 输出：`FailureAttribution{CaseID, Category, Reason, Evidence}`

**阶段 3：PromptIter 优化**
- 复用 [engine.New](file:///home/cyl/agent/trpc-agent-go/evaluation/workflow/promptiter/engine/engine.go) L170-196
- 构造 `RunRequest`：Train/Validation EvalSetInput、InitialProfile（baseline prompt）、TargetSurfaceIDs（Instruction/GlobalInstruction/FewShot/Model）
- 注入 fake model 作为 candidate/judge/worker runner
- 通过 Observer 收集每轮 RoundResult

**阶段 4：候选验证与逐 case delta**
- 候选 prompt 跑 validation evalset
- `CaseDeltaCalculator`：逐 case 比较 baseline vs candidate
  - `NewlyPassed` / `NewlyFailed` / `ScoreImproved` / `ScoreRegressed`
- 复用 [evaluation/evalresult/evalresult.go](file:///home/cyl/agent/trpc-agent-go/evaluation/evalresult/evalresult.go) `EvalCaseResult`

**阶段 5：多维接受门禁（MultiDimensionGate）**
- 替换/包装 [accept.go](file:///home/cyl/agent/trpc-agent-go/evaluation/workflow/promptiter/engine/accept.go) L33-37 单维判定
- 四类 gate（可配置）：
  - `ValidationScoreGain`：validation 总分提升 ≥ 阈值（保留原 MinScoreGain 语义）
  - `NoNewHardFail`：不新增 hard fail case（critical case 不得退化）
  - `CriticalCasePreserve`：指定 case ID 列表不得退化
  - `BudgetConstraint`：成本/调用次数 ≤ 预算
- 决策：全部 gate 通过才 `Accepted: true`

**阶段 6：审计报告**
- `AuditReportBuilder` 生成：
  - `optimization_report.json`：baseline/candidate/delta/gate_decision/failure_attribution_stats/cost_latency
  - `optimization_report.md`：人读版，含是否接受建议与理由
- 字段：每轮候选 prompt、eval result、delta、接受/拒绝理由、运行成本、耗时、随机种子、模型配置/fake engine 配置

**阶段 7：确定性保证**
- FakeEngine 包装 QueueModel：按调用次数消费（而非回放整个队列）
- 固定随机种子
- 禁用真实 API Key，全程 fake model + trace mode

**阶段 8：测试覆盖**
- 单测：gate 决策（各 gate 通过/失败组合）、逐 case delta（四类）、失败归因（六类）、报告生成（JSON/MD 字段校验）
- 集成测：6 条样例 case 全跑通 + fake model/trace mode ≤ 3 分钟

#### 第 30 章 验收标准对齐
- 逐条映射赛题验收标准 1-6 到实现阶段
- 重点：验收标准 3（过拟合拒绝）→ 阶段 5 `NoNewHardFail` + `CriticalCasePreserve` gate
- 验收标准 5（≤3 分钟）→ 阶段 7 fake model + trace mode 跳过真实推理

#### 第 31 章 方案设计说明（300-500 字）
- 失败归因方法：基于 TerminalLoss + ExecutionTrace 终态 step 的规则归类
- 接受策略：四维 gate（总分提升 + 不新增 hard fail + 关键 case 不退化 + 预算）
- 防过拟合策略：train/validation 分离 + 逐 case delta + critical case 保留
- PromptIter 接入方式：复用 engine.Run + Observer，外层包装 RegressionLoop
- 产物审计方式：JSON + MD 双格式，含每轮快照与决策理由

### 第 10 部分：附录

#### 附录 A：完整文件超链接索引
- 按模块分组（agent/、internal/flow/、skill/、tool/、codeexecutor/、memory/、session/、evaluation/、graph/、runner/、team/、model/hedge/、evolution/、test/、examples/）

#### 附录 B：关键数据流图（文字版）
- Agent Loop 事件流
- PromptIter engine 数据流
- Graph BSP/DAG 执行流
- Hedge fan-in 流

#### 附录 C：构建与测试速查
- 引用 [AGENTS.md](file:///home/cyl/agent/trpc-agent-go/AGENTS.md) 命令表
- gofmt/goimports/golangci-lint 注意事项
- 多模块测试脚本

## 四、假设与决策

1. **续写策略**：使用 Edit 工具替换 `<!-- 文档正文将在后续章节继续 -->` 标记，每次追加一个部分 + 新标记，最后移除标记
2. **语言**：正文中文，代码标识符/SQL/路径/类型名保持英文原文
3. **超链接格式**：`file:///home/cyl/agent/trpc-agent-go/...` 绝对路径 markdown 链接，行号用 `#L123-145` 或文中"约 L123"
4. **赛题方案边界**：仅设计，不修改业务代码；如需示意代码用 markdown 代码块（非 file 链接）
5. **行号精度**：大文件用"约 L123"，小文件用精确 `#L123-145`
6. **不创建新文件**：所有内容追加到现有 [PROJECT_DEEP_DIVE.zh.md](file:///home/cyl/agent/trpc-agent-go/PROJECT_DEEP_DIVE.zh.md)
7. **澄清点**：`evolution/` 不用于 prompt 优化；`EvalModeTrace` 在 evalset 非 agent/trace；tooltrajectory 用 Kuhn 二部图匹配非加权 Kuhn-Munkres

## 五、验证步骤

1. **结构完整性**：文档应包含 10 部分、31 章、3 附录，TOC 与正文对应
2. **超链接有效性**：所有 `file:///` 链接指向真实存在的文件，行号在合理范围
3. **代码准确性**：关键行号与实际代码一致（已通过 search subagent 校验）
4. **赛题覆盖**：六阶段、六验收标准、300-500 字说明均有对应章节
5. **格式规范**：markdown 代码块带语言标签，三反引号不缩进，链接文本不用反引号
6. **最终校对**：用 Grep 检查残留的 `<!-- 文档正文将在后续章节继续 -->` 标记是否已移除

## 六、执行顺序

1. 追加第 7 部分（第 17-21 章 Evaluation）—— 替换标记
2. 追加第 8 部分（第 22-26 章 高并发）—— 替换标记
3. 追加第 9 部分（第 27-31 章 赛题方案）—— 替换标记
4. 追加第 10 部分（附录 A/B/C）—— 移除标记
5. 整体校对：Grep 检查残留标记 + 抽查超链接
