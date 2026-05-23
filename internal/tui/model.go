package tui

import (
	"fmt"
	"strconv"
	"strings"

	gitpkg "github.com/ByteTrue/commit-now-myfriend/internal/git"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type Screen string

const (
	ScreenScopeReview      Screen = "scope_review"
	ScreenAgentInstruction Screen = "agent_instruction"
	ScreenAIActivity       Screen = "ai_activity"
	ScreenMessageReview    Screen = "message_review"
	ScreenMessageEdit      Screen = "message_edit"
	ScreenRepairReview     Screen = "repair_review"
)

type ModelInput struct {
	Width         int
	Height        int
	NoColor       bool
	RepairContext *RepairContext
	Scope         gitpkg.CommitScope
	CommitPlan    CommitPlanView
	PlanCommits   PlanCommitsFunc
	DiffProvider  DiffProvider
	FileReader    FileReader
}

type DiffProvider func(path string) (string, error)

type FileReader func(path string) (string, error)

type PlanCommitsFunc func(input PlanCommitsInput) (CommitPlanView, error)

type PlanCommitsInput struct {
	Scope            gitpkg.CommitScope
	ScopeFiles       []string
	AgentInstruction string
}

type RepairContext struct {
	Reason        string
	EligibleFiles []string
}

type CommitPlanView struct {
	Kind    string
	Commits []CommitView
}

type CommitView struct {
	Message string
	Files   []string
}

type AIActivityDoneMsg struct {
	Plan CommitPlanView
	Err  error
}

type Result struct {
	Accepted   bool
	Cancelled  bool
	CommitPlan CommitPlanView
	ScopeFiles []string
}

type Model struct {
	Width            int
	Height           int
	NoColor          bool
	RepairContext    *RepairContext
	Screen           Screen
	Scope            gitpkg.CommitScope
	ScopeCursor      int
	ScopeIncluded    map[string]bool
	CommitPlan       CommitPlanView
	PlanCommits      PlanCommitsFunc
	DiffProvider     DiffProvider
	FileReader       FileReader
	AgentInstruction string
	AIActivityError  string
	MessageDraft     string
	Accepted         bool
	Cancelled        bool

	spinner    spinner.Model
	preview    viewport.Model
	previewSet bool
	theme      Theme
}

func NewModel(input ModelInput) Model {
	width := input.Width
	if width <= 0 {
		width = 96
	}
	height := input.Height
	if height <= 0 {
		height = 30
	}
	screen := ScreenScopeReview
	if input.RepairContext != nil {
		screen = ScreenRepairReview
	}
	included := map[string]bool{}
	for _, file := range input.Scope.Files {
		included[file.Path] = true
	}

	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	if !input.NoColor {
		sp.Style = lipgloss.NewStyle().Foreground(colorAccentBright).Bold(true)
	}

	vp := viewport.New(40, 10)

	return Model{
		Width:         width,
		Height:        height,
		NoColor:       input.NoColor,
		RepairContext: input.RepairContext,
		Screen:        screen,
		Scope:         input.Scope,
		ScopeIncluded: included,
		CommitPlan:    input.CommitPlan,
		PlanCommits:   input.PlanCommits,
		DiffProvider:  input.DiffProvider,
		FileReader:    input.FileReader,
		spinner:       sp,
		preview:       vp,
		theme:         Theme{NoColor: input.NoColor},
	}
}

func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m Model) Result() Result {
	return Result{
		Accepted:   m.Accepted,
		Cancelled:  m.Cancelled,
		CommitPlan: m.CommitPlan,
		ScopeFiles: m.selectedScopeFilePaths(),
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case AIActivityDoneMsg:
		if msg.Err != nil {
			m.AIActivityError = msg.Err.Error()
			m.Screen = ScreenAgentInstruction
			return m, nil
		}
		m.AIActivityError = ""
		m.CommitPlan = msg.Plan
		m.Screen = ScreenMessageReview
		m.refreshPreview()
		return m, nil
	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			m.Width = msg.Width
		}
		if msg.Height > 0 {
			m.Height = msg.Height
		}
		m.refreshPreview()
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		m.Cancelled = true
		return m, tea.Quit
	case tea.KeyRunes:
		s := msg.String()
		if s == "q" && m.Screen != ScreenAgentInstruction && m.Screen != ScreenMessageEdit {
			m.Cancelled = true
			return m, tea.Quit
		}
		if m.Screen == ScreenScopeReview {
			switch s {
			case "j":
				m.moveScopeCursor(1)
				return m, nil
			case "k":
				m.moveScopeCursor(-1)
				return m, nil
			case "a":
				m.toggleAllScope(true)
				return m, nil
			case "n":
				m.toggleAllScope(false)
				return m, nil
			}
		}
		if m.Screen == ScreenMessageReview && s == "e" && len(m.CommitPlan.Commits) > 0 {
			m.MessageDraft = m.CommitPlan.Commits[0].Message
			m.Screen = ScreenMessageEdit
			return m, nil
		}
		if m.Screen == ScreenAgentInstruction {
			m.AgentInstruction += s
		}
		if m.Screen == ScreenMessageEdit {
			m.MessageDraft += s
		}
	case tea.KeySpace:
		if m.Screen == ScreenScopeReview {
			m.toggleCurrentScopeFile()
			return m, nil
		}
		if m.Screen == ScreenAgentInstruction {
			m.AgentInstruction += " "
		}
		if m.Screen == ScreenMessageEdit {
			m.MessageDraft += " "
		}
	case tea.KeyEnter:
		switch m.Screen {
		case ScreenRepairReview:
			m.Accepted = true
			return m, tea.Quit
		case ScreenScopeReview:
			if m.selectedScopeFileCount() == 0 {
				return m, nil
			}
			m.CommitPlan = filterCommitPlanByIncluded(m.CommitPlan, m.ScopeIncluded)
			m.Screen = ScreenAgentInstruction
			m.refreshPreview()
		case ScreenAgentInstruction:
			m.Screen = ScreenAIActivity
			return m, planAIActivity(m.PlanCommits, PlanCommitsInput{Scope: m.Scope, ScopeFiles: m.selectedScopeFilePaths(), AgentInstruction: m.AgentInstruction}, m.CommitPlan)
		case ScreenMessageEdit:
			if len(m.CommitPlan.Commits) > 0 && strings.TrimSpace(m.MessageDraft) != "" {
				m.CommitPlan.Commits[0].Message = strings.TrimSpace(m.MessageDraft)
			}
			m.Screen = ScreenMessageReview
			m.refreshPreview()
		case ScreenMessageReview:
			m.Accepted = true
			return m, tea.Quit
		}
	case tea.KeyBackspace, tea.KeyDelete:
		if m.Screen == ScreenAgentInstruction && len(m.AgentInstruction) > 0 {
			runes := []rune(m.AgentInstruction)
			m.AgentInstruction = string(runes[:len(runes)-1])
		}
		if m.Screen == ScreenMessageEdit && len(m.MessageDraft) > 0 {
			runes := []rune(m.MessageDraft)
			m.MessageDraft = string(runes[:len(runes)-1])
		}
	case tea.KeyUp:
		if m.Screen == ScreenScopeReview {
			m.moveScopeCursor(-1)
			return m, nil
		}
		if m.previewSet {
			m.preview.LineUp(1)
			return m, nil
		}
	case tea.KeyDown:
		if m.Screen == ScreenScopeReview {
			m.moveScopeCursor(1)
			return m, nil
		}
		if m.previewSet {
			m.preview.LineDown(1)
			return m, nil
		}
	case tea.KeyPgUp:
		if m.previewSet {
			m.preview.HalfViewUp()
		}
	case tea.KeyPgDown:
		if m.previewSet {
			m.preview.HalfViewDown()
		}
	}
	return m, nil
}

