研究完成，完整报告已写入：

`/tmp/cnm-34-dataset-research.md`

核心结论：

- **推荐只使用 CommitChronicle**：commit-level `mods` 包含完整多文件修改，10.7M commits、20 种语言，逐行保留 repo/SHA/license，来源限 MIT、Apache-2.0、BSD-3-Clause。
- 最小方案：流式抽取约 **8,000 个完整 commits**，超过上下文的整行丢弃而非截断，并按 repository hash 隔离 train/valid/test。
- **排除 CommitPackFT**：其记录是单文件 slice；`/tmp/cnm-train31/build_dataset.py` 正是把每个 slice 错当独立 commit family。
- **排除 CommitBench 作为默认训练源**：dataset 为 CC BY-NC 4.0，且 standard/long 版本分别受 512/2,048-token 限制，不能保证完整 commit，也不适合潜在商业分发。
- 训练前应人工盲审 200 行，并通过既定 26-case puncture test。

运行环境没有 shell/`gh`，且 `fetch_content`、`get_search_content` 未配置，因此无法直接读取 GitHub #34 或下载候选数据做现场抽样；这些限制和后续验证步骤已列入报告。