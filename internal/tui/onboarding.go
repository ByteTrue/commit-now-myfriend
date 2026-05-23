package tui

import (
	"fmt"
	"strings"

	"github.com/ByteTrue/commit-now-myfriend/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

type OnboardingInput struct {
	Width           int
	Height          int
	NoColor         bool
	CurrentProvider config.ProviderType
	CurrentModel    string
	CurrentBaseURL  string
	CurrentStyle    config.PromptStyle
	CurrentLanguage config.MessageLanguage
	CurrentStanding string
}

type OnboardingResult struct {
	Cancelled           bool
	Provider            config.ProviderType
	Model               string
	BaseURL             string
	PromptStyle         config.PromptStyle
	MessageLanguage     config.MessageLanguage
	StandingInstruction string
	APIKey              string
}

type OnboardingModel struct {
	width   int
	height  int
	theme   Theme
	step    int
	steps   []onboardStep
	answers map[string]string
	cursor  int
	choices []string
	buf     string
	noColor bool
	cancel  bool
	done    bool
	input   OnboardingInput
}

type onboardStepKind int

const (
	stepKindChoice onboardStepKind = iota
	stepKindText
	stepKindSecret
)

type onboardStep struct {
	id          string
	title       string
	hint        string
	kind        onboardStepKind
	choices     []string
	defaultText string
	required    bool
	skipUnless  func(answers map[string]string) bool
}

func defaultOnboardingSteps(input OnboardingInput) []onboardStep {
	provider := string(input.CurrentProvider)
	if provider == "" {
		provider = string(config.DefaultProvider)
	}
	model := input.CurrentModel
	if model == "" {
		model = config.GetDefaultModel(config.ProviderType(provider))
	}
	style := string(input.CurrentStyle)
	if style == "" {
		style = string(config.DefaultPromptStyle)
	}
	language := string(input.CurrentLanguage)
	if language == "" {
		language = string(config.DefaultMessageLanguage)
	}
	return []onboardStep{
		{
			id:      "provider",
			title:   "Choose AI provider",
			hint:    "Pick the provider that will run cnm's Tool Call Loop.",
			kind:    stepKindChoice,
			choices: []string{"openai-responses", "openai-compatible", "anthropic-messages", "google-gemini"},
		},
		{
			id:          "model",
			title:       "Choose model",
			hint:        "Type a model name. Press enter to keep the default.",
			kind:        stepKindText,
			defaultText: model,
			required:    true,
		},
		{
			id:          "baseURL",
			title:       "OpenAI-compatible base URL",
			hint:        "Required only for openai-compatible providers.",
			kind:        stepKindText,
			defaultText: input.CurrentBaseURL,
			required:    true,
			skipUnless: func(answers map[string]string) bool {
				return answers["provider"] == "openai-compatible"
			},
		},
		{
			id:      "promptStyle",
			title:   "Commit message style",
			hint:    "Pick the convention cnm should follow when drafting messages.",
			kind:    stepKindChoice,
			choices: []string{"auto", "conventional", "angular", "google", "atom", "plain", "custom"},
		},
		{
			id:      "messageLanguage",
			title:   "Commit message language",
			hint:    "auto detects the repo's language; explicit values force it.",
			kind:    stepKindChoice,
			choices: []string{"auto", "en", "zh-CN", "zh-TW"},
		},
		{
			id:          "standingInstruction",
			title:       "Standing Instruction (optional)",
			hint:        "Long-lived guidance for every cnm run. Leave empty to skip.",
			kind:        stepKindText,
			defaultText: input.CurrentStanding,
			required:    false,
		},
		{
			id:       "apiKey",
			title:    "API key",
			hint:     "Stored in the Secret Store. Type your key and press enter.",
			kind:     stepKindSecret,
			required: true,
		},
	}
}

func NewOnboardingModel(input OnboardingInput) *OnboardingModel {
	width := input.Width
	if width <= 0 {
		width = 96
	}
	height := input.Height
	if height <= 0 {
		height = 30
	}
	steps := defaultOnboardingSteps(input)
	answers := map[string]string{
		"provider":            string(input.CurrentProvider),
		"model":               input.CurrentModel,
		"baseURL":             input.CurrentBaseURL,
		"promptStyle":         string(input.CurrentStyle),
		"messageLanguage":     string(input.CurrentLanguage),
		"standingInstruction": input.CurrentStanding,
	}
	if answers["provider"] == "" {
		answers["provider"] = string(config.DefaultProvider)
	}
	if answers["promptStyle"] == "" {
		answers["promptStyle"] = string(config.DefaultPromptStyle)
	}
	if answers["messageLanguage"] == "" {
		answers["messageLanguage"] = string(config.DefaultMessageLanguage)
	}
	model := &OnboardingModel{
		width:   width,
		height:  height,
		theme:   Theme{NoColor: input.NoColor},
		noColor: input.NoColor,
		input:   input,
		steps:   steps,
		answers: answers,
	}
	model.advanceTo(0)
	return model
}

func (m *OnboardingModel) advanceTo(step int) {
	for step < len(m.steps) && m.steps[step].skipUnless != nil && !m.steps[step].skipUnless(m.answers) {
		step++
	}
	if step >= len(m.steps) {
		m.done = true
		return
	}
	m.step = step
	current := m.steps[step]
	switch current.kind {
	case stepKindChoice:
		m.choices = current.choices
		m.cursor = 0
		for i, c := range current.choices {
			if c == m.answers[current.id] {
				m.cursor = i
				break
			}
		}
		m.buf = ""
	case stepKindText:
		if existing, ok := m.answers[current.id]; ok && existing != "" {
			m.buf = existing
		} else {
			m.buf = current.defaultText
		}
	case stepKindSecret:
		m.buf = ""
	}
}

func (m *OnboardingModel) Init() tea.Cmd { return nil }

func (m *OnboardingModel) Result() OnboardingResult {
	if m.cancel {
		return OnboardingResult{Cancelled: true}
	}
	provider := config.ProviderType(m.answers["provider"])
	style := config.PromptStyle(m.answers["promptStyle"])
	language := config.MessageLanguage(m.answers["messageLanguage"])
	return OnboardingResult{
		Provider:            provider,
		Model:               m.answers["model"],
		BaseURL:             m.answers["baseURL"],
		PromptStyle:         style,
		MessageLanguage:     language,
		StandingInstruction: m.answers["standingInstruction"],
		APIKey:              m.answers["apiKey"],
	}
}

func (m *OnboardingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			m.width = msg.Width
		}
		if msg.Height > 0 {
			m.height = msg.Height
		}
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *OnboardingModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.done {
		return m, tea.Quit
	}
	step := m.steps[m.step]
	switch msg.Type {
	case tea.KeyCtrlC:
		m.cancel = true
		return m, tea.Quit
	case tea.KeyEsc:
		if m.step > 0 {
			m.step--
			m.advanceTo(m.step)
		}
		return m, nil
	}
	switch step.kind {
	case stepKindChoice:
		return m.handleChoiceKey(msg)
	case stepKindText, stepKindSecret:
		return m.handleTextKey(msg, step)
	}
	return m, nil
}

