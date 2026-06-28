# PromptIter Regression Loop Example

This example is a deterministic Evaluation + PromptIter regression-loop demo. It reads fixed train / validation evalsets, metric definitions, a baseline prompt, PromptIter gate config, and a fake model queue from `configs/`, then writes:

```text
report/optimization_report.json
report/optimization_report.md
```

Run without API keys:

```bash
cd examples/evaluation/promptiter_regression_loop
go run .
```

Run tests:

```bash
cd examples/evaluation
go test ./promptiter_regression_loop
```

## Design Notes

本示例实现一个可复现的“评测 - 失败归因 - prompt 优化 - 回归验证 - 产物审计”闭环。输入侧包含 `train.evalset.json`、`validation.evalset.json`、`metrics.json`、`promptiter.json`、`baseline_prompt.txt` 和 `fake_model_queue.json`；样例共 6 条 case，训练集和验证集各 3 条，覆盖可优化成功、优化无效以及验证集关键样本退化。运行时使用 `FakeModel` 按调用次数顺序消费预设响应，并记录调用次数，因此不依赖任何真实 API key。

失败归因基于 PromptIter engine 的 `EvaluationResult`，遍历每条 case 的 failed metric，并按 metric name 与 reason 归入六类：最终回复不匹配、工具调用错误、工具参数错误、route 错误、格式错误、知识召回不足。候选验证使用 engine 风格的逐 case delta，区分 newly passed、newly failed、score improved、score regressed 和 unchanged。

接受策略使用多维 gate：验证集总分提升必须达到阈值；不能新增 hard fail；`critical_case_ids` 中的关键 case 不得退化；调用次数不得超过预算。样例报告故意构造了候选总分提升但 `validation_overfit_guard` 从 passed 变为 failed 的场景，因此最终拒绝候选，展示防过拟合逻辑。审计报告记录 baseline/candidate 分数、逐 case delta、gate 决策、失败归因统计、成本、耗时、随机种子、模型配置和 fake queue 配置。