func (m *Model) moveScopeCursor(delta int) {
	if len(m.Scope.Files) == 0 {
		m.ScopeCursor = 0
		return
	}
	next := m.ScopeCursor + delta
	if next < 0 {
		next = 0
	}
	if next >= len(m.Scope.Files) {
		next = len(m.Scope.Files) - 1
	}
	m.ScopeCursor = next
	m.refreshPreview()
}

func (m *Model) toggleCurrentScopeFile() {
	if len(m.Scope.Files) == 0 || m.ScopeCursor < 0 || m.ScopeCursor >= len(m.Scope.Files) {
		return
	}
	if m.ScopeIncluded == nil {
		m.ScopeIncluded = map[string]bool{}
		for _, file := range m.Scope.Files {
			m.ScopeIncluded[file.Path] = true
		}
	}
	path := m.Scope.Files[m.ScopeCursor].Path
	m.ScopeIncluded[path] = !m.scopeFileIncluded(path)
	m.CommitPlan = filterCommitPlanByIncluded(m.CommitPlan, m.ScopeIncluded)
}

func (m *Model) toggleAllScope(include bool) {
	if m.ScopeIncluded == nil {
		m.ScopeIncluded = map[string]bool{}
	}
	for _, file := range m.Scope.Files {
		m.ScopeIncluded[file.Path] = include
	}
	m.CommitPlan = filterCommitPlanByIncluded(m.CommitPlan, m.ScopeIncluded)
}

func (m Model) scopeFileIncluded(path string) bool {
	if m.ScopeIncluded == nil {
		return true
	}
	included, ok := m.ScopeIncluded[path]
	return !ok || included
}

func (m Model) selectedScopeFileCount() int {
	count := 0
	for _, file := range m.Scope.Files {
		if m.scopeFileIncluded(file.Path) {
			count++
		}
	}
	return count
}

func (m Model) selectedScopeFilePaths() []string {
	paths := make([]string, 0)
	for _, file := range m.Scope.Files {
		if m.scopeFileIncluded(file.Path) {
			paths = append(paths, file.Path)
		}
	}
	return paths
}

func filterCommitPlanByIncluded(plan CommitPlanView, included map[string]bool) CommitPlanView {
	if included == nil || len(plan.Commits) == 0 {
		return plan
	}
	next := CommitPlanView{Kind: plan.Kind}
	for _, commit := range plan.Commits {
		files := make([]string, 0, len(commit.Files))
		for _, file := range commit.Files {
			if selected, ok := included[file]; !ok || selected {
				files = append(files, file)
			}
		}
		if len(files) == 0 {
			continue
		}
		next.Commits = append(next.Commits, CommitView{Message: commit.Message, Files: files})
	}
	if len(next.Commits) == 1 && next.Kind == "file_split" {
		next.Kind = "single"
	}
	return next
}

func planAIActivity(planner PlanCommitsFunc, input PlanCommitsInput, fallback CommitPlanView) tea.Cmd {
	return func() tea.Msg {
		if planner == nil {
			return AIActivityDoneMsg{Plan: fallback}
		}
		plan, err := planner(input)
		return AIActivityDoneMsg{Plan: plan, Err: err}
	}
}

func (m *Model) refreshPreview() {
	previewWidth, previewHeight := m.previewDims()
	m.preview.Width = previewWidth
	m.preview.Height = previewHeight
	m.preview.SetContent(m.previewContent())
	m.previewSet = true
}

func (m Model) previewDims() (int, int) {
	if !m.useTwoColumn() {
		return m.contentWidth(), m.bodyHeight() / 2
	}
	left, right := m.columnWidths()
	_ = left
	return right - 4, m.bodyHeight() - 4
}

func (m Model) previewContent() string {
	if m.RepairContext != nil {
		return m.repairPreview()
	}
	switch m.Screen {
	case ScreenScopeReview, ScreenAgentInstruction:
		return m.diffPreview()
	case ScreenAIActivity:
		return m.theme.Muted().Render("Planning commits with local Domain Tools…")
	case ScreenMessageReview, ScreenMessageEdit:
		return m.commitPlanPreview()
	}
	return ""
}

