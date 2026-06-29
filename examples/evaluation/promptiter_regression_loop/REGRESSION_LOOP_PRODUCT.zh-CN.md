# PromptIter 回归优化闭环产品文档

## 1. 文档目的

本文档说明 tRPC-Agent-Go 原有评测与 PromptIter 优化能力，以及本次为完成赛题新增的“评测 - 失败归因 - PromptIter 优化 - 验证集回归 - 接受门禁 - 审计报告”闭环能力。文档面向产品、算法、工程和评测平台使用者，重点回答：

- 项目原本已经具备哪些 Agent、评测和优化基础能力。
- 本次新增了哪些产品能力、开发接口和示例资产。
- 如何在无真实 API Key 的环境中复现完整流程。
- 系统如何判断候选 prompt 是否值得接受，如何防止训练集提升但验证集退化。
- 审计报告包含哪些字段，如何支撑上线前评审。

## 2. 项目原有内容概览

tRPC-Agent-Go 是一个 Go 原生 Agent 框架，用于构建生产级 AI Agent 系统。它不是单一应用，而是多模块 monorepo，根模块路径为 `trpc.group/trpc-go/trpc-agent-go`。项目原有核心能力包括：

| 能力域 | 原有能力 | 典型位置 |
| --- | --- | --- |
| Agent Runtime | LLM Agent、Chain Agent、Graph Agent、Parallel Agent、Cycle Agent、多 Agent 编排 | `agent/`、`graph/`、`runner/` |
| 工具系统 | function tool、MCP、web fetch、code execution、file tool、workspace tool | `tool/`、`codeexecutor/` |
| 状态与记忆 | session、memory、artifact、knowledge retrieval、多种存储后端 | `session/`、`memory/`、`knowledge/`、`storage/` |
| 可观测性 | OpenTelemetry trace/metric、Langfuse 示例、trace capture | `telemetry/`、`agent/trace/` |
| Evaluation | evalset、metric、evaluation service、本地和 MySQL 存储、trace mode、tool trajectory、LLM rubric、ROUGE | `evaluation/` |
| PromptIter | 基于评测结果抽取 loss、反向传播 failure signal、聚合 gradient、生成 prompt patch、多轮优化 | `evaluation/workflow/promptiter/` |
| 管理与持久化 | PromptIter manager、observer、store，可保存运行事件和 round 状态 | `evaluation/workflow/promptiter/manager/`、`store/` |

在本次改进前，项目已经能运行评测、生成 PromptIter patch，并基于验证集总分增量做基础接受判断。但对于真实业务所需的回归闭环仍存在几个缺口：

- 接受策略主要依赖总分提升，缺少“不能新增 hard fail”“关键 case 不退化”“预算不超限”等多维门禁。
- 验证集结果缺少逐 case delta，无法明确候选 prompt 是修复了哪些 case，又破坏了哪些 case。
- 失败归因没有面向赛题要求输出六类可解释分类。
- PromptIter 运行结果虽可被 manager 持久化，但没有面向评审和归档的 `optimization_report.json` / `optimization_report.md`。
- 缺少一个无 API Key 可复现的端到端示例，来证明闭环在 fake model / trace mode / deterministic runner 下可运行。

## 3. 本次新增内容概览

本次新增内容由两部分组成：PromptIter engine 的通用能力扩展，以及一个可运行的 deterministic 示例。

### 3.1 Engine 通用能力扩展

| 新增能力 | 文件 | 产品意义 |
| --- | --- | --- |
| 多维接受策略 | `evaluation/workflow/promptiter/engine/accept.go` | 候选 prompt 不再只看总分，而是同时检查分数提升、新增失败、关键 case 退化和预算 |
| 逐 case delta | `evaluation/workflow/promptiter/engine/case_delta.go` | 输出每条验证 case 的变化类型、分数变化和状态变化 |
| 预算使用快照 | `evaluation/workflow/promptiter/engine/option.go` | 调用方可向 engine 注入 cost、API calls、latency，用于接受门禁 |
| Run 链路接入 gate | `evaluation/workflow/promptiter/engine/engine.go` | 每轮验证完成后立即计算 case delta 并执行多维 gate |
| 单元测试 | `accept_gate_test.go`、`case_delta_test.go` | 覆盖 gate 决策、缺失候选 case、跨 eval set 重名 case 等边界 |

### 3.2 可复现示例

新增目录：

```text
examples/evaluation/promptiter_regression_loop/
```

