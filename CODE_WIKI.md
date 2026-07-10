# tRPC-Agent-Go Code Wiki

> 本文档为 `tRPC-Agent-Go` 仓库的结构化代码 Wiki，涵盖项目整体架构、主要模块职责、关键类与函数说明、依赖关系以及项目运行方式。
>
> - 仓库根模块路径：`trpc.group/trpc-go/trpc-agent-go`
> - 远程仓库：`https://github.com/cyl6/trpc-agent-go.git`（分支 `main`）
> - License：Apache 2.0

---

## 目录

1. [项目概述](#1-项目概述)
2. [整体架构](#2-整体架构)
3. [目录结构与子模块](#3-目录结构与子模块)
4. [核心模块职责与关键 API](#4-核心模块职责与关键-api)
   - 4.1 [agent —— 智能体核心](#41-agent--智能体核心)
   - 4.2 [runner —— 执行编排器](#42-runner--执行编排器)
   - 4.3 [model —— LLM 模型抽象](#43-model--llm-模型抽象)
   - 4.4 [tool —— 工具生态](#44-tool--工具生态)
   - 4.5 [session —— 会话状态](#45-session--会话状态)
   - 4.6 [memory —— 长期记忆](#46-memory--长期记忆)
   - 4.7 [knowledge —— RAG 知识检索](#47-knowledge--rag-知识检索)
   - 4.8 [graph —— 图工作流引擎](#48-graph--图工作流引擎)
   - 4.9 [skill —— 智能体技能](#49-skill--智能体技能)
   - 4.10 [evolution —— 智能体自演化](#410-evolution--智能体自演化)
   - 4.11 [evaluation —— 评测与基准](#411-evaluation--评测与基准)
   - 4.12 [planner —— 规划器](#412-planner--规划器)
   - 4.13 [event —— 事件流](#413-event--事件流)
   - 4.14 [artifact —— 版本化产物存储](#414-artifact--版本化产物存储)
   - 4.15 [codeexecutor —— 安全代码执行](#415-codeexecutor--安全代码执行)
   - 4.16 [server —— 多协议 HTTP 服务](#416-server--多协议-http-服务)
   - 4.17 [telemetry —— 可观测性](#417-telemetry--可观测性)
   - 4.18 [plugin —— 插件机制](#418-plugin--插件机制)
   - 4.19 [prompt —— 提示词模板](#419-prompt--提示词模板)
   - 4.20 [team —— 多智能体协作](#420-team--多智能体协作)
   - 4.21 [log / internal / storage —— 基础设施](#421-log--internal--storage--基础设施)
5. [internal/flow —— LLM 执行流水线](#5-internalflow--llm-执行流水线)
6. [模块依赖关系](#6-模块依赖关系)
7. [项目运行方式](#7-项目运行方式)
8. [示例索引](#8-示例索引)

---

## 1. 项目概述

**tRPC-Agent-Go** 是一个用于构建生产级 AI 智能体系统的 Go 框架（库/框架，而非单一可运行应用，根目录无 `main.go`）。它在一个 Go 原生技术栈中提供：

- LLM 智能体（`LLMAgent`）与图工作流（`GraphAgent`，功能对标 LangGraph）
- 多智能体编排：链式（Chain）、并行（Parallel）、循环（Cycle）、Swarm/Coordinator 协作
- 丰富的工具生态：函数工具、MCP 工具、Web 搜索、代码执行、自定义服务工具
- 持久化状态：Session、Memory、Artifact、Knowledge 检索
- Agent Skills：可复用的 `SKILL.md` 工作流及其安全执行
- Agent 自演化：Hermes 风格的会话回顾，提取、过闸、发布可复用技能
- Prompt 缓存（最高 90% 成本节省）与 Prompt 迭代（PromptIter）
- 评测（Evaluation）与基准（Benchmark）
- 协议集成：AG-UI（前端）、A2A（智能体互通）、MCP（工具）
- 生产级可观测性：OpenTelemetry tracing/metrics、Langfuse 示例

仓库采用 **Go 多模块 monorepo** 形态，根模块为 `trpc.group/trpc-go/trpc-agent-go`，全仓约 80+ 个 `go.mod` 子模块。根模块依赖 SQLite（CGO），需保证 `CGO_ENABLED=1` 与 C 编译器可用。

---

## 2. 整体架构

### 架构分层

```
┌──────────────────────────────────────────────────────────────────────┐
│  协议接入层  server/ (trpcagent网关 / agui / a2a / openai兼容)        │
│                openclaw/ (OpenClaw 网关分发)                          │
├──────────────────────────────────────────────────────────────────────┤
│  编排层      runner.Runner (会话管理 / 生命周期 / 可取消 / 可中断)    │
├──────────────────────────────────────────────────────────────────────┤
│  智能体层    agent.Agent                                             │
│   ├─ LLMAgent (内置主力)   ├─ ChainAgent  ├─ ParallelAgent           │
│   ├─ CycleAgent            ├─ GraphAgent  └─ Team (Swarm/Coordinator)│
├──────────────────────────────────────────────────────────────────────┤
│  执行引擎    internal/flow/llmflow (请求/响应处理器链)                │
│   RequestProcessors: Basic/Identity/Instruction/Content/Time/        │
│      Planning/PostTool/Skills/OnDemandSession/WorkspaceExec/...      │
│   ResponseProcessors: FunctionCall/CodeExecution/Output/Planning/    │
│      Transfer/...                                                    │
├──────────────────────────────────────────────────────────────────────┤
│  能力层      model(多LLM) tool(工具) session memory knowledge         │
│              skill evolution evaluation planner artifact              │
│              codeexecutor graph prompt plugin                        │
├──────────────────────────────────────────────────────────────────────┤
│  存储层      storage/ (redis s3 qdrant milvus pgvector mysql ...)    │
│  基础设施    log telemetry internal/ (flow/state/jsonrepair/...)     │
└──────────────────────────────────────────────────────────────────────┘
```

### 执行流程（Execution Flow）

1. **Runner** 编排整个执行流水线，负责会话管理与生命周期
2. **Agent** 使用多个专用组件处理请求
3. **Planner** 决定最优策略与工具选择
4. **Tools** 执行具体任务（API 调用、计算、Web 搜索等）
5. **Memory** 维护上下文并从交互中学习
6. **Knowledge** 提供 RAG 文档理解能力
7. **Evolution** 复盘已完成会话，将可复用流程沉淀为托管技能

事件流贯穿全程：`Runner.Run` 返回 `<-chan *event.Event`，所有协议层、智能体层、执行引擎都围绕该事件通道协作。

---

## 3. 目录结构与子模块

### 顶层目录

| 目录 | 类别 | 说明 |
| --- | --- | --- |
| `agent/` | 源码包 | 智能体核心接口与内置实现（llmagent/chainagent/parallelagent/cycleagent/graphagent） |
| `runner/` | 源码包 | 执行编排器 |
| `model/` | 源码包 | LLM 模型抽象与多 provider 实现 |
| `tool/` | 源码包 | 工具生态（function/mcp/duckduckgo/skill/...） |
| `session/` | 源码包 | 会话状态管理 |
| `memory/` | 源码包 | 长期记忆 |
| `knowledge/` | 源码包 | RAG 知识检索 |
| `graph/` | 源码包 | 图工作流引擎 |
| `skill/` | 源码包 | Agent Skills 仓库 |
| `evolution/` | 源码包 | 智能体自演化 |
| `evaluation/` | 源码包 | 评测框架 |
| `planner/` | 源码包 | 规划器 |
| `event/` | 源码包 | 事件类型定义 |
| `artifact/` | 源码包 | 版本化产物存储 |
| `codeexecutor/` | 源码包 | 安全代码执行 |
| `server/` | 源码包 | 多协议 HTTP 服务 |
| `telemetry/` | 源码包 | OpenTelemetry 可观测性 |
| `plugin/` | 源码包 | 插件机制 |
| `prompt/` | 源码包 | 提示词模板 |
| `team/` | 源码包 | 多智能体协作 |
| `log/` | 源码包 | 日志（zap 封装） |
| `internal/` | 源码包 | 框架内部管线（flow/state/jsonrepair/...） |
| `storage/` | 子模块集合 | 后端客户端（redis/s3/qdrant/milvus/...） |
| `openclaw/` | 独立子模块 | OpenClaw 网关分发应用 |
| `examples/` | 示例 | 90+ 可运行示例 |
| `test/` | 独立子模块 | E2E 测试 |
| `docs/` | 文档 | mkdocs 站点源码 |
| `.github/` | CI | workflows 与脚本 |

### 多模块说明

仓库约 80+ 个 `go.mod`。在根目录运行 `go test ./...` 只测试根模块；跨模块测试使用 CI 脚本 `.github/scripts/run-go-tests.sh`，它会：

1. `find` 所有 `go.mod`（排除 `.resource`、`docs`、`examples`、`test`）
2. 进入每个模块目录执行 `go test -coverprofile=coverage.out ./...`
3. 用 Python 脚本把模块限定路径改写为仓库相对路径
4. 合并所有 `coverage/*.out` 为根目录 `coverage.out`

子模块主要分布在：`storage/`（11 个）、`memory/`（7 个）、`session/`（7 个）、`model/`（6 个）、`knowledge/`、`codeexecutor/`（jupyter/container）、`server/`（promptiter/evaluation/agui）、`tool/`、`agent/`（4 个）、`openclaw`、`evaluation`、`test`。

---

## 4. 核心模块职责与关键 API

### 4.1 agent —— 智能体核心

**位置**：`/workspace/agent/`

所有智能体实现统一接口 `agent.Agent`，定义于 [agent.go](file:///workspace/agent/agent.go)。

```go
// agent/agent.go:61-83
type Agent interface {
    Run(ctx context.Context, invocation *Invocation) (<-chan *event.Event, error)
    Tools() []tool.Tool
    Info() Info
    SubAgents() []Agent
    FindSubAgent(name string) Agent
}
```

- `Info` 结构（[agent.go:23-32](file:///workspace/agent/agent.go)）：`Name`、`Description`、`InputSchema`、`OutputSchema`
- `StopError`（[agent.go:39-59](file:///workspace/agent/agent.go)）：表示智能体主动停止
- `SubAgentSetter` 接口（[agent.go:91-96](file:///workspace/agent/agent.go)）：动态设置子智能体
- `CodeExecutor` 接口（[agent.go:100-104](file:///workspace/agent/agent.go)）：暴露代码执行器

#### 子包与内置智能体

| 子包 | 文件 | 类型 | 构造函数 |
| --- | --- | --- | --- |
| `agent/llmagent` | [llm_agent.go](file:///workspace/agent/llmagent/llm_agent.go) | `LLMAgent` | `New(name string, opts ...Option) *LLMAgent`（[L96-289](file:///workspace/agent/llmagent/llm_agent.go)） |
| `agent/chainagent` | [chain_agent.go](file:///workspace/agent/chainagent/chain_agent.go) | `ChainAgent`（顺序执行） | `New(name, opts...) *ChainAgent`（[L40-55](file:///workspace/agent/chainagent/chain_agent.go)） |
| `agent/parallelagent` | [parallel_agent.go](file:///workspace/agent/parallelagent/parallel_agent.go) | `ParallelAgent`（并发合并） | `New(name, opts...) *ParallelAgent`（[L48-61](file:///workspace/agent/parallelagent/parallel_agent.go)） |
| `agent/cycleagent` | [cycle_agent.go](file:///workspace/agent/cycleagent/cycle_agent.go) | `CycleAgent`（循环到终止） | `New(name, opts...) *CycleAgent`（[L40-55](file:///workspace/agent/cycleagent/cycle_agent.go)） |
| `agent/graphagent` | [graph_agent.go](file:///workspace/agent/graphagent/graph_agent.go) | `GraphAgent`（图工作流） | `New(name, g *graph.Graph, opts...) (*GraphAgent, error)`（[L52-100](file:///workspace/agent/graphagent/graph_agent.go)） |

#### LLMAgent 关键选项（[option.go](file:///workspace/agent/llmagent/option.go)）

`Options` 结构非常庞大（[L226-670](file:///workspace/agent/llmagent/option.go)），关键 `WithXxx`：

- `WithModel` / `WithModels` / `WithModelSelector`
- `WithInstruction` / `WithGlobalInstruction` / `WithDescription`
- `WithGenerationConfig` / `WithMaxLLMCalls` / `WithMaxToolIterations`
- `WithTools` / `WithToolSets` / `WithToolFilter`
- `WithSkills` / `WithCodeExecutor` / `WithKnowledge`
- `WithPlanner` / `WithSubAgents`
- `WithAgentCallbacks` / `WithModelCallbacks` / `WithToolCallbacks` / `WithExtensions`
- `WithOutputKey` / `WithOutputSchema` / `WithInputSchema`
- `WithEnableParallelTools`

#### Invocation 与 RunOptions（[invocation.go](file:///workspace/agent/invocation.go)）

- `Invocation` 结构（[L126-237](file:///workspace/agent/invocation.go)）：单次调用上下文，携带 session、memory、artifact、plugins、runOptions
- `RunOption` 类型（[L266](file:///workspace/agent/invocation.go)）、`RunOptions` 结构（[L1104-1409](file:///workspace/agent/invocation.go)）
- 关键 per-run 选项：
  - `WithInstruction`（[L803-807](file:///workspace/agent/invocation.go)）
  - `WithRequestID`（[L658-662](file:///workspace/agent/invocation.go)）
  - `WithSpanAttributes`（[L698-706](file:///workspace/agent/invocation.go)）
  - `WithModel` / `WithModelName` / `WithModelSelector`
  - `WithStream` / `WithMaxRunDuration` / `WithDetachedCancel`
  - `WithStructuredOutputJSONSchema` / `WithStructuredOutputJSON`
  - `WithToolFilter` / `WithAdditionalTools` / `WithExternalTools`
  - `WithGlobalInstruction`
- `InvocationOptions`（[invocation_options.go:22](file:///workspace/agent/invocation_options.go)）含 `WithInvocation*` 构造器

#### 插件管理（[agent/plugins.go:26-46](file:///workspace/agent/plugins.go)）

`PluginManager` 接口暴露 `AgentCallbacks()`、`ModelCallbacks()`、`ToolCallbacks()`、`OnEvent(...)`、`Close(ctx)`，由 `plugin.Manager` 实现。

---

### 4.2 runner —— 执行编排器

**位置**：`/workspace/runner/runner.go`

Runner 是连接 Agent 与 Session/Memory/Evolution 等能力的编排核心。

#### 接口

```go
// runner/runner.go:219-233
type Runner interface {
    Run(ctx context.Context, userID string, sessionID string,
        message model.Message, runOpts ...agent.RunOption,
    ) (<-chan *event.Event, error)
    Close() error
}
```

- `ManagedRunner`（[L241-251](file:///workspace/runner/runner.go)）：扩展 `Runner`，增加 `Cancel(ctx, sessionID)` 与 `RunStatus(ctx, sessionID)`
- `SteerableRunner`（[L258-263](file:///workspace/runner/runner.go)）：再扩展 `EnqueueUserMessage`（安全边界内的运行时转向）

#### 构造函数

- `NewRunner(appName string, ag agent.Agent, opts ...Option) Runner`（[L382-423](file:///workspace/runner/runner.go)）：单一固定 Agent
- `NewRunnerWithAgentFactory(appName, defaultAgentName string, factory AgentFactory, opts ...Option) Runner`（[L431-479](file:///workspace/runner/runner.go)）：按 `RunOptions` 为每次运行构造或复用 Agent

#### 关键 Option（[runner.go](file:///workspace/runner/runner.go)）

| Option | 行号 | 作用 |
| --- | --- | --- |
| `WithSessionService` | L84-88 | 注入会话服务 |
| `WithMemoryService` | L97-101 | 注入记忆服务 |
| `WithSessionIngestor` | L111-115 | 注入会话摄入器 |
| `WithArtifactService` | L131-135 | 注入产物服务 |
| `WithEvolutionService` | L140-144 | 注入自演化服务 |
| `WithAgent` | L147-151 | 注入 Agent |
| `WithAgentFactory` | L158-162 | 注入 Agent 工厂 |
| `WithPlugins` | L165-169 | 注入插件 |
| `WithAwaitUserReplyRouting` | L181-185 | 等待用户回复路由 |
| `WithPersistInterruptedAssistant` | L194-198 | 持久化被中断的助手消息 |
| `WithCandidateSelector` | L205-217 | 候选选择器 |

`AgentFactory` 类型（[L94](file:///workspace/runner/runner.go)）：`func(ctx, ro agent.RunOptions) (agent.Agent, error)`。

`runner.Run`（[L527-777](file:///workspace/runner/runner.go)）是核心编排方法：解析/构造 Agent → 构建 Invocation → 执行 before-runner 回调 → 驱动 Agent 事件通道 → 持久化事件到会话 → 处理中断/等待用户回复 → 执行 after-runner 回调。

---

### 4.3 model —— LLM 模型抽象

**位置**：`/workspace/model/`

#### Model 接口（[model.go:45-57](file:///workspace/model/model.go)）

```go
type Model interface {
    GenerateContent(ctx context.Context, request *Request) (<-chan *Response, error)
    Info() Info
}
```

- `IterModel`（[model.go:66-69](file:///workspace/model/model.go)）：可选扩展，通过回调式 `Seq[*Response]` 迭代器流式输出，降低 goroutine/channel 开销
- `Info`（[model.go:72-78](file:///workspace/model/model.go)）：`Name`、`ContextWindow`
- 双层错误处理：函数级 `error`（系统级失败）+ 响应级 `Response.Error`（API 级失败）

#### 关键类型（[request.go](file:///workspace/model/request.go)）

- `Role`（[L24](file:///workspace/model/request.go)）及常量：`RoleSystem`/`RoleUser`/`RoleAssistant`/`RoleTool`（[L27-32](file:///workspace/model/request.go)）
- `Message`（[L68-89](file:///workspace/model/request.go)）：含多模态助手 `AddFilePath`/`AddFileData`/`AddFileURL`/`AddImageURL`/`AddAudioData` 等
- 消息构造：`NewSystemMessage`/`NewUserMessage`/`NewToolMessage`/`NewAssistantMessage`（[L364-396](file:///workspace/model/request.go)）
- `GenerationConfig`（[L399-459](file:///workspace/model/request.go)）：`MaxTokens`、`Temperature`、`TopP`、`Stream`（[L410](file:///workspace/model/request.go)，控制流式）、`Stop`、`ReasoningEffort`、`ThinkingEnabled`、`ThinkingTokens` 等
- `Request`（[L536-560](file:///workspace/model/request.go)）：`Messages`、内嵌 `GenerationConfig`、`StructuredOutput`、`Tools map[string]tool.Tool`
- `ToolCall`（[L593-607](file:///workspace/model/request.go)）、`StructuredOutput`（[L713-718](file:///workspace/model/request.go)）

#### Response（[response.go](file:///workspace/model/response.go)）

- `Response`（[L174-209](file:///workspace/model/response.go)）：`ID`、`Object`、`Choices`、`Usage`、`Error`、`Done`、`IsPartial`
- `Choice`（[L59-71](file:///workspace/model/response.go)）：含 `Message`（非流式）与 `Delta`（流式增量）
- `Usage`（[L117-135](file:///workspace/model/response.go)）含 `ReasoningTokens` 等明细
- Object 常量（[L25-56](file:///workspace/model/response.go)）：`ObjectTypeChatCompletionChunk = "chat.completion.chunk"`（流式，[L53](file:///workspace/model/response.go)）、`ObjectTypeChatCompletion`（非流式，[L55](file:///workspace/model/response.go)）
- 错误类型常量（[L15-22](file:///workspace/model/response.go)）：`StreamError`/`APIError`/`FlowError`/`RunError`/`Cancelled`

#### 模型实现

| 实现 | 文件 | 构造函数 |
| --- | --- | --- |
| OpenAI（含 DeepSeek/Qwen/GLM/Hunyuan 变体） | [openai/openai.go](file:///workspace/model/openai/openai.go) | `New(name, opts...) *Model`（[L289](file:///workspace/model/openai/openai.go)） |
| Anthropic (Claude) | [anthropic/anthropic.go](file:///workspace/model/anthropic/anthropic.go) | `New(name, opts...) *Model`（[L82](file:///workspace/model/anthropic/anthropic.go)） |
| Gemini | [gemini/gemini.go](file:///workspace/model/gemini/gemini.go) | `New(ctx, name, opts...) (*Model, error)`（[L74](file:///workspace/model/gemini/gemini.go)） |
| Ollama | [ollama/ollama.go](file:///workspace/model/ollama/ollama.go) | `New(name, opts...) *Model`（[L68](file:///workspace/model/ollama/ollama.go)） |
| Hunyuan | [hunyuan/hunyuan.go](file:///workspace/model/hunyuan/hunyuan.go) | `New(name, opts...) *Model`（[L128](file:///workspace/model/hunyuan/hunyuan.go)） |
| AWS Bedrock | [bedrock/bedrock.go](file:///workspace/model/bedrock/bedrock.go) | `New(modelID, opts...) *Model`（[L52](file:///workspace/model/bedrock/bedrock.go)） |
| HuggingFace | [huggingface/huggingface.go](file:///workspace/model/huggingface/huggingface.go) | `New(modelName, opts...) (*Model, error)`（[L55](file:///workspace/model/huggingface/huggingface.go)） |

> **注意**：DeepSeek/Qwen/GLM 并非独立包，而是 OpenAI 兼容适配器的变体（`VariantDeepSeek`/`VariantQwen`/`VariantGLM`，[openai.go:62-77](file:///workspace/model/openai/openai.go)），DeepSeek 通过 base URL 自动识别（`inferVariant`/`isDeepSeekBaseURL`）。

**元模型（容错）**：`model/hedge`（`New`，[hedge.go:65](file:///workspace/model/hedge/hedge.go)）、`model/failover`（`New`，[failover.go:33](file:///workspace/model/failover/failover.go)）。

#### Provider 注册表（[provider/provider.go](file:///workspace/model/provider/provider.go)）

- `Provider` 类型（[L35](file:///workspace/model/provider/provider.go)）
- `Register`/`Get`/`Model(providerName, modelName, opt...)` 统一构造（[L43/50/58](file:///workspace/model/provider/provider.go)）
- 已注册：`openai`、`anthropic`、`gemini`、`ollama`、`hunyuan`

#### 上下文窗口注册表（[registry.go](file:///workspace/model/registry.go)）

`RegisterModelContextWindow`/`LookupModelContextWindow`（[L20/L38](file:///workspace/model/registry.go)）。

---

### 4.4 tool —— 工具生态

**位置**：`/workspace/tool/`

#### 核心接口（[tool.go](file:///workspace/tool/tool.go)）

```go
// tool/tool.go:17-20
type Tool interface {
    Declaration() *Declaration
}

// tool/tool.go:23-29 —— 可调用工具
type CallableTool interface {
    Tool
    Call(ctx context.Context, jsonArgs []byte) (any, error)
}

// tool/tool.go:34-41 —— 流式工具
type StreamableTool interface {
    StreamableCall(ctx context.Context, jsonArgs []byte) (*StreamReader, error)
}
```

> 仓库中**没有 `Invoker` 类型**，工具调用契约是 `CallableTool.Call`。

- `Declaration`（[tool.go:44-56](file:///workspace/tool/tool.go)）：`Name`、`Description`、`InputSchema`、`OutputSchema`
- `Schema`（[tool.go:62-83](file:///workspace/tool/tool.go)）：JSON-Schema 表示
- `ToolSet` 接口（[toolset.go:16-25](file:///workspace/tool/toolset.go)）：`Tools(ctx)`、`Close()`、`Name()`

#### function 子包（[tool/function/function_tool.go](file:///workspace/tool/function/function_tool.go)）

- `FunctionTool[I, O any]` 泛型结构（[L30-41](file:///workspace/tool/function/function_tool.go)）
- `NewFunctionTool[I, O any](fn, opts...) *FunctionTool[I, O]`（[L120](file:///workspace/tool/function/function_tool.go)）：通过反射自动生成 JSON Schema
- 选项：`WithName`（[L66](file:///workspace/tool/function/function_tool.go)）、`WithDescription`（[L73](file:///workspace/tool/function/function_tool.go)）、`WithLongRunning`、`WithSkipSummarization`、`WithInputSchema`、`WithOutputSchema`
- 流式变体 `StreamableFunctionTool` 与 `NewStreamableFunctionTool`（[L244](file:///workspace/tool/function/function_tool.go)）

#### mcp 子包（[tool/mcp/](file:///workspace/tool/mcp/)）

- 入口为 `NewMCPToolSet(config ConnectionConfig, opts ...ToolSetOption) *ToolSet`（[toolset.go:68](file:///workspace/tool/mcp/toolset.go)）（**不是 `New`**）
- `ConnectionConfig`（[config.go:58-80](file:///workspace/tool/mcp/config.go)）：`Transport`（stdio/sse/streamable）、`ServerURL`、`Command`、`Args`、`Headers`、`Timeout`
- `Init(ctx)`（[toolset.go:102](file:///workspace/tool/mcp/toolset.go)）建立会话并预加载工具

#### 其他工具子包

| 子包 | 说明 |
| --- | --- |
| `tool/duckduckgo` | DuckDuckGo 搜索（`NewTool`，[duckduckgo.go:145](file:///workspace/tool/duckduckgo/duckduckgo.go)，支持 api/html/lite 后端） |
| `tool/google/search` | Google Custom Search（ToolSet） |
| `tool/webfetch/{httpfetch,claudefetch,geminifetch}` | Web 抓取 |
| `tool/arxivsearch`、`tool/wikipedia` | 学术/百科搜索 |
| `tool/claudecode` | Claude Code 工具集（web_search/web_fetch/bash/read 等） |
| `tool/email`、`tool/file`、`tool/todo` | 通用工具 |
| `tool/codeexec`、`tool/hostexec`、`tool/workspaceexec` | 代码执行类工具 |
| `tool/openapi` | OpenAPI 规范转工具 |
| `tool/transfer`、`tool/awaitreply` | 智能体转接/等待回复 |
| `tool/skill` | Agent Skills 工具（见 [4.9](#49-skill--智能体技能)） |
| `tool/agent` | 智能体即工具 |

---

### 4.5 session —— 会话状态

**位置**：`/workspace/session/`

#### Session 结构（[session.go:63-90](file:///workspace/session/session.go)）

```go
type Session struct {
    ID, AppName, UserID string
    State    StateMap          // map[string][]byte
    Events   []event.Event
    EventMu  sync.RWMutex
    Tracks   map[Track]*TrackEvents
    TracksMu sync.RWMutex
    SummariesMu sync.RWMutex
    Summaries   map[string]*Summary
    UpdatedAt, CreatedAt time.Time
    Hash        int
    ServiceMeta map[string]string
    stateMu     sync.RWMutex
}
```

- `NewSession(appName, userID, sessionID, opts...) *Session`（[L231](file:///workspace/session/session.go)）
- 方法：`Clone`、`GetState`/`SetState`/`DeleteState`/`SnapshotState`、`GetEvents`/`AppendTrackEvent`/`GetTrackEvents`、`ApplyEventFiltering`、`ApplyEventStateDelta`
- `Summary`（[L588-593](file:///workspace/session/session.go)）、`SummaryBoundary`（[L610-615](file:///workspace/session/session.go)）

#### Service 接口（[session.go:1048-1114](file:///workspace/session/session.go)）

```go
type Service interface {
    CreateSession(ctx, key Key, state StateMap, options ...Option) (*Session, error)
    GetSession(ctx, key Key, options ...Option) (*Session, error)
    ListSessions(ctx, userKey UserKey, options ...Option) ([]*Session, error)
    DeleteSession(ctx, key Key, options ...Option) error
    UpdateAppState / DeleteAppState / ListAppStates
    UpdateUserState / ListUserStates / DeleteUserState
    UpdateSessionState(ctx, key Key, state StateMap) error
    AppendEvent(ctx, session *Session, event *event.Event, options ...Option) error
    CreateSessionSummary / EnqueueSummaryJob / GetSessionSummaryText
    Close() error
}
```

> 接口名为 `Service`（非 `SessionService`）。

- 扩展接口：`SearchableService`（[L989-1001](file:///workspace/session/session.go)，向量/语义搜索）、`WindowService`（[L1038-1045](file:///workspace/session/session.go)）
- `Key`（`AppName`/`UserID`/`SessionID`，[L1117-1121](file:///workspace/session/session.go)）、`UserKey`（[L1134-1137](file:///workspace/session/session.go)）
- `SearchMode`：`SearchModeDense`/`SearchModeHybrid`（[L901-907](file:///workspace/session/session.go)）

#### 实现

- 内存：`session/inmemory`，结构 `SessionService`，`NewSessionService(opts...)`（[inmemory/service.go:193](file:///workspace/session/inmemory/service.go)），实现 `Service`/`TrackService`/`WindowService`，支持 TTL 与异步摘要 worker
- 后端：`session/redis`、`session/mysql`、`session/postgres`、`session/sqlite`、`session/mongodb`、`session/clickhouse`、`session/pgvector`、`session/noop`

---

### 4.6 memory —— 长期记忆

**位置**：`/workspace/memory/`

#### Service 接口（[memory.go:169-202](file:///workspace/memory/memory.go)）

```go
type Service interface {
    Reader  // ReadMemories, SearchMemories
    AddMemory(ctx, userKey UserKey, memory string, topics []string, opts ...AddOption) error
    UpdateMemory(ctx, memoryKey Key, memory string, topics []string, opts ...UpdateOption) error
    DeleteMemory(ctx, memoryKey Key) error
    ClearMemories(ctx, userKey UserKey) error
    Tools() []tool.Tool
    EnqueueAutoMemoryJob(ctx, sess *session.Session) error
    Close() error
}
```

- `Reader`（[L156-166](file:///workspace/memory/memory.go)）：`ReadMemories`、`SearchMemories`
- 工具名常量（[L23-30](file:///workspace/memory/memory.go)）：`memory_add`/`memory_update`/`memory_delete`/`memory_clear`/`memory_search`/`memory_load`
- `Kind`：`KindFact = "fact"`、`KindEpisode = "episode"`（[L209-217](file:///workspace/memory/memory.go)）
- `Memory`（[L221-231](file:///workspace/memory/memory.go)）、`Entry`（[L234-242](file:///workspace/memory/memory.go)）
- `SearchOptions`（[L273-315](file:///workspace/memory/memory.go)）：支持 hybrid/RRF、时间过滤、相似度阈值

#### 实现

- 内存：`memory/inmemory`，`NewMemoryService(opts...) *MemoryService`（[inmemory/service.go:59](file:///workspace/memory/inmemory/service.go)）
  > 注意：构造函数是 `inmemory.NewMemoryService`，**不存在 `memorysvc.NewInMemoryService`**。
- Redis：`memory/redis`，`NewService(opts...) (*Service, error)`（[redis/service.go:51](file:///workspace/memory/redis/service.go)），存储布局 `appName+userID -> hash[memoryID->Entry]`
- 其他后端：`postgres`、`mysql`、`sqlite`、`sqlitevec`、`mysqlvec`、`pgvector`、`tencentdb`、`mem0`

#### 记忆工具集成（[memory/tool/tool.go](file:///workspace/memory/tool/tool.go)）

六个工具均为 `function.NewFunctionTool`，从 invocation 上下文获取 `MemoryService` 与 app/user 身份：

| 构造函数 | 工具名 | 作用 |
| --- | --- | --- |
| `NewAddTool()` | `memory_add` | 新增持久记忆 |
| `NewUpdateTool()` | `memory_update` | 按 ID 更新记忆 |
| `NewDeleteTool()` | `memory_delete` | 按 ID 删除记忆 |
| `NewClearTool()` | `memory_clear` | 清空用户记忆 |
| `NewSearchTool()` | `memory_search` | 语义/hybrid 搜索 |
| `NewLoadTool()` | `memory_load` | 加载最近记忆 |

工具注册表在 [memory/internal/memory/memory.go](file:///workspace/memory/internal/memory/memory.go)：`AllToolCreators`（[L192-199](file:///workspace/memory/internal/memory/memory.go)）、`DefaultEnabledTools`（[L203-208](file:///workspace/memory/internal/memory/memory.go)）、`BuildToolsList`（[L281](file:///workspace/memory/internal/memory/memory.go)）。

---

### 4.7 knowledge —— RAG 知识检索

**位置**：`/workspace/knowledge/`

#### Knowledge 接口（[knowledge.go:23-28](file:///workspace/knowledge/knowledge.go)）

```go
type Knowledge interface {
    Search(ctx context.Context, req *SearchRequest) (*SearchResult, error)
}
```

- `SearchRequest`（[L31-57](file:///workspace/knowledge/knowledge.go)）：`Query`、`History`、`UserID`、`SessionID`、`MaxResults`、`MinScore`、`SearchFilter`、`SearchMode`
- `SearchResult`（[L64-74](file:///workspace/knowledge/knowledge.go)）

#### BuiltinKnowledge（[default.go](file:///workspace/knowledge/default.go)）

- 结构持有 `vectorStore`、`embedder`、`retriever`、`queryEnhancer`、`reranker`、`sources`（[L53-70](file:///workspace/knowledge/default.go)）
- `New(opts ...Option)`（[L114-140](file:///workspace/knowledge/default.go)）
- `AddSource`/`ReloadSource`/`RemoveSource`/`Load`/`Search`（[L193/L226/L347/L386/L1067](file:///workspace/knowledge/default.go)）

#### 核心子接口

| 子包 | 接口 | 文件 |
| --- | --- | --- |
| `embedder` | `Embedder`（`GetEmbedding`/`GetEmbeddingWithUsage`/`GetDimensions`） | [embedder/embedder.go:44-65](file:///workspace/knowledge/embedder/embedder.go) |
| `retriever` | `Retriever`（`Retrieve`/`Close`） | [retriever/retriever.go:22-28](file:///workspace/knowledge/retriever/retriever.go) |
| `vectorstore` | `VectorStore`（`Add/Get/Update/Delete/Search/Count/...`） | [vectorstore/vectorstore.go:22-55](file:///workspace/knowledge/vectorstore/vectorstore.go) |
| `source` | `Source`（`ReadDocuments`/`Name`/`Type`） | [source/source.go:101-114](file:////workspace/knowledge/source/source.go) |
| `document` | `Document` 结构 | [document/document.go:18-42](file:///workspace/knowledge/document/document.go) |
| `reranker` | `Reranker` | [reranker/reranker.go](file:///workspace/knowledge/reranker/reranker.go) |
| `query` | 查询增强器 | [query/query.go](file:///workspace/knowledge/query/query.go) |
| `chunking` | 文档分块策略 | [chunking/chunking.go](file:///workspace/knowledge/chunking/chunking.go) |

- `SearchMode` 常量（[vectorstore.go:277-286](file:///workspace/knowledge/vectorstore/vectorstore.go)）：`SearchModeHybrid`/`SearchModeVector`/`SearchModeKeyword`/`SearchModeFilter`
- Source 类型：`TypeAuto`/`TypeFile`/`TypeDir`/`TypeRepo`/`TypeURL`（[source.go:23-29](file:///workspace/knowledge/source/source.go)）
- FileReader 类型：text/markdown/json/csv/pdf/docx/proto/go（[source.go:35-58](file:///workspace/knowledge/source/source.go)）
- 向量库后端：in-memory、pgvector、milvus、qdrant、elasticsearch

---

### 4.8 graph —— 图工作流引擎

**位置**：`/workspace/graph/`

`GraphAgent` 底层运行时，对标 LangGraph。

#### 核心类型（[graph.go](file:///workspace/graph/graph.go)）

- 特殊节点：`Start = "__start__"`（[L32](file:///workspace/graph/graph.go)）、`End = "__end__"`（[L34](file:///workspace/graph/graph.go)）
- `NodeFunc func(ctx, state State) (any, error)`（[L54](file:///workspace/graph/graph.go)）
- `ConditionalFunc`（[L62](file:///workspace/graph/graph.go)）、`MultiConditionalFunc`（[L65](file:///workspace/graph/graph.go)）、`UniversalCondFunc`（[L68](file:///workspace/graph/graph.go)）
- `Node`（[L111-206](file:///workspace/graph/graph.go)）、`Edge`（[L210-213](file:///workspace/graph/graph.go)）、`ConditionalEdge`（[L216-220](file:///workspace/graph/graph.go)）
- `Graph` 结构（[L229-246](file:///workspace/graph/graph.go)）：不可变编译产物
- `Command`（[L559-564](file:///workspace/graph/graph.go)）：`Update`/`GoTo`/`Resume`/`ResumeMap`
- `validate()`（[L447-484](file:///workspace/graph/graph.go)）

#### StateGraph 流式构建器（[state_graph.go](file:///workspace/graph/state_graph.go)）

- `StateGraph` 结构（[L72-75](file:///workspace/graph/state_graph.go)）
- `NewStateGraph(schema *StateSchema) *StateGraph`（[L82-86](file:///workspace/graph/state_graph.go)）
- 流式方法（均返回 `*StateGraph` 便于链式）：
  - `AddNode(id, function NodeFunc, opts...)`（[L603-629](file:///workspace/graph/state_graph.go)）
  - `AddLLMNode`/`AddToolsNode`/`AddAgentNode`/`AddSubgraphNode`
  - `AddEdge(from, to)`（[L822](file:///workspace/graph/state_graph.go)）、`AddJoinEdge(fromNodes, to)`（[L848](file:///workspace/graph/state_graph.go)）
  - `AddConditionalEdges(from, condFunc, pathMap)`（[L939-954](file:///workspace/graph/state_graph.go)）：单目标条件路由
  - `AddMultiConditionalEdges(from, condFunc, pathMap)`（[L958-973](file:///workspace/graph/state_graph.go)）：并行多目标条件路由
  - `AddToolsConditionalEdges(fromLLMNode, toToolsNode, fallbackNode)`（[L978-1009](file:///workspace/graph/state_graph.go)）
  - `SetEntryPoint(nodeID)`（[L1013-1021](file:///workspace/graph/state_graph.go)）
  - `SetFinishPoint(nodeID)`（[L1025-1028](file:///workspace/graph/state_graph.go)）
  - `Compile() (*Graph, error)`（[L1031-1039](file:///workspace/graph/state_graph.go)）/ `MustCompile()`（[L1091-1097](file:///workspace/graph/state_graph.go)）

#### 端到端构建流程

`NewStateGraph(schema)` → `AddNode`/`AddEdge`/`AddConditionalEdges`/`AddMultiConditionalEdges`/`SetEntryPoint`/`SetFinishPoint` → `Compile()` → `graph.NewExecutor(g, opts...)` 或 `graphagent.New(name, g, opts...)` 包装为 `agent.Agent`。

#### 其他文件

- [executor.go](file:///workspace/graph/executor.go)：`Executor`、`NewExecutor`、`WithChannelBufferSize`/`WithMaxConcurrency`/`WithCheckpointSaver`/`WithMaxSteps`/`WithStepTimeout` 等
- [checkpoint.go](file:///workspace/graph/checkpoint.go)：`Checkpoint`、`CheckpointSaver` 接口
- [execution_engine.go](file:///workspace/graph/execution_engine.go)：`ExecutionEngine`（BSP 等）
- [interrupt.go](file:///workspace/graph/interrupt.go)、[resume.go](file:///workspace/graph/resume.go)、[time_travel.go](file:///workspace/graph/time_travel.go)：中断/恢复/时间旅行
- 检查点后端：`graph/checkpoint/inmemory`、`graph/checkpoint/redis`、`graph/checkpoint/sqlite`

#### GraphAgent 选项（[graphagent/option.go](file:///workspace/agent/graphagent/option.go)）

`Options`（[L93-172](file:///workspace/agent/graphagent/option.go)）支持：`WithSubAgents`、`WithInitialState`、`WithMaxConcurrency`、`WithCheckpointSaver`、`WithExecutionEngine`、`WithAddSessionSummary`、`WithEnableContextCompaction`、`WithMaxHistoryRuns`、`WithExecutorOptions`（转发 `graph.WithMaxSteps` 等）等。

---

### 4.9 skill —— 智能体技能

**位置**：`/workspace/skill/` 与 `/workspace/tool/skill/`

#### 仓库（[skill/repository.go](file:///workspace/skill/repository.go)）

- 常量：`skillFile = "SKILL.md"`（[L37](file:///workspace/skill/repository.go)）、`EnvSkillsRoot = "SKILLS_ROOT"`（[L46](file:///workspace/skill/repository.go)）
- `Repository` 接口（[L68-76](file:///workspace/skill/repository.go)）：`Summaries()`/`Get(name)`/`Path(name)`
- `RefreshableRepository` 接口（[L86-89](file:///workspace/skill/repository.go)）：扩展 `Refresh() error`
- `FSRepository` 结构（[L92-97](file:///workspace/skill/repository.go)）
- `NewFSRepository(roots...)`（[L112-131](file:///workspace/skill/repository.go)）：支持多 root、HTTP(S) URL（zip/tar.gz 归档，缓存目录可由 `SKILLS_CACHE_DIR` 覆盖）
- `Refresh()`（[L134-143](file:///workspace/skill/repository.go)）：写锁下重扫并原子替换索引

#### 技能工具（tool/skill/）

> "skilltool" 实际位于 `/workspace/tool/skill/`，包名 `skill`，导入路径 `trpc.group/trpc-go/trpc-agent-go/tool/skill`。

| 工具 | 文件 | 构造函数 | 作用 |
| --- | --- | --- | --- |
| `skill_load` | [load.go](file:///workspace/tool/skill/load.go) | `NewLoadTool(repo)`（[L63](file:///workspace/tool/skill/load.go)） | 加载技能与文档 |
| `skill_list_docs` | [list_docs.go](file:///workspace/tool/skill/list_docs.go) | `NewListDocsTool(repo)`（[L34](file:///workspace/tool/skill/list_docs.go)） | 列出技能文档 |
| `skill_select_docs` | [select_docs.go](file:///workspace/tool/skill/select_docs.go) | `NewSelectDocsTool(repo)`（[L51](file:///workspace/tool/skill/select_docs.go)） | 选择文档（add/replace/clear） |
| `skill_run` | [run.go](file:///workspace/tool/skill/run.go) | `NewRunTool(...)`（[L93-119](file:///workspace/tool/skill/run.go)） | 隔离工作区执行命令 |
| `skill_exec` | [exec.go](file:///workspace/tool/skill/exec.go) | `NewExecTool(...)`（[L131-138](file:///workspace/tool/skill/exec.go)） | 交互式会话执行 |
| `skill_write_stdin` | exec.go | `NewWriteStdinTool(...)`（[L141](file:///workspace/tool/skill/exec.go)） | 写入会话 stdin |
| `skill_poll_session` | exec.go | `NewPollSessionTool(...)`（[L146](file:///workspace/tool/skill/exec.go)） | 轮询会话输出 |
| `skill_kill_session` | exec.go | `NewKillSessionTool(...)`（[L151](file:///workspace/tool/skill/exec.go)） | 终止会话 |

`skill_run` 选项：`WithAllowedCommands`、`WithDeniedCommands`、`WithForceSaveArtifacts`、`WithRunOutputLimits`、`WithRequireSkillLoaded`、`WithWorkspaceRegistry`。

---

### 4.10 evolution —— 智能体自演化

**位置**：`/workspace/evolution/`

异步管线：复盘已完成会话 → LLM 审阅提取技能 → 质量过闸 → 发布托管技能。

#### Service 与构造（[service.go](file:///workspace/evolution/service.go)）

- `NewService(reviewModel model.Model, opts ...Option) (Service, error)`（[L25-73](file:////workspace/evolution/service.go)）
- `Service` 接口（[types.go:35-38](file:///workspace/evolution/types.go)）：`EnqueueLearningJob`/`Close`
- `service.Close()`（[L82-85](file:///workspace/evolution/service.go)）、`ApprovalGateMetrics()`

#### 选项（[options.go](file:///workspace/evolution/options.go)）

- 仓库/作用域：`WithManagedSkillsDir`（[L59-61](file:///workspace/evolution/options.go)）、`WithSkillRepository`（[L65-67](file:///workspace/evolution/options.go)）、`WithSkillRepositoryProvider`、`WithSkillScopeMode`
- 管线：`WithReviewPolicy`、`WithPublisher`、`WithWorkerNum`、`WithQueueSize`、`WithReviewer`、`WithReviewerOptions`
- 质量过闸：`WithSpecGate`、`WithSafetyGate`、`WithEffectivenessGate`、`WithHumanGate`、`WithApprovalGateShadow`、`WithApprovalTimeout`、`WithApprovalSweepInterval`

#### 核心类型（[types.go](file:///workspace/evolution/types.go)）

- `LearningJob`（[L45-65](file:///workspace/evolution/types.go)）、`Outcome`（[L116-134](file:///workspace/evolution/types.go)）
- `OutcomeStatus`（[L101-107](file:///workspace/evolution/types.go)）：`Unknown`/`Success`/`Partial`/`Fail`/`AgentError`
- `ReviewInput`（[L137-156](file:///workspace/evolution/types.go)）、`ReviewDecision`（[L182-191](file:///workspace/evolution/types.go)）、`SkillSpec`（[L206-212](file:///workspace/evolution/types.go)）

#### Worker（[worker.go](file:///workspace/evolution/worker.go)）

- `processJob`（[L324-425](file:///workspace/evolution/worker.go)）：扫描会话 delta → 调用审阅器 → 应用决策
- `runGates`（[L793-818](file:///workspace/evolution/worker.go)）：按 spec → safety → effectiveness → human 顺序执行
- `publishRevision`（[L904-978](file:///workspace/evolution/worker.go)）：写入技能文件、更新活跃指针、追加审计记录

#### Publisher（[publisher.go](file:///workspace/evolution/publisher.go)）

- `Publisher` 接口（[L22-28](file:///workspace/evolution/publisher.go)）：`UpsertSkill`/`DeleteSkill`
- `NewFilePublisher(root)`（[L42-44](file:///workspace/evolution/publisher.go)）

#### Reviewer（[reviewer.go](file:///workspace/evolution/reviewer.go)）

- `Reviewer` 接口（[L24-26](file:///workspace/evolution/reviewer.go)）
- `LLMReviewer`（[L99-102](file:///workspace/evolution/reviewer.go)）、`NewLLMReviewer(m, opts...)`（[L117-126](file:///workspace/evolution/reviewer.go)）
- `Review`（[L130-184](file:///workspace/evolution/reviewer.go)）：构建 prompt → 流式模型响应 → 解析 JSON 决策

---

### 4.11 evaluation —— 评测与基准

**位置**：`/workspace/evaluation/`

#### 主评测器（[evaluation.go](file:///workspace/evaluation/evaluation.go)）

- `AgentEvaluator` 接口（[L42-47](file:///workspace/evaluation/evaluation.go)）：`Evaluate(ctx, evalSetID, opt...) (*EvaluationResult, error)`、`Close()`
- `New(appName, runner, opt...)`（[L50-117](file:///workspace/evaluation/evaluation.go)）
- `Evaluate`（[L186-214](file:///workspace/evaluation/evaluation.go)）：收集 per-case 结果并汇总
- `runEvaluationInParallel`（[L395-419](file:///workspace/evaluation/evaluation.go)）：`errgroup` 并行
- `aggregateCaseRuns`（[L529-588](file:///workspace/evaluation/evaluation.go)）：跨 run 平均指标

#### 选项（[options.go](file:///workspace/evaluation/options.go)）

- `WithNumRuns(numRuns int)`（[L163-167](file:///workspace/evaluation/options.go)）：每 case 运行次数（用于平均，默认 1）
- 其他：`WithEvalSetManager`、`WithMetricManager`、`WithUserSimulator`、`WithJudgeRunner`、`WithExpectedRunner`、`WithToolMockRunner`、`WithRunOptions`、`WithEvalCaseParallelism`

#### EvalSet（[evalset/evalset.go](file:///workspace/evaluation/evalset/evalset.go)）

- `EvalSet`（[L20-31](file:///workspace/evaluation/evalset/evalset.go)）、`Manager` 接口（[L34-52](file:///workspace/evaluation/evalset/evalset.go)）
- 后端：in-memory、MySQL

#### Metrics（[metric/metric.go](file:///workspace/evaluation/metric/metric.go)）

- `EvalMetric`（[L20-25](file:///workspace/evaluation/metric/metric.go)）、`Manager` 接口（[L28-40](file:///workspace/evaluation/metric/metric.go)）

#### PromptIter（[evaluation/workflow/promptiter/](file:///workspace/evaluation/workflow/promptiter/)）

自动 Prompt 迭代域模型与工作流：`Profile`、`SurfaceGradient`、`PatchSet`、`LossSeverity`（P0-P3）；子包 `engine/`（`Engine` 接口）、`optimizer/`、`backwarder/`、`aggregator/`、`manager/`、`store/`。

---

### 4.12 planner —— 规划器

**位置**：`/workspace/planner/`

#### 接口（[planner.go:25-41](file:///workspace/planner/planner.go)）

```go
type Planner interface {
    BuildPlanningInstruction(ctx, invocation, llmRequest) string
    ProcessPlanningResponse(ctx, invocation, response) *model.Response
}
```

#### 内置实现（[builtin/builtin_planner.go](file:///workspace/planner/builtin/builtin_planner.go)）

面向思考型模型，不生成显式指令，而是配置模型内部思考：

- `Planner` 结构（[L51-70](file:///workspace/planner/builtin/builtin_planner.go)）：`reasoningEffort`、`thinkingEnabled`、`thinkingTokens`
- `New(opts Options)`（[L107-113](file:///workspace/planner/builtin/builtin_planner.go)）
- 支持模型：OpenAI o-series（`reasoning_effort`）、DeepSeek v4（`reasoning_effort` + `thinking_enabled`）、Claude/Gemini 经 OpenAI API（`thinking_enabled` + `thinking_tokens`）
- `BuildPlanningInstruction`（[L123-143](file:///workspace/planner/builtin/builtin_planner.go)）应用思考配置后返回空串

其他实现：`planner/react`（ReAct）、`planner/a2ui`（A2UI）。

---

### 4.13 event —— 事件流

**位置**：`/workspace/event/`

#### Event 结构（[event.go:99-169](file:///workspace/event/event.go)）

内嵌 `*model.Response`，额外字段：`RequestID`、`InvocationID`、`ParentInvocationID`、`ParentMetadata`、`Author`、`ID`、`Timestamp`、`Branch`、`Tag`、`RequiresCompletion`、`LongRunningToolIDs`、`StateDelta`、`Extensions`、`StructuredOutput`、`ExecutionTrace`、`Actions`、`FilterKey`、`Version`。

- `ParentInvocationMetadata`（[L84-96](file:///workspace/event/event.go)）：`TriggerType`/`TriggerID`/`TriggerName`
- `TriggerType` 常量（[L63-73](file:///workspace/event/event.go)）：`TriggerTypeToolCall`/`TriggerTypeTransfer`/`TriggerTypeDynamicWorkflow`
- `EventActions`（[L189-194](file:///workspace/event/event.go)）：`SkipSummarization`
- 构造/辅助：`New`（[L328-341](file:///workspace/event/event.go)）、`NewErrorEvent`、`NewResponseEvent`、`EmitEvent`（[L423](file:///workspace/event/event.go)）、`EmitEventWithTimeout`（[L472](file:///workspace/event/event.go)）、`IsRunnerCompletion`、`IsError`、`IsTerminalError`

#### State Delta 机制

工具响应通过 `Event.StateDelta` 字段将状态变更传回运行时（如 `skill_load` 通报已加载技能状态）。

---

### 4.14 artifact —— 版本化产物存储

**位置**：`/workspace/artifact/`

#### 核心类型（[artifact.go](file:///workspace/artifact/artifact.go)）

- `Artifact`（[L16-27](file:///workspace/artifact/artifact.go)）：`Data`、`MimeType`、`URL`、`Name`
- `SessionInfo`（[L30-37](file:///workspace/artifact/artifact.go)）：`AppName`/`UserID`/`SessionID`

#### Service 接口（[service.go:15-77](file:///workspace/artifact/service.go)）

`SaveArtifact`（返回 revision ID，首次为 0）、`LoadArtifact`（`version=nil` 取最新）、`ListArtifactKeys`、`DeleteArtifact`、`ListVersions`。

#### 后端

- 内存：[inmemory/service.go](file:///workspace/artifact/inmemory/service.go)，`NewService()`（[L34](file:///workspace/artifact/inmemory/service.go)）
- S3：[s3/service.go](file:///workspace/artifact/s3/service.go)，`NewService(ctx, bucket, opts...)`（[L52-81](file:///workspace/artifact/s3/service.go)），支持 AWS S3/MinIO/DigitalOcean Spaces/Cloudflare R2
  - 对象 key：`{app}/{user}/user/{filename}/{version}` 或 `{app}/{user}/{session}/{filename}/{version}`
  - 注意：同 filename 写入非并发安全（[L95-98](file:///workspace/artifact/s3/service.go) 文档）
- COS：[cos/service.go](file:///workspace/artifact/cos/service.go)（腾讯云 COS）

---

### 4.15 codeexecutor —— 安全代码执行

**位置**：`/workspace/codeexecutor/`

#### 核心接口（[codeexecutor.go](file:///workspace/codeexecutor/codeexecutor.go)）

```go
// L24
type CodeExecutor interface {
    ExecuteCode(ctx, CodeExecutionInput) (CodeExecutionResult, error)
    CodeBlockDelimiter() CodeBlockDelimiter
}
```

- `InteractiveProgramRunner` 接口（[interactive.go:85](file:///workspace/codeexecutor/interactive.go)）：`StartProgram(ctx, ws, spec) (ProgramSession, error)`，支持多轮交互
- `ProgramSession` 接口（[interactive.go:55](file:///workspace/codeexecutor/interactive.go)）：`ID()`/`Poll`/`Log`/`Write`/`Kill`/`Close`
- `Workspace`（[workspace.go:52](file:///workspace/codeexecutor/workspace.go)）、`WorkspaceManager`（[L117](file:///workspace/codeexecutor/workspace.go)）、`WorkspaceFS`（[L124](file:///workspace/codeexecutor/workspace.go)）、`ProgramRunner`（[L145](file:///workspace/codeexecutor/workspace.go)）
- `Engine` 接口（[L175](file:///workspace/codeexecutor/workspace.go)）、`EngineProvider`（[L185](file:///workspace/codeexecutor/workspace.go)）、`Capabilities`（[L151](file:///workspace/codeexecutor/workspace.go)，含安全相关 `SupportsCleanEnv`）
- `NewEngine`（[L206](file:///workspace/codeexecutor/workspace.go)）/ `NewEngineWithCapabilities`（[L222](file:///workspace/codeexecutor/workspace.go)）
- `WorkspaceRegistry`（[registry.go:19](file:////workspace/codeexecutor/registry.go)）、`NewWorkspaceRegistry()`（[L32](file:///workspace/codeexecutor/registry.go)）

#### 后端

| 后端 | 文件 | 说明 |
| --- | --- | --- |
| local | [local/local.go](file:///workspace/codeexecutor/local/local.go) | 本地执行（`New`，[L99-111](file:///workspace/codeexecutor/local/local.go)），文档明确标注 **不安全** |
| sandbox | [sandbox/](file:///workspace/codeexecutor/sandbox/) | OS 沙箱（Linux bubblewrap / macOS seatbelt），`New`（[executor.go:34](file:///workspace/codeexecutor/sandbox/executor.go)），含权限/网络/文件系统策略 |
| e2b | [e2b/](file:///workspace/codeexecutor/e2b/) | 云沙箱 |
| jupyter | [jupyter/](file:///workspace/codeexecutor/jupyter/) | Jupyter kernel（独立 go.mod） |
| container | [container/](file:///workspace/codeexecutor/container/) | 容器化执行（独立 go.mod） |
| codeact | [codeact/gateway.go](file:///workspace/codeexecutor/codeact/gateway.go) | `NewGateway(tools...)`（[L40](file:///workspace/codeexecutor/codeact/gateway.go)），宿主侧工具白名单网关 |

> 根 `codeexecutor` 包**没有 `New` 函数**，构造函数在子包中。"localexec" 是 `codeexecutor/local` 的别名。

---

### 4.16 server —— 多协议 HTTP 服务

**位置**：`/workspace/server/`

| 子包 | 文件 | 构造函数 | 说明 |
| --- | --- | --- | --- |
| `server/trpcagent` | [trpcagent/server.go](file:///workspace/server/trpcagent/server.go) | `New(opts...) (*Server, error)`（[L55](file:///workspace/server/trpcagent/server.go)） | 网关服务器，base path `/trpc-agent/v1/apps` |
| `server/agui` | [agui/agui.go](file:///workspace/server/agui/agui.go) | `New(runner, opt...) (*Server, error)`（[L35](file:///workspace/server/agui/agui.go)） | AG-UI 协议服务器 |
| `server/a2a` | [a2a/server.go](file:////workspace/server/a2a/server.go) | `New(opts...) (*a2a.A2AServer, error)`（[L40](file:///workspace/server/a2a/server.go)） | A2A 智能体互通，封装 `trpc-a2a-go` |
| `server/openai` | [openai/server.go](file:///workspace/server/openai/server.go) | `New(opts...) (*Server, error)`（[L74](file:///workspace/server/openai/server.go)） | OpenAI 兼容 `/v1/chat/completions`（流式+非流式） |
| `server/promptiter` | [promptiter/server.go](file:///workspace/server/promptiter/server.go) | `New(opts...)`（[L47](file:///workspace/server/promptiter/server.go)） | PromptIter 控制面（独立 go.mod） |
| `server/evaluation` | [evaluation/server.go](file:///workspace/server/evaluation/server.go) | — | 评测服务（独立 go.mod），含 Langfuse 子包 |

所有服务器统一暴露 `runner.Runner`，调用 `runner.Run(...)` 返回 `<-chan *event.Event`。

**OpenClaw 网关分发**（[openclaw/](file:///workspace/openclaw/)）：

- 入口 `openclaw/cmd/openclaw/main.go` → `app.Main` → `gateway.New(runner, opts...)`（[app/app.go:1417](file:///workspace/openclaw/app/app.go)）
- `gateway.Server`（[openclaw/internal/gateway/server.go:128](file:///workspace/openclaw/internal/gateway/server.go)）、`New(r, opts...)`（[L166](file:///workspace/openclaw/internal/gateway/server.go)）
- 路由：`POST /gateway/messages`、`POST /gateway/messages/stream`、`GET /gateway/status`、`POST /gateway/cancel`、`GET /healthz`（base path 默认 `/v1`）
- 安全控制：用户白名单（`WithAllowUsers`）、mention 门控（`WithRequireMentionInThreads`/`WithMentionPatterns`）、内容 URL 域名白名单、body 大小限制（默认 12 MiB）
- 会话串行化：`laneLocker`（[server.go:903](file:///workspace/openclaw/internal/gateway/server.go)）按 session key 加锁

---

### 4.17 telemetry —— 可观测性

**位置**：`/workspace/telemetry/`

#### Tracing（[trace/trace.go](file:///workspace/telemetry/trace/trace.go)）

- `TracerProvider`（[L36](file:///workspace/telemetry/trace/trace.go)）、`Tracer`（[L39](file:///workspace/telemetry/trace/trace.go)）
- `Start(ctx, opts...) (clean func() error, err error)`（[L46](file:///workspace/telemetry/trace/trace.go)）：主入口，默认 gRPC、endpoint `localhost:4317`，支持 `OTEL_EXPORTER_OTLP_ENDPOINT` 环境变量
- 选项：`WithEndpoint`/`WithProtocol`/`WithServiceName`/`WithResourceAttributes` 等
- 采样：`AlwaysSample`，传播器 `propagation.TraceContext{}`

#### Metrics（[metric/metric.go](file:///workspace/telemetry/metric/metric.go)）

- `InitMeterProvider(mp)`（[L33](file:///workspace/telemetry/metric/metric.go)）：注册 chat/execute_tool/invoke_agent/workflow scopes 的计数器与直方图（token 用量、操作耗时、TTFT）
- `NewMeterProvider(ctx, opts...)`（[L319](file:///workspace/telemetry/metric/metric.go)）

#### Langfuse（[langfuse/](file:///workspace/telemetry/langfuse/)）

- `Start(ctx, opts...) (clean func(context.Context) error, err error)`（[tracer.go:31](file:///workspace/telemetry/langfuse/tracer.go)），使用 OTLP HTTP 导出至 `/api/public/otel/v1/traces`，HTTP Basic 认证
- 配置（[config.go](file:///workspace/telemetry/langfuse/config.go)）：`LANGFUSE_SECRET_KEY`/`LANGFUSE_PUBLIC_KEY`/`LANGFUSE_HOST`
- 自定义 `baggageBatchSpanProcessor`（[spanprocessor.go:31](file:///workspace/telemetry/langfuse/spanprocessor.go)）：把 baggage 成员复制到 span 属性

#### 跨层 span 创建（[internal/telemetry/trace.go](file:///workspace/internal/telemetry/trace.go)）

- 操作名常量：`OperationChat`/`OperationGenerateContent`/`OperationExecuteTool`/`OperationInvokeAgent`/`OperationWorkflow` 等（[L47-53](file:///workspace/internal/telemetry/trace.go)）
- Model 层：`TraceChat(span, *TraceChatAttributes)`（[L459](file:///workspace/internal/telemetry/trace.go)）
- Tool 层：`TraceToolCall`（[L198](file:///workspace/internal/telemetry/trace.go)）/ `TraceMergedToolCalls`（[L248](file:///workspace/internal/telemetry/trace.go)）
- Runner/Agent 层：`TraceBeforeInvokeAgent`（[L289](file:///workspace/internal/telemetry/trace.go)）/ `TraceAfterInvokeAgent`（[L387](file:///workspace/internal/telemetry/trace.go)）
- Workflow 层：`TraceWorkflow(span, *Workflow)`（[L151](file:///workspace/internal/telemetry/trace.go)）

---

### 4.18 plugin —— 插件机制

**位置**：`/workspace/plugin/`

#### Plugin 接口与注册（[manager.go](file:///workspace/plugin/manager.go)）

```go
// L33-39
type Plugin interface {
    Name() string
    Register(r *Registry)
}
```

插件在 Runner 上注册一次，自动应用于该 Runner 创建的所有 Invocation。

#### Registry 钩子（[manager.go:54-57](file:///workspace/plugin/manager.go)）

- `BeforeAgent`/`AfterAgent`（[L60-93](file:///workspace/plugin/manager.go)）
- `BeforeModel`/`AfterModel`（[L96-129](file:///workspace/plugin/manager.go)）
- `BeforeTool`/`AfterTool`（[L132-165](file:///workspace/plugin/manager.go)）
- `AfterToolMessages`（[L169-177](file:///workspace/plugin/manager.go)）
- `OnEvent`（[L180-188](file:///workspace/plugin/manager.go)）

#### Manager（[manager.go:193-200](file:///workspace/plugin/manager.go)）

- `NewManager(plugins ...Plugin)`（[L213-236](file:///workspace/plugin/manager.go)）/ `MustNewManager`
- 实现 `agent.PluginManager`

#### 注册入口

`WithPlugins(...extension.Plugin)`（[plugin/options.go:18-28](file:///workspace/plugin/options.go)）作为 `agent.RunOption`，将 `Manager` 附加到每个 Invocation。

#### 内置插件

`debuglog`、`errormessage`、`guardrail`（含 `approval`/`promptinjection`/`unsafeintent` 及各自 `review`）、`identity`、`messagemerger`、`toolcallid`、`toolsearch`；顶层 `GlobalInstruction`（[global_instruction.go:28-31](file:///workspace/plugin/global_instruction.go)）通过 `BeforeModel` 钩子前置系统消息。

---

### 4.19 prompt —— 提示词模板

**位置**：`/workspace/prompt/`

#### 核心类型（[text.go](file:///workspace/prompt/text.go)）

- `Text`（[L65-69](file:///workspace/prompt/text.go)）：`Template`、`Meta`、`Syntax`
- `Syntax`（[L37-62](file:///workspace/prompt/text.go)）：`SyntaxMixedBrace`（默认，识别 `{name}` 与 `{{name}}`）、`SyntaxSingleBrace`、`SyntaxDoubleBrace`；尾随 `?` 标记可选占位符
- `Meta`（[L28-31](file:///workspace/prompt/text.go)）：`Name`、`Version`
- `Resolver` 接口（[L83-96](file:///workspace/prompt/text.go)）：`Resolve(ref Ref) (string, bool, error)`，解析 `{user:name}` 类占位符
- `Source` 接口（[L23-25](file:///workspace/prompt/text.go)）：`FetchPrompt(ctx) (Text, error)`
- `Render`（[L124-145](file:///workspace/prompt/text.go)）/ `ValidateRequired`（[L148-174](file:///workspace/prompt/text.go)）

#### Langfuse provider（[provider/langfuse/langfuse.go](file:///workspace/prompt/provider/langfuse/langfuse.go)）

- `Client`（[L39-44](file:///workspace/prompt/provider/langfuse/langfuse.go)）从 Langfuse REST API 拉取文本 prompt
- `FetchTextPrompt`（[L102-161](file:///workspace/prompt/provider/langfuse/langfuse.go)）：仅接受 `type=="text"`
- `TextPromptSource`（[L196-213](file:///workspace/prompt/provider/langfuse/langfuse.go)）：返回带缓存（默认 TTL 60s）的 `prompt.Source`

---

### 4.20 team —— 多智能体协作

**位置**：`/workspace/team/`

`Team` 本身是 `agent.Agent`，可直接传给 `runner.NewRunner`。

#### Team 结构（[team.go:32-49](file:///workspace/team/team.go)）

实现 `agent.Agent`：`Run`（[L183-195](file:///workspace/team/team.go)）、`Tools`、`Info`、`SubAgents`、`FindSubAgent`。

#### 模式（[team.go:51-62](file:///workspace/team/team.go)）

- `ModeCoordinator`（iota 0）：协调者智能体把成员当工具调用
- `ModeSwarm`（iota 1）：从入口成员开始，成员间通过 `transfer_to_agent` 交接控制

#### 构造函数

- `New(coordinator, members, opts...)`（[L88-134](file:///workspace/team/team.go)）：协调者模式
- `NewSwarm(name, entryName, members, opts...)`（[L139-180](file:///workspace/team/team.go)）：swarm 模式

#### Swarm 配置（[swarm.go](file:///workspace/team/swarm.go)）

- `SwarmConfig`（[L24-40](file:///workspace/team/swarm.go)）：`MaxHandoffs`、`NodeTimeout`、`RepetitiveHandoffWindow`、`RepetitiveHandoffMinUnique`
- `swarmRuntime`（[runtime.go:29-40](file:///workspace/team/runtime.go)）：跟踪交接次数与循环检测

#### 选项（[options.go](file:///workspace/team/options.go)）

- `HistoryScope`：`HistoryScopeDefault`/`HistoryScopeIsolated`/`HistoryScopeParentBranch`
- `InnerTextMode`、`MemberToolConfig`

---

### 4.21 log / internal / storage —— 基础设施

#### log（[log/log.go](file:///workspace/log/log.go)）

- `Logger` 接口（[L108](file:///workspace/log/log.go)）：`Debug/Info/Warn/Error/Fatal` 及 `f` 变体，与 `trpc-a2a-go/log.Logger` 一致
- `Default`（[L42](file:///workspace/log/log.go)）：zap sugared logger（console encoder、stdout、`AddCaller`）
- `SetLevel`（[L71](file:///workspace/log/log.go)）、`Tracef`/`SetTraceEnabled`
- Context 变体为可赋值 var（`DebugContext` 等）便于插桩

#### internal（[internal/](file:///workspace/internal/)）

框架内部管线，非公开 API。关键子包：

- `internal/flow/`：见 [第 5 节](#5-internalflow--llm-执行流水线)
- `internal/telemetry/`：telemetry 辅助（span 创建、metric 全局）
- `internal/session/`：会话外部化、hook、sqldb、tool/recall
- `internal/state/`：状态管理（steer/barrier/sessionroute/appender/summaryfork/livesession/eventstream/flush）
- `internal/tool/`、`internal/toolcache/`、`internal/toolsurface/`、`internal/toolorder/`、`internal/toolcall/`
- `internal/jsonschema/`、`internal/jsonutils/`、`internal/jsonrepair/`（修复畸形 LLM JSON）
- `internal/structuredoutput/`、`internal/modelcontext/`、`internal/profilecompiler/`
- `internal/workspaceprep/`、`internal/workspacesession/`、`internal/workspacefacade/`、`internal/workspaceinput/`
- `internal/transfer/`（子智能体转接）、`internal/programsession/`、`internal/envscrub/`、`internal/trace/`

#### storage（[storage/](file:///workspace/storage/)）

根目录无 `.go` 文件、无统一 `Storage` 接口；每个子模块是独立 go.mod，提供后端客户端，被 `session/`、`memory/`、`knowledge/vectorstore/` 消费。

| 子模块 | 用途 |
| --- | --- |
| `storage/redis` | Redis 客户端构建器（builder 模式） |
| `storage/s3`、`storage/qdrant` | 对象/向量存储客户端（registry 模式） |
| `storage/clickhouse`、`storage/elasticsearch`、`storage/milvus`、`storage/mongodb` | 各 DB 客户端 |
| `storage/gorm`、`storage/mysql`、`storage/postgres` | SQL 客户端 |
| `storage/tcvector` | 腾讯云向量 DB |

---

## 5. internal/flow —— LLM 执行流水线

**位置**：`/workspace/internal/flow/`

这是 `LLMAgent` 的执行引擎核心，理解它对理解整个框架至关重要。

### 核心接口（[flow.go](file:///workspace/internal/flow/flow.go)）

```go
// L22-26
type Flow interface {
    Run(ctx context.Context, invocation *agent.Invocation) (<-chan *event.Event, error)
}

// L29-37
type RequestProcessor interface {
    ProcessRequest(ctx, invocation, req *model.Request, ch chan<- *event.Event)
}

type ResponseProcessor interface {
    ProcessResponse(ctx, invocation, req *model.Request, rsp *model.Response, ch chan<- *event.Event)
}
```

处理器直接向 channel 发送事件，而非返回值。

### llmflow 实现（[llmflow/llmflow.go](file:///workspace/internal/flow/llmflow/llmflow.go)）

- `Flow` 结构（[L112-123](file:///workspace/internal/flow/llmflow/llmflow.go)）：持有 `requestProcessors`、`responseProcessors` 及配置；创建后不可变
- `New(requestProcessors, responseProcessors, opts)`（[L152-171](file:///workspace/internal/flow/llmflow/llmflow.go)）
- `Options`（[L82-91](file:///workspace/internal/flow/llmflow/llmflow.go)）：`ChannelBufferSize`、`ModelCallbacks`、`BaseModelResolver`、`ModelSelector`、`SyncSummaryIntraRun`、`EnableContextCompaction`、`ContextCompactionThresholdRatio`、`ToolActivationApplier`

### 执行流程（`Flow.Run`，[L174-290](file:///workspace/internal/flow/llmflow/llmflow.go)）

在 goroutine 中循环直到完成：

1. `maybeResumePendingToolCalls`（[L449-501](file:///workspace/internal/flow/llmflow/llmflow.go)）：可选恢复上次运行的待处理工具调用
2. 每轮迭代：
   - `emitStartEventAndWait`（[L545-574](file:///workspace/internal/flow/llmflow/llmflow.go)）
   - `maybeSyncSummaryIntraRun`（[L503-543](file:///workspace/internal/flow/llmflow/llmflow.go)）
   - `maybeConsumeQueuedUserMessages`（[L357-435](file:///workspace/internal/flow/llmflow/llmflow.go)）：drain 排队用户消息（steer）
   - `runOneStep`（[L644-733](file:///workspace/internal/flow/llmflow/llmflow.go)）：
     - **Preprocess**（`preprocess`，[L1363-1431](file:///workspace/internal/flow/llmflow/llmflow.go)）：`populateRequestTools` 解析过滤工具 → **按序运行所有 `RequestProcessor`** → 消息净化
     - `maybeCompactContextBeforeLLM`（[L1440-1497](file:///workspace/internal/flow/llmflow/llmflow.go)）：上下文压缩（默认阈值比 0.7）
     - **Call LLM**（`callLLM`，[L2178-2230](file:///workspace/internal/flow/llmflow/llmflow.go)）：before-model 回调 → `generateContentSeq`（优先 `IterModel.GenerateContentIter`）
     - **Process streaming**（`processStreamingResponses`，[L736-804](file:///workspace/internal/flow/llmflow/llmflow.go)）：after-model 回调 → 修复工具调用参数 → 发射 LLM 响应事件 → **按序运行所有 `ResponseProcessor`**
   - 退出条件：无事件产生 / `EndInvocation` / `lastEvent.IsFinalResponse()`
3. panic 恢复（`recoverFlowRunPanic`，[L292-319](file:///workspace/internal/flow/llmflow/llmflow.go)）转为错误事件
4. context 取消视为优雅终止

### 处理器清单（`internal/flow/processor/`）

**请求处理器**（在 `preprocess` 中按序运行）：

| 处理器 | 文件:行 | 职责 |
| --- | --- | --- |
| `BasicRequestProcessor` | basic.go:23 | 应用默认 `GenerationConfig`（如 `Stream: true`） |
| `IdentityRequestProcessor` | identity.go:23 | 注入智能体名称/描述作为身份到 system prompt |
| `InstructionRequestProcessor` | instruction.go:26 | 注入指令/system prompt 与 JSON 输出 schema 指令 |
| `ContentRequestProcessor` | content.go:152 | 从会话事件组装请求消息（包含/过滤/重排历史、注入摘要与预加载记忆）——核心上下文构建器 |
| `TimeRequestProcessor` | time.go:28 | 添加当前时间到 system prompt |
| `PlanningRequestProcessor` | planning.go:25 | 用 `planner.Planner` 生成并注入规划指令 |
| `PostToolRequestProcessor` | posttool.go:52 | 追加稳定工具结果指引，避免破坏 prompt 缓存 |
| `SkillsRequestProcessor` | skills.go:280 | 注入技能概览、已加载 `SKILL.md` 正文、选中文档 |
| `SkillsToolResultRequestProcessor` | skills_tool_result.go:115 | 把已加载技能内容作为工具结果消息附加 |
| `OnDemandSessionRequestProcessor` | ondemand_session.go:41 | 注入渐进式披露工具（`session_search`/`session_load`）指引 |
| `WorkspaceExecRequestProcessor` | workspaceexec.go:87 | 注入 `workspace_exec` 工具的系统指引 |
| `ConditionalRequestProcessor` | conditional.go:25 | 谓词守卫包装器，条件满足才运行委托处理器 |

**响应处理器**（在 `postprocess` 中按序运行）：

| 处理器 | 文件:行 | 职责 |
| --- | --- | --- |
| `FunctionCallResponseProcessor` | functioncall.go:124 | 检测模型响应中的工具/函数调用并执行（并行、重试、post-tool 钩子）——工具执行主驱动 |
| `CodeExecutionResponseProcessor` | codeexecution.go:42 | 处理代码执行响应（提取并运行代码块） |
| `OutputResponseProcessor` | output.go:24 | 处理 `output_key`/`output_schema`（state-delta 模式） |
| `PlanningResponseProcessor` | planning.go:128 | 用 `planner.Planner` 处理规划响应 |
| `TransferResponseProcessor` | transfer.go:33 | 处理智能体转接（swarm `transfer_to_agent`） |
| `ConditionalResponseProcessor` | conditional.go:58 | 谓词守卫包装器 |

支持文件：`context_compact.go`（`ContextCompactionConfig`，[L210-218](file:///workspace/internal/flow/processor/context_compact.go)）、`latency_diagnostics.go`、`preload_memory_prompt.go`、`skills_state_migration.go`。

---

## 6. 模块依赖关系

### 依赖层次（自顶向下）

```
server/* , openclaw
   └─> runner
        ├─> agent (+ llmagent/chainagent/parallelagent/cycleagent/graphagent)
        │     ├─> internal/flow/llmflow  (LLM 执行引擎)
        │     │     └─> internal/flow/processor/* (请求/响应处理器)
        │     ├─> model (+ openai/anthropic/gemini/ollama/hunyuan/bedrock/huggingface/hedge/failover)
        │     ├─> tool (+ function/mcp/duckduckgo/skill/agent/...)
        │     ├─> planner (+ builtin/react/a2ui)
        │     ├─> knowledge (+ embedder/retriever/vectorstore/source/reranker/chunking)
        │     ├─> codeexecutor (+ local/sandbox/e2b/jupyter/container/codeact)
        │     ├─> skill (仓库) ─> tool/skill (工具)
        │     ├─> prompt
        │     └─> plugin (Manager) ─> agent.PluginManager
        ├─> session (+ inmemory/redis/postgres/sqlite/...)
        ├─> memory (+ inmemory/redis/postgres/sqlite/...) ─> memory/tool, memory/internal/memory
        ├─> artifact (+ inmemory/s3/cos)
        ├─> evolution ─> skill, reviewer(LLM)
        ├─> evaluation ─> evalset, metric, workflow/promptiter
        └─> event

graph (图引擎) <─ graphagent, knowledge(GraphSource)
storage/* <─ session/*, memory/*, knowledge/vectorstore/*, graph/checkpoint/*
telemetry <─ 跨层 (model/tool/runner/agent)
log <─ 全局
internal/* <─ 框架内部（不对外暴露）
```

### 关键依赖说明

- **agent → model/tool**：`Agent.Tools() []tool.Tool`，`Request.Tools map[string]tool.Tool`，模型请求内嵌工具声明
- **runner → session/memory/artifact/evolution**：通过 `With*Service` 注入，Runner 在 `Run` 中持久化事件、调度记忆与自演化
- **llmagent → internal/flow/llmflow**：`LLMAgent.Run` 委托给 `Flow.Run`，处理器链完成上下文构建、工具执行、转接等
- **tool/skill → skill + codeexecutor**：技能工具从 `skill.Repository` 读取，通过 `codeexecutor` 执行
- **evolution → skill + model**：审阅器用 LLM 复盘会话，发布到 `skill` 仓库
- **server/* → runner**：所有协议服务器统一调用 `runner.Run`
- **graphagent → graph**：`GraphAgent` 包装 `graph.Graph` 与 `graph.Executor`
- **telemetry 跨层**：`internal/telemetry` 在 model（`TraceChat`）、tool（`TraceToolCall`）、runner（`TraceBeforeInvokeAgent`/`TraceAfterInvokeAgent`）、workflow（`TraceWorkflow`）层创建 span

### 外部关键依赖（根 go.mod）

- LLM SDK：`github.com/openai/openai-go`
- 并发：`github.com/panjf2000/ants/v2`、`golang.org/x/sync`
- 遥测：`go.opentelemetry.io/otel`（trace/metric/sdk/otlp exporters）
- 协议：`trpc.group/trpc-go/trpc-a2a-go`（A2A）、`trpc.group/trpc-go/trpc-mcp-go`（MCP）
- 存储：`github.com/mattn/go-sqlite3`（CGO）、`github.com/tencentyun/cos-go-sdk-v5`
- 文档处理：`github.com/gomutex/godocx`、`github.com/yuin/goldmark`、`github.com/go-ego/gse`
- 日志：`go.uber.org/zap`
- Schema：`github.com/santhosh-tekuri/jsonschema/v6`
- gRPC/Protobuf：`google.golang.org/grpc`、`google.golang.org/protobuf`、`github.com/bufbuild/protocompile`

---

## 7. 项目运行方式

### 环境前置

- Go 1.21+
- `CGO_ENABLED=1`（根模块依赖 `go-sqlite3`）与 C 编译器
- LLM provider API key（仅运行 `examples/` 需要；测试全用 mock，无需 key）
- 工具链：`golangci-lint`、`goimports`（需在 PATH，`/home/cyl/.local/bin` 与 `$(go env GOPATH)/bin`）

### 常用命令

| 任务 | 命令 | 说明 |
| --- | --- | --- |
| 构建（根模块） | `go build ./...` | 仅根模块 |
| 单元测试（根模块） | `go test ./...` | 全用 mock，无需 API key |
| E2E 测试 | `cd test && go test ./...` | `test/` 是独立模块 |
| 跨所有子模块测试（CI 风格） | `bash .github/scripts/run-go-tests.sh` | 排除 examples/docs/test，合并 coverage |
| 检查示例构建 | `bash .github/scripts/check-examples.sh` | |
| Lint | `golangci-lint run --timeout=10m` | 配置在 `.golangci.yml` |
| gofmt 检查 | `gofmt -r 'interface{} -> any' -l .` | CI 强制 `any` 而非 `interface{}` |
| goimports 检查 | `goimports -l .` | |
| Trace eval 示例（无 key） | `cd examples/evaluation/trace && go run .` | Trace 模式跳过 LLM 调用 |

### 快速开始（运行第一个智能体）

```bash
# 1. 克隆
git clone https://github.com/trpc-group/trpc-agent-go.git
cd trpc-agent-go

# 2. 配置 LLM
export OPENAI_API_KEY="your-api-key-here"
export OPENAI_BASE_URL="your-base-url-here"  # 可选

# 3. 运行
cd examples/runner
go run . -model="gpt-4o-mini" -streaming=true
```

### 最小用法代码

```go
package main

import (
    "context"
    "fmt"
    "log"

    "trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
    "trpc.group/trpc-go/trpc-agent-go/model"
    "trpc.group/trpc-go/trpc-agent-go/model/openai"
    "trpc.group/trpc-go/trpc-agent-go/runner"
    "trpc.group/trpc-go/trpc-agent-go/tool"
    "trpc.group/trpc-go/trpc-agent-go/tool/function"
)

func main() {
    modelInstance := openai.New("deepseek-chat",
        openai.WithVariant(openai.VariantDeepSeek))

    calculatorTool := function.NewFunctionTool(calculator,
        function.WithName("calculator"),
        function.WithDescription("Execute add/sub/mul/div."))

    agent := llmagent.New("assistant",
        llmagent.WithModel(modelInstance),
        llmagent.WithTools([]tool.Tool{calculatorTool}),
        llmagent.WithGenerationConfig(model.GenerationConfig{Stream: true}))

    r := runner.NewRunner("calculator-app", agent)

    ctx := context.Background()
    events, err := r.Run(ctx, "user-001", "session-001",
        model.NewUserMessage("Calculate what 2+3 equals"))
    if err != nil {
        log.Fatal(err)
    }
    for event := range events {
        if event.Object == "chat.completion.chunk" {
            fmt.Print(event.Response.Choices[0].Delta.Content)
        }
    }
    fmt.Println()
}
```

### 每请求动态 Agent

```go
r := runner.NewRunnerWithAgentFactory("my-app", "assistant",
    func(ctx context.Context, ro agent.RunOptions) (agent.Agent, error) {
        return llmagent.New("assistant",
            llmagent.WithInstruction(ro.Instruction)), nil
    })
```

### 停止 / 取消运行

- **推荐**：取消传给 `Runner.Run` 的 context，并继续 drain 事件通道直到关闭
- 终端程序：`signal.NotifyContext` 把 Ctrl+C 转为取消
- 代码取消：`context.WithCancel` + `go cancel()`
- 按 `requestID` 取消（服务端/后台运行）：

```go
requestID := "req-123"
events, _ := r.Run(ctx, userID, sessionID, message, agent.WithRequestID(requestID))
mr := r.(runner.ManagedRunner)
_ = mr.Cancel(requestID)
```

### 多智能体协作

```go
base := llmagent.New("assistant", llmagent.WithModel(openai.New("gpt-4o-mini")))
translator := llmagent.New("translator",
    llmagent.WithInstruction("Translate everything to French"),
    llmagent.WithModel(openai.New("gpt-3.5-turbo")))

pipeline := chainagent.New("pipeline",
    chainagent.WithSubAgents([]agent.Agent{base, translator}))

run := runner.NewRunner("demo-app", pipeline)
events, _ := run.Run(ctx, "user-1", "sess-1", model.NewUserMessage("Hello!"))
for ev := range events { /* ... */ }
```

### 开发约定

- 所有 `.go` 文件须含 Tencent Apache 2.0 license header（CI 检查）
- 提交 PR 前本地 `go test ./...` 与 `go vet ./...`
- Commit message 首行格式：`package: summary`，正文说明 why
- PR 须打 type 标签：`type/bug`/`type/feature`/`type/enhancement`/`type/documentation`/`type/api-change`/`type/performance`/`type/ci`
- 用户可见变更须填 RELEASE NOTES

---

## 8. 示例索引

`examples/` 下有 90+ 可运行示例，按类别：

| 类别 | 示例目录 |
| --- | --- |
| A2A / UI 协议 | `a2aadk`、`a2aagent`、`a2acodeexecution`、`a2amultipath`、`a2asubagent`、`a2ui`、`agui` |
| 多智能体 / 团队 / 转接 | `agenttool`、`dynamicagenttool`、`team`、`transfer`、`humaninloop` |
| 代码执行 | `codeact`、`codeexecution`、`sandboxcodeexecution` |
| 上下文管理 | `context_compaction`、`context_compaction_recovery`、`runwithmessages` |
| MCP | `mcpbroker`、`mcptool`、`claudecode`、`codex` |
| 技能 | `skill`、`skilldynamicschema`、`skillfind`、`skillisolation`、`skillloadmode`、`skillrun`、`skilltoolactivation`、`skilltoolprofile` |
| 工具 | `tool`、`multitools`、`openapitool`、`toolcallid`、`toolfilter`、`toolinterrupt`、`toolpipe`、`toolpolicy` |
| 记忆 / 知识 | `memory`、`knowledge` |
| 模型 / LLM | `model`、`llmagent`、`llmagent_tool_call_retry` |
| 图 / 工作流 | `graph`、`dynamicworkflow` |
| 会话 | `session` |
| 输出处理 | `outputkey`、`outputkeystate`、`outputschema`、`structuredoutput`、`structuredoutputskills` |
| 评测 / Prompt 迭代 | `evaluation`（含 `promptiter`、`promptiter_regression_loop` 等） |
| Prompt / 缓存 | `prompt`、`promptcache` |
| 遥测 / 回调 | `telemetry`、`callbacks`、`tokentracker` |
| 第三方集成 | `arxivsearch`、`dify`、`duckduckgo`、`email`、`google`、`n8n`、`provider`、`trpcagent`、`openaiserver`、`weknora` |
| 其他模式 | `artifact`、`builtinexplorer`、`cancelrun`、`customagent`、`debugagent`、`evolution`、`fileinput`、`goal`、`managedrunner`、`plugin`、`ralphloop`、`react`、`runner`、`steer`、`summary`、`tailor`、`taskrun`、`thinking`、`timeaware`、`todo`、`todoenforcer`、`usermessagerewriter`、`wiki`、`workspace_io`、`guardrail` |

---

> 本 Wiki 基于仓库当前磁盘状态生成，所有文件路径与行号均来自源码实测。如代码演进，请以源码为准。
