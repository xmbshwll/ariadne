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
	if err := validation.Run(validation.RunConfig[validationInputs, validationArtifacts]{
		Args:    args,
		Stdout:  os.Stdout,
		Timeout: 30 * time.Second,
		Load:    loadValidationInputs,
		Collect: collectValidationArtifacts,
		Write:   writeValidationArtifacts,
	}); err != nil {
		return fmt.Errorf("run apple music validation: %w", err)
	}
	return nil
}