func (m Model) diffPreview() string {
	if len(m.Scope.Files) == 0 {
		return m.theme.Muted().Render("No selected changes")
	}
	if m.ScopeCursor < 0 || m.ScopeCursor >= len(m.Scope.Files) {
		return ""
	}
	file := m.Scope.Files[m.ScopeCursor]
	header := m.theme.Section().Render(file.Path)
	if m.DiffProvider == nil {
		statusLine := m.fileStatus(file)
		return joinNonEmpty([]string{header, m.theme.Subtle().Render(statusLine), m.theme.Muted().Render("(no diff provider attached)")})
	}
	content, err := m.DiffProvider(file.Path)
	if err != nil {
		return joinNonEmpty([]string{header, m.theme.Error().Render("diff error: " + err.Error())})
	}
	if strings.TrimSpace(content) == "" {
		statusLine := m.fileStatus(file)
		return joinNonEmpty([]string{header, m.theme.Subtle().Render(statusLine), m.theme.Muted().Render("(no diff yet)")})
	}
	return joinNonEmpty([]string{header, m.colorizeDiff(content)})
}

func (m Model) repairPreview() string {
	if m.RepairContext == nil {
		return ""
	}
	lines := []string{m.theme.Section().Render("Conflict context")}
	if strings.TrimSpace(m.RepairContext.Reason) != "" {
		lines = append(lines, m.theme.Text().Render(m.RepairContext.Reason))
	}
	if len(m.RepairContext.EligibleFiles) == 0 {
		return joinNonEmpty(lines)
	}
	if m.FileReader == nil {
		lines = append(lines, m.theme.Muted().Render("Eligible files:"))
		for _, file := range m.RepairContext.EligibleFiles {
			lines = append(lines, "  "+file)
		}
		return joinNonEmpty(lines)
	}
	for _, path := range m.RepairContext.EligibleFiles {
		lines = append(lines, m.theme.Section().Render(path))
		body, err := m.FileReader(path)
		if err != nil {
			lines = append(lines, m.theme.Error().Render("read error: "+err.Error()))
			continue
		}
		lines = append(lines, m.colorizeConflict(body))
	}
	return joinNonEmpty(lines)
}

func (m Model) commitPlanPreview() string {
	if len(m.CommitPlan.Commits) == 0 {
		return m.theme.Muted().Render("No plan yet")
	}
	out := []string{m.theme.Section().Render(fmt.Sprintf("mode %s  %d commit(s)", emptyDefault(m.CommitPlan.Kind, "single"), len(m.CommitPlan.Commits)))}
	for index, commit := range m.CommitPlan.Commits {
		out = append(out, m.theme.Accent().Render(fmt.Sprintf("%d. %s", index+1, commit.Message)))
		for _, file := range commit.Files {
			out = append(out, m.theme.Subtle().Render("    "+file))
		}
	}
	return strings.Join(out, "\n")
}

func (m Model) colorizeDiff(diff string) string {
	lines := strings.Split(diff, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") || strings.HasPrefix(line, "@@") || strings.HasPrefix(line, "diff "):
			out = append(out, m.theme.DiffMeta().Render(line))
		case strings.HasPrefix(line, "+"):
			out = append(out, m.theme.DiffAdd().Render(line))
		case strings.HasPrefix(line, "-"):
			out = append(out, m.theme.DiffDel().Render(line))
		default:
			out = append(out, m.theme.Text().Render(line))
		}
	}
	return strings.Join(out, "\n")
}

func (m Model) colorizeConflict(body string) string {
	lines := strings.Split(body, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "<<<<<<<") || strings.HasPrefix(line, ">>>>>>>"):
			out = append(out, m.theme.DiffMeta().Render(line))
		case strings.HasPrefix(line, "======="):
			out = append(out, m.theme.DiffMeta().Render(line))
		default:
			out = append(out, m.theme.Text().Render(line))
		}
	}
	return strings.Join(out, "\n")
}

func (m Model) renderAIErrorBlock(raw string, width int) []string {
	summary, detail := humanizeAIError(raw)
	lines := []string{m.theme.Error().Render("⚠ AI activity failed")}
	for _, l := range wordWrap(summary, width) {
		lines = append(lines, m.theme.Text().Render(l))
	}
	if detail != "" {
		paragraphs := strings.Split(detail, "\n\n")
		for _, paragraph := range paragraphs {
			lines = append(lines, "")
			isRawSection := strings.HasPrefix(paragraph, "Raw error:")
			for _, l := range wordWrap(paragraph, width) {
				if isRawSection {
					lines = append(lines, m.theme.Subtle().Render(l))
				} else {
					lines = append(lines, m.theme.Text().Render(l))
				}
			}
		}
	}
	lines = append(lines, "")
	for _, l := range wordWrap("Press q to exit. Try `cnm doctor` to verify provider, model, and API key.", width) {
		lines = append(lines, m.theme.Muted().Render(l))
	}
	return lines
}

