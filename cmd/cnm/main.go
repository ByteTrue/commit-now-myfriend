package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ByteTrue/commit-now-myfriend/internal/api"
	"github.com/ByteTrue/commit-now-myfriend/internal/config"
	"github.com/ByteTrue/commit-now-myfriend/internal/git"
	"github.com/ByteTrue/commit-now-myfriend/internal/style"
)

var (
	flagStyle    = flag.String("style", "", "Commit message style (auto/conventional/angular/google/atom/plain/custom)")
	flagPrompt   = flag.String("prompt", "", "Custom prompt guidance")
	flagProvider = flag.String("provider", "", "API provider (openai-chat/openai-response/anthropic-message)")
	flagModel    = flag.String("model", "", "Model name")
	flagAPIKey   = flag.String("api-key", "", "API key")
	flagAPIURL   = flag.String("api-url", "", "API base URL")
	flagYes      = flag.Bool("yes", false, "Skip confirmation and commit directly")
	flagVersion  = flag.Bool("version", false, "Show version")
	flagDryRun   = flag.Bool("dry-run", false, "Show prompts without calling API")
)

const version = "0.3.0"

func main() {
	flag.Parse()

	if *flagVersion {
		fmt.Println("cnm", version)
		return
	}

	// Subcommand: cnm setup
	if flag.Arg(0) == "setup" {
		runSetup()
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cnm:", err)
		os.Exit(1)
	}
	cfg.OverrideFromCLI(*flagStyle, *flagPrompt, *flagProvider, *flagModel, *flagAPIKey, *flagAPIURL)

	if cfg.APIKey == "" && !*flagDryRun {
		fmt.Fprintln(os.Stderr, "cnm: no API key configured.")
		fmt.Fprintln(os.Stderr, "  Run: cnm setup")
		fmt.Fprintln(os.Stderr, "  Or set: CNM_API_KEY, OPENAI_API_KEY, or ANTHROPIC_API_KEY")
		os.Exit(1)
	}

	diff, files, err := git.StagedDiff()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cnm:", err)
		os.Exit(1)
	}

	if strings.TrimSpace(diff) == "" {
		fmt.Fprintln(os.Stderr, "cnm: no changes to commit")
		os.Exit(1)
	}

	input := style.CommitInput{
		Diff:         diff,
		Files:        files,
		Repo:         git.RepoInfo(),
		Style:        cfg.Style,
		CustomPrompt: cfg.Prompt,
	}
	sysPrompt, userPrompt := style.BuildPrompt(input)

	fmt.Fprintf(os.Stderr, "cnm: %s style via %s/%s...\n", cfg.Style, cfg.Provider, cfg.Model)

	if *flagDryRun {
		fmt.Println("=== SYSTEM PROMPT ===")
		fmt.Println(sysPrompt)
		fmt.Println()
		fmt.Println("=== USER PROMPT (truncated) ===")
		if len(userPrompt) > 2000 {
			fmt.Println(userPrompt[:2000] + "\n... (truncated)")
		} else {
			fmt.Println(userPrompt)
		}
		return
	}

	req := api.Request{
		Provider:  cfg.Provider,
		APIKey:    cfg.APIKey,
		APIURL:    cfg.APIURL,
		Model:     cfg.Model,
		System:    sysPrompt,
		Prompt:    userPrompt,
		MaxTokens: cfg.MaxTokens,
		Timeout:   60 * time.Second,
	}

	start := time.Now()
	msg, err := req.Send()
	elapsed := time.Since(start)

	if err != nil {
		fmt.Fprintln(os.Stderr, "cnm: API error:", err)
		os.Exit(1)
	}

	msg = cleanMessage(msg)

	fmt.Fprintf(os.Stderr, "cnm: done in %.1fs\n\n", elapsed.Seconds())
	fmt.Println(msg)

	if *flagYes {
		commitNow(msg)
		return
	}

	fmt.Print("\nCommit with this message? [Y/n] ")
	var answer string
	fmt.Scanln(&answer)
	if answer == "" || strings.ToLower(answer)[:1] == "y" {
		commitNow(msg)
	} else {
		fmt.Fprintln(os.Stderr, "cnm: message printed but not committed")
	}
}

