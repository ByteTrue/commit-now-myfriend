package commands

import (
	"flag"
	"fmt"
	"io"

	"github.com/ByteTrue/commit-now-myfriend/internal/config"
	"github.com/ByteTrue/commit-now-myfriend/internal/output"
)

const plaintextAPIKeyWarning = "Warning: API keys are stored in plaintext in the user config file. Prefer environment variables when possible."

type ConfigRuntime struct {
	CWD         string
	Env         map[string]string
	Stdout      io.Writer
	Stderr      io.Writer
	SecretStore config.SecretStore
}

func RunConfig(args []string, runtime ConfigRuntime) int {
	if len(args) == 0 {
		return runConfigList(nil, runtime)
	}

	switch args[0] {
	case "get":
		return runConfigGet(args[1:], runtime)
	case "list":
		return runConfigList(args[1:], runtime)
	case "set":
		return runConfigSet(args[1:], runtime)
	case "unset":
		return runConfigUnset(args[1:], runtime)
	default:
		fmt.Fprintf(runtime.Stderr, "error: unknown config subcommand %q\n", args[0])
		return int(output.Error)
	}
}

func runConfigGet(args []string, runtime ConfigRuntime) int {
	fs := flag.NewFlagSet("cnm config get", flag.ContinueOnError)
	fs.SetOutput(runtime.Stderr)
	jsonMode := fs.Bool("json", false, "emit JSON output")
	if err := fs.Parse(args); err != nil {
		return int(output.UsageError)
	}

	router := output.NewRouter(*jsonMode, runtime.Stdout, runtime.Stderr)
	resolved, err := config.ResolveEffectiveConfig(config.ResolveConfigOptions{CWD: runtime.CWD, Env: runtime.Env, SecretStore: runtime.SecretStore})
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.Error)
	}
	for _, warning := range resolved.Warnings {
		_ = router.WriteHuman("Warning: "+warning, output.StderrTarget)
	}

	var key string
	if fs.NArg() > 0 {
		key = fs.Arg(0)
	}
	if fs.NArg() > 1 {
		fmt.Fprintf(runtime.Stderr, "error: unexpected config arguments: %v\n", fs.Args()[1:])
		return int(output.UsageError)
	}

	if key != "" {
		configKey, err := config.AssertConfigKey(key)
		if err != nil {
			fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
			return int(output.Error)
		}
		value := config.GetConfigValue(resolved.Values, configKey)
		if router.IsJSON() {
			_ = router.WriteJSON(map[string]*string{string(configKey): value})
		} else if value == nil {
			_ = router.WriteHuman(string(configKey)+"=(unset)", output.StdoutTarget)
		} else {
			_ = router.WriteHuman(string(configKey)+"="+*value, output.StdoutTarget)
		}
		return int(output.Success)
	}

	if router.IsJSON() {
		_ = router.WriteJSON(config.ToJSONConfigView(resolved.Values))
	} else {
		for _, line := range config.ToHumanConfigLines(resolved.Values) {
			_ = router.WriteHuman(line, output.StdoutTarget)
		}
	}
	return int(output.Success)
}

func runConfigList(args []string, runtime ConfigRuntime) int {
	fs := flag.NewFlagSet("cnm config list", flag.ContinueOnError)
	fs.SetOutput(runtime.Stderr)
	jsonMode := fs.Bool("json", false, "emit JSON output")
	if err := fs.Parse(args); err != nil {
		return int(output.UsageError)
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(runtime.Stderr, "error: unexpected config arguments: %v\n", fs.Args())
		return int(output.UsageError)
	}
	if *jsonMode {
		return runConfigGet([]string{"--json"}, runtime)
	}
	return runConfigGet(nil, runtime)
}

