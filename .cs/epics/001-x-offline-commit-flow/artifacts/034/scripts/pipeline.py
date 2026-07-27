#!/usr/bin/env python3
"""Frozen semantic-record and deterministic rendering path for cnm #34."""
from __future__ import annotations

import json
import re
from dataclasses import dataclass

TYPES = {"build", "chore", "ci", "docs", "feat", "fix", "perf", "refactor", "revert", "style", "test"}
RENDER_STYLES = {"conventional", "angular", "google", "atom", "plain", "custom"}
CONVENTIONAL = re.compile(r"^(?:build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)(?:\([^)]+\))?!?:\s*", re.I)
SCOPE = re.compile(r"^[a-z0-9][a-z0-9._/-]{0,31}$")
EXPLANATION = re.compile(r"^(?:here(?:'s| is)|commit message|message|output)\s*[:：]", re.I)

JSON_SCHEMA = {
    "type": "object",
    "additionalProperties": False,
    "required": ["type", "scope", "subject", "body"],
    "properties": {
        "type": {"anyOf": [{"type": "string", "enum": sorted(TYPES)}, {"type": "null"}]},
        "scope": {"anyOf": [{"type": "string", "pattern": SCOPE.pattern}, {"type": "null"}]},
        "subject": {"type": "string", "minLength": 1, "maxLength": 72},
        "body": {"type": "string"},
    },
}

STYLE_INSTRUCTIONS = {
    "conventional": "Write semantic content for a Conventional Commit. Choose a valid type and optional scope. The subject field must not include the type/scope prefix.",
    "angular": "Write semantic content for an Angular commit. Choose a valid lowercase type and optional scope. Use an imperative lowercase subject without a trailing period. The subject field must not include the type/scope prefix.",
    "google": "Use a short, specific, imperative subject without a trailing period. Use body only when it adds useful what/why context.",
    "atom": "Use a concise imperative subject and an optional supporting body. Do not put a Conventional prefix in the subject.",
    "plain": "Use a concise natural-language subject and optional useful body. Do not put a Conventional prefix in the subject.",
    "custom": "Follow the user guidance for subject and body content. Do not add labels, Markdown wrappers, or explanations.",
}


@dataclass(frozen=True)
class SemanticRecord:
    type: str | None
    scope: str | None
    subject: str
    body: str


def _validate_text(record: SemanticRecord) -> None:
    if not record.subject or record.subject != record.subject.strip() or "\n" in record.subject or "\r" in record.subject:
        raise ValueError("subject must be one trimmed non-empty line")
    if len(record.subject) > 72:
        raise ValueError("subject exceeds 72 characters")
    if record.body != record.body.strip():
        raise ValueError("body must be trimmed")
    combined = record.subject + ("\n" + record.body if record.body else "")
    if "```" in combined or EXPLANATION.match(record.subject):
        raise ValueError("wrapper or explanation text is not a commit message")
    if record.type is not None and record.type not in TYPES:
        raise ValueError("invalid type")
    if record.scope is not None and not SCOPE.fullmatch(record.scope):
        raise ValueError("invalid scope")


def parse_record(raw: str) -> SemanticRecord:
    try:
        value = json.loads(raw)
    except json.JSONDecodeError as error:
        raise ValueError("record is not JSON") from error
    if not isinstance(value, dict) or set(value) != {"type", "scope", "subject", "body"}:
        raise ValueError("record keys differ from frozen schema")
    if value["type"] is not None and not isinstance(value["type"], str):
        raise ValueError("type must be string or null")
    if value["scope"] is not None and not isinstance(value["scope"], str):
        raise ValueError("scope must be string or null")
    if not isinstance(value["subject"], str) or not isinstance(value["body"], str):
        raise ValueError("subject/body must be strings")
    record = SemanticRecord(value["type"], value["scope"], value["subject"], value["body"])
    _validate_text(record)
    return record


def resolve_auto(history: list[str]) -> str:
    """Resolve only from history; callers cannot provide the current diff or guidance."""
    subjects = [message.strip().splitlines()[0] for message in history if message.strip()]
    if subjects and sum(bool(CONVENTIONAL.match(subject)) for subject in subjects) * 2 >= len(subjects):
        return "conventional"
    return "plain"


