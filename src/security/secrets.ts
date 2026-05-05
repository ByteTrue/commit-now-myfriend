export type SecretSeverity = "warning" | "blocking";

export interface SecretFinding {
  code: string;
  description: string;
  line: number;
  excerpt: string;
  severity: SecretSeverity;
}

export interface SecretScanResult {
  findings: SecretFinding[];
  hasBlockingFindings: boolean;
}

interface SecretPattern {
  code: string;
  description: string;
  expression: RegExp;
  severity: SecretSeverity;
}

const secretPatterns: SecretPattern[] = [
  {
    code: "private_key",
    description: "Private key material appears in the diff.",
    expression: /-----BEGIN (?:RSA |DSA |EC |OPENSSH |PGP )?PRIVATE KEY-----/i,
    severity: "warning"
  },
  {
    code: "aws_access_key",
    description: "AWS access key-like token appears in the diff.",
    expression: /\b(?:AKIA|ASIA)[A-Z0-9]{16}\b/,
    severity: "warning"
  },
  {
    code: "provider_api_key",
    description: "Provider API key-like token appears in the diff.",
    expression: /\b(?:sk|xox[baprs]|ghp|github_pat)_[A-Za-z0-9_\-]{16,}\b/i,
    severity: "warning"
  },
  {
    code: "assigned_secret",
    description: "Secret-looking assignment appears in the diff.",
    expression: /(?:api[_-]?key|secret|token|password)\s*[:=]\s*["'][^"'\s]{12,}["']/i,
    severity: "warning"
  }
];

function redactExcerpt(line: string): string {
  return line
    .replace(/(["'=:\s])([A-Za-z0-9_\-\/+]{8})[A-Za-z0-9_\-\/+]{4,}(["'\s]|$)/g, "$1$2…$3")
    .slice(0, 160);
}

export function scanTextForSecrets(text: string): SecretScanResult {
  const findings: SecretFinding[] = [];
  const lines = text.split(/\r?\n/);

  for (const [index, line] of lines.entries()) {
    for (const pattern of secretPatterns) {
      if (pattern.expression.test(line)) {
        findings.push({
          code: pattern.code,
          description: pattern.description,
          line: index + 1,
          excerpt: redactExcerpt(line),
          severity: pattern.severity
        });
      }
    }
  }

  return {
    findings,
    hasBlockingFindings: findings.some((finding) => finding.severity === "blocking")
  };
}
