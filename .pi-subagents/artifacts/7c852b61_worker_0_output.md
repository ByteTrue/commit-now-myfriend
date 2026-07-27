# 翻译数据报告

已将 `/tmp/cnm-train31/data/translation-input.json` 中的 112 条英文 Git commit subject/body 翻译为简体中文，并写入 `/tmp/cnm-train31/data/translations.json`。未修改仓库文件。

## 校验

运行命令：

```bash
python3 - <<'PY'
import json
from pathlib import Path
src = json.loads(Path('/tmp/cnm-train31/data/translation-input.json').read_text())
out = json.loads(Path('/tmp/cnm-train31/data/translations.json').read_text())
assert isinstance(out, list)
assert len(src) == len(out) == 112
src_ids = [x['id'] for x in src]
out_ids = [x['id'] for x in out]
assert len(set(out_ids)) == 112
assert set(src_ids) == set(out_ids)
assert all(set(x) == {'id', 'subject', 'body'} for x in out)
by_id = {x['id']: x for x in out}
assert all(by_id[x['id']]['body'] == '' for x in src if x['body'] == '')
assert all(by_id[x['id']]['body'] != '' for x in src if x['body'] != '')
assert all(not x['subject'].endswith(('.', '。')) for x in out)
print('PASS: valid JSON array')
print(f'PASS: entries={len(out)}, unique_ids={len(set(out_ids))}, id_sets_equal={set(src_ids) == set(out_ids)}')
print('PASS: every entry has exactly id/subject/body')
print(f"PASS: {sum(x['body'] == '' for x in src)} empty input bodies remain empty")
print('PASS: non-empty input bodies remain non-empty')
print('PASS: no translated subject ends with a period')
PY
git diff --cached --name-only
git status --short
```

结果：

- PASS：JSON 数组可解析
- PASS：112 条，112 个唯一 id，输入输出 id 集合完全一致
- PASS：每条仅含 `id`、`subject`、`body`
- PASS：101 个空 body 仍为空；所有非空 body 仍非空
- PASS：所有 subject 均不以句号结尾
- PASS：仓库无 staged 文件
- 仓库已有与本任务无关的未暂存/未跟踪内容：`.github/workflows/release.yml`、`.cs/`、`.pi-subagents/`；本任务未触碰这些内容