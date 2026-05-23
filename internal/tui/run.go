package tui

import (
	"io"

	tea "github.com/charmbracelet/bubbletea"
)

type Runtime struct {
	Input    io.Reader
	Output   io.Writer
	NoColor  bool
	Headless bool
}

func Run(input ModelInput, runtime Runtime) (Result, error) {
	if runtime.NoColor {
		input.NoColor = true
	}
	model := NewModel(input)
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
		return Result{}, err
	}
	if resultModel, ok := finalModel.(Model); ok {
		return resultModel.Result(), nil
	}
	return Result{}, nil
}