func humanizeAIError(raw string) (summary, detail string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "Unknown error from AI provider.", ""
	}
	lower := strings.ToLower(raw)
	hint := ""
	switch {
	case strings.Contains(raw, "provider response parse error") && (strings.Contains(raw, "invalid character '<'") || strings.Contains(lower, "<!doctype") || strings.Contains(lower, "<html")):
		summary = "The provider returned HTML instead of JSON."
		hint = "Your base URL is likely wrong, or a proxy in front of the provider returned an error page. Run `cnm doctor`, then `cnm config` to check `baseURL`."
	case strings.Contains(raw, "provider response parse error"):
		summary = "The provider returned a response that cnm could not parse."
		hint = "This usually means the provider responded with an error page or the wrong API shape (chat-completions vs responses vs anthropic-messages). Verify provider and base URL."
	case strings.Contains(lower, "no tool calls"):
		summary = "The AI provider stopped without calling a tool."
		hint = "This usually means the provider, model, or system prompt rejects native tool calls. Run `cnm doctor` and verify your provider supports function/tool calling."
	case containsAny(lower, "context_limit", "context window", "context length", "maximum context", "token limit"):
		summary = "The provider says the request exceeded its context window."
		hint = "Try a model with a larger context window, narrow the Commit Scope (--staged, --no-untracked, or pathspecs), or set --diff-only."
	case matchesHTTPStatus(raw, 401) || containsAny(lower, "unauthorized", "invalid api key", "invalid_api_key", "authentication", "authenticate"):
		summary = "The provider rejected the request (looks like an auth problem)."
		hint = "If you confirm this is an auth error, run `cnm init` to update the API key, or set CNM_API_KEY for this run. Otherwise see the raw response below — cnm may have misclassified it."
	case matchesHTTPStatus(raw, 404) || containsAny(lower, "model not found", "model_not_found", "no such model"):
		summary = "The provider does not have the configured model."
		hint = "Run `cnm config` and pick a model that the provider exposes, or update CNM_MODEL. If this is a different 404, see the raw response below."
	case matchesHTTPStatus(raw, 429) || containsAny(lower, "rate limit", "rate_limit", "too many requests"):
		summary = "The provider is rate-limiting cnm."
		hint = "Wait a moment and try again, or switch provider."
	case matchesHTTPStatus(raw, 503) || containsAny(lower, "service unavailable", "service_unavailable"):
		summary = "The provider reported a temporary outage."
		hint = "Try again in a moment, or switch provider."
	case matchesHTTPStatus(raw, 500, 502, 504):
		summary = "The provider returned a server error."
		hint = "The provider or gateway has a problem. Try again, or switch provider."
	case strings.Contains(lower, "tool choice") && strings.Contains(lower, "tools"):
		summary = "The provider rejected the tool_choice configuration."
		hint = "Your provider gateway may not support tool_choice 'required'. Switch provider or upgrade the gateway."
	case containsAny(lower, "connection refused", "no such host", "dial tcp", "i/o timeout", "context deadline"):
		summary = "Could not reach the AI provider."
		hint = "Check the base URL, your network, and proxy settings, then try again."
	default:
		summary = "AI provider call failed."
	}
	detail = composeErrorDetail(hint, raw)
	if len(detail) > 1200 {
		detail = detail[:1200] + "..."
	}
	return summary, detail
}

func composeErrorDetail(hint, raw string) string {
	parts := []string{}
	if hint != "" {
		parts = append(parts, hint)
	}
	if raw != "" {
		parts = append(parts, "Raw error: "+raw)
	}
	return strings.Join(parts, "\n\n")
}

func containsAny(haystack string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(haystack, n) {
			return true
		}
	}
	return false
}

func matchesHTTPStatus(raw string, codes ...int) bool {
	for _, code := range codes {
		c := strconv.Itoa(code)
		patterns := []string{
			"http " + c + ":",
			"http " + c + " ",
			"status " + c,
			"status: " + c,
			"status:" + c,
			"\"status\":" + c,
			"code " + c,
			"code: " + c,
			"\"code\":" + c,
			" " + c + " ",
			"(" + c + ")",
			"[" + c + "]",
		}
		lower := strings.ToLower(raw)
		for _, p := range patterns {
			if strings.Contains(lower, p) {
				return true
			}
		}
	}
	return false
}

func extractParseDetail(raw string) string {
	idx := strings.Index(raw, "response body:")
	if idx < 0 {
		return raw
	}
	body := strings.TrimSpace(raw[idx+len("response body:"):])
	body = strings.TrimSuffix(body, ")")
	body = strings.TrimSpace(body)
	if body == "" {
		return raw
	}
	return body
}

var _ = extractParseDetail

func wordWrap(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	out := []string{}
	for _, paragraph := range strings.Split(text, "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		current := words[0]
		for _, word := range words[1:] {
			if ansi.StringWidth(current)+1+ansi.StringWidth(word) > width {
				out = append(out, current)
				current = word
				continue
			}
			current += " " + word
		}
		out = append(out, current)
	}
	return out
}

func (m Model) fileStatus(file gitpkg.FileStatus) string {
	parts := []string{}
	if file.Untracked {
		parts = append(parts, "untracked")
	}
	if file.Staged != nil {
		parts = append(parts, "staged:"+string(*file.Staged))
	}
	if file.Unstaged != nil {
		parts = append(parts, "unstaged:"+string(*file.Unstaged))
	}
	if isFileConflict(file) {
		parts = append(parts, "conflict")
	}
	if file.Binary {
		parts = append(parts, "binary")
	}
	if len(parts) == 0 {
		return "modified"
	}
	return strings.Join(parts, " ")
}

func isFileConflict(file gitpkg.FileStatus) bool {
	if file.Staged != nil && *file.Staged == gitpkg.ChangeUnmerged {
		return true
	}
	if file.Unstaged != nil && *file.Unstaged == gitpkg.ChangeUnmerged {
		return true
	}
	return false
}

func (m Model) View() string {
	if m.NoColor || m.Width < 80 {
		return m.renderPlain()
	}
	return m.renderRich()
}

func (m Model) renderRich() string {
	header := m.renderHeader()
	body := m.renderBody()
	footer := m.renderFooter()
	pieces := []string{header, body, footer}
	return strings.Join(pieces, "\n")
}

func (m Model) renderPlain() string {
	pieces := []string{
		m.renderHeaderPlain(),
		m.renderScopePlain(),
		m.renderRepairContextPlain(),
		m.renderActiveScreenPlain(),
		m.renderExposurePlain(),
		m.renderCommitPlanPlain(),
		m.renderFooterPlain(),
	}
	return strings.TrimSpace(strings.Join(filterNonEmpty(pieces), "\n\n")) + "\n"
}

func (m Model) renderHeader() string {
	mode := "Interactive Commit"
	if m.RepairContext != nil {
		mode = "Interactive Repair"
	}
	left := m.theme.Title().Render("cnm") + "  " + m.theme.Mode().Render(mode) + "  " + m.theme.Subtle().Render(m.progressLabel())
	indicator := m.aiIndicator()
	right := m.theme.Subtle().Render(indicator)

	innerWidth := m.Width - 2
	if innerWidth < 10 {
		innerWidth = 10
	}
	leftFit := fitLine(left, innerWidth)
	leftWidth := ansi.StringWidth(leftFit)
	rightWidth := ansi.StringWidth(right)
	if leftWidth+rightWidth+1 > innerWidth {
		right = ""
		rightWidth = 0
	}
	gap := innerWidth - leftWidth - rightWidth
	if gap < 1 {
		gap = 1
	}
	bar := leftFit + strings.Repeat(" ", gap) + right
	return m.theme.HeaderBar(m.Width).Render(bar)
}

