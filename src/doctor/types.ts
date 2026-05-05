import type { ConfigValues } from "../config/index.js";
import type { GitInspection, InspectGitRepositoryOptions } from "../git/types.js";

export type DoctorCheckStatus = "pass" | "warning" | "error";
export type DoctorIssueSeverity = "warning" | "error";
export type DoctorConfigSource = "user" | "project" | "env" | "default" | "missing";

export interface DoctorCheckBase {
  status: DoctorCheckStatus;
  message: string;
  issueCodes: string[];
}

export interface DoctorIssue {
  check: keyof DoctorReport["checks"];
  code: string;
  severity: DoctorIssueSeverity;
  message: string;
}

export interface DoctorConfigSnapshot {
  apiKey: string | null;
  baseURL: string | null;
  customPrompt: string | null;
  model: string | null;
  promptStyle: string | null;
  provider: string | null;
}

export interface DoctorNodeCheck extends DoctorCheckBase {
  currentVersion: string;
  requiredVersion: string | null;
  supported: boolean | null;
}

export interface DoctorGitCheck extends DoctorCheckBase {
  available: boolean;
  version: string | null;
}

export interface DoctorRepositoryCheck extends DoctorCheckBase {
  branchName: string | null;
  gitIdentity: {
    email: string | null;
    name: string | null;
  };
  isBare: boolean;
  isRepository: boolean;
  rootPath: string | null;
}

export interface DoctorConfigDirectoryCheck extends DoctorCheckBase {
  exists: boolean;
  isDirectory: boolean;
  path: string;
  readable: boolean;
  writable: boolean;
}

export interface DoctorConfigFileCheck extends DoctorCheckBase {
  config: DoctorConfigSnapshot | null;
  exists: boolean;
  mode: string | null;
  path: string;
  valid: boolean;
}

export interface DoctorEffectiveConfigCheck extends DoctorCheckBase {
  config: DoctorConfigSnapshot;
  sources: {
    apiKey: DoctorConfigSource;
    baseURL: DoctorConfigSource;
    model: DoctorConfigSource;
    provider: DoctorConfigSource;
  };
}

export interface DoctorReport {
  bin: {
    command: string;
    packageName: string;
  };
  checks: {
    configDirectory: DoctorConfigDirectoryCheck;
    effectiveConfig: DoctorEffectiveConfigCheck;
    git: DoctorGitCheck;
    node: DoctorNodeCheck;
    projectConfig: DoctorConfigFileCheck;
    repository: DoctorRepositoryCheck;
    userConfig: DoctorConfigFileCheck;
  };
  command: "cnm doctor";
  guidance: string[];
  issues: DoctorIssue[];
  ok: boolean;
  paths: {
    cwd: string;
    projectConfigPath: string;
    userConfigHome: string;
    userConfigPath: string;
  };
  readOnly: true;
  status: "ok" | "issues_found";
  summary: {
    errors: number;
    warnings: number;
  };
}

export interface GitVersionResult {
  available: boolean;
  version: string | null;
}

export interface DoctorDependencies {
  inspectGitRepository(options: InspectGitRepositoryOptions): Promise<GitInspection>;
  runGitVersion(cwd: string, env?: NodeJS.ProcessEnv): Promise<GitVersionResult>;
}

export interface RunDoctorOptions {
  cwd: string;
  dependencies?: Partial<DoctorDependencies>;
  env?: NodeJS.ProcessEnv;
  nodeEngine?: string;
  nodeVersion?: string;
}

export interface SafeConfigSourceResult {
  config: ConfigValues;
  error: Error | null;
  exists: boolean;
  warningMessages: string[];
}
