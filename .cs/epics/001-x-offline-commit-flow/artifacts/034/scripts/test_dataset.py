#!/usr/bin/env python3
import unittest

from build_dataset import (
    Candidate,
    canonical_repo_components,
    canonical_repo,
    clean_message,
    grounded_body,
    grounded,
    leakage_signatures,
    make_diff,
    repo_group,
    semantic_record,
    sensitive_categories,
    signature_pairs,
)


class DatasetBuilderTest(unittest.TestCase):
    def test_complete_multi_file_diff(self):
        patch = "@@ -1 +1 @@\n" + "-old parser line\n+new parser line\n" * 12
        result = make_diff([
            {"change_type": "MODIFY", "old_path": "a.go", "new_path": "a.go", "diff": patch},
            {"change_type": "ADD", "old_path": "", "new_path": "b.go", "diff": patch},
        ])
        self.assertIsNotNone(result)
        diff, paths = result
        self.assertEqual(paths, ["a.go", "b.go"])
        self.assertEqual(diff.count("diff --git "), 2)
        self.assertIn("--- /dev/null\n+++ b/b.go", diff)

    def test_incomplete_commit_is_rejected_whole(self):
        self.assertIsNone(make_diff([
            {"change_type": "MODIFY", "old_path": "a.go", "new_path": "a.go", "diff": "@@ -1 +1 @@\n-old\n+new"},
            {"change_type": "MODIFY", "old_path": "b.go", "new_path": "b.go", "diff": ""},
        ]))

    def test_message_and_conventional_record(self):
        self.assertEqual(clean_message("fix(cli): preserve staged changes\n\nAvoid unrelated files."),
                         ("fix(cli): preserve staged changes", "Avoid unrelated files."))
        style, record = semantic_record("fix(cli): preserve staged changes", "Avoid unrelated files.", "01" * 32)
        self.assertIn(style, {"conventional", "angular"})
        self.assertEqual(record, {"type": "fix", "scope": "cli", "subject": "preserve staged changes", "body": "Avoid unrelated files."})

    def test_message_rejects_identity_trailers(self):
        self.assertIsNone(clean_message("Fix parser\n\nSigned-off-by: A <a@example.com>"))

    def test_secret_and_pii_patterns(self):
        self.assertIn("private_key", sensitive_categories("-----BEGIN PRIVATE KEY-----"))
        self.assertIn("email", sensitive_categories("owner@example.com"))
        self.assertIn("ipv4", sensitive_categories("connect to 192.168.1.1"))

    def test_repo_normalization_groups_forks_by_basename(self):
        self.assertEqual(canonical_repo("Owner/Project.git"), "owner/project")
        self.assertEqual(repo_group("owner/project"), "project")
        self.assertEqual(repo_group("fork/project"), "project")

    def candidate(self, repo: str, commit: str, patch: str, family: str) -> Candidate:
        return Candidate(
            family=family, repo=repo, repo_group=repo_group(repo), commit=commit,
            language="Go", license="MIT License", paths=["x.go"], file_count=1,
            diff="@@ -1 +1 @@\n-old\n+new", diff_sha256=family, message_sha256=family,
            normalized_patch_sha256=patch, changed_token_minhash=(family,),
            normalized_target_sha256=family, target_token_minhash=(family,),
            style="plain", record={"type": None, "scope": None, "subject": "update code", "body": ""}, input_tokens=10,
        )

    def test_canonical_components_link_renamed_fork_by_commit(self):
        rows = [
            self.candidate("upstream/project", "same-commit", "p1", "a"),
            self.candidate("fork/renamed", "same-commit", "p2", "b"),
        ]
        mapping, evidence = canonical_repo_components(rows)
        self.assertEqual(mapping["upstream/project"], mapping["fork/renamed"])
        self.assertEqual(evidence["shared_commit_links"], 1)

    def test_path_independent_leakage_signatures(self):
        left = "diff --git a/a.go b/a.go\n@@ -1 +1 @@\n-old value\n+new parser value\n"
        right = "diff --git a/renamed.go b/renamed.go\n@@ -40,2 +99,2 @@\n-old   value\n+new parser value\n"
        left_patch, left_minhash, _, _ = leakage_signatures(left, "Fix parser value")
        right_patch, right_minhash, _, _ = leakage_signatures(right, "Fix parser value")
        self.assertEqual(left_patch, right_patch)
        self.assertEqual(left_minhash, right_minhash)
        self.assertEqual(signature_pairs(left_minhash), signature_pairs(right_minhash))

    def test_unsupported_body_and_home_path_are_rejected(self):
        diff = "diff --git a/parser.go b/parser.go\n+func parseInput() {}\n"
        self.assertEqual(grounded_body("This is faster because of caching.", diff, ["parser.go"]), "")
        self.assertIn("home_path_pii", sensitive_categories("fixture: /Users/alice/project"))

    def test_dangling_issue_trailer_is_removed(self):
        self.assertEqual(clean_message("Handle parser input\n\nFixes"), ("Handle parser input", ""))

    def test_grounding_needs_content_overlap(self):
        diff = "diff --git a/parser.go b/parser.go\n+func parseInput() {}\n"
        self.assertTrue(grounded("Handle parser input", diff, ["parser.go"]))
        self.assertFalse(grounded("Improve payment settlement", diff, ["parser.go"]))


if __name__ == "__main__":
    unittest.main()
