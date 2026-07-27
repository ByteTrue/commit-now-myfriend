#!/usr/bin/env python3
import json
import unittest

from pipeline import SemanticRecord, build_messages, check_requirements, parse_record, render, resolve_auto


class PipelineTest(unittest.TestCase):
    def test_parse_and_render_conventional(self):
        raw = json.dumps({"type": "fix", "scope": "git", "subject": "preserve staged changes", "body": "Avoid staging unrelated files."})
        record = parse_record(raw)
        self.assertEqual(render("conventional", record), "fix(git): preserve staged changes\n\nAvoid staging unrelated files.")

    def test_renderer_does_not_rewrite_content(self):
        record = SemanticRecord(None, None, "Preserve GitHub casing", "Why: Keep API names exact.")
        self.assertEqual(render("google", record), "Preserve GitHub casing\n\nWhy: Keep API names exact.")

    def test_custom_content_passes_unchanged(self):
        record = SemanticRecord(None, None, "SECURITY: reject exposed keys", "- Reject private keys\n- Explain the affected path")
        self.assertEqual(render("custom", record), record.subject + "\n\n" + record.body)

    def test_style_requiring_type_rejects_null(self):
        with self.assertRaisesRegex(ValueError, "type required"):
            render("angular", SemanticRecord(None, None, "handle empty input", ""))

    def test_renderer_rejects_prefixed_semantic_subject(self):
        with self.assertRaisesRegex(ValueError, "already contains"):
            render("conventional", SemanticRecord("fix", "git", "fix(git): preserve staged changes", ""))

    def test_parse_rejects_surrounding_or_extra_output(self):
        good = '{"type":null,"scope":null,"subject":"Update docs","body":""}'
        for raw in ("Result: " + good, good + "\nextra", good[:-1], good[:-1] + ',"extra":1}'):
            with self.subTest(raw=raw):
                with self.assertRaises(ValueError):
                    parse_record(raw)

    def test_parse_rejects_markdown_and_explanation(self):
        for subject in ("```json", "Here is: update docs", "Commit message: update docs"):
            raw = json.dumps({"type": None, "scope": None, "subject": subject, "body": ""})
            with self.subTest(subject=subject):
                with self.assertRaises(ValueError):
                    parse_record(raw)

    def test_parse_rejects_invalid_type_scope_and_multiline_subject(self):
        values = [
            {"type": "feature", "scope": None, "subject": "add cache", "body": ""},
            {"type": "feat", "scope": "Bad Scope", "subject": "add cache", "body": ""},
            {"type": "feat", "scope": None, "subject": "add\ncache", "body": ""},
        ]
        for value in values:
            with self.subTest(value=value):
                with self.assertRaises(ValueError):
                    parse_record(json.dumps(value))

    def test_auto_uses_history_only(self):
        self.assertEqual(resolve_auto(["fix(cli): handle empty input", "feat: add cache", "Update docs"]), "conventional")
        self.assertEqual(resolve_auto(["Handle empty input", "Update docs"]), "plain")

    def test_deleted_body_remains_a_failure(self):
        record = SemanticRecord("fix", "git", "preserve staged changes", "")
        message = render("conventional", record)
        self.assertIn("body_required", check_requirements(record, message, {"body_required": True}))
        self.assertNotIn("Why:", message)

    def test_wrong_type_and_swapped_subject_are_not_repaired(self):
        record = SemanticRecord("docs", None, "speed up parser", "")
        message = render("conventional", record)
        self.assertEqual(message, "docs: speed up parser")
        self.assertNotIn("fix", message)

    def test_guidance_checks_use_raw_record_content(self):
        record = SemanticRecord(None, None, "安全：拒绝泄露密钥", "- 拒绝私钥\n- 记录影响路径")
        message = render("custom", record)
        self.assertEqual(check_requirements(record, message, {
            "subject_prefix": "安全：",
            "body_required": True,
            "exact_body_bullets": 2,
            "simplified_chinese": True,
        }), [])
        self.assertIn("reference", check_requirements(record, message, {"reference": "#123"}))

    def test_build_messages_keeps_diff_out_of_renderer_contract(self):
        messages = build_messages("plain", "diff --git a/a b/a\n+x", "Use Chinese", ["Update docs"])
        self.assertEqual([item["role"] for item in messages], ["system", "user"])
        self.assertIn("Use Chinese", messages[0]["content"])
        self.assertIn("diff --git", messages[1]["content"])


if __name__ == "__main__":
    unittest.main()
