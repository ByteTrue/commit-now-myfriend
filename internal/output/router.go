package output

import (
	"encoding/json"
	"fmt"
	"io"
)

type HumanTarget string

const (
	StdoutTarget HumanTarget = "stdout"
	StderrTarget HumanTarget = "stderr"
)

type RoutedJSONResult struct {
	OK      bool   `json:"ok"`
	Status  string `json:"status"`
	Command string `json:"command"`
	Message string `json:"message"`
	DryRun  bool   `json:"dryRun"`
}

type Router struct {
	isJSON bool
	stdout io.Writer
	stderr io.Writer
}

func NewRouter(jsonMode bool, stdout, stderr io.Writer) *Router {
	return &Router{
		isJSON: jsonMode,
		stdout: stdout,
		stderr: stderr,
	}
}

func (r *Router) IsJSON() bool {
	return r.isJSON
}

func (r *Router) WriteHuman(message string, target HumanTarget) error {
	stream := r.stderr
	if target == StdoutTarget {
		stream = r.stdout
	}

	_, err := fmt.Fprintln(stream, message)
	return err
}

func (r *Router) WriteJSON(payload any) error {
	encoder := json.NewEncoder(r.stdout)
	return encoder.Encode(payload)
}

func (r *Router) WriteStructured(payload any, message string, target HumanTarget) error {
	if r.isJSON {
		return r.WriteJSON(payload)
	}

	return r.WriteHuman(message, target)
}
