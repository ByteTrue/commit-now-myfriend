#!/usr/bin/env python3
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


class ScoreGateTest(unittest.TestCase):
    def run_gate(self, bad: bool, string_boolean: bool = False) -> tuple[int, dict | None]:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            results = root / "results.jsonl"
            scores = root / "scores.jsonl"
            out = root / "summary.json"
            with results.open("w") as result_stream, scores.open("w") as score_stream:
                for index in range(30):
                    case_id = f"case-{index}"
                    result_stream.write(json.dumps({
                        "id": case_id,
                        "status": "generated",
                        "expected_rejection": None,
                        "mechanical_pass": True,
                        "requirements": {},
                    }) + "\n")
                    score_stream.write(json.dumps({
                        "id": case_id,
                        "raw_score": 0 if bad and index == 0 else 2,
                        "rendered_score": 0 if bad and index == 0 else 2,
                        "guidance_pass": "false" if string_boolean and index == 0 else True,
                        "body_pass": True,
                    }) + "\n")
            completed = subprocess.run([
                sys.executable,
                str(Path(__file__).with_name("score_gate.py")),
                "--results", str(results),
                "--scores", str(scores),
                "--mode", "shadow",
                "--out", str(out),
            ], stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
            return completed.returncode, json.loads(out.read_text()) if out.is_file() else None

    def test_all_semantic_scores_pass(self):
        code, summary = self.run_gate(False)
        self.assertEqual(code, 0)
        self.assertTrue(summary["quality_gate_pass"])

    def test_swapped_or_wrong_semantics_cannot_pass(self):
        code, summary = self.run_gate(True)
        self.assertEqual(code, 2)
        self.assertFalse(summary["quality_gate_pass"])
        self.assertEqual(summary["zero"], 1)
    def test_string_boolean_is_rejected(self):
        code, summary = self.run_gate(False, string_boolean=True)
        self.assertNotEqual(code, 0)
        self.assertIsNone(summary)


if __name__ == "__main__":
    unittest.main()
