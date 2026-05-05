import type { DoctorConfigSnapshot, DoctorReport } from "./types.js";

function labelForStatus(status: DoctorReport["checks"][keyof DoctorReport["checks"]]["status"]): string {
  switch (status) {
    case "pass":
      return "PASS";
    case "warning":
      return "WARN";
    case "error":
      return "ERROR";
    default:
      return status;
  }
}

function formatValue(value: string | boolean | null): string {
  if (value === null) {
    return "(unset)";
  }

  if (typeof value === "boolean") {
    return value ? "true" : "false";
  }

  return value;
}

function pushConfigLines(lines: string[], config: DoctorConfigSnapshot | null): void {
  if (!config) {
    lines.push("  config=(unavailable)");
    return;
  }

  lines.push(`  provider=${formatValue(config.provider)}`);
  lines.push(`  model=${formatValue(config.model)}`);
  lines.push(`  baseURL=${formatValue(config.baseURL)}`);
  lines.push(`  apiKey=${formatValue(config.apiKey)}`);
}

export function renderDoctorReport(report: DoctorReport): string {
  const lines: string[] = [];

  lines.push("cnm doctor");
  lines.push("");
  lines.push(`Status: ${report.status} (${report.summary.errors} errors, ${report.summary.warnings} warnings)`);
  lines.push("");
  lines.push(`Node: ${labelForStatus(report.checks.node.status)} - ${report.checks.node.message}`);
  lines.push(`  current=${report.checks.node.currentVersion}`);
  lines.push(`  required=${formatValue(report.checks.node.requiredVersion)}`);
  lines.push(`Git: ${labelForStatus(report.checks.git.status)} - ${report.checks.git.message}`);
  lines.push(`  available=${formatValue(report.checks.git.available)}`);
  lines.push(`  version=${formatValue(report.checks.git.version)}`);
  lines.push(`Repository: ${labelForStatus(report.checks.repository.status)} - ${report.checks.repository.message}`);
  lines.push(`  root=${formatValue(report.checks.repository.rootPath)}`);
  lines.push(`  branch=${formatValue(report.checks.repository.branchName)}`);
  lines.push(`  gitIdentity.name=${formatValue(report.checks.repository.gitIdentity.name)}`);
  lines.push(`  gitIdentity.email=${formatValue(report.checks.repository.gitIdentity.email)}`);
  lines.push(`Config directory: ${labelForStatus(report.checks.configDirectory.status)} - ${report.checks.configDirectory.message}`);
  lines.push(`  path=${report.checks.configDirectory.path}`);
  lines.push(`  exists=${formatValue(report.checks.configDirectory.exists)}`);
  lines.push(`  readable=${formatValue(report.checks.configDirectory.readable)}`);
  lines.push(`  writable=${formatValue(report.checks.configDirectory.writable)}`);
  lines.push(`User config: ${labelForStatus(report.checks.userConfig.status)} - ${report.checks.userConfig.message}`);
  lines.push(`  path=${report.checks.userConfig.path}`);
  lines.push(`  exists=${formatValue(report.checks.userConfig.exists)}`);
  lines.push(`  valid=${formatValue(report.checks.userConfig.valid)}`);
  lines.push(`  mode=${formatValue(report.checks.userConfig.mode)}`);
  pushConfigLines(lines, report.checks.userConfig.config);
  lines.push(`Project config: ${labelForStatus(report.checks.projectConfig.status)} - ${report.checks.projectConfig.message}`);
  lines.push(`  path=${report.checks.projectConfig.path}`);
  lines.push(`  exists=${formatValue(report.checks.projectConfig.exists)}`);
  lines.push(`  valid=${formatValue(report.checks.projectConfig.valid)}`);
  lines.push(`  mode=${formatValue(report.checks.projectConfig.mode)}`);
  pushConfigLines(lines, report.checks.projectConfig.config);
  lines.push(`Effective config: ${labelForStatus(report.checks.effectiveConfig.status)} - ${report.checks.effectiveConfig.message}`);
  lines.push(`  provider=${formatValue(report.checks.effectiveConfig.config.provider)} (source=${report.checks.effectiveConfig.sources.provider})`);
  lines.push(`  model=${formatValue(report.checks.effectiveConfig.config.model)} (source=${report.checks.effectiveConfig.sources.model})`);
  lines.push(`  baseURL=${formatValue(report.checks.effectiveConfig.config.baseURL)} (source=${report.checks.effectiveConfig.sources.baseURL})`);
  lines.push(`  apiKey=${formatValue(report.checks.effectiveConfig.config.apiKey)} (source=${report.checks.effectiveConfig.sources.apiKey})`);

  if (report.issues.length > 0) {
    lines.push("");
    lines.push("Issues:");

    for (const issue of report.issues) {
      lines.push(`- [${issue.severity}] ${issue.code}: ${issue.message}`);
    }
  }

  if (report.guidance.length > 0) {
    lines.push("");
    lines.push("Guidance:");

    for (const guidance of report.guidance) {
      lines.push(`- ${guidance}`);
    }
  }

  return lines.join("\n");
}
