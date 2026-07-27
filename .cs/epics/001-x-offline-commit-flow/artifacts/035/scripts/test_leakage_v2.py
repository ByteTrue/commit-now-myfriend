#!/usr/bin/env python3
import importlib.util
import sys
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name("leakage_v2.py")
spec = importlib.util.spec_from_file_location("leakage_v2_tested", SCRIPT)
module = importlib.util.module_from_spec(spec)
sys.modules[spec.name] = module
spec.loader.exec_module(module)


class LeakageV2Tests(unittest.TestCase):
    def test_metadata_only_rename_is_signed(self):
        diff = "diff --git a/old.txt b/new.txt\nsimilarity index 100%\nrename from old.txt\nrename to new.txt\n"
        value = module.signatures(diff)
        self.assertGreaterEqual(value["content_lines"], 4)
        self.assertTrue(value["minhash"])

    def test_seven_of_eight_match_has_shared_key(self):
        left = list(range(8))
        right = list(range(7)) + [99]
        self.assertTrue(module.near_keys(left) & module.near_keys(right))

    def test_six_of_eight_match_has_no_shared_key(self):
        left = list(range(8))
        right = list(range(6)) + [98, 99]
        self.assertFalse(module.near_keys(left) & module.near_keys(right))


if __name__ == "__main__":
    unittest.main()