func runConfigSet(args []string, runtime ConfigRuntime) int {
	fs := flag.NewFlagSet("cnm config set", flag.ContinueOnError)
	fs.SetOutput(runtime.Stderr)
	jsonMode := fs.Bool("json", false, "emit JSON output")
	dryRun := fs.Bool("dry-run", false, "preview command execution without side effects")
	if err := fs.Parse(args); err != nil {
		return int(output.UsageError)
	}
	if fs.NArg() != 2 {
		fmt.Fprintln(runtime.Stderr, "error: cnm config set requires <key> <value>")
		return int(output.UsageError)
	}

	key := fs.Arg(0)
	value := fs.Arg(1)
	router := output.NewRouter(*jsonMode, runtime.Stdout, runtime.Stderr)
	configKey, err := config.AssertConfigKey(key)
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.Error)
	}
	if configKey == config.ConfigKeyAPIKey {
		_ = router.WriteHuman(plaintextAPIKeyWarning, output.StderrTarget)
	}
	masked := value
	if configKey == config.ConfigKeyAPIKey {
		masked = "[redacted]"
	}
	paths := config.GetConfigPaths(config.ConfigEnvironment{CWD: runtime.CWD, Env: runtime.Env})
	if *dryRun {
		payload := map[string]any{
			"command": "cnm config set",
			"dryRun":  true,
			"key":     string(configKey),
			"ok":      true,
			"path":    paths.UserConfigPath,
			"status":  "dry_run",
			"value":   masked,
		}
		_ = router.WriteStructured(payload, "Dry-run: would update "+string(configKey)+" in user config at "+paths.UserConfigPath+".", output.StdoutTarget)
		return int(output.Success)
	}

	patch, err := config.ParseKeyValue(key, value)
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.Error)
	}
	result, err := config.WriteUserConfigPatch(patch, config.ConfigEnvironment{CWD: runtime.CWD, Env: runtime.Env})
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.Error)
	}
	for _, warning := range result.Warnings {
		_ = router.WriteHuman("Warning: "+warning, output.StderrTarget)
	}
	payload := map[string]any{
		"command": "cnm config set",
		"dryRun":  false,
		"key":     string(configKey),
		"ok":      true,
		"path":    result.Path,
		"status":  "updated",
		"value":   masked,
	}
	_ = router.WriteStructured(payload, "Updated user config at "+result.Path+".\n"+string(configKey)+"="+masked, output.StdoutTarget)
	return int(output.Success)
}

func runConfigUnset(args []string, runtime ConfigRuntime) int {
	fs := flag.NewFlagSet("cnm config unset", flag.ContinueOnError)
	fs.SetOutput(runtime.Stderr)
	jsonMode := fs.Bool("json", false, "emit JSON output")
	dryRun := fs.Bool("dry-run", false, "preview command execution without side effects")
	if err := fs.Parse(args); err != nil {
		return int(output.UsageError)
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(runtime.Stderr, "error: cnm config unset requires <key>")
		return int(output.UsageError)
	}

	key := fs.Arg(0)
	configKey, err := config.AssertConfigKey(key)
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.Error)
	}
	router := output.NewRouter(*jsonMode, runtime.Stdout, runtime.Stderr)
	paths := config.GetConfigPaths(config.ConfigEnvironment{CWD: runtime.CWD, Env: runtime.Env})
	if *dryRun {
		payload := map[string]any{
			"command": "cnm config unset",
			"dryRun":  true,
			"key":     string(configKey),
			"ok":      true,
			"path":    paths.UserConfigPath,
			"status":  "dry_run",
		}
		_ = router.WriteStructured(payload, "Dry-run: would remove "+string(configKey)+" from user config at "+paths.UserConfigPath+".", output.StdoutTarget)
		return int(output.Success)
	}

	result, err := config.UnsetUserConfigKey(configKey, config.ConfigEnvironment{CWD: runtime.CWD, Env: runtime.Env})
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.Error)
	}
	for _, warning := range result.Warnings {
		_ = router.WriteHuman("Warning: "+warning, output.StderrTarget)
	}
	payload := map[string]any{
		"command": "cnm config unset",
		"dryRun":  false,
		"key":     string(configKey),
		"ok":      true,
		"path":    result.Path,
		"status":  "removed",
	}
	_ = router.WriteStructured(payload, "Removed "+string(configKey)+" from user config at "+result.Path+".", output.StdoutTarget)
	return int(output.Success)
}
