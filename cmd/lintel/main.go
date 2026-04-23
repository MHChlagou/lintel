package main

import (
	"fmt"
	"os"

	"github.com/MHChlagou/lintel/internal/cli"
)

func main() {
	if err := cli.NewRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "✖", err)
		os.Exit(1)
	}
}
