package commands

import (
	"flag"
	"fmt"
	"io"

	"github.com/ByteTrue/commit-now-myfriend/internal/config"
	"github.com/ByteTrue/commit-now-myfriend/internal/doctor"
	"github.com/ByteTrue/commit-now-myfriend/internal/output"
)

type DoctorRuntime struct {
	CWD         string
	Env         map[string]string
	Stdout      io.Writer
	Stderr      io.Writer
	IsTTY       bool
	SecretStore config.SecretStore
	RenderRich  func(report doctor.Report) string
}

func RunDoctor(args []string, runtime DoctorRuntime) int {
	fs := flag.NewFlagSet("cnm doctor", flag.ContinueOnError)
	fs.SetOutput(runtime.Stderr)

	jsonMode := fs.Bool("json", false, "emit JSON output")
	probeProvider := fs.Bool("probe-provider", false, "send a fixed non-repository probe to verify provider connectivity")

	if err := fs.Parse(args); err != nil {
		return int(output.UsageError)
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(runtime.Stderr, "error: unexpected doctor arguments: %v\n", fs.Args())
		return int(output.UsageError)
	}

	report, err := doctor.Run(doctor.RunOptions{CWD: runtime.CWD, Env: runtime.Env, SecretStore: runtime.SecretStore, ProbeProvider: *probeProvider})
	if err != nil {
		fmt.Fprintf(runtime.Stderr, "error: %v\n", err)
		return int(output.Error)
	}

	router := output.NewRouter(*jsonMode, runtime.Stdout, runtime.Stderr)
	if *jsonMode {
		payload := map[string]any{}
		payload["command"] = report.Command
		payload["status"] = report.Status
		payload["ok"] = report.OK
		payload["readOnly"] = report.ReadOnly
		payload["paths"] = report.Paths
		payload["checks"] = report.Checks
		payload["issues"] = report.Issues
		payload["summary"] = report.Summary
		if report.Probe != nil {
			payload["probe"] = report.Probe
		}
		_ = router.WriteJSON(payload)
	} else if runtime.IsTTY && runtime.RenderRich != nil {
		_ = router.WriteHuman(runtime.RenderRich(report), output.StdoutTarget)
	} else {
		_ = router.WriteHuman(doctor.RenderReport(report), output.StdoutTarget)
		if report.Probe != nil {
			_ = router.WriteHuman("\nProvider probe: "+report.Probe.Status, output.StdoutTarget)
		}
	}

	if !report.OK {
		return int(output.Error)
	}
	return int(output.Success)
}