func (m Model) renderBody() string {
	if !m.useTwoColumn() {
		return m.renderBodySingleColumn()
	}
	leftWidth, rightWidth := m.columnWidths()
	left := m.renderScopePanel(leftWidth)
	right := m.renderRightColumn(rightWidth)
	row := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	exposure := m.renderExposureLine()
	if exposure == "" {
		return row
	}
	return strings.Join([]string{row, exposure}, "\n")
}

func (m Model) renderBodySingleColumn() string {
	width := m.contentWidth()
	available := m.bodyHeight()
	parts := []string{}
	used := 0
	tryAdd := func(block string) bool {
		if block == "" {
			return false
		}
		height := strings.Count(block, "\n") + 1
		if used+height > available {
			return false
		}
		parts = append(parts, block)
		used += height
		return true
	}
	exposure := m.renderExposureLine()
	exposureHeight := 0
	if exposure != "" {
		exposureHeight = 1
	}
	scopeRows := m.singleColumnScopeRows(width)
	scopeBudget := len(scopeRows) + 2
	maxScope := available / 3
	if maxScope < 4 {
		maxScope = 4
	}
	if scopeBudget > maxScope {
		scopeBudget = maxScope
	}
	if scopeBudget < 4 {
		scopeBudget = 4
	}
	tryAdd(m.renderScopePanelFixedHeight(width, scopeBudget, scopeRows))
	planBlock := m.renderCommitPlanPanel(width)
	planHeight := strings.Count(planBlock, "\n") + 1
	reservedForPlan := 0
	if len(m.CommitPlan.Commits) > 0 && planHeight <= 8 {
		reservedForPlan = planHeight
	}
	previewBudget := available - used - exposureHeight - reservedForPlan - 1
	if previewBudget < 5 {
		previewBudget = 5
	}
	previewTitle, previewBody := m.activePreviewForSingleColumn()
	tryAdd(m.renderViewportPanel(width, previewBudget, previewTitle, previewBody))
	if reservedForPlan > 0 {
		tryAdd(planBlock)
	}
	tryAdd(exposure)
	return strings.Join(parts, "\n")
}

func (m Model) singleColumnScopeRows(width int) []string {
	innerWidth := width - 4
	if innerWidth < 8 {
		innerWidth = 8
	}
	heading := fmt.Sprintf("Commit Scope  %d files  %d selected", len(m.Scope.Files), m.selectedScopeFileCount())
	if m.RepairContext == nil && m.Screen == ScreenScopeReview {
		heading = m.theme.Section().Render(heading)
	} else {
		heading = m.theme.Subtle().Render(heading)
	}
	rows := []string{fitLine(heading, innerWidth)}
	if len(m.Scope.Files) == 0 {
		rows = append(rows, fitLine(m.theme.Muted().Render("No selected changes"), innerWidth))
		return rows
	}
	visible, start, end := m.visibleFiles()
	for index, file := range visible {
		active := start+index == m.ScopeCursor && m.Screen == ScreenScopeReview && m.RepairContext == nil
		cursor := m.theme.CursorMarker(active)
		check := m.theme.Checkbox(m.scopeFileIncluded(file.Path))
		status := padRight(m.theme.StatusBadge(simpleStatus(file)), 11)
		row := cursor + " " + check + " " + status + " " + file.Path
		rows = append(rows, fitLine(row, innerWidth))
	}
	if start > 0 || end < len(m.Scope.Files) {
		rows = append(rows, fitLine(m.theme.Muted().Render(fmt.Sprintf("…showing %d-%d of %d (j/k to scroll)", start+1, end, len(m.Scope.Files))), innerWidth))
	}
	return rows
}

func (m Model) renderScopePanelFixedHeight(width, budget int, rows []string) string {
	innerHeight := budget - 2
	if innerHeight < 1 {
		innerHeight = 1
	}
	if len(rows) > innerHeight {
		rows = rows[:innerHeight]
	}
	for len(rows) < innerHeight {
		rows = append(rows, "")
	}
	return m.theme.Panel(width).Render(strings.Join(rows, "\n"))
}

func (m Model) activePreviewForSingleColumn() (string, string) {
	if m.RepairContext != nil {
		header := m.theme.Section().Render("Repair Review") + "\n" +
			m.theme.Text().Render("Confirm AI-assisted conflict repair for eligible files before commit execution.") + "\n" +
			m.theme.Muted().Render("Enter accepts. q cancels.")
		return "Conflict Files", header + "\n\n" + m.repairPreview()
	}
	switch m.Screen {
	case ScreenScopeReview:
		return "Diff Preview", m.diffPreview()
	case ScreenAgentInstruction:
		caret := m.theme.Accent().Render("│")
		header := m.theme.Section().Render("Agent Instruction") + "\n"
		if m.AgentInstruction == "" {
			header += m.theme.Muted().Render("Optional. Tell the AI how to plan this commit. Enter continues.") + "\n" + caret + " "
		} else {
			header += m.theme.Text().Render(m.AgentInstruction) + caret
		}
		if strings.TrimSpace(m.AIActivityError) != "" {
			innerWidth := m.contentWidth() - 4
			if innerWidth < 8 {
				innerWidth = 8
			}
			header += "\n\n" + strings.Join(m.renderAIErrorBlock(m.AIActivityError, innerWidth), "\n")
		}
		return "Diff Preview", header + "\n\n" + m.diffPreview()
	case ScreenAIActivity:
		return "AI Activity", m.spinner.View() + " " + m.theme.Muted().Render("Planning commits with local Domain Tools…")
	case ScreenMessageReview:
		return "Commit Plan", m.commitPlanPreview() + "\n\n" + m.theme.Muted().Render("Enter accepts. e edits the first commit message. q cancels.")
	case ScreenMessageEdit:
		caret := m.theme.Accent().Render("│")
		return "Edit Message", m.theme.Text().Render(m.MessageDraft) + caret + "\n\n" + m.theme.Muted().Render("Enter saves and returns to review.")
	}
	return "Diff Preview", m.diffPreview()
}

