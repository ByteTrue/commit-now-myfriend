package interactive

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type Prompter struct {
	reader *bufio.Reader
	out    io.Writer
}

func NewPrompter(input io.Reader, output io.Writer) *Prompter {
	return &Prompter{reader: bufio.NewReader(input), out: output}
}

func (p *Prompter) AskChoice(prompt string, allowed []string) (string, error) {
	for {
		if _, err := fmt.Fprintln(p.out, prompt); err != nil {
			return "", err
		}
		if _, err := fmt.Fprint(p.out, "> "); err != nil {
			return "", err
		}
		line, err := p.reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", err
		}
		choice := strings.TrimSpace(line)
		for _, candidate := range allowed {
			if strings.EqualFold(choice, candidate) {
				return candidate, nil
			}
		}
		if _, err := fmt.Fprintf(p.out, "Invalid choice %q. Allowed: %s\n", choice, strings.Join(allowed, ", ")); err != nil {
			return "", err
		}
		if err == io.EOF {
			return "", io.EOF
		}
	}
}

func (p *Prompter) AskYesNo(prompt string, defaultNo bool) (bool, error) {
	allowed := []string{"y", "yes", "n", "no"}
	if defaultNo {
		prompt += " [y/N]"
	} else {
		prompt += " [Y/n]"
	}
	choice, err := p.AskChoice(prompt, allowed)
	if err != nil {
		return false, err
	}
	return choice == "y" || choice == "yes", nil
}

func (p *Prompter) AskText(prompt string, defaultValue string) (string, error) {
	if strings.TrimSpace(defaultValue) != "" {
		prompt += " [" + defaultValue + "]"
	}
	if _, err := fmt.Fprintln(p.out, prompt); err != nil {
		return "", err
	}
	if _, err := fmt.Fprint(p.out, "> "); err != nil {
		return "", err
	}
	line, err := p.reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	value := strings.TrimSpace(line)
	if value == "" {
		return strings.TrimSpace(defaultValue), nil
	}
	return value, nil
}

func (p *Prompter) EditText(current string, label string, validationMessage string) (*string, error) {
	if validationMessage != "" {
		if _, err := fmt.Fprintf(p.out, "%s\n", validationMessage); err != nil {
			return nil, err
		}
	}
	if _, err := fmt.Fprintf(p.out, "%s\n", label); err != nil {
		return nil, err
	}
	if _, err := fmt.Fprintln(p.out, "Current value:"); err != nil {
		return nil, err
	}
	if _, err := fmt.Fprintln(p.out, current); err != nil {
		return nil, err
	}
	if _, err := fmt.Fprintln(p.out, "Enter a replacement. Finish with a single '.' on its own line."); err != nil {
		return nil, err
	}
	if _, err := fmt.Fprintln(p.out, "Press Enter on the first line to keep the current value. Type 'cancel' on the first line to cancel."); err != nil {
		return nil, err
	}
	if _, err := fmt.Fprint(p.out, "> "); err != nil {
		return nil, err
	}
	firstLine, err := p.reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, err
	}
	firstLine = strings.TrimRight(firstLine, "\r\n")
	if strings.TrimSpace(firstLine) == "" {
		value := current
		return &value, nil
	}
	if strings.EqualFold(strings.TrimSpace(firstLine), "cancel") {
		return nil, nil
	}
	if strings.TrimSpace(firstLine) == "." {
		trimmed := strings.TrimSpace(firstLine)
		return &trimmed, nil
	}
	lines := []string{firstLine}
	for {
		if err == io.EOF {
			break
		}
		line, readErr := p.reader.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return nil, readErr
		}
		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "." {
			break
		}
		lines = append(lines, trimmed)
		err = readErr
		if readErr == io.EOF {
			break
		}
	}
	value := strings.TrimSpace(strings.Join(lines, "\n"))
	return &value, nil
}
