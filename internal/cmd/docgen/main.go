package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra/doc"

	"github.com/nobl9/sloctl/internal"
)

func main() {
	cmd := internal.NewRootCmd()
	err := doc.GenMarkdownTree(cmd, "./docs")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating markdown docs: %v\n", err)
		os.Exit(1)
	}
}
