#!/usr/bin/env python3
import importlib.util
import sys
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name("build_pilot.py")
spec = importlib.util.spec_from_file_location("build_pilot", SCRIPT)
module = importlib.util.module_from_spec(spec)
sys.modules[spec.name] = module
spec.loader.exec_module(module)


class PilotTests(unittest.TestCase):
    def test_file_bins_are_bounded(self):
        self.assertEqual(module.file_bin(1), "single")
        self.assertEqual(module.file_bin(2), "two_three")
        self.assertEqual(module.file_bin(3), "two_three")
        self.assertEqual(module.file_bin(4), "four_eight")
        self.assertEqual(module.file_bin(8), "four_eight")
        self.assertIsNone(module.file_bin(9))

    def test_changed_lines_exclude_headers_and_context(self):
        diff = """diff --git a/a b/a
--- a/a
+++ b/a
@@ -1,2 +1,2 @@
-old
+new
 context
"""
        self.assertEqual(module.changed_line_texts(diff), ["old", "new"])

    def test_smallest_heap_is_deterministic(self):
        candidate = lambda name: module.Candidate(name, name, name, name, "Go", "MIT License", ["a"], 1, "d", name, name, (), name, (), 1)
        heap = []
        for score, name in [(8, "b"), (2, "a"), (5, "c")]:
            module.push_smallest(heap, score, candidate(name), limit=2)
        self.assertEqual({entry[2].family for entry in heap}, {"a", "c"})


if __name__ == "__main__":
    unittest.main()
