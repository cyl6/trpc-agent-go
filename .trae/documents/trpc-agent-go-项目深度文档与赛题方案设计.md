# 计划：撰写 trpc-agent-go 项目深度中文文档 + 赛题方案设计

## 一、摘要（Summary）

**交付物**：一份中文 Markdown 项目文档（单文件），路径建议为 `/home/cyl/agent/trpc-agent-go/PROJECT_DEEP_DIVE.zh.md`。

该文档同时承担两个目标：
1. **项目深度解读**：系统性阐述 trpc-agent-go 各核心子系统的架构与代码实现细节，包含关键代码走读与可点击的文件超链接索引（`file:///` 协议）。覆盖：Agent Loop、Sandbox/CodeExecutor、Memory、Skills & Tools、Prompt、Evaluation、高并发实现、数据库/会话存储实现。
2. **赛题方案设计**：针对 `/home/cyl/agent/trpc-agent-go/赛题.md`（"评测-失败归因-prompt 优化-回归验证-产物审计"自动闭环），给出详细的解决步骤、实现阶段与详细方案——**仅设计层面，不修改任何代码**。

**本任务为文档撰写任务，唯一产出为一份 .md 文件，不改动任何源代码。**

---

## 二、当前状态分析（Current State Analysis）

已通过 5 个并行 search 子代理完成对仓库的深度探索，掌握以下事实：

### 2.1 已探索的核心子系统与关键文件

| 子系统 | 关键文件（绝对路径） |
|---|---|
| Agent Loop | `agent/agent.go`、`agent/invocation.go`、`agent/callbacks.go`、`agent/plugins.go`、`agent/run_with_plugins.go`、`agent/lazy_agent.go`、`agent/stream_hub.go`、`agent/stream_mode.go`、`agent/execution_trace.go`、`agent/toolcontext.go`、`agent/callbackcontext.go`、`agent/llmagent/llm_agent.go`、`internal/flow/flow.go`、`internal/flow/llmflow/llmflow.go`、`internal/flow/processor/functioncall.go`、`internal/tracecapture/capture.go`、`agent/trace/trace.go` |
| Sandbox/CodeExecutor | `codeexecutor/codeexecutor.go`、`codeexecutor/workspace.go`、`codeexecutor/registry.go`、`codeexecutor/metadata.go`、`codeexecutor/manifest.go`、`codeexecutor/mime.go`、`codeexecutor/e2b/e2b.go` |
| Skills | `skill/repository.go`、`skill/scope.go`、`skill/state_keys.go`、`skill/state_order.go`、`skill/url_root.go`、`skill/context_repository.go` |
| Tools | `tool/tool.go`、`tool/toolset.go`、`tool/context.go`、`tool/stream.go`、`tool/callbacks.go`、`tool/filter.go`、`tool/merge.go`、`tool/metadata.go`、`tool/permission.go`、`tool/retry.go`、`tool/final_result.go`、`tool/skill/{load,run,exec,stager,schema,list_docs}.go`、`tool/file/file.go`、`tool/mcp/{config,tool,toolset}.go`、`tool/mcpbroker/broker.go`、`tool/agent/agent_tool.go`、`tool/todo/todo.go`、`tool/hostexec/`、`tool/taskrun/`、`tool/openapi/`、`tool/openviking/`、`internal/tool/toolset.go` |
| Prompt | `prompt/doc.go`、`prompt/text.go` |
| Evaluation | `evaluation/evaluation.go`、`evaluation/options.go`、`evaluation/pass.go`、`evaluation/evalset/{evalset,evalcase}.go`、`evaluation/metric/metric.go`、`evaluation/evalresult/evalresult.go`、`evaluation/metric/criterion/rouge/rouge.go`、`evaluation/metric/criterion/llm/llm.go`、`evaluation/metric/criterion/tooltrajectory/tooltrajectory.go`、`evaluation/service/local/{local.go,trace_mode_additional_test.go}`、`evaluation/internal/multirun/multirun.go` |
| PromptIter 引擎 | `evaluation/workflow/promptiter/{promptiter,gradient,loss,patch,profile}.go`、`evaluation/workflow/promptiter/engine/{engine,accept,stop,evaluate,loss,backward,aggregate,optimize,event,option}.go`、`evaluation/workflow/promptiter/manager/manager.go`、`server/promptiter/{server,handler}.go`、`server/evaluation/server.go` |
| PromptIter 示例 | `examples/evaluation/promptiter/{syncrun,asyncrun,server,multinode}/` 及其 `data/promptiter-nba-commentary-app/` 配置 |
| Fake/Trace | `test/mock_model.go`（QueueModel）、`evalset.EvalModeTrace="trace"` |
| Memory | `memory/memory.go`、`memory/sqlite/service.go`、`memory/mysql/service.go`、`memory/redis/service.go`、`memory/pgvector/service.go`、`memory/sqlitevec/service.go`、`memory/mysqlvec/vec.go`、`memory/mem0/{client,service,tools,types,ingest_worker,session_scan,helpers}.go`、`memory/tool/tool.go`、`memory/internal/memory/{memory,auto}.go` |
| Session | `session/session.go`、`session/state.go`、`session/track.go`、`session/hook.go`、`session/ingestor.go`、`session/sqlite/`、`session/mysql/`、`session/postgres/schema.sql`、`session/redis/{service,options}.go`、`session/redis/internal/{hashidx,zset,util}/`、`session/mongodb/`、`session/noop/service.go`、`session/summary/summarizer.go`、`session/pgvector/search.go` |
| Graph/并发 | `graph/graph.go`、`graph/executor.go`、`graph/executor_dag.go`、`graph/checkpoint.go`、`graph/time_travel.go`、`graph/cache.go`、`graph/retry.go`、`graph/trace_task.go`、`graph/internal/channel/channel.go` |
| Runner/并发 | `runner/runner.go`、`runner/ralph_loop.go`、`runner/candidate_selector.go`、`runner/candidate_selector_session.go`、`runner/candidate_selector_effect.go`、`runner/bestofn/{evaluation,options}.go`、`runner/plugin.go`、`runner/diagnostics.go` |
| Team | `team/team.go`、`team/runtime.go`、`team/swarm.go`、`team/swarm_members.go`、`team/options.go` |
| Hedge | `model/hedge/{hedge,failure,options}.go` |
| Evolution（澄清：非 prompt 优化） | `evolution/{types,gates,policy,revision,service,promotion,reviewer,worker}.go` |