def render(resolved_style: str, record: SemanticRecord) -> str:
    if resolved_style not in RENDER_STYLES:
        raise ValueError("style must be resolved before rendering")
    _validate_text(record)
    subject = record.subject
    if resolved_style in {"conventional", "angular"}:
        if record.type not in TYPES:
            raise ValueError("type required by resolved style")
        if CONVENTIONAL.match(subject):
            raise ValueError("semantic subject already contains a style prefix")
        prefix = record.type + (f"({record.scope})" if record.scope else "") + ": "
        subject = prefix + subject
    message = subject + ("\n\n" + record.body if record.body else "")
    if "```" in message or EXPLANATION.match(message):
        raise ValueError("rendered wrapper/explanation rejected")
    return message


def build_messages(resolved_style: str, diff: str, guidance: str = "", history: list[str] | None = None) -> list[dict[str, str]]:
    if resolved_style not in RENDER_STYLES:
        raise ValueError("unresolved style")
    system = [
        "Generate one semantic record for the complete selected Git diff.",
        "Return exactly one JSON object matching the supplied schema and nothing else.",
        "Describe only behavior supported by the current diff; never copy a history message as the current change.",
        "Subject is semantic content only; type/scope prefixes are rendered separately.",
        STYLE_INSTRUCTIONS[resolved_style],
    ]
    if guidance:
        system.append("Additional user guidance for subject/body content:\n" + guidance)
    if history:
        system.append("Recent commit messages are style/language reference only:\n" + "\n".join(history))
    return [
        {"role": "system", "content": "\n".join(system)},
        {"role": "user", "content": "Generate the semantic commit record for this complete diff:\n\n" + diff},
    ]


def check_requirements(record: SemanticRecord, message: str, requirements: dict) -> list[str]:
    """Mechanical checks only; semantic correctness remains independently scored."""
    failures: list[str] = []
    body_required = requirements.get("body_required") or requirements.get("required_body")
    body_forbidden = requirements.get("body_forbidden") or requirements.get("subject_only")
    if body_required and not record.body:
        failures.append("body_required")
    if body_forbidden and record.body:
        failures.append("body_forbidden")
    prefix = requirements.get("subject_prefix") or requirements.get("required_subject_prefix")
    if prefix and not record.subject.startswith(prefix):
        failures.append("subject_prefix")
    suffix = requirements.get("required_subject_suffix")
    if suffix and not record.subject.endswith(suffix):
        failures.append("subject_suffix")
    body_prefix = requirements.get("body_prefix")
    if body_prefix and not record.body.startswith(body_prefix):
        failures.append("body_prefix")
    reference = requirements.get("reference") or requirements.get("required_issue_reference")
    if reference and reference not in record.body:
        failures.append("reference")
    bullets = requirements.get("exact_body_bullets")
    if bullets is not None:
        actual = [line for line in record.body.splitlines() if line.startswith("- ")]
        if len(actual) != bullets or len(actual) != len([line for line in record.body.splitlines() if line.strip()]):
            failures.append("exact_body_bullets")
    shape = requirements.get("body_shape")
    if isinstance(shape, str) and "bullet" in shape.lower():
        wanted = 2 if "two" in shape.lower() or "2" in shape else None
        actual = [line for line in record.body.splitlines() if line.startswith("- ")]
        nonempty = [line for line in record.body.splitlines() if line.strip()]
        if len(actual) != len(nonempty) or (wanted is not None and len(actual) != wanted):
            failures.append("body_shape")
    language = requirements.get("language")
    if (requirements.get("simplified_chinese") or (isinstance(language, str) and language.lower().startswith("zh"))) and not re.search(r"[\u4e00-\u9fff]", record.subject + record.body):
        failures.append("simplified_chinese")
    maximum = requirements.get("subject_max_characters")
    if isinstance(maximum, int) and len(record.subject) > maximum:
        failures.append("subject_max_characters")
    if requirements.get("lowercase_subject") and record.subject[:1].isalpha() and not record.subject[:1].islower():
        failures.append("lowercase_subject")
    if requirements.get("message_contains_history"):
        failures.append("history_substitution")
    if not message.strip():
        failures.append("empty_message")
    return failures
