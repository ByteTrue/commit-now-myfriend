#!/usr/bin/env python3
import json
import unittest

from augment_guidance import make_variant, validate_variant
from pipeline import build_messages


class Tokenizer:
    def apply_chat_template(self, messages, tokenize, add_generation_prompt):
        return list(range(100))


class GuidanceAugmentationTest(unittest.TestCase):
    def base(self) -> dict:
        diff = "diff --git a/app.py b/app.py\n@@ -1 +1 @@\n-old parser\n+new secure parser\n"
        record = {"type": "fix", "scope": "parser", "subject": "replace the parser", "body": "Avoid stale parsing behavior. Preserve secure defaults."}
        messages = build_messages("conventional", diff)
        messages.append({"role": "assistant", "content": json.dumps(record, separators=(",", ":"))})
        return {"messages": messages, "meta": {"family": "fixture", "style": "conventional"}}

    def test_every_deterministic_variant_validates(self):
        for kind in ("issue", "no_body", "bullets", "body", "prefix"):
            with self.subTest(kind=kind):
                row = make_variant(self.base(), "train", kind, 0, Tokenizer())
                self.assertIsNotNone(row)
                validate_variant(row)


if __name__ == "__main__":
    unittest.main()