该示例提供完整输入、运行入口、报告生成和测试：

| 文件或目录 | 作用 |
| --- | --- |
| `main.go` | CLI 入口，默认读取 `configs/` 并写入 `report/` |
| `pipeline.go` | 串联配置加载、PromptIter 运行、delta、失败归因、报告落盘 |
| `promptiter_runtime.go` | deterministic PromptIter runtime，包含 scripted evaluator、backwarder、aggregator、optimizer |
| `fake_model.go` | 按调用次数消费队列的 fake model |
| `failure_attribution.go` | 失败归因分类器 |
| `audit_report.go` | JSON / Markdown 审计报告 builder |
| `configs/` | 样例 train / validation evalset、metrics、PromptIter 配置、baseline prompt、fake queue |
| `report/` | 示例输出 `optimization_report.json` 和 `optimization_report.md` |
| `*_test.go` | 覆盖配置、fake model、失败归因、报告生成、pipeline 行为 |

## 4. 产品场景

该闭环适用于需要把 prompt 优化纳入工程评审和上线门禁的场景：

1. 团队维护一个线上 Agent 的 system prompt、agent instruction、skill 描述或 router prompt。
2. 团队已经积累 train / validation evalset，用于衡量 Agent 行为质量。
3. PromptIter 根据 train failures 生成候选 prompt。
4. 系统在 validation 上重新评估候选 prompt。
5. 系统比较 baseline 与 candidate 的逐 case 变化。
6. 系统根据多维 gate 自动给出接受或拒绝建议。
7. 系统输出审计报告，供研发、评测、产品和业务 owner 共同确认。

本次样例特意构造了“候选总分提升，但验证集关键 case 退化”的过拟合场景：`validation_prompt_fixable` 被修复，`validation_overfit_guard` 从通过变为失败。最终 gate 拒绝候选 prompt，展示防过拟合逻辑。

## 5. 闭环流程

端到端流程如下：

```text
输入配置
  ├─ baseline_prompt.txt
  ├─ train.evalset.json
  ├─ validation.evalset.json
  ├─ metrics.json
  ├─ promptiter.json
  └─ fake_model_queue.json
        ↓
Baseline 验证集评测
        ↓
训练集评测与 loss 提取
        ↓
Backward / Aggregation / Optimizer
        ↓
生成候选 prompt
        ↓
候选 prompt 验证集回归
        ↓
逐 case delta
        ↓
多维接受门禁
        ↓
JSON / Markdown 审计报告
```

关键设计点：

- PromptIter engine 是主编排器，示例不是绕过 engine 直接拼报告。
- baseline validation 由 engine 在 run 开始时执行并记录。
- 每轮 candidate validation 完成后，engine 计算相对当前 accepted validation 的 case delta。
- gate 决策在 engine 内执行，报告只消费 engine 输出的 `AcceptanceDecision`。
- 示例 evaluator 为 deterministic scripted evaluator，用于无 API Key 环境复现；这保证了赛题核心流程可稳定运行。

## 6. 输入资产

### 6.1 `train.evalset.json`

训练集包含 3 条 case：

| Case | 设计目的 | Baseline 行为 |
| --- | --- | --- |
| `train_prompt_fixable` | 可通过 prompt 格式约束修复 | 工具调用正确，但最终回复不是 JSON |
| `train_tool_argument_error` | 优化无效类问题 | 工具参数错误，prompt 格式优化无法修复 |
| `train_already_passed` | 已通过样本 | baseline 已满足要求 |

### 6.2 `validation.evalset.json`

验证集包含 3 条 case：

| Case | 设计目的 | Candidate 行为 |
| --- | --- | --- |
| `validation_prompt_fixable` | 验证可优化成功 | 从 failed 变为 passed |
| `validation_no_effect` | 验证优化无效 | 保持 failed |
| `validation_overfit_guard` | 验证退化 / 过拟合保护 | 从 passed 变为 failed |

### 6.3 `metrics.json`

样例配置三类评测信号：

- `final_response_exact_json`：检查最终回复是否符合精确 JSON。
- `tool_trajectory_avg_score`：检查工具名称、参数和结果轨迹。
- `llm_rubric_json_grounding`：表达格式合规和知识召回 rubric。

示例 runtime 会读取 metric 文件中的 `finalResponse` 和 `toolTrajectory` metric 名称，并用它们生成 engine 风格评测结果。

### 6.4 `promptiter.json`

该文件配置优化和门禁策略：

