package tui

import (
	"fmt"
	"strings"

	"github.com/ByteTrue/commit-now-myfriend/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

type ConfigPanelInput struct {
	Width      int
	Height     int
	NoColor    bool
	Effective  config.EffectiveConfig
	Sources    config.ConfigSourceSummary
	UserPath   string
	WriteValue func(key config.ConfigKey, value string) error
	UnsetValue func(key config.ConfigKey) error
	Reload     func() (config.EffectiveConfig, config.ConfigSourceSummary, error)
}

type ConfigPanelResult struct {
	Cancelled bool
	Saved     int
	LastError string
}

type configPanelScreen int

const (
	configScreenList configPanelScreen = iota
	configScreenEdit
	configScreenChoice
)

type ConfigPanelModel struct {
	width   int
	height  int
	theme   Theme
	input   ConfigPanelInput
	cursor  int
	screen  configPanelScreen
	editKey config.ConfigKey
	editBuf string
	choices []string
	choice  int
	status  string
	saved   int
	cancel  bool
}

func NewConfigPanelModel(input ConfigPanelInput) ConfigPanelModel {
	width := input.Width
	if width <= 0 {
		width = 96
	}
	height := input.Height
	if height <= 0 {
		height = 30
	}
	return ConfigPanelModel{
		width:  width,
		height: height,
		theme:  Theme{NoColor: input.NoColor},
		input:  input,
		screen: configScreenList,
	}
}

func (m ConfigPanelModel) Init() tea.Cmd { return nil }

func (m ConfigPanelModel) Result() ConfigPanelResult {
	return ConfigPanelResult{Cancelled: m.cancel, Saved: m.saved, LastError: m.status}
}

func (m ConfigPanelModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m ConfigPanelModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case configScreenList:
		return m.handleListKey(msg)
	case configScreenEdit:
		return m.handleEditKey(msg)
	case configScreenChoice:
		return m.handleChoiceKey(msg)
	}
	return m, nil
}

func (m ConfigPanelModel) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		m.cancel = true
		return m, tea.Quit
	case tea.KeyRunes:
		s := msg.String()
		switch s {
		case "q":
			m.cancel = true
			return m, tea.Quit
		case "j":
			m.moveCursor(1)
		case "k":
			m.moveCursor(-1)
		case "d":
			return m.startUnset()
		}
	case tea.KeyUp:
		m.moveCursor(-1)
	case tea.KeyDown:
		m.moveCursor(1)
	case tea.KeyEnter:
		return m.startEdit()
	}
	return m, nil
}

func (m *ConfigPanelModel) moveCursor(delta int) {
	keys := config.ConfigKeys
	next := m.cursor + delta
	if next < 0 {
		next = 0
	}
	if next >= len(keys) {
		next = len(keys) - 1
	}
	m.cursor = next
}

func (m ConfigPanelModel) startEdit() (tea.Model, tea.Cmd) {
	key := config.ConfigKeys[m.cursor]
	choices := configChoiceList(key)
	if len(choices) > 0 {
		m.screen = configScreenChoice
		m.editKey = key
		m.choices = choices
		m.choice = 0
		current := stringFromPointer(config.GetConfigValue(m.input.Effective, key))
		for i, c := range choices {
			if c == current {
				m.choice = i
				break
			}
		}
		return m, nil
	}
	m.screen = configScreenEdit
	m.editKey = key
	current := config.GetConfigValue(m.input.Effective, key)
	if current != nil && key != config.ConfigKeyAPIKey {
		m.editBuf = *current
	} else {
		m.editBuf = ""
	}
	return m, nil
}

func (m *ConfigPanelModel) reloadEffective() {
	if m.input.Reload == nil {
		return
	}
	values, sources, err := m.input.Reload()
	if err != nil {
		m.status = "reload error: " + err.Error()
		return
	}
	m.input.Effective = values
	m.input.Sources = sources
}

func (m ConfigPanelModel) startUnset() (tea.Model, tea.Cmd) {
	key := config.ConfigKeys[m.cursor]
	if m.input.UnsetValue == nil {
		m.status = "no unset handler attached"
		return m, nil
	}
	if err := m.input.UnsetValue(key); err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.saved++
	m.status = fmt.Sprintf("removed %s", key)
	m.reloadEffective()
	return m, nil
}

