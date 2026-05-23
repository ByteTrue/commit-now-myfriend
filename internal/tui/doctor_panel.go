package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type DoctorView struct {
	Status   string
	Errors   int
	Warnings int
	OK       bool

	GitMessage        string
	GitStatus         string
	RepoMessage       string
	RepoStatus        string
	ConfigMessage     string
	ConfigStatus      string
	CapabilityMessage string
	CapabilityStatus  string

	ProviderName  string
	ModelName     string
	APIKeySource  string
	UserConfig    string
	ProjectConfig string

	ProbeAttempted bool
	ProbeStatus    string
	ProbeMessage   string

	Issues []DoctorIssue
}

type DoctorIssue struct {
	Severity string
	Check    string
	Message  string
}

func RenderDoctorRich(view DoctorView, width int, noColor bool) string {
	if width <= 0 {
		width = 96
	}
	theme := Theme{NoColor: noColor}
	if noColor || width < 80 {
		return RenderDoctorPlain(view)
	}
	header := theme.HeaderBar(width).Render(
		theme.Title().Render("cnm") + "  " + theme.Mode().Render("Doctor") +
			"  " + theme.Subtle().Render(fmt.Sprintf("[%s]", view.Status)),
	)
	statusLine := renderDoctorStatusLine(theme, view)
	checks := renderDoctorChecks(theme, view, width)
	configPanel := renderDoctorConfig(theme, view, width)
	probePanel := renderDoctorProbe(theme, view, width)
	issues := renderDoctorIssues(theme, view, width)
	footer := theme.FooterBar(width).Render(theme.KeyDesc().Render("doctor is read-only by default. Run with --probe-provider for a fixed provider probe."))
	parts := []string{header, statusLine, checks, configPanel}
	if probePanel != "" {
		parts = append(parts, probePanel)
	}
	if issues != "" {
		parts = append(parts, issues)
	}
	parts = append(parts, footer)
	return strings.Join(parts, "\n") + "\n"
}

func RenderDoctorPlain(view DoctorView) string {
	lines := []string{
		"cnm doctor",
		"",
		fmt.Sprintf("Status: %s", view.Status),
		fmt.Sprintf("Errors: %d", view.Errors),
		fmt.Sprintf("Warnings: %d", view.Warnings),
		"",
		fmt.Sprintf("Git: %s", view.GitMessage),
		fmt.Sprintf("Repository: %s", view.RepoMessage),
		fmt.Sprintf("Effective config: %s", view.ConfigMessage),
		fmt.Sprintf("Provider capability: %s", view.CapabilityMessage),
	}
	if view.ProbeAttempted {
		lines = append(lines, "", fmt.Sprintf("Provider probe: %s — %s", view.ProbeStatus, view.ProbeMessage))
	}
	if len(view.Issues) > 0 {
		lines = append(lines, "", "Issues:")
		for _, issue := range view.Issues {
			lines = append(lines, fmt.Sprintf("  - [%s] %s: %s", issue.Severity, issue.Check, issue.Message))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func renderDoctorStatusLine(theme Theme, view DoctorView) string {
	statusStyle := theme.Success()
	statusLabel := "ok"
	if view.Errors > 0 {
		statusStyle = theme.Error()
		statusLabel = "issues found"
	} else if view.Warnings > 0 {
		statusStyle = theme.Warn()
		statusLabel = "warnings"
	}
	return statusStyle.Render("●") + " " + theme.Text().Render(statusLabel) +
		theme.Subtle().Render(fmt.Sprintf("    errors %d  warnings %d", view.Errors, view.Warnings))
}

func renderDoctorChecks(theme Theme, view DoctorView, width int) string {
	innerWidth := width - 4
	if innerWidth < 8 {
		innerWidth = 8
	}
	rows := []string{fitLine(theme.Section().Render("Checks"), innerWidth)}
	rows = append(rows, fitLine(doctorCheckLine(theme, "git", view.GitStatus, view.GitMessage), innerWidth))
	rows = append(rows, fitLine(doctorCheckLine(theme, "repository", view.RepoStatus, view.RepoMessage), innerWidth))
	rows = append(rows, fitLine(doctorCheckLine(theme, "effective config", view.ConfigStatus, view.ConfigMessage), innerWidth))
	rows = append(rows, fitLine(doctorCheckLine(theme, "provider capability", view.CapabilityStatus, view.CapabilityMessage), innerWidth))
	return theme.Panel(width).Render(strings.Join(rows, "\n"))
}

func doctorCheckLine(theme Theme, label, status, message string) string {
	icon := "✔"
	style := theme.Success()
	switch status {
	case "warning":
		icon = "!"
		style = theme.Warn()
	case "error":
		icon = "✖"
		style = theme.Error()
	}
	return style.Render(icon) + " " + theme.Accent().Render(label) + " " + theme.Subtle().Render(message)
}

func renderDoctorConfig(theme Theme, view DoctorView, width int) string {
	innerWidth := width - 4
	if innerWidth < 8 {
		innerWidth = 8
	}
	rows := []string{fitLine(theme.Section().Render("Effective Configuration"), innerWidth)}
	rows = append(rows, fitLine(theme.Text().Render("provider:    ")+theme.Accent().Render(view.ProviderName), innerWidth))
	rows = append(rows, fitLine(theme.Text().Render("model:       ")+theme.Accent().Render(view.ModelName), innerWidth))
	rows = append(rows, fitLine(theme.Text().Render("api key:     ")+theme.Subtle().Render(view.APIKeySource), innerWidth))
	if view.UserConfig != "" {
		rows = append(rows, fitLine(theme.Text().Render("user config: ")+theme.Subtle().Render(view.UserConfig), innerWidth))
	}
	if view.ProjectConfig != "" {
		rows = append(rows, fitLine(theme.Text().Render("project cfg: ")+theme.Subtle().Render(view.ProjectConfig), innerWidth))
	}
	return theme.Panel(width).Render(strings.Join(rows, "\n"))
}

func renderDoctorProbe(theme Theme, view DoctorView, width int) string {
	if !view.ProbeAttempted {
		return ""
	}
	innerWidth := width - 4
	if innerWidth < 8 {
		innerWidth = 8
	}
	style := theme.Success()
	switch view.ProbeStatus {
	case "warning":
		style = theme.Warn()
	case "error":
		style = theme.Error()
	}
	rows := []string{
		fitLine(theme.Section().Render("Provider Probe"), innerWidth),
		fitLine(style.Render(view.ProbeStatus)+" "+theme.Subtle().Render(view.ProbeMessage), innerWidth),
	}
	return theme.Panel(width).Render(strings.Join(rows, "\n"))
}

func renderDoctorIssues(theme Theme, view DoctorView, width int) string {
	if len(view.Issues) == 0 {
		return ""
	}
	innerWidth := width - 4
	if innerWidth < 8 {
		innerWidth = 8
	}
	rows := []string{fitLine(theme.Section().Render("Issues"), innerWidth)}
	for _, issue := range view.Issues {
		style := theme.Subtle()
		marker := "·"
		switch issue.Severity {
		case "error":
			style = theme.Error()
			marker = "✖"
		case "warning":
			style = theme.Warn()
			marker = "!"
		}
		rows = append(rows, fitLine(style.Render(marker)+" "+theme.Accent().Render(issue.Check)+" "+theme.Subtle().Render(issue.Message), innerWidth))
	}
	return theme.Panel(width).Render(strings.Join(rows, "\n"))
}

var _ = lipgloss.NewStyle