```json
{
  "max_rounds": 2,
  "min_score_gain": 0.01,
  "target_score": 0.95,
  "no_new_hard_fail": true,
  "critical_case_ids": ["validation_overfit_guard"],
  "max_api_calls": 200
}
```

产品含义：

- `min_score_gain`：候选验证集总分提升必须达到阈值。
- `no_new_hard_fail`：候选不能引入新增失败。
- `critical_case_ids`：关键验证样本不能退化。
- `max_api_calls`：候选运行不能超过调用次数预算。

### 6.5 `fake_model_queue.json`

该文件提供 fake model 响应队列。当前示例的 optimizer 使用 `FakeModel` 按调用次数消费 patch 响应；candidate、judge、backwarder、aggregator 字段保留为 fake engine 配置的一部分，便于扩展为更完整的多角色 fake runner。

## 7. 输出资产

### 7.1 `optimization_report.json`

结构化报告包含：

| 字段 | 内容 |
| --- | --- |
| `metadata` | 生成时间、随机种子、模型配置、fake queue 配置、pipeline 版本 |
| `baseline` | baseline validation 总分、通过数、失败数、总 case 数 |
| `candidate` | candidate validation 总分、通过数、失败数、总 case 数 |
| `delta.case_deltas` | 每条验证 case 的 `eval_set_id`、`case_id`、变化类型、分数和状态 |
| `gate_decision` | 是否接受、score delta、拒绝/接受理由、每个 gate 的明细 |
| `failure_attribution_stats` | 失败归因统计和逐 case 归因 |
| `cost_latency` | cost、API calls、latency |
| `rounds` | 每轮候选 prompt、训练评测、验证评测、case delta、acceptance |

### 7.2 `optimization_report.md`

人读版报告包含：

- 接受 / 拒绝结论。
- baseline 与 candidate 分数对比。
- 逐 case delta 表格。
- 失败归因表格。
- cost / latency 摘要。

当前样例输出结论为拒绝：

```text
Accepted: false
Reason: newly failed cases: validation_overfit_guard; regressed critical cases: validation_overfit_guard
```

## 8. 多维接受策略

Engine 新增的 `AcceptancePolicy` 支持以下维度：

| Gate | 输入 | 通过条件 |
| --- | --- | --- |
| `ValidationScoreGain` | baseline score、candidate score、`MinScoreGain` | `candidate - baseline >= MinScoreGain` |
| `NoNewHardFail` | case delta | 不存在 `newly_failed` |
| `CriticalCasePreserve` | case delta、`CriticalCaseIDs` | 关键 case 不出现 `newly_failed` 或 `score_regressed` |
| `BudgetConstraint` | `BudgetUsage`、`BudgetLimit` | cost、API calls、latency 不超过配置上限 |

该策略解决了“平均分提升掩盖关键样本退化”的问题。即使 candidate 总分提升，只要新增 hard fail 或关键 case 退化，系统仍会拒绝候选。

## 9. 逐 case delta 语义

`CompareCaseDeltas` 输出五类变化：

| 类型 | 含义 |
| --- | --- |
| `newly_passed` | baseline 非 passed，candidate passed |
| `newly_failed` | baseline passed，candidate 非 passed |
| `score_improved` | pass/fail 状态未改变，但分数提升 |
| `score_regressed` | pass/fail 状态未改变，但分数下降 |
| `unchanged` | 状态和分数均未变化 |

边界处理：

- baseline 中存在、candidate 缺失的 passed case 会被视为 `newly_failed`。
- candidate 中新增的 case 会进入 delta 输出。
- 跨 eval set 重名 case 使用 `(eval_set_id, eval_case_id)` 复合键区分，避免覆盖。
- 空 metric 或未评估 metric 会按 `not_evaluated` 处理。

## 10. 失败归因

示例的失败归因器基于 failed metric 的 `MetricName` 和 `Reason` 进行语义分类，覆盖赛题要求的六类：

| 分类 | 触发信号示例 |
| --- | --- |
| `final_response_mismatch` | final response / ROUGE / exact JSON 失败 |
| `tool_call_error` | tool trajectory 中工具名称或调用错误 |
| `tool_arg_error` | tool trajectory 中参数不匹配 |
| `route_error` | route / transfer 相关失败 |
| `format_error` | format / schema / JSON 相关失败 |
| `knowledge_recall` | knowledge / recall / fact 相关失败 |

每个失败 attribution 包含：