func (m ConfigPanelModel) handleEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		m.cancel = true
		return m, tea.Quit
	case tea.KeyEsc:
		m.screen = configScreenList
		m.editBuf = ""
		return m, nil
	case tea.KeyEnter:
		return m.commitEdit()
	case tea.KeyBackspace, tea.KeyDelete:
		if len(m.editBuf) > 0 {
			runes := []rune(m.editBuf)
			m.editBuf = string(runes[:len(runes)-1])
		}
	case tea.KeySpace:
		m.editBuf += " "
	case tea.KeyRunes:
		m.editBuf += msg.String()
	}
	return m, nil
}

func (m ConfigPanelModel) commitEdit() (tea.Model, tea.Cmd) {
	if m.input.WriteValue == nil {
		m.status = "no write handler attached"
		m.screen = configScreenList
		return m, nil
	}
	value := strings.TrimSpace(m.editBuf)
	if value == "" {
		m.status = "value is empty (use d to unset)"
		m.screen = configScreenList
		return m, nil
	}
	if err := m.input.WriteValue(m.editKey, value); err != nil {
		m.status = err.Error()
		m.screen = configScreenList
		return m, nil
	}
	m.saved++
	m.status = fmt.Sprintf("saved %s", m.editKey)
	m.editBuf = ""
	m.screen = configScreenList
	m.reloadEffective()
	return m, nil
}

func (m ConfigPanelModel) handleChoiceKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		m.cancel = true
		return m, tea.Quit
	case tea.KeyEsc:
		m.screen = configScreenList
		return m, nil
	case tea.KeyEnter:
		if m.input.WriteValue == nil {
			m.status = "no write handler attached"
		} else if err := m.input.WriteValue(m.editKey, m.choices[m.choice]); err != nil {
			m.status = err.Error()
		} else {
			m.saved++
			m.status = fmt.Sprintf("saved %s = %s", m.editKey, m.choices[m.choice])
			m.reloadEffective()
		}
		m.screen = configScreenList
		return m, nil
	case tea.KeyUp:
		if m.choice > 0 {
			m.choice--
		}
	case tea.KeyDown:
		if m.choice < len(m.choices)-1 {
			m.choice++
		}
	case tea.KeyRunes:
		switch msg.String() {
		case "j":
			if m.choice < len(m.choices)-1 {
				m.choice++
			}
		case "k":
			if m.choice > 0 {
				m.choice--
			}
		case "q":
			m.screen = configScreenList
		}
	}
	return m, nil
}

func (m ConfigPanelModel) View() string {
	if m.input.NoColor || m.width < 80 {
		return m.viewPlain()
	}
	return m.viewRich()
}

func (m ConfigPanelModel) viewRich() string {
	header := m.theme.HeaderBar(m.width).Render(
		m.theme.Title().Render("cnm") + "  " + m.theme.Mode().Render("Configuration") +
			"  " + m.theme.Subtle().Render(m.input.UserPath),
	)
	body := m.renderListPanel(m.width)
	overlay := ""
	switch m.screen {
	case configScreenEdit:
		overlay = m.renderEditPanel(m.width)
	case configScreenChoice:
		overlay = m.renderChoicePanel(m.width)
	}
	footer := m.renderFooter()
	parts := []string{header, body}
	if overlay != "" {
		parts = append(parts, overlay)
	}
	if m.status != "" {
		parts = append(parts, m.theme.Subtle().Render(fitLine(m.status, m.width-2)))
	}
	parts = append(parts, footer)
	return strings.Join(parts, "\n")
}

func (m ConfigPanelModel) renderListPanel(width int) string {
	lines := []string{m.theme.Section().Render("Preferences")}
	for index, key := range config.ConfigKeys {
		value := stringFromPointer(config.GetConfigValue(m.input.Effective, key))
		display := value
		if key == config.ConfigKeyAPIKey && value != "" {
			display = "[redacted]"
		}
		if display == "" {
			display = "(unset)"
		}
		source := m.sourceFor(key)
		active := index == m.cursor && m.screen == configScreenList
		cursor := m.theme.CursorMarker(active)
		keyLabel := fmt.Sprintf("%-22s", string(key))
		if active {
			keyLabel = m.theme.Accent().Render(keyLabel)
		} else {
			keyLabel = m.theme.Text().Render(keyLabel)
		}
		valueText := m.theme.Subtle().Render(display)
		sourceTag := ""
		if source != "" {
			sourceTag = " " + m.theme.Muted().Render("from "+source)
		}
		row := cursor + " " + keyLabel + " " + valueText + sourceTag
		lines = append(lines, fitLine(row, width-4))
	}
	return m.theme.Panel(width).Render(strings.Join(lines, "\n"))
}