func (m Model) renderRightColumn(width int) string {
	headerSection := m.renderActivePanel(width)
	headerHeight := strings.Count(headerSection, "\n") + 1
	available := m.bodyHeight()
	previewHeight := available - headerHeight
	if previewHeight < 4 {
		previewHeight = 4
	}
	previewBody := ""
	previewTitle := ""
	switch m.Screen {
	case ScreenScopeReview, ScreenAgentInstruction:
		previewTitle = "Diff Preview"
		previewBody = m.diffPreview()
	case ScreenAIActivity:
		previewTitle = "AI Activity"
		previewBody = m.spinner.View() + " " + m.theme.Muted().Render("Planning commits with local Domain Tools…")
	case ScreenMessageReview, ScreenMessageEdit:
		previewTitle = "Commit Plan"
		previewBody = m.commitPlanPreview()
	case ScreenRepairReview:
		previewTitle = "Conflict Files"
		previewBody = m.repairPreview()
	}
	if previewTitle == "" {
		return headerSection
	}
	panel := m.renderViewportPanel(width, previewHeight, previewTitle, previewBody)
	return strings.Join([]string{headerSection, panel}, "\n")
}

func (m Model) renderViewportPanel(width, height int, title, body string) string {
	if width < 10 {
		width = 10
	}
	if height < 4 {
		height = 4
	}
	innerWidth := width - 4
	if innerWidth < 4 {
		innerWidth = 4
	}
	innerHeight := height - 3
	if innerHeight < 1 {
		innerHeight = 1
	}
	heading := fitLine(m.theme.Section().Render(title), innerWidth)
	bodyLines := []string{}
	for _, ln := range strings.Split(body, "\n") {
		bodyLines = append(bodyLines, fitLine(ln, innerWidth))
	}
	if len(bodyLines) > innerHeight {
		bodyLines = bodyLines[:innerHeight]
		if innerHeight > 0 {
			bodyLines[innerHeight-1] = fitLine(m.theme.Muted().Render("…more (pgdn)"), innerWidth)
		}
	}
	for len(bodyLines) < innerHeight {
		bodyLines = append(bodyLines, "")
	}
	return m.theme.Panel(width).Render(heading + "\n" + strings.Join(bodyLines, "\n"))
}

func (m Model) rightPreviewHeight() int {
	body := m.bodyHeight()
	available := body - 6
	if available < 6 {
		available = 6
	}
	return available
}

func (m Model) renderScopePanel(width int) string {
	innerWidth := width - 4
	if innerWidth < 8 {
		innerWidth = 8
	}
	heading := fmt.Sprintf("Commit Scope  %d files  %d selected", len(m.Scope.Files), m.selectedScopeFileCount())
	if m.RepairContext == nil && m.Screen == ScreenScopeReview {
		heading = m.theme.Section().Render(heading)
	} else {
		heading = m.theme.Subtle().Render(heading)
	}
	rows := []string{fitLine(heading, innerWidth)}
	if len(m.Scope.Files) == 0 {
		rows = append(rows, fitLine(m.theme.Muted().Render("No selected changes"), innerWidth))
	} else {
		visible, start, end := m.visibleFiles()
		for index, file := range visible {
			active := start+index == m.ScopeCursor && m.Screen == ScreenScopeReview && m.RepairContext == nil
			cursor := m.theme.CursorMarker(active)
			check := m.theme.Checkbox(m.scopeFileIncluded(file.Path))
			status := padRight(m.theme.StatusBadge(simpleStatus(file)), 11)
			row := cursor + " " + check + " " + status + " " + file.Path
			rows = append(rows, fitLine(row, innerWidth))
		}
		if start > 0 || end < len(m.Scope.Files) {
			rows = append(rows, fitLine(m.theme.Muted().Render(fmt.Sprintf("…showing %d-%d of %d (j/k to scroll)", start+1, end, len(m.Scope.Files))), innerWidth))
		}
	}
	leftHeight := m.bodyHeight() - 2
	if leftHeight < 4 {
		leftHeight = 4
	}
	if len(rows) > leftHeight {
		rows = rows[:leftHeight]
	}
	for len(rows) < leftHeight {
		rows = append(rows, "")
	}
	return m.theme.Panel(width).Render(strings.Join(rows, "\n"))
}

func (m Model) visibleFiles() ([]gitpkg.FileStatus, int, int) {
	files := m.Scope.Files
	if len(files) == 0 {
		return nil, 0, 0
	}
	maxRows := m.bodyHeight() - 5
	if maxRows < 5 {
		maxRows = 5
	}
	if maxRows >= len(files) {
		return files, 0, len(files)
	}
	start := m.ScopeCursor - maxRows/2
	if start < 0 {
		start = 0
	}
	end := start + maxRows
	if end > len(files) {
		end = len(files)
		start = end - maxRows
		if start < 0 {
			start = 0
		}
	}
	return files[start:end], start, end
}

