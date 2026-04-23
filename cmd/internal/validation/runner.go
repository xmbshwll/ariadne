package validation

import (
	"context"
	"fmt"
	"io"
	"time"
)

type RunInputs interface {
	OutputDir() string
	SuccessMessage() string
}

func Run[Inputs RunInputs, Artifacts any](args []string, stdout io.Writer, timeout time.Duration, load func([]string) (Inputs, error), collect func(context.Context, Inputs) (Artifacts, error), write func(string, Artifacts) error) error {
	inputs, err := load(args)
	if err != nil {
		return err
	}

	if stdout == nil {
		stdout = io.Discard
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	artifacts, err := collect(ctx, inputs)
	if err != nil {
		return err
	}
	if err := write(inputs.OutputDir(), artifacts); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(stdout, inputs.SuccessMessage()); err != nil {
		return fmt.Errorf("write success message: %w", err)
	}
	return nil
}