func runSetup() {
	path := config.Path()
	fmt.Printf("cnm setup — configure your API and style preferences\n")
	fmt.Printf("Config will be saved to: %s\n\n", path)

	reader := bufio.NewReader(os.Stdin)
	cfg, _ := config.Load()

	// Provider
	fmt.Printf("API provider [%s]:\n", cfg.Provider)
	fmt.Println("  1. openai-chat (OpenAI Chat Completions)")
	fmt.Println("  2. openai-response (OpenAI Responses)")
	fmt.Println("  3. anthropic-message (Anthropic Messages)")
	fmt.Print("Choose (1-3): ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	switch choice {
	case "2":
		cfg.Provider = api.OpenAIResponse
	case "3":
		cfg.Provider = api.AnthropicMessage
	}

	// API Key
	fmt.Printf("\nAPI key [%s]: ", maskKey(cfg.APIKey))
	key, _ := reader.ReadString('\n')
	key = strings.TrimSpace(key)
	if key != "" {
		cfg.APIKey = key
	}

	// API URL
	fmt.Printf("API base URL [%s]: ", cfg.APIURL)
	url, _ := reader.ReadString('\n')
	url = strings.TrimSpace(url)
	if url != "" {
		cfg.APIURL = url
	}

	// Model
	fmt.Printf("Model [%s]: ", cfg.Model)
	model, _ := reader.ReadString('\n')
	model = strings.TrimSpace(model)
	if model != "" {
		cfg.Model = model
	}

	// Style
	fmt.Printf("\nCommit message style [%s]:\n", cfg.Style)
	for i, s := range style.All {
		mark := " "
		if s == cfg.Style {
			mark = "*"
		}
		fmt.Printf("  %s %d. %s\n", mark, i+1, s)
	}
	fmt.Print("Choose (1-7): ")
	choice, _ = reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	if idx := choice; idx != "" {
		switch idx {
		case "1": cfg.Style = style.Auto
		case "2": cfg.Style = style.Conventional
		case "3": cfg.Style = style.Angular
		case "4": cfg.Style = style.Google
		case "5": cfg.Style = style.Atom
		case "6": cfg.Style = style.Plain
		case "7": cfg.Style = style.Custom
		}
	}

	// Custom prompt (only relevant for custom style, but can be used as additional guidance)
	fmt.Printf("\nCustom prompt guidance (optional) [%s]: ", cfg.Prompt)
	prompt, _ := reader.ReadString('\n')
	prompt = strings.TrimSpace(prompt)
	if prompt != "" {
		cfg.Prompt = prompt
	}

	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "\ncnm: failed to save config: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\n✓ Config saved to %s\n", path)
	fmt.Println("Run `cnm` to generate a commit message.")
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

func cleanMessage(msg string) string {
	msg = strings.TrimSpace(msg)
	if strings.HasPrefix(msg, "```") {
		msg = strings.TrimPrefix(msg, "```")
		if idx := strings.Index(msg, "\n"); idx > 0 {
			msg = msg[idx+1:]
		}
		msg = strings.TrimSuffix(msg, "```")
	}
	return strings.TrimSpace(msg)
}

func commitNow(msg string) {
	tmp, err := os.CreateTemp("", "cnm-commit-*.txt")
	if err != nil {
		fmt.Fprintln(os.Stderr, "cnm: temp file error:", err)
		os.Exit(1)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(msg + "\n"); err != nil {
		fmt.Fprintln(os.Stderr, "cnm: write error:", err)
		os.Exit(1)
	}
	tmp.Close()

	cmd := exec.Command("git", "commit", "-F", tmp.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "cnm: commit failed:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "cnm: committed ✓")
}

// dryRun shows the prompt without calling the API
func dryRun(sysPrompt, userPrompt string) {
	fmt.Println("=== SYSTEM PROMPT ===")
	fmt.Println(sysPrompt)
	fmt.Println()
	fmt.Println("=== USER PROMPT ===")
	fmt.Println(userPrompt)
	fmt.Println()
	fmt.Println("=== WOULD SEND TO API ===")
	fmt.Printf("Provider: %s\n", *flagProvider)
	fmt.Printf("Model: %s\n", *flagModel)
}