func (m Model) renderActivePanel(width int) string {
	switch m.Screen {
	case ScreenScopeReview:
		body := m.theme.Muted().Render("Review the Commit Scope, then enter to write an Agent Instruction. Space toggles a file.")
		return m.theme.Panel(width).Render(m.theme.Section().Render("Agent Instruction") + "\n" + body)
	case ScreenAgentInstruction:
		value := m.AgentInstruction
		caret := m.theme.Accent().Render("│")
		lines := []string{m.theme.Section().Render("Agent Instruction")}
		if value == "" {
			lines = append(lines, caret+" "+m.theme.Muted().Render("Optional. Tell the AI how to plan this commit. Enter continues."))
		} else {
			lines = append(lines, m.theme.Text().Render(value)+caret)
		}
		if strings.TrimSpace(m.AIActivityError) != "" {
			innerWidth := width - 4
			if innerWidth < 8 {
				innerWidth = 8
			}
			lines = append(lines, "")
			lines = append(lines, m.renderAIErrorBlock(m.AIActivityError, innerWidth)...)
		}
		return m.theme.Panel(width).Render(strings.Join(lines, "\n"))
	case ScreenAIActivity:
		body := m.spinner.View() + " " + m.theme.Text().Render("Planning commits with local Domain Tools…")
		return m.theme.Panel(width).Render(m.theme.Section().Render("AI Activity") + "\n" + body)
	case ScreenMessageReview:
		body := m.theme.Muted().Render("Enter accepts this plan. e edits the first commit message. q cancels.")
		return m.theme.Panel(width).Render(m.theme.Section().Render("Message Review") + "\n" + body)
	case ScreenMessageEdit:
		caret := m.theme.Accent().Render("│")
		body := m.theme.Text().Render(m.MessageDraft) + caret
		return m.theme.Panel(width).Render(m.theme.Section().Render("Edit Message") + "\n" + body + "\n" + m.theme.Muted().Render("Enter saves and returns to review."))
	case ScreenRepairReview:
		body := m.theme.Text().Render("Confirm AI-assisted conflict repair for eligible files before commit execution.")
		hint := m.theme.Muted().Render("Enter accepts. q cancels.")
		return m.theme.Panel(width).Render(m.theme.Section().Render("Repair Review") + "\n" + body + "\n" + hint)
	}
	return ""
}

func (m Model) renderDiffPanel(width int) string {
	heading := m.theme.Section().Render("Diff Preview")
	body := ""
	if m.previewSet {
		body = m.preview.View()
	} else {
		body = m.diffPreview()
	}
	return m.theme.Panel(width).Render(heading + "\n" + body)
}

func (m Model) renderRepairPanel(width int) string {
	heading := m.theme.Section().Render("Conflict Files")
	body := ""
	if m.useTwoColumn() && m.previewSet {
		body = m.preview.View()
	} else {
		body = m.repairPreview()
	}
	return m.theme.Panel(width).Render(heading + "\n" + body)
}

func (m Model) renderCommitPlanPanel(width int) string {
	heading := m.theme.Section().Render("Commit Plan")
	body := ""
	if m.Screen == ScreenMessageReview || m.Screen == ScreenMessageEdit {
		if m.previewSet {
			body = m.preview.View()
		} else {
			body = m.commitPlanPreview()
		}
	} else {
		body = m.commitPlanPreview()
	}
	return m.theme.Panel(width).Render(heading + "\n" + body)
}

func (m Model) renderExposureLine() string {
	exposure := m.Scope.AIExposure
	provider := emptyDefault(exposure.PreferenceSources.Provider, "default")
	apiKey := emptyDefault(exposure.PreferenceSources.APIKey, "missing")
	line := fmt.Sprintf("AI Exposure  files %d  diff %s  read %s  provider %s  api key %s",
		exposure.SelectedFileCount,
		formatBytes(exposure.DiffBudget.UsedBytes, exposure.DiffBudget.MaxBytes),
		formatBytes(exposure.ReadBudget.UsedBytes, exposure.ReadBudget.MaxBytes),
		provider,
		apiKey,
	)
	return m.theme.Subtle().Render(fitLine(line, m.contentWidth()))
}

func (m Model) renderFooter() string {
	keys := m.footerKeys()
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, m.theme.KeyHint().Render(k.Key)+" "+m.theme.KeyDesc().Render(k.Desc))
	}
	bar := strings.Join(parts, m.theme.Subtle().Render("  ·  "))
	return m.theme.FooterBar(m.Width).Render(fitLine(bar, m.Width-2))
}

type keyHint struct{ Key, Desc string }

func (m Model) footerKeys() []keyHint {
	switch m.Screen {
	case ScreenScopeReview:
		return []keyHint{
			{"↑/↓", "navigate"},
			{"space", "toggle"},
			{"a", "all"},
			{"n", "none"},
			{"enter", "continue"},
			{"q", "quit"},
		}
	case ScreenAgentInstruction:
		return []keyHint{{"type", "instruction"}, {"enter", "continue"}, {"q", "quit"}}
	case ScreenAIActivity:
		return []keyHint{{"q", "quit"}}
	case ScreenMessageReview:
		return []keyHint{{"e", "edit"}, {"enter", "accept"}, {"pgup/pgdn", "scroll"}, {"q", "quit"}}
	case ScreenMessageEdit:
		return []keyHint{{"type", "edit message"}, {"enter", "save"}, {"q", "quit"}}
	case ScreenRepairReview:
		return []keyHint{{"enter", "confirm repair"}, {"pgup/pgdn", "scroll"}, {"q", "quit"}}
	}
	return []keyHint{{"enter", "continue"}, {"q", "quit"}}
}

func (m Model) renderHeaderPlain() string {
	mode := "Interactive Commit"
	if m.RepairContext != nil {
		mode = "Interactive Repair"
	}
	return "cnm " + mode + "  " + m.progressLabel()
}

