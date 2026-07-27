#!/usr/bin/env python3
import importlib.util
import json
import sys
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name("validate_labels.py")
spec = importlib.util.spec_from_file_location("validate_labels", SCRIPT)
module = importlib.util.module_from_spec(spec)
sys.modules[spec.name] = module
spec.loader.exec_module(module)

ROW = {
    "family": "f",
    "diff_sha256": "d",
    "body_policy": "required",
    "diff": "diff --git a/a b/a\n--- a/a\n+++ b/a\n@@ -1 +1 @@\n-old value\n+new value\n",
}


def saved(record):
    return {
        "family": "f",
        "diff_sha256": "d",
        "body_policy": "required",
        "server_pid": 1,
        "content": json.dumps(record),
        "finish_reason": "stop",
    }


class ValidationTests(unittest.TestCase):
    def test_valid_evidence_backed_record(self):
        record = {
            "type": "fix",
            "scope": None,
            "subject": "Use new value",
            "body": "Replace the old value.",
            "subject_evidence": ["new value"],
            "body_evidence": ["old value"],
        }
        parsed, errors = module.validate_record(ROW, saved(record))
        self.assertEqual(parsed, record)
        self.assertEqual(errors, [])

    def test_rejects_non_changed_evidence(self):
        record = {
            "type": None,
            "scope": None,
            "subject": "Use new value",
            "body": "Required body",
            "subject_evidence": ["not in diff"],
            "body_evidence": ["new value"],
        }
        _, errors = module.validate_record(ROW, saved(record))
        self.assertIn("evidence_not_exact_changed_line", errors)

    def test_accepts_metadata_only_rename_evidence(self):
        row = {
            **ROW,
            "body_policy": "optional",
            "diff": "diff --git a/old.txt b/new.txt\nsimilarity index 100%\nrename from old.txt\nrename to new.txt\n",
        }
        record = {
            "type": "chore",
            "scope": None,
            "subject": "Rename old.txt to new.txt",
            "body": "",
            "subject_evidence": ["rename from old.txt", "rename to new.txt"],
            "body_evidence": [],
        }
        envelope = saved(record)
        envelope["body_policy"] = "optional"
        self.assertEqual(module.validate_record(row, envelope)[1], [])

    def test_rejects_missing_required_body(self):
        record = {
            "type": None,
            "scope": None,
            "subject": "Use new value",
            "body": "",
            "subject_evidence": ["new value"],
            "body_evidence": [],
        }
        _, errors = module.validate_record(ROW, saved(record))
        self.assertIn("body_required", errors)


if __name__ == "__main__":
    unittest.main()
