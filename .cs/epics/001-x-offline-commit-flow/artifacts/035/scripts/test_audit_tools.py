#!/usr/bin/env python3
import hashlib
import importlib.util
import json
import sys
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name("merge_scores.py")
spec = importlib.util.spec_from_file_location("merge_scores", SCRIPT)
module = importlib.util.module_from_spec(spec)
sys.modules[spec.name] = module
spec.loader.exec_module(module)


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def score(index, input_hash):
    return {
        "index": index,
        "reviewer": "A",
        "input_slice_sha256": input_hash,
        "critical_error": False,
        "fully_grounded": True,
        "subject_quality": 2,
        "body_quality": None,
        "evidence_quality": 2,
        "reason": "grounded",
    }


def write_jsonl(path, rows):
    path.write_text("".join(json.dumps(row) + "\n" for row in rows))


class AuditTests(unittest.TestCase):
    def make_slices(self, directory):
        slices = []
        for number in range(10):
            input_path = directory / f"input-{number}.jsonl"
            score_path = directory / f"score-{number}.jsonl"
            indices = list(range(number * 20, number * 20 + 20))
            write_jsonl(input_path, [{"index": index} for index in indices])
            input_hash = sha(input_path)
            write_jsonl(score_path, [score(index, input_hash) for index in indices])
            slices.append((input_path, score_path, input_hash))
        return slices

    def test_bound_complete_scores_load(self):
        with tempfile.TemporaryDirectory() as directory:
            values, hashes = module.read_scores("A", self.make_slices(Path(directory)))
            self.assertEqual(len(values), 200)
            self.assertEqual(len(hashes), 10)

    def test_expected_pid_counts_reject_any_third_pid(self):
        expected = module.expected_label_pid_counts(10, 20)
        self.assertEqual(expected, {"10": 1, "20": 199})
        tampered = {"10": 1, "20": 198, "30": 1}
        self.assertNotEqual(tampered, expected)

    def test_reused_pid_still_requires_all_200(self):
        self.assertEqual(module.expected_label_pid_counts(10, 10), {"10": 200})

    def test_duplicate_score_rejects(self):
        with tempfile.TemporaryDirectory() as directory:
            slices = self.make_slices(Path(directory))
            input_path, score_path, input_hash = slices[0]
            with score_path.open("a") as stream:
                stream.write(json.dumps(score(0, input_hash)) + "\n")
            with self.assertRaises(SystemExit):
                module.read_scores("A", slices)


if __name__ == "__main__":
    unittest.main()