### 2.2 赛题现状与差距分析（关键结论）

1. **PromptIter 引擎已存在且较完整**：`evaluation/workflow/promptiter/engine/engine.go` 已实现"baseline 验证 → 训练评测 → loss 提取 → backward → aggregation → optimize → 候选验证 → 接受门禁 → 停止策略"闭环。这是赛题的核心依赖，**不必从零写**。
2. **`evolution/` 目录与赛题无关**：它是"从 session 异步抽取可复用 SKILL.md"的技能库管理子系统，其 gates（SpecGate/SafetyGate/EffectivenessGate/HumanGate）作用于 skill revision 而非 prompt candidate。赛题所指"接受门禁"应对应 PromptIter 引擎的 `AcceptancePolicy`。
3. **现有接受门禁非常简陋**：`engine/accept.go` 仅做 `scoreDelta >= MinScoreGain` 单维判断，**不满足**赛题要求的"不能新增 hard fail、关键 case 不能退化、成本/调用次数预算"等多维门禁。
4. **关键缺失项**（赛题需新建，本方案设计将详细说明）：
   - 失败归因分类（6 类：最终回复不匹配/工具调用错误/工具参数错误/route 错误/格式错误/知识召回不足）
   - 候选验证的**逐 case delta**（新增通过/新增失败/分数提升/分数下降）
   - 多维接受门禁（hard fail / 关键 case / 预算）
   - `optimization_report.json` 与 `optimization_report.md` 审计落盘
   - 集成 fake model / trace mode / deterministic runner 的可复现 pipeline（现有示例依赖真实 OpenAI Key）
   - 6 条样例 case（3 训练 + 3 验证，含可优化成功/优化无效/验证集退化三类场景）
   - 单元测试覆盖（gate 决策/逐 case delta/失败归因/报告生成）
   - 竞赛建议目录 `examples/evaluation/promptiter_regression_loop/` 不存在

---

## 三、文档结构设计（Proposed Changes）

文档将创建为单文件 `/home/cyl/agent/trpc-agent-go/PROJECT_DEEP_DIVE.zh.md`，采用如下章节结构。每章均包含：架构图/数据流、关键类型与函数（带行号）、可点击文件超链接、关键代码走读片段。

### 第一部分：项目总览

**第 1 章 项目定位与模块拓扑**
- Go 多模块 monorepo（~80 个 go.mod），根模块 `trpc.group/trpc-go/trpc-agent-go`，Go 1.21+
- 顶层目录矩阵表（agent/evaluation/memory/session/tool/skill/graph/runner/team/model/codeexecutor 等）
- 链接：`AGENTS.md`、`README.zh_CN.md`、`go.mod`
- 构建与测试命令速查（`go build ./...`、`go test ./...`、`bash .github/scripts/run-go-tests.sh`）

### 第二部分：Agent 核心运行时