func (m *OnboardingModel) handleChoiceKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
	case tea.KeyDown:
		if m.cursor < len(m.choices)-1 {
			m.cursor++
		}
	case tea.KeyRunes:
		switch msg.String() {
		case "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "k":
			if m.cursor > 0 {
				m.cursor--
			}
		}
	case tea.KeyEnter:
		current := m.steps[m.step]
		m.answers[current.id] = m.choices[m.cursor]
		next := m.step + 1
		m.advanceTo(next)
		if m.done {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *OnboardingModel) handleTextKey(msg tea.KeyMsg, step onboardStep) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		value := strings.TrimSpace(m.buf)
		if value == "" && step.required {
			if step.kind == stepKindSecret {
				return m, nil
			}
			value = step.defaultText
		}
		m.answers[step.id] = value
		next := m.step + 1
		m.advanceTo(next)
		if m.done {
			return m, tea.Quit
		}
		return m, nil
	case tea.KeyBackspace, tea.KeyDelete:
		if len(m.buf) > 0 {
			runes := []rune(m.buf)
			m.buf = string(runes[:len(runes)-1])
		}
	case tea.KeySpace:
		m.buf += " "
	case tea.KeyRunes:
		m.buf += msg.String()
	}
	return m, nil
}

func (m *OnboardingModel) View() string {
	if m.noColor || m.width < 80 {
		return m.viewPlain()
	}
	return m.viewRich()
}

