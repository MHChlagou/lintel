package main

import (
	"fmt"
	"os"

	"github.com/aegis-sec/aegis/internal/cli"
)

func main() {
	if err := cli.NewRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "✖", err)
		os.Exit(1)
	}
}
