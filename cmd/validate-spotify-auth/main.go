package main

import (
	"fmt"
	"os"
	"time"

	"github.com/xmbshwll/ariadne/cmd/internal/validation"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if err := validation.Run(args, os.Stdout, 30*time.Second, loadValidationInputs, collectValidationArtifacts, writeValidationArtifacts); err != nil {
		return fmt.Errorf("run spotify validation: %w", err)
	}
	return nil
}