- `case_id`
- `category`
- `reason`
- `evidence`

## 11. 运行方式

在无需真实 API Key 的环境中运行：

```bash
cd examples/evaluation/promptiter_regression_loop
go run .
```

指定配置和输出目录：

```bash
go run . \
  -config-dir configs \
  -output-dir report
```

运行后生成：

```text
report/optimization_report.json
report/optimization_report.md
```

## 12. 测试与验证

本次新增测试覆盖：

| 测试面 | 文件 |
| --- | --- |
| Gate 决策 | `evaluation/workflow/promptiter/engine/accept_gate_test.go` |
| 逐 case delta | `evaluation/workflow/promptiter/engine/case_delta_test.go` |
| Fake model 队列消费 | `examples/evaluation/promptiter_regression_loop/fake_model_test.go` |
| 失败归因六类分类 | `examples/evaluation/promptiter_regression_loop/failure_attribution_test.go` |
| 报告字段 | `examples/evaluation/promptiter_regression_loop/audit_report_test.go` |
| 配置文件完整性 | `examples/evaluation/promptiter_regression_loop/config_files_test.go` |
| 端到端 pipeline | `examples/evaluation/promptiter_regression_loop/pipeline_test.go` |

已执行的验证命令：

```bash
go test ./...
cd evaluation && go test -count=1 ./workflow/promptiter/...
cd examples/evaluation && go test -count=1 ./promptiter_regression_loop
cd examples/evaluation/promptiter_regression_loop && go run .
goimports -l evaluation/workflow/promptiter/engine examples/evaluation/promptiter_regression_loop
gofmt -r 'interface{} -> any' -l evaluation/workflow/promptiter/engine examples/evaluation/promptiter_regression_loop
git diff --check
bash .github/scripts/check-examples.sh
bash .github/scripts/run-go-tests.sh
```

环境说明：

- 当前环境缺少系统级 `sqlite3.h`，直接运行 `run-go-tests.sh` 会在 `knowledge/vectorstore/sqlitevec` 的 CGO 编译处失败。
- 已确认该失败不是本次改动导致，根因是系统未安装 `libsqlite3-dev` 且当前用户无免密 sudo。
- 为完成全量测试，本次使用 `github.com/mattn/go-sqlite3` 模块自带的 `sqlite3-binding.h` 作为临时 include，设置 `CGO_CFLAGS=-I<tmp_include>` 后完整 `run-go-tests.sh` 通过。

## 13. 审查结论

从赛题要求看，本次实现已经覆盖核心交付：

- 有 Go pipeline 入口：`main.go`。
- 有 train / validation evalset、metrics、baseline prompt、PromptIter 配置和 README。
- 有 6 条样例 case，覆盖可优化成功、优化无效、优化后验证集退化。
- 有 JSON 和 Markdown 示例报告。
- 有 300-500 字级别方案说明：`README.md` 的 Design Notes。
- 单元测试覆盖 gate、delta、失败归因、报告生成。
- fake model / trace mode / deterministic runner 下可无 API Key 运行。
- 过拟合场景被拒绝，拒绝理由可审计。

剩余边界：

- 示例 evaluator 是 deterministic scripted evaluator，适合赛题复现和 CI，不等同于对真实线上 Agent 运行全量外部模型评测。
- `fake_model_queue.json` 当前主要由 optimizer 消费，candidate / judge / backwarder / aggregator 队列作为配置资产保留，后续可扩展为所有角色都由 fake model 驱动。
- hidden sample 的“接受/拒绝准确率 >= 80%”需要隐藏集执行结果证明，当前仓库只能通过公开样例、gate 单测和端到端测试证明策略行为。

## 14. 后续产品化建议

后续如果要从赛题示例推进到生产能力，建议按以下方向演进：

1. 将示例中的 `FailureAttributor` 沉淀为 evaluation 或 promptiter 的可复用组件，支持可配置规则和业务自定义分类。
2. 将 `optimization_report` builder 抽象为通用报告接口，支持 manager store 直接导出。
3. 将 budget usage 与真实模型 telemetry / token usage / latency 采集打通，减少示例中的固定 usage。
4. 支持多目标 surface 同时优化，并在报告中按 surface 展示 patch、delta 和风险。
5. 支持将 gate 决策接入 CI 或发布系统，只有接受的候选 prompt 才允许回写源 prompt。
6. 在真实 evaluation service 模式下提供同样的 deterministic trace replay，用于线上 regression suite。
