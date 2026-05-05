import { confirm, isCancel, password, select, text } from "@clack/prompts";

import { PROMPT_STYLES, PROVIDER_TYPES, type ConfigValues, type PromptStyle } from "../config/index.js";
import {
  getResetTargets,
  type ConfigPanelAction,
  type ConfigPanelPrompts,
  type ConfigPanelResetTarget
} from "./config-panel.js";

export interface CreateConfigPromptsOptions {
  stdout?: NodeJS.WriteStream;
}

function unexpectedPromptCall(name: keyof ConfigPanelPrompts): never {
  throw new Error(`Internal error: config prompt '${name}' should not run in non-interactive mode.`);
}

function validateRequiredValue(value: string | undefined, message: string): string | undefined {
  return value?.trim() ? undefined : message;
}

function toResetTargetLabel(target: ConfigPanelResetTarget): string {
  return target === "all" ? "Reset all user config values" : `Unset ${target}`;
}

function toPromptStyleLabel(value: PromptStyle): string {
  return value === "auto" ? "auto (default)" : value;
}

function toPromptStyleHint(value: PromptStyle): string {
  switch (value) {
    case "auto":
      return "Infer style when possible; fallback to Conventional Commits.";
    case "conventional":
      return "type(scope)?: subject with optional body.";
    case "angular":
      return "Angular-style type(scope): subject.";
    case "google":
      return "Short imperative subject; optional what/why body.";
    case "atom":
      return "Concise imperative subject; optional body, no strict prefix.";
    case "plain":
      return "Plain natural-language commit message.";
    case "custom":
      return "No predefined style instructions; rely on custom prompt.";
  }
}

function resolveCurrentConfigLabel(userConfig: ConfigValues): string {
  const configuredKeys = Object.keys(userConfig).length;
  return configuredKeys === 0 ? "No user overrides saved" : `${configuredKeys} user override${configuredKeys === 1 ? "" : "s"}`;
}

export function createClackConfigPrompts({ stdout }: CreateConfigPromptsOptions = {}): ConfigPanelPrompts {
  return {
    async confirmReset({ target }) {
      const confirmed = await confirm({
        initialValue: false,
        message: target === "all"
          ? "Reset all user config values?"
          : `Unset ${target} from user config?`,
        output: stdout
      });

      if (isCancel(confirmed)) {
        return null;
      }

      return confirmed;
    },
    async inputApiKey({ hasExistingValue }) {
      const value = await password({
        clearOnError: false,
        mask: "*",
        message: hasExistingValue ? "Enter the replacement API key." : "Enter the API key.",
        output: stdout,
        validate(nextValue) {
          return validateRequiredValue(nextValue, "API key cannot be empty.");
        }
      });

      if (isCancel(value)) {
        return null;
      }

      return value.trim();
    },
    async inputBaseURL({ currentValue }) {
      const value = await text({
        initialValue: currentValue,
        message: "Enter the base URL.",
        output: stdout,
        placeholder: "https://api.example.com/v1",
        validate(nextValue) {
          return validateRequiredValue(nextValue, "baseURL cannot be empty. Use reset/unset to remove it.");
        }
      });

      if (isCancel(value)) {
        return null;
      }

      return value.trim();
    },
    async inputCustomPrompt({ currentValue }) {
      const value = await text({
        initialValue: currentValue,
        message: "Enter the custom prompt.",
        output: stdout,
        placeholder: "Give extra instructions for commit generation.",
        validate(nextValue) {
          return validateRequiredValue(nextValue, "Custom prompt cannot be empty. Use reset/unset to remove it.");
        }
      });

      if (isCancel(value)) {
        return null;
      }

      return value.trim();
    },
    async inputModel({ currentValue, provider }) {
      const value = await text({
        initialValue: currentValue,
        message: `Enter the default model for ${provider}.`,
        output: stdout,
        validate(nextValue) {
          return validateRequiredValue(nextValue, "Model cannot be empty.");
        }
      });

      if (isCancel(value)) {
        return null;
      }

      return value.trim();
    },
    async selectAction({ effectiveConfig, userConfig }) {
      const action = await select({
        initialValue: "configureProviderModel",
        message: `Choose a config action (${effectiveConfig.provider} / ${effectiveConfig.model}; ${resolveCurrentConfigLabel(userConfig)}).`,
        options: [
          { label: "Configure provider and model", value: "configureProviderModel" },
          { label: "Set API key", value: "setApiKey" },
          { label: "Set baseURL", value: "setBaseURL" },
          { label: "Set commit style", value: "setPromptStyle" },
          { label: "Set custom prompt", value: "setCustomPrompt" },
          { label: "View effective config", value: "viewEffectiveConfig" },
          { label: "Test current config", value: "testCurrentConfig" },
          { label: "Reset or unset values", value: "resetUnset" },
          { label: "Exit", value: "exit" }
        ],
        output: stdout
      });

      if (isCancel(action)) {
        return null;
      }

      return action as ConfigPanelAction;
    },
    async selectPromptStyle({ currentPromptStyle }) {
      const promptStyle = await select({
        initialValue: currentPromptStyle,
        message: "Choose the commit message style.",
        options: PROMPT_STYLES.map((value) => ({
          hint: toPromptStyleHint(value),
          label: toPromptStyleLabel(value),
          value
        })),
        output: stdout
      });

      if (isCancel(promptStyle)) {
        return null;
      }

      return promptStyle;
    },
    async selectProvider({ currentProvider }) {
      const provider = await select({
        initialValue: currentProvider,
        message: "Choose the default provider.",
        options: PROVIDER_TYPES.map((value) => ({ label: value, value })),
        output: stdout
      });

      if (isCancel(provider)) {
        return null;
      }

      return provider;
    },
    async selectResetTarget({ userConfig }) {
      const target = await select({
        initialValue: "all",
        message: "Choose what to reset or unset.",
        options: getResetTargets(userConfig).map((value) => ({ label: toResetTargetLabel(value), value })),
        output: stdout
      });

      if (isCancel(target)) {
        return null;
      }

      return target as ConfigPanelResetTarget;
    }
  };
}

export function createNonInteractiveConfigPrompts(): ConfigPanelPrompts {
  return {
    async confirmReset() {
      return unexpectedPromptCall("confirmReset");
    },
    async inputApiKey() {
      return unexpectedPromptCall("inputApiKey");
    },
    async inputBaseURL() {
      return unexpectedPromptCall("inputBaseURL");
    },
    async inputCustomPrompt() {
      return unexpectedPromptCall("inputCustomPrompt");
    },
    async inputModel() {
      return unexpectedPromptCall("inputModel");
    },
    async selectAction() {
      return unexpectedPromptCall("selectAction");
    },
    async selectPromptStyle() {
      return unexpectedPromptCall("selectPromptStyle");
    },
    async selectProvider() {
      return unexpectedPromptCall("selectProvider");
    },
    async selectResetTarget() {
      return unexpectedPromptCall("selectResetTarget");
    }
  };
}