func (m ConfigPanelModel) renderEditPanel(width int) string {
	heading := m.theme.Section().Render(fmt.Sprintf("Edit %s", m.editKey))
	caret := m.theme.Accent().Render("│")
	body := m.editBuf + caret
	hint := m.theme.Muted().Render("Enter saves. Esc cancels. Ctrl-C quits.")
	return m.theme.Panel(width).Render(heading + "\n" + body + "\n" + hint)
}

func (m ConfigPanelModel) renderChoicePanel(width int) string {
	heading := m.theme.Section().Render(fmt.Sprintf("Choose %s", m.editKey))
	lines := []string{heading}
	for index, choice := range m.choices {
		active := index == m.choice
		marker := m.theme.CursorMarker(active)
		text := choice
		if active {
			text = m.theme.Accent().Render(text)
		} else {
			text = m.theme.Text().Render(text)
		}
		lines = append(lines, marker+" "+text)
	}
	lines = append(lines, m.theme.Muted().Render("↑/↓ to choose. Enter saves. Esc cancels."))
	return m.theme.Panel(width).Render(strings.Join(lines, "\n"))
}

func (m ConfigPanelModel) renderFooter() string {
	keys := []keyHint{
		{"↑/↓", "navigate"},
		{"enter", "edit"},
		{"d", "unset"},
		{"esc", "back"},
		{"q", "quit"},
	}
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, m.theme.KeyHint().Render(k.Key)+" "+m.theme.KeyDesc().Render(k.Desc))
	}
	bar := strings.Join(parts, m.theme.Subtle().Render("  ·  "))
	return m.theme.FooterBar(m.width).Render(fitLine(bar, m.width-2))
}

func (m ConfigPanelModel) sourceFor(key config.ConfigKey) string {
	switch key {
	case config.ConfigKeyProvider:
		return string(m.input.Sources.Provider)
	case config.ConfigKeyModel:
		return string(m.input.Sources.Model)
	case config.ConfigKeyPromptStyle:
		return string(m.input.Sources.PromptStyle)
	case config.ConfigKeyMessageLanguage:
		return string(m.input.Sources.MessageLanguage)
	case config.ConfigKeyStandingInstruction:
		return m.input.Sources.StandingInstruction
	case config.ConfigKeyAPIKey:
		return string(m.input.Sources.APIKey)
	}
	return ""
}

func (m ConfigPanelModel) viewPlain() string {
	lines := []string{"cnm Configuration"}
	lines = append(lines, "user config: "+m.input.UserPath)
	for index, key := range config.ConfigKeys {
		value := stringFromPointer(config.GetConfigValue(m.input.Effective, key))
		display := value
		if key == config.ConfigKeyAPIKey && value != "" {
			display = "[redacted]"
		}
		if display == "" {
			display = "(unset)"
		}
		cursor := " "
		if index == m.cursor {
			cursor = ">"
		}
		lines = append(lines, fmt.Sprintf("%s %-22s %s", cursor, string(key), display))
	}
	if m.status != "" {
		lines = append(lines, m.status)
	}
	lines = append(lines, "enter edit  d unset  q quit")
	return strings.Join(lines, "\n") + "\n"
}

func configChoiceList(key config.ConfigKey) []string {
	switch key {
	case config.ConfigKeyProvider, config.ConfigKeyRecommendedProvider:
		return []string{"openai-responses", "openai-compatible", "anthropic-messages", "google-gemini"}
	case config.ConfigKeyPromptStyle:
		return []string{"auto", "conventional", "angular", "google", "atom", "plain", "custom"}
	case config.ConfigKeyMessageLanguage:
		return []string{"auto", "en", "zh-CN", "zh-TW"}
	}
	return nil
}

func stringFromPointer(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func RunConfigPanel(input ConfigPanelInput, runtime Runtime) (ConfigPanelResult, error) {
	if runtime.NoColor {
		input.NoColor = true
	}
	model := NewConfigPanelModel(input)
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
		return ConfigPanelResult{}, err
	}
	if resultModel, ok := finalModel.(ConfigPanelModel); ok {
		return resultModel.Result(), nil
	}
	return ConfigPanelResult{}, nil
}
