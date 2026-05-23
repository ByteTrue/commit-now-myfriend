package main

import (
	"os"

	"github.com/ByteTrue/commit-now-myfriend/internal/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:], os.Stdout, os.Stderr))
}