func (m *OnboardingModel) viewRich() string {
	header := m.theme.HeaderBar(m.width).Render(
		m.theme.Title().Render("cnm") + "  " + m.theme.Mode().Render("Onboarding") +
			"  " + m.theme.Subtle().Render(m.progressLabel()),
	)
	footer := m.renderFooter()
	if m.done {
		body := m.theme.Panel(m.width).Render(m.theme.Section().Render("All set") + "\n" + m.theme.Text().Render("Saving configuration…"))
		return strings.Join([]string{header, body, footer}, "\n")
	}
	step := m.steps[m.step]
	heading := m.theme.Section().Render(step.title)
	hint := m.theme.Muted().Render(step.hint)
	body := ""
	switch step.kind {
	case stepKindChoice:
		lines := []string{heading, hint}
		for index, choice := range m.choices {
			active := index == m.cursor
			marker := m.theme.CursorMarker(active)
			text := choice
			if active {
				text = m.theme.Accent().Render(text)
			} else {
				text = m.theme.Text().Render(text)
			}
			lines = append(lines, marker+" "+text)
		}
		body = strings.Join(lines, "\n")
	case stepKindText:
		caret := m.theme.Accent().Render("│")
		preview := m.buf + caret
		body = strings.Join([]string{heading, hint, preview}, "\n")
	case stepKindSecret:
		caret := m.theme.Accent().Render("│")
		masked := strings.Repeat("•", len(m.buf)) + caret
		body = strings.Join([]string{heading, hint, masked}, "\n")
	}
	panel := m.theme.Panel(m.width).Render(body)
	return strings.Join([]string{header, panel, footer}, "\n")
}

func (m *OnboardingModel) renderFooter() string {
	keys := []keyHint{}
	step := m.currentStep()
	switch step.kind {
	case stepKindChoice:
		keys = append(keys, keyHint{"↑/↓", "navigate"}, keyHint{"enter", "select"})
	default:
		keys = append(keys, keyHint{"type", "answer"}, keyHint{"enter", "continue"})
	}
	keys = append(keys, keyHint{"esc", "back"}, keyHint{"ctrl-c", "cancel"})
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, m.theme.KeyHint().Render(k.Key)+" "+m.theme.KeyDesc().Render(k.Desc))
	}
	bar := strings.Join(parts, m.theme.Subtle().Render("  ·  "))
	return m.theme.FooterBar(m.width).Render(fitLine(bar, m.width-2))
}

func (m *OnboardingModel) currentStep() onboardStep {
	if m.step < 0 || m.step >= len(m.steps) {
		return onboardStep{}
	}
	return m.steps[m.step]
}

func (m *OnboardingModel) progressLabel() string {
	return fmt.Sprintf("[step %d/%d]", m.step+1, len(m.steps))
}

func (m *OnboardingModel) viewPlain() string {
	if m.done {
		return "Onboarding done. Saving configuration…\n"
	}
	step := m.steps[m.step]
	lines := []string{
		fmt.Sprintf("cnm Onboarding %s", m.progressLabel()),
		step.title,
		step.hint,
	}
	switch step.kind {
	case stepKindChoice:
		for index, choice := range m.choices {
			cursor := "  "
			if index == m.cursor {
				cursor = "> "
			}
			lines = append(lines, cursor+choice)
		}
		lines = append(lines, "↑/↓ navigate  enter select  esc back  ctrl-c cancel")
	case stepKindText:
		lines = append(lines, m.buf+"|")
		lines = append(lines, "type answer  enter continue  esc back  ctrl-c cancel")
	case stepKindSecret:
		lines = append(lines, strings.Repeat("•", len(m.buf))+"|")
		lines = append(lines, "type answer  enter continue  esc back  ctrl-c cancel")
	}
	return strings.Join(lines, "\n") + "\n"
}

func RunOnboarding(input OnboardingInput, runtime Runtime) (OnboardingResult, error) {
	if runtime.NoColor {
		input.NoColor = true
	}
	model := NewOnboardingModel(input)
	options := []tea.ProgramOption{}
	if runtime.Input != nil {
		options = append(options, tea.WithInput(runtime.Input))
	}
	if runtime.Output != nil {
		options = append(options, tea.WithOutput(runtime.Output))
	}
	if runtime.NoColor {
		options = append(options, tea.WithoutRenderer(), tea.WithoutBracketedPaste())
	} else if !runtime.Headless {
		options = append(options, tea.WithAltScreen(), tea.WithMouseCellMotion())
	}
	program := tea.NewProgram(model, options...)
	finalModel, err := program.Run()
	if err != nil {
		return OnboardingResult{}, err
	}
	if resultModel, ok := finalModel.(*OnboardingModel); ok {
		return resultModel.Result(), nil
	}
	return OnboardingResult{}, nil
}