func (m Model) renderScopePlain() string {
	heading := fmt.Sprintf("Commit Scope  %d files  %d selected", len(m.Scope.Files), m.selectedScopeFileCount())
	lines := []string{heading}
	if len(m.Scope.Files) == 0 {
		lines = append(lines, "No selected changes")
		return strings.Join(lines, "\n")
	}
	for index, file := range m.Scope.Files {
		marker := simpleStatus(file)
		cursor := " "
		if index == m.ScopeCursor {
			cursor = ">"
		}
		check := "[x]"
		if !m.scopeFileIncluded(file.Path) {
			check = "[ ]"
		}
		row := fmt.Sprintf("%s %s %-10s  %s", cursor, check, marker, file.Path)
		lines = append(lines, fitLine(row, m.plainWidth()))
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderRepairContextPlain() string {
	if m.RepairContext == nil {
		return ""
	}
	lines := []string{"Repair"}
	if strings.TrimSpace(m.RepairContext.Reason) != "" {
		lines = append(lines, fitLine(m.RepairContext.Reason, m.plainWidth()))
	}
	if len(m.RepairContext.EligibleFiles) > 0 {
		lines = append(lines, "eligible files")
		for _, file := range m.RepairContext.EligibleFiles {
			lines = append(lines, fitLine("   "+file, m.plainWidth()))
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderActiveScreenPlain() string {
	switch m.Screen {
	case ScreenRepairReview:
		return "Repair Review\n" + fitLine("Confirm AI-assisted conflict repair for eligible files before commit execution.", m.plainWidth())
	case ScreenAgentInstruction:
		value := m.AgentInstruction
		if value == "" {
			value = "optional instruction; enter continues"
		}
		out := []string{"Agent Instruction", fitLine(value, m.plainWidth())}
		if strings.TrimSpace(m.AIActivityError) != "" {
			summary, detail := humanizeAIError(m.AIActivityError)
			out = append(out, "", "AI activity failed.")
			out = append(out, wordWrap(summary, m.plainWidth())...)
			if detail != "" {
				out = append(out, "")
				out = append(out, wordWrap(detail, m.plainWidth())...)
			}
			out = append(out, "", "Press q to exit. Try `cnm doctor` to verify provider, model, and API key.")
		}
		return strings.Join(out, "\n")
	case ScreenAIActivity:
		return "AI Activity\n" + fitLine("Planning commits with local Domain Tools", m.plainWidth())
	case ScreenMessageReview:
		return "Message Review\n" + fitLine("Enter accepts this plan. e edits the first message.", m.plainWidth())
	case ScreenMessageEdit:
		return "Edit Message\n" + fitLine(m.MessageDraft, m.plainWidth())
	default:
		return "Agent Instruction\n" + fitLine("review scope first; enter opens instruction input", m.plainWidth())
	}
}

func (m Model) renderExposurePlain() string {
	exposure := m.Scope.AIExposure
	return strings.Join([]string{
		"AI Exposure",
		fitLine(fmt.Sprintf("files %d  diff %d/%d bytes  read %d/%d bytes",
			exposure.SelectedFileCount,
			exposure.DiffBudget.UsedBytes, exposure.DiffBudget.MaxBytes,
			exposure.ReadBudget.UsedBytes, exposure.ReadBudget.MaxBytes), m.plainWidth()),
		fitLine(fmt.Sprintf("provider %s  api key %s",
			emptyDefault(exposure.PreferenceSources.Provider, "default"),
			emptyDefault(exposure.PreferenceSources.APIKey, "missing")), m.plainWidth()),
	}, "\n")
}

func (m Model) renderCommitPlanPlain() string {
	if len(m.CommitPlan.Commits) == 0 {
		return "Commit Plan\nNo plan yet"
	}
	lines := []string{"Commit Plan", fmt.Sprintf("mode %s", emptyDefault(m.CommitPlan.Kind, "single"))}
	for index, commit := range m.CommitPlan.Commits {
		lines = append(lines, fitLine(fmt.Sprintf("%d. %s", index+1, commit.Message), m.plainWidth()))
		for _, file := range commit.Files {
			lines = append(lines, fitLine("   "+file, m.plainWidth()))
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderFooterPlain() string {
	label := "enter continue  space toggle  e edit  q quit"
	if m.plainWidth() < len(label) {
		label = "enter continue  space toggle  q quit"
	}
	if m.plainWidth() < len(label) {
		label = "enter continue  q quit"
	}
	return fitLine(label, m.plainWidth())
}

func (m Model) progressLabel() string {
	screens := []Screen{ScreenScopeReview, ScreenAgentInstruction, ScreenAIActivity, ScreenMessageReview}
	if m.RepairContext != nil {
		screens = []Screen{ScreenRepairReview}
	}
	for index, screen := range screens {
		if m.Screen == screen {
			return fmt.Sprintf("[%d/%d]", index+1, len(screens))
		}
	}
	return fmt.Sprintf("[%d/%d]", len(screens), len(screens))
}

func (m Model) aiIndicator() string {
	if m.Screen == ScreenAIActivity {
		return m.spinner.View() + " AI working"
	}
	return ""
}

func (m Model) useTwoColumn() bool {
	return m.Width >= 100 && m.Height >= 18 && !m.NoColor
}

func (m Model) columnWidths() (int, int) {
	available := m.Width
	if available <= 0 {
		available = 96
	}
	left := available * 38 / 100
	if left < 32 {
		left = 32
	}
	right := available - left
	if right < 32 {
		right = 32
		left = available - right
	}
	return left, right
}

func (m Model) contentWidth() int {
	if m.NoColor {
		return m.plainWidth()
	}
	if m.Width <= 0 {
		return 96
	}
	if m.Width < 40 {
		return 40
	}
	return m.Width
}

func (m Model) plainWidth() int {
	if m.Width <= 0 {
		return 88
	}
	if m.Width < 40 {
		return 40
	}
	if m.Width > 92 {
		return 92
	}
	return m.Width
}

func (m Model) bodyHeight() int {
	if m.Height <= 0 {
		return 24
	}
	body := m.Height - 4
	if body < 10 {
		body = 10
	}
	return body
}

func simpleStatus(file gitpkg.FileStatus) string {
	if isFileConflict(file) {
		return "conflict"
	}
	if file.Untracked {
		return "untracked"
	}
	if file.Staged != nil {
		return string(*file.Staged)
	}
	if file.Unstaged != nil {
		return string(*file.Unstaged)
	}
	return "modified"
}

func formatBytes(used, max int) string {
	if max <= 0 {
		return fmt.Sprintf("%d/%d B", used, max)
	}
	return fmt.Sprintf("%d/%d B", used, max)
}

func filterNonEmpty(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, value)
		}
	}
	return result
}

func joinNonEmpty(values []string) string {
	return strings.Join(filterNonEmpty(values), "\n")
}

func fitLine(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if ansi.StringWidth(value) <= width {
		return value
	}
	if width <= 3 {
		return ansi.Truncate(value, width, "")
	}
	return ansi.Truncate(value, width, "...")
}

func padRight(value string, width int) string {
	current := ansi.StringWidth(value)
	if current >= width {
		return value
	}
	return value + strings.Repeat(" ", width-current)
}

func stripANSI(value string) string {
	return ansi.Strip(value)
}

func emptyDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
