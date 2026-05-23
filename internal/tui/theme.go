package tui

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	NoColor bool
}

var (
	colorAccent       = lipgloss.Color("#C77DFF")
	colorAccentBright = lipgloss.Color("#E0AAFF")
	colorAccentDim    = lipgloss.Color("#9D4EDD")
	colorSuccess      = lipgloss.Color("#5EEAD4")
	colorWarn         = lipgloss.Color("#FBBF24")
	colorError        = lipgloss.Color("#F472B6")
	colorMuted        = lipgloss.Color("#6B7280")
	colorText         = lipgloss.Color("#E5E7EB")
	colorSubtle       = lipgloss.Color("#9CA3AF")
	colorDiffAdd      = lipgloss.Color("#5EEAD4")
	colorDiffDel      = lipgloss.Color("#F472B6")
	colorDiffMeta     = lipgloss.Color("#9D4EDD")
)

func (t Theme) Title() lipgloss.Style {
	style := lipgloss.NewStyle().Bold(true)
	if t.NoColor {
		return style
	}
	return style.Foreground(colorAccentBright)
}

func (t Theme) Mode() lipgloss.Style {
	style := lipgloss.NewStyle()
	if t.NoColor {
		return style
	}
	return style.Foreground(colorAccent)
}

func (t Theme) Section() lipgloss.Style {
	style := lipgloss.NewStyle().Bold(true)
	if t.NoColor {
		return style
	}
	return style.Foreground(colorAccent)
}

func (t Theme) Muted() lipgloss.Style {
	if t.NoColor {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(colorMuted)
}

func (t Theme) Subtle() lipgloss.Style {
	if t.NoColor {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(colorSubtle)
}

func (t Theme) Text() lipgloss.Style {
	if t.NoColor {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(colorText)
}

func (t Theme) Success() lipgloss.Style {
	if t.NoColor {
		return lipgloss.NewStyle().Bold(true)
	}
	return lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
}

func (t Theme) Warn() lipgloss.Style {
	if t.NoColor {
		return lipgloss.NewStyle().Bold(true)
	}
	return lipgloss.NewStyle().Foreground(colorWarn).Bold(true)
}

func (t Theme) Error() lipgloss.Style {
	if t.NoColor {
		return lipgloss.NewStyle().Bold(true)
	}
	return lipgloss.NewStyle().Foreground(colorError).Bold(true)
}

func (t Theme) Accent() lipgloss.Style {
	if t.NoColor {
		return lipgloss.NewStyle().Bold(true)
	}
	return lipgloss.NewStyle().Foreground(colorAccentBright).Bold(true)
}

func (t Theme) CursorMarker(active bool) string {
	if !active {
		return " "
	}
	if t.NoColor {
		return ">"
	}
	return lipgloss.NewStyle().Foreground(colorAccentBright).Bold(true).Render(">")
}

func (t Theme) Checkbox(included bool) string {
	if included {
		if t.NoColor {
			return "[x]"
		}
		return lipgloss.NewStyle().Foreground(colorAccent).Render("[") +
			lipgloss.NewStyle().Foreground(colorAccentBright).Bold(true).Render("x") +
			lipgloss.NewStyle().Foreground(colorAccent).Render("]")
	}
	if t.NoColor {
		return "[ ]"
	}
	return lipgloss.NewStyle().Foreground(colorMuted).Render("[ ]")
}

func (t Theme) StatusBadge(status string) string {
	if t.NoColor {
		return status
	}
	color := colorMuted
	switch status {
	case "untracked":
		color = colorSuccess
	case "modified":
		color = colorAccentBright
	case "added":
		color = colorSuccess
	case "deleted":
		color = colorError
	case "renamed":
		color = colorAccent
	case "conflict", "unmerged":
		color = colorError
	}
	return lipgloss.NewStyle().Foreground(color).Render(status)
}

func (t Theme) Panel(width int) lipgloss.Style {
	inner := width - 2
	if inner < 1 {
		inner = 1
	}
	style := lipgloss.NewStyle().
		Padding(0, 1).
		Width(inner)
	if t.NoColor {
		return style.Border(lipgloss.NormalBorder())
	}
	return style.
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccentDim)
}

func (t Theme) HeaderBar(width int) lipgloss.Style {
	style := lipgloss.NewStyle().Width(width).Padding(0, 1)
	if t.NoColor {
		return style
	}
	return style.Foreground(colorText).Background(lipgloss.Color("#1F1B2E"))
}

func (t Theme) FooterBar(width int) lipgloss.Style {
	style := lipgloss.NewStyle().Width(width).Padding(0, 1)
	if t.NoColor {
		return style
	}
	return style.Foreground(colorMuted).Background(lipgloss.Color("#15121F"))
}

func (t Theme) KeyHint() lipgloss.Style {
	if t.NoColor {
		return lipgloss.NewStyle().Bold(true)
	}
	return lipgloss.NewStyle().Foreground(colorAccentBright).Bold(true)
}

func (t Theme) KeyDesc() lipgloss.Style {
	if t.NoColor {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(colorSubtle)
}

func (t Theme) DiffAdd() lipgloss.Style {
	if t.NoColor {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(colorDiffAdd)
}

func (t Theme) DiffDel() lipgloss.Style {
	if t.NoColor {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(colorDiffDel)
}

func (t Theme) DiffMeta() lipgloss.Style {
	if t.NoColor {
		return lipgloss.NewStyle().Bold(true)
	}
	return lipgloss.NewStyle().Foreground(colorDiffMeta).Bold(true)
}