**第 2 章 Agent 接口与调用上下文**
- `Agent` 接口（[agent.go](file:///home/cyl/agent/trpc-agent-go/agent/agent.go)）：`Run/Tools/Info/SubAgents/FindSubAgent`
- `Invocation` 上下文（[invocation.go](file:///home/cyl/agent/trpc-agent-go/agent/invocation.go)）：字段表、`NewInvocation`/`Clone`/`View`、计数与限额（`IncLLMCallCount`/`IncToolIteration`）、state KV、事件注入
- `RunOptions` 配置项（模型/流式/工具过滤/超时/trace）
- `InvocationContext` 传播机制（[invocationcontext.go](file:///home/cyl/agent/trpc-agent-go/agent/invocationcontext.go)）
- 走读：`Runner.Run → NewInvocation → RunWithPlugins → ag.Run`

**第 3 章 ReAct 主循环（Agent Loop）**
- Flow 接口（[flow.go](file:///home/cyl/agent/trpc-agent-go/internal/flow/flow.go)）
- LLMAgent 装配（[llm_agent.go](file:///home/cyl/agent/trpc-agent-go/agent/llmagent/llm_agent.go)）：request/response processor 链
- 主循环 `Flow.Run`（[llmflow.go](file:///home/cyl/agent/trpc-agent-go/internal/flow/llmflow/llmflow.go)）：`for { emitStart → runOneStep → 退出判定 }`
- 单步 `runOneStep`：`preprocess → callLLM → processStreamingResponses`
- 工具调用迭代 `FunctionCallResponseProcessor`（[functioncall.go](file:///home/cyl/agent/trpc-agent-go/internal/flow/processor/functioncall.go)）：`IncToolIteration` 限额、deferred/executable 判定、并行/串行执行、`EndInvocation` 触发条件
- 走读：ReAct 循环本质（tool.response 写回 session → 下轮 preprocess 读取 → LLM 决策）
- 退出条件汇总：`lastEvent==nil || EndInvocation || IsFinalResponse`

**第 4 章 回调与插件体系**
- 三层回调：Agent（[callbacks.go](file:///home/cyl/agent/trpc-agent-go/agent/callbacks.go)）/ Model / Tool
- `PluginManager` 接口（[plugins.go](file:///home/cyl/agent/trpc-agent-go/agent/plugins.go)）
- `RunWithPlugins` 编排（[run_with_plugins.go](file:///home/cyl/agent/trpc-agent-go/agent/run_with_plugins.go)）：BeforeAgent 短路、AfterAgent 流后触发
- `CallbackContext`/`ToolContext`（[callbackcontext.go](file:///home/cyl/agent/trpc-agent-go/agent/callbackcontext.go)、[toolcontext.go](file:///home/cyl/agent/trpc-agent-go/agent/toolcontext.go)）：artifact 与 state 操作
- panic 恢复与错误聚合

**第 5 章 延迟 Agent、StreamHub 与 StreamMode**
- `LazyAgent`（[lazy_agent.go](file:///home/cyl/agent/trpc-agent-go/agent/lazy_agent.go)）：调用时才实例化
- `StreamHub`（[stream_hub.go](file:///home/cyl/agent/trpc-agent-go/agent/stream_hub.go)）：invocation 作用域临时流注册表，Clone 时保留
- `StreamMode`（[stream_mode.go](file:///home/cyl/agent/trpc-agent-go/agent/stream_mode.go)）：messages/updates/checkpoints/tasks/debug/custom

**第 6 章 执行 Trace 机制**
- 公共模型（[trace.go](file:///home/cyl/agent/trpc-agent-go/agent/trace/trace.go)）：`Trace`/`Step`/`Snapshot`
- Invocation API（[execution_trace.go](file:///home/cyl/agent/trpc-agent-go/agent/execution_trace.go)）：`StartExecutionTraceStep`/`FinishExecutionTraceStep`/`BuildExecutionTrace`
- 捕获器（[capture.go](file:///home/cyl/agent/trpc-agent-go/internal/tracecapture/capture.go)）：跨调用 trace 树构建
- 在 `runOneStep` 的埋点

### 第三部分：Skills & Tools

**第 7 章 技能系统（Skills）**
- `SKILL.md` + YAML front matter 载体
- `Repository`/`FSRepository`（[repository.go](file:///home/cyl/agent/trpc-agent-go/skill/repository.go)）：多 root 扫描、front matter 解析
- `SkillScope` 与多租户隔离（[scope.go](file:///home/cyl/agent/trpc-agent-go/skill/scope.go)）
- 状态键设计（[state_keys.go](file:///home/cyl/agent/trpc-agent-go/skill/state_keys.go)）：agent 名前缀防子 agent 污染
- URL root 下载与路径穿越防护（[url_root.go](file:///home/cyl/agent/trpc-agent-go/skill/url_root.go)）
- `ContextRepository` + `VisibilityFilter`

**第 8 章 工具框架核心**
- 三层接口：`Tool`/`CallableTool`/`StreamableTool`（[tool.go](file:///home/cyl/agent/trpc-agent-go/tool/tool.go)）
- `ToolSet` 注册单元（[toolset.go](file:///home/cyl/agent/trpc-agent-go/tool/toolset.go)）
- Stream 与 FinalResultChunk（[stream.go](file:///home/cyl/agent/trpc-agent-go/tool/stream.go)、[final_result.go](file:///home/cyl/agent/trpc-agent-go/tool/final_result.go)）
- 回调机制（[callbacks.go](file:///home/cyl/agent/trpc-agent-go/tool/callbacks.go)）：结构化 before/after/result-messages
- 过滤/合并/元数据/权限/重试（[filter.go](file:///home/cyl/agent/trpc-agent-go/tool/filter.go)、[merge.go](file:///home/cyl/agent/trpc-agent-go/tool/merge.go)、[metadata.go](file:///home/cyl/agent/trpc-agent-go/tool/metadata.go)、[permission.go](file:///home/cyl/agent/trpc-agent-go/tool/permission.go)、[retry.go](file:///home/cyl/agent/trpc-agent-go/tool/retry.go)）

**第 9 章 工具实现走读**
- skill 工具族（[load.go](file:///home/cyl/agent/trpc-agent-go/tool/skill/load.go)、[run.go](file:///home/cyl/agent/trpc-agent-go/tool/skill/run.go)、[exec.go](file:///home/cyl/agent/trpc-agent-go/tool/skill/exec.go)、[stager.go](file:///home/cyl/agent/trpc-agent-go/tool/skill/stager.go)）
- file 工具族（[file.go](file:///home/cyl/agent/trpc-agent-go/tool/file/file.go)、[readfile.go](file:///home/cyl/agent/trpc-agent-go/tool/file/readfile.go)）
- MCP 工具与 Broker（[tool/mcp/](file:///home/cyl/agent/trpc-agent-go/tool/mcp/)、[tool/mcpbroker/](file:///home/cyl/agent/trpc-agent-go/tool/mcpbroker/)）：会话重连 + singleflight
- agent-as-tool（[agent_tool.go](file:///home/cyl/agent/trpc-agent-go/tool/agent/agent_tool.go)）：history scope / response mode
- todo / hostexec / taskrun / openapi / openviking

**第 10 章 工具注册、过滤、鉴权与调用全流程**
- `compileTools` 四步（[llm_agent.go](file:///home/cyl/agent/trpc-agent-go/agent/llmagent/llm_agent.go)）
- `NamedToolSet` 前缀防冲突（[internal/tool/toolset.go](file:///home/cyl/agent/trpc-agent-go/internal/tool/toolset.go)）
- `FilterTools` 区分用户/框架工具
- 双层鉴权：`PermissionChecker`（逐工具）+ `PermissionPolicy`（逐运行）于 `checkToolPermission`（[functioncall.go](file:///home/cyl/agent/trpc-agent-go/internal/flow/processor/functioncall.go)）
- 完整调用链走读：`ProcessResponse → handleFunctionCallsWithRequest → executeToolWithCallbacks → executeTool`
- 并行执行 panic 恢复、状态增量 delta 回放、HTML 转义关闭、SkipSummarization

### 第四部分：Sandbox / 代码执行

**第 11 章 CodeExecutor 架构**
- 核心抽象（[codeexecutor.go](file:///home/cyl/agent/trpc-agent-go/codeexecutor/codeexecutor.go)）：`CodeExecutor`/`CodeExecutionInput`/`CodeExecutionResult`
- Workspace 三能力接口（[workspace.go](file:///home/cyl/agent/trpc-agent-go/codeexecutor/workspace.go)）：`WorkspaceManager`/`WorkspaceFS`/`ProgramRunner` + `Engine` 聚合
- `WorkspaceRegistry` single-flight 合并并发创建（[registry.go](file:///home/cyl/agent/trpc-agent-go/codeexecutor/registry.go)）
- 标准目录布局与 metadata.json 原子写（[metadata.go](file:///home/cyl/agent/trpc-agent-go/codeexecutor/metadata.go)）
- 输入输出清单与 MIME（[manifest.go](file:///home/cyl/agent/trpc-agent-go/codeexecutor/manifest.go)、[mime.go](file:///home/cyl/agent/trpc-agent-go/codeexecutor/mime.go)）
- E2B 沙箱实现（[e2b.go](file:///home/cyl/agent/trpc-agent-go/codeexecutor/e2b/e2b.go)）：PerTurn/PerSession 持久化、`SupportsCleanEnv`

### 第五部分：Memory 与数据库实现

**第 12 章 记忆层架构**
- `Service` 接口（[memory.go](file:///home/cyl/agent/trpc-agent-go/memory/memory.go)）：`AddMemory`/`SearchMemories`/`EnqueueAutoMemoryJob` 等
- `Memory`/`Entry`/`Kind`(fact/episode)/`SearchOptions`（HybridSearch/RRF/Deduplicate）
- 后端矩阵：SQLite/MySQL/Postgres/Redis/pgvector/sqlitevec/mysqlvec/mem0
- SQL 后端 Schema 走读（[sqlite/init.go](file:///home/cyl/agent/trpc-agent-go/memory/sqlite/init.go)、[mysql/init.go](file:///home/cyl/agent/trpc-agent-go/memory/mysql/init.go)、[postgres/init.go](file:///home/cyl/agent/trpc-agent-go/memory/postgres/init.go)）：软删 `deleted_at`、JSON/JSONB 列、`expectedSchema` 校验
- Redis Hash 结构与哈希标签（[redis/service.go](file:///home/cyl/agent/trpc-agent-go/memory/redis/service.go)）

**第 13 章 向量存储与混合检索**
- pgvector（[pgvector/service.go](file:///home/cyl/agent/trpc-agent-go/memory/pgvector/service.go)）：HNSW + `vector_cosine_ops`、`tsvector`+GIN、`1-(embedding <=> $1)`
- sqlitevec（[sqlitevec/service.go](file:///home/cyl/agent/trpc-agent-go/memory/sqlitevec/service.go)）：`vec0` 虚表、`vec_f32`、score=1-distance、LRU 驱逐
- mysqlvec（[mysqlvec/vec.go](file:///home/cyl/agent/trpc-agent-go/memory/mysqlvec/vec.go)）：小端 float32 二进制 + 暴力余弦
- 混合检索 RRF + 内存 BM25 + CJK 分词（[memory/internal/memory/memory.go](file:///home/cyl/agent/trpc-agent-go/memory/internal/memory/memory.go)）
- 自动记忆抽取 worker（[auto.go](file:///home/cyl/agent/trpc-agent-go/memory/internal/memory/auto.go)）：reconcile 阈值
- mem0 后端异步 ingest（[mem0/](file:///home/cyl/agent/trpc-agent-go/memory/mem0/)）
- 记忆工具六件套（[memory/tool/tool.go](file:///home/cyl/agent/trpc-agent-go/memory/tool/tool.go)）

**第 14 章 会话层架构**
- `Session`/`Service`/`SearchableService`/`WindowService`（[session.go](file:///home/cyl/agent/trpc-agent-go/session/session.go)）
- State/Track/Hook/Ingestor（[state.go](file:///home/cyl/agent/trpc-agent-go/session/state.go)、[track.go](file:///home/cyl/agent/trpc-agent-go/session/track.go)、[hook.go](file:///home/cyl/agent/trpc-agent-go/session/hook.go)、[ingestor.go](file:///home/cyl/agent/trpc-agent-go/session/ingestor.go)）
- SQL 后端六表 Schema（[sqlite/init.go](file:///home/cyl/agent/trpc-agent-go/session/sqlite/init.go)、[postgres/schema.sql](file:///home/cyl/agent/trpc-agent-go/session/postgres/schema.sql)）：部分唯一索引 `WHERE deleted_at IS NULL`、TTL 索引
- Redis facade + hashidx/zset 双实现 + Lua 原子脚本（[redis/service.go](file:///home/cyl/agent/trpc-agent-go/session/redis/service.go)、[redis/internal/hashidx/lua.go](file:///home/cyl/agent/trpc-agent-go/session/redis/internal/hashidx/lua.go)）
- MongoDB 后端事务探测与索引（[mongodb/](file:///home/cyl/agent/trpc-agent-go/session/mongodb/)）
- noop/summary/pgvector 事件搜索

**第 15 章 记忆 ↔ 会话耦合**
- `EnqueueAutoMemoryJob(sess)` 桥接
- `SessionStateKeyAutoMemoryLastExtractAt` 增量抽取
- `IngestSession` 接口（mem0 实现）
- 共享 `UserKey`/`Key` 类型

### 第六部分：Prompt 子系统

**第 16 章 Prompt 与 Surface 抽象**
- [prompt/doc.go](file:///home/cyl/agent/trpc-agent-go/prompt/doc.go)、[prompt/text.go](file:///home/cyl/agent/trpc-agent-go/prompt/text.go)
- PromptIter 领域模型：`Profile`/`SurfaceOverride`/`PatchSet`/`SurfacePatch`（[promptiter/profile.go](file:///home/cyl/agent/trpc-agent-go/evaluation/workflow/promptiter/profile.go)、[patch.go](file:///home/cyl/agent/trpc-agent-go/evaluation/workflow/promptiter/patch.go)）
- 可优化 surface 类型：`Instruction`/`GlobalInstruction`/`FewShot`/`Model`（[server/promptiter/handler.go](file:///home/cyl/agent/trpc-agent-go/server/promptiter/handler.go)）

### 第七部分：Evaluation 子系统

**第 17 章 评测服务核心**
- `AgentEvaluator` 接口与 `New`（[evaluation.go](file:///home/cyl/agent/trpc-agent-go/evaluation/evaluation.go)）
- `Evaluate` 数据流：`collectCaseResults → runEvaluation → runEvaluationOnce → aggregateCaseRuns`
- `EvaluationResult`/`EvaluationCaseResult`/`EvaluationCaseRunDetails`（含 trace）
- Option 体系（[options.go](file:///home/cyl/agent/trpc-agent-go/evaluation/options.go)）：`WithJudgeRunner`/`WithRunDetailsEnabled`/`WithNumRuns`
- pass@k / pass^k（[pass.go](file:///home/cyl/agent/trpc-agent-go/evaluation/pass.go)）

**第 18 章 评测数据结构与指标**
- EvalSet/EvalCase/Invocation/Tool（[evalset.go](file:///home/cyl/agent/trpc-agent-go/evaluation/evalset/evalset.go)、[evalcase.go](file:///home/cyl/agent/trpc-agent-go/evaluation/evalset/evalcase.go)）：`EvalModeTrace`
- EvalMetric/Manager（[metric.go](file:///home/cyl/agent/trpc-agent-go/evaluation/metric/metric.go)）
- EvalResult 结构（[evalresult.go](file:///home/cyl/agent/trpc-agent-go/evaluation/evalresult/evalresult.go)）：`EvalMetricResultDetails.Reason`（失败归因原始文本）
- 多轮汇总（[multirun.go](file:///home/cyl/agent/trpc-agent-go/evaluation/internal/multirun/multirun.go)）

**第 19 章 指标实现走读**
- ROUGE（[rouge.go](file:///home/cyl/agent/trpc-agent-go/evaluation/metric/criterion/rouge/rouge.go)）
- LLM Rubric（[llm.go](file:///home/cyl/agent/trpc-agent-go/evaluation/metric/criterion/llm/llm.go)）：`JudgeRunnerOptions` 可注入 fake
- Tool Trajectory（[tooltrajectory.go](file:///home/cyl/agent/trpc-agent-go/evaluation/metric/criterion/tooltrajectory/tooltrajectory.go)）：Kuhn-Munkras 无序匹配
- finalresponse/json/text/length/xml

**第 20 章 PromptIter 优化引擎**
- Engine 接口与 `RunRequest`/`RunResult`/`RoundResult`（[engine.go](file:///home/cyl/agent/trpc-agent-go/evaluation/workflow/promptiter/engine/engine.go)）
- 主循环 `run`：describeStructure → baseline 验证 → 多轮 executeRound → stop
- `executeRound`：evaluate(Train) → loss → mergeLossHints → backward → aggregate → optimize → applyPatchSet → evaluate(Validation) → accept
- 接受门禁现状（[accept.go](file:///home/cyl/agent/trpc-agent-go/evaluation/workflow/promptiter/engine/accept.go)）：仅 `MinScoreGain` 单维
- 停止策略（[stop.go](file:///home/cyl/agent/trpc-agent-go/evaluation/workflow/promptiter/engine/stop.go)）
- 失败信号提取（[loss.go](file:///home/cyl/agent/trpc-agent-go/evaluation/workflow/promptiter/engine/loss.go)）：`TerminalLoss{MetricName,Reason,StepID}`
- Observer 事件流（[event.go](file:///home/cyl/agent/trpc-agent-go/evaluation/workflow/promptiter/engine/event.go)）：审计钩子
- 异步管理（[manager.go](file:///home/cyl/agent/trpc-agent-go/evaluation/workflow/promptiter/manager/manager.go)）
- 示例走读（[examples/evaluation/promptiter/syncrun/engine.go](file:///home/cyl/agent/trpc-agent-go/examples/evaluation/promptiter/syncrun/engine.go)）：runtime 构造模式

**第 21 章 Fake Model / Trace Mode / 确定性运行**
- `QueueModel`（[test/mock_model.go](file:///home/cyl/agent/trpc-agent-go/test/mock_model.go)）
- `EvalModeTrace`（trace mode 跳过 runner）
- 确定性运行组合策略

### 第八部分：高并发与多智能体

**第 22 章 Graph 执行引擎**
- `Graph`/`Node`/`ExecutionContext`（[graph.go](file:///home/cyl/agent/trpc-agent-go/graph/graph.go)）：多重锁分离
- 两种引擎：BSP（屏障同步）vs DAG（无屏障）
- BSP Worker Pool（[executor.go](file:///home/cyl/agent/trpc-agent-go/graph/executor.go)）：`executeStep` 用 `sync.WaitGroup` + channel
- DAG 信号量调度（[executor_dag.go](file:///home/cyl/agent/trpc-agent-go/graph/executor_dag.go)）：`sem chan struct{}` + `done` channel + select 多路复用
- Channel 4 种行为（[channel.go](file:///home/cyl/agent/trpc-agent-go/graph/internal/channel/channel.go)）：LastValue/Topic/Ephemeral/Barrier
- Checkpoint 与时间旅行（[checkpoint.go](file:///home/cyl/agent/trpc-agent-go/graph/checkpoint.go)、[time_travel.go](file:///home/cyl/agent/trpc-agent-go/graph/time_travel.go)）
- Retry/Interrupt/Resume/TraceTask/Visualize

**第 23 章 Runner 与 Best-of-N**
- Runner 核心（[runner.go](file:///home/cyl/agent/trpc-agent-go/runner/runner.go)）：`runsMu`/`closeOnce`/`runHandle.mu`
- Ralph Loop 验证外循环（[ralph_loop.go](file:///home/cyl/agent/trpc-agent-go/runner/ralph_loop.go)）
- Best-of-N 并行候选（[candidate_selector.go](file:///home/cyl/agent/trpc-agent-go/runner/candidate_selector.go)）：`errgroup.Group` + `SetLimit` + 会话克隆 + 只读服务隔离
- 评估选择 pointwise/pairwise（[bestofn/evaluation.go](file:///home/cyl/agent/trpc-agent-go/runner/bestofn/evaluation.go)）
- 插件链（[plugin.go](file:///home/cyl/agent/trpc-agent-go/runner/plugin.go)）

**第 24 章 Team 多智能体协调**
- Team 两种模式（[team.go](file:///home/cyl/agent/trpc-agent-go/team/team.go)）：Coordinator（AgentTool 调用）vs Swarm（transfer_to_agent 转移）
- Swarm 运行时与会话隔离（[runtime.go](file:///home/cyl/agent/trpc-agent-go/team/runtime.go)）：`swarmRuntime` 锁、per-agent session、跨请求转移
- 动态成员管理（[swarm_members.go](file:///home/cyl/agent/trpc-agent-go/team/swarm_members.go)）
- 配置选项（[options.go](file:///home/cyl/agent/trpc-agent-go/team/options.go)、[swarm.go](file:///home/cyl/agent/trpc-agent-go/team/swarm.go)）

**第 25 章 Hedge 对冲请求**
- `hedgeModel` 跨候选对冲（[hedge.go](file:///home/cyl/agent/trpc-agent-go/model/hedge/hedge.go)）：fan-in channel + 首到先赢 + cancelLosers
- 失败聚合（[failure.go](file:///home/cyl/agent/trpc-agent-go/model/hedge/failure.go)）
- 错峰启动与选项（[options.go](file:///home/cyl/agent/trpc-agent-go/model/hedge/options.go)）

**第 26 章 并发原语汇总**
- 跨组件并发模式对照表：`sync.RWMutex`/`sync.Mutex`/`sync.Once`/`atomic.Int64`/信号量 channel/`WaitGroup` Worker Pool/`errgroup.SetLimit`/Fan-in channel/`context.WithCancel`/select 多路复用/双锁模式

### 第九部分：赛题方案设计（不修改代码）

**第 27 章 赛题解读与差距对照**
- 赛题要求 6 阶段：Baseline 评测 / 失败归因 / PromptIter 优化 / 候选验证 / 接受策略 / 审计落盘
- 输入输出要求、交付物、验收标准逐条列出
- 现状对照表（哪些已具备、哪些需新建）——基于第 2.2 节差距分析

**第 28 章 总体方案架构**
- 复用 > 新建原则：以 `evaluation/workflow/promptiter/engine` 为核心，外层补齐"失败归因 + 逐 case delta + 多维门禁 + 审计报告 + fake 集成"
- 目标目录：`examples/evaluation/promptiter_regression_loop/`（新建）
- 架构图（文字版数据流）：
  ```
  配置加载(train/validation evalset + metrics + promptiter.json + baseline prompt)
    → Baseline 评测(train+validation, WithRunDetailsEnabled+WithExecutionTraceEnabled)
    → 失败归因(Attributor: MetricName+Reason+Trace+Tools → 6 类)
    → PromptIter engine.Run(扩展 AcceptancePolicy)
    → 逐 case Delta(baseline vs candidate validation)
    → 多维 Gate 链(ScoreGain/NoNewHardFail/CriticalCase/Budget)
    → 审计报告(optimization_report.json + .md)
  ```
- fake model / trace mode / deterministic runner 接入策略

**第 29 章 详细实现阶段**

**阶段 0：目录与配置骨架**
- 新建 `examples/evaluation/promptiter_regression_loop/`，含 `main.go`、`go.mod`、`README.md`
- `data/` 下放 6 条样例 case（3 训练 + 3 验证）+ `metrics.json` + baseline prompt + `promptiter.json`
- `promptiter.json` schema 设计：`trainEvalSetID`/`validationEvalSetID`/`metricFileID`/`maxRounds`/`minScoreGain`/`gate`/`targetSurfaceIDs`/`seed`/`modelConfig`/`fakeModelResponses`
- 6 条 case 刻意构造：训练集 3 条（可优化成功场景）；验证集 3 条（1 条可优化成功 + 1 条优化无效 + 1 条优化后退化）

**阶段 1：Fake/Deterministic Runtime 构造**
- 仿 [syncrun/engine.go](file:///home/cyl/agent/trpc-agent-go/examples/evaluation/promptiter/syncrun/engine.go) 的 `buildPromptIterRuntime`，但把 `loadOpenAIModel` 替换为 `test/mock_model.go` 的 `QueueModel`
- 为 5 个 runner（candidate/judge/backwarder/aggregator/optimizer）分别注入预录响应队列
- 固定 `math/rand` 种子保证可复现
- `WithRunDetailsEnabled(true)` + `WithExecutionTraceEnabled(true)` 确保 trace 可用

**阶段 2：Baseline 评测模块**
- 调 `agentEvaluator.Evaluate(ctx, validationEvalSetID, ...)` 与 train 集
- 落盘 `baseline_evalresult.json`
- 提取每 case 的 metric 分、pass/fail、`EvalMetricResultDetails.Reason`、`ExecutionTraces`

**阶段 3：失败归因模块（新建 `pkg/attribution/`）**
- 输入：`[]promptiter.CaseLoss` + `[]CaseResult`（含 trace 与 tool trajectory）
- 输出：`[]Attribution{CaseID, Category, Evidence, Reason}`
- 6 类分类法与判定规则：
  - `FinalResponseMismatch`：finalresponse/text/rouge/llm rubric 失败 + Reason 含"不匹配/missing/incorrect"
  - `ToolCallError`：tooltrajectory 失败（工具名错误/未调用/多调用）
  - `ToolParamError`：tooltrajectory 失败 + Reason 含"arguments/parameter"
  - `RouteError`：transfer/router 相关 trace step + Reason 含"route/transfer/wrong agent"
  - `FormatError`：json/xml/length metric 失败 + Reason 含"format/schema/invalid"
  - `KnowledgeRecallInsufficient`：rubricknowledgerecall/hallucination 失败 + Reason 含"knowledge/recall/hallucinate"
- 判定优先级与兜底（无法分类 → FinalResponseMismatch）
- 订阅 `EventKindRoundLosses` 事件触发归因

**阶段 4：PromptIter 优化集成**
- 直接调 `engine.Run(ctx, RunRequest{...})`
- `RunRequest` 构造：`Train`/`Validation`/`InitialProfile`(baseline)/`Teacher`/`Judge`/`AcceptancePolicy`(扩展)/`StopPolicy`/`MaxRounds`/`TargetSurfaceIDs`
- 复用现有 backwarder/aggregator/optimizer

**阶段 5：逐 case Delta 模块（新建 `pkg/delta/`）**
- 输入：baseline 与 candidate 的 `*engine.EvaluationResult`
- 按 `EvalCaseID` join，输出 `[]CaseDelta{CaseID, BaselineStatus, CandidateStatus, BaselineScore, CandidateScore, Category}`
- 5 类分类：`NewlyPassed`/`NewlyFailed`/`ScoreUp`/`ScoreDown`/`Unchanged`
- 阈值判定（如 `ScoreUp` 需 score 提升 ≥ epsilon）

**阶段 6：多维接受门禁（新建 `pkg/gate/`）**
- 扩展 `AcceptancePolicy` 或在引擎外层包 `GateChain`（借鉴 [evolution/gates.go](file:///home/cyl/agent/trpc-agent-go/evolution/gates.go) 模式）
- 四类 gate 链式调用：
  - `ScoreGainGate`：验证集总分提升 ≥ `MinScoreGain`（复用现有逻辑）
  - `NoNewHardFailGate`：候选不得新增 hard fail（score=0 或 status=fail 的 case）
  - `CriticalCaseGate`：`CriticalCaseIDs` 列表中的 case 不得退化超过 `CriticalCaseMaxRegression`
  - `BudgetGate`：`MaxCost`/`MaxCalls` 不超预算
- 每个 gate 返回 `GateDecision{Passed, Reason}`，任一失败则拒绝候选
- 输出接受/拒绝理由（结构化）

**阶段 7：审计报告模块（新建 `pkg/report/`）**
- 订阅 `Observer` 事件累积 + 接收最终 `RunResult`
- 输出 `optimization_report.json`：
  - `baseline`（分数/逐 case）
  - `candidate`（每轮候选 prompt/分数/逐 case）
  - `delta`（逐 case delta 分类）
  - `gateDecision`（每个 gate 的 Passed/Reason）
  - `failureAttribution`（按类别统计 + 每 case 归因）
  - `cost`/`latency` 摘要（LLM 调用次数/耗时）
  - `seed`/`modelConfig`/`fakeEngineConfig`
- 输出 `optimization_report.md`（人读摘要）：是否接受、理由、关键指标对比表、过拟合检测结论

**阶段 8：单元测试（新建 `*_test.go`）**
- `gate_test.go`：四类 gate 决策（接受/拒绝边界）
- `case_delta_test.go`：5 类 delta 分类
- `attributor_test.go`：6 类归因准确率
- `report_test.go`：JSON/MD 生成与字段完整性
- 全部用 fake/离线数据，无 API Key

**第 30 章 验收标准对齐**
- 逐条对照赛题 6 条验收标准，说明本方案如何满足：
  1. 6 条样例 case 可运行 + 完整报告 → 阶段 0+8
  2. 隐藏样本接受/拒绝决策准确率 ≥ 80% → 多维 gate + 验证集回归
  3. 过拟合场景（训练提升验证退化）必须拒绝 → `NoNewHardFailGate` + 验证集 delta
  4. 失败归因分类准确率 ≥ 75% + 每 case 至少一原因 → 阶段 3 规则覆盖 6 类
  5. fake/trace mode 全流程 ≤ 3 分钟 → 阶段 1 离线 QueueModel
  6. 报告含 baseline/candidate/delta/gate decision/理由 → 阶段 7

**第 31 章 300-500 字方案设计说明**
- 失败归因方法：基于 metric name + Reason 关键词 + trace step 类型的规则分类
- 接受策略：四维 gate 链（ScoreGain + NoNewHardFail + CriticalCase + Budget）
- 防过拟合策略：验证集回归 + 逐 case delta + 关键 case 不可退化 + 训练/验证分数背离检测
- PromptIter 接入方式：复用 `engine.Run`，扩展 `AcceptancePolicy`，订阅 Observer 事件
- 产物审计方式：Observer 事件累积 + RunResult 落盘 JSON/MD，含 seed/modelConfig/cost

### 第十部分：附录

**附录 A：完整文件超链接索引**
- 按子系统分组的所有引用文件可点击链接清单

**附录 B：关键数据流图**
- Agent Loop 数据流、工具调用链、PromptIter 引擎主循环、Graph BSP/DAG 调度、Best-of-N、Hedge、Swarm 转移

**附录 C：构建与测试速查**
- 引用 `AGENTS.md` 命令表

---

## 四、假设与决策（Assumptions & Decisions）

1. **交付形式**：单文件 Markdown 文档，路径 `/home/cyl/agent/trpc-agent-go/PROJECT_DEEP_DIVE.zh.md`（可在用户确认时调整）。
2. **语言**：正文中文，代码标识符/SQL/路径/类型名保持原文。
3. **超链接**：所有文件引用使用 `file:///home/cyl/agent/trpc-agent-go/...` 绝对路径 markdown 链接，链接文本用 basename；行范围用 `#L起-止`。
4. **代码走读**：仅引用"承重"片段（接口签名、关键算法、并发原语、SQL、阈值常量），配行号，不整段复述源文件。
5. **赛题方案**：仅设计层面（架构/阶段/模块划分/判定规则/数据流），**不写实际可运行代码、不修改任何源文件**。方案中的代码示例仅以伪代码/接口签名形式呈现，用于说明设计意图。
6. **覆盖深度**：对用户显式列出的每个组件（agent loop/sandbox/memory/skills & tools/prompt/evaluation/高并发/数据库）均给"架构+关键文件+关键类型/函数行号+数据流+代码走读"五要素。
7. **行号准确性**：行号基于 Phase 1 探索时实际读取的结果；少数大文件取首读窗口行号，文档中标注"约"。
8. **不引入新探索**：本计划基于已完成的全量探索，撰写阶段不再开新 search 子代理。
9. **evolution 澄清**：文档明确说明 `evolution/` 是技能库管理而非 prompt 优化，避免读者混淆；但会借鉴其 gates/AuditEvent 设计模式用于赛题方案。
10. **赛题方案目录**：采用竞赛建议的 `examples/evaluation/promptiter_regression_loop/`，与现有 `examples/evaluation/promptiter/syncrun/` 平行。

---

## 五、验证步骤（Verification）

1. **文件存在性**：文档中每个 `file:///` 链接可通过 Read 工具复核对应文件存在。
2. **行号复核**：关键函数行号可通过 `Grep -n "func.*XXX"` 复核（如 `func.*AddMemory`、`func (f \*Flow) Run`、`func.*executeStep`）。
3. **赛题对齐**：逐条核对赛题 6 阶段要求、6 条交付物、6 条验收标准是否在文档第 27-31 章全部覆盖。
4. **超链接可点击**：在 IDE 中预览 .md，验证所有 `file:///` 链接可跳转。
5. **完整性**：对照用户列出的组件清单（agent loop/sandbox/memory/skills & tools/prompt/evaluation/高并发/数据库）确认每项均有专章。
6. **不修改代码**：撰写完成后 `git status` 应仅显示新增的 `PROJECT_DEEP_DIVE.zh.md`（及本计划文件），无源码改动。

---

## 六、撰写执行顺序（用户批准后）

1. 创建 `PROJECT_DEEP_DIVE.zh.md`，写入第 1 章总览 + 附录 A 链接索引骨架
2. 撰写第二部分（第 2-6 章 Agent 运行时）
3. 撰写第三部分（第 7-10 章 Skills & Tools）
4. 撰写第四部分（第 11 章 Sandbox）
5. 撰写第五部分（第 12-15 章 Memory & 数据库）
6. 撰写第六部分（第 16 章 Prompt）
7. 撰写第七部分（第 17-21 章 Evaluation）
8. 撰写第八部分（第 22-26 章 高并发）
9. 撰写第九部分（第 27-31 章赛题方案设计）
10. 补全附录 B/C，整体校对超链接与行号
