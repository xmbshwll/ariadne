package validation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"
)

type RunInputs interface {
	OutputDir() string
	SuccessMessage() string
}

type RunConfig[Inputs RunInputs, Artifacts any] struct {
	Args    []string
	Stdout  io.Writer
	Timeout time.Duration
	Load    func([]string) (Inputs, error)
	Collect func(context.Context, Inputs) (Artifacts, error)
	Write   func(string, Artifacts) error
}

var errNilRunConfigField = errors.New("nil RunConfig field")

func Run[Inputs RunInputs, Artifacts any](cfg RunConfig[Inputs, Artifacts]) error {
	if err := validateRunConfig(cfg); err != nil {
		return err
	}

	inputs, err := cfg.Load(cfg.Args)
	if err != nil {
		return err
	}

	ctx, cancel := runContext(cfg.Timeout)
	defer cancel()

	artifacts, err := cfg.Collect(ctx, inputs)
	if err != nil {
		return err
	}
	if err := cfg.Write(inputs.OutputDir(), artifacts); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(runStdout(cfg.Stdout), inputs.SuccessMessage()); err != nil {
		return fmt.Errorf("write success message: %w", err)
	}
	return nil
}

func runStdout(stdout io.Writer) io.Writer {
	if stdout == nil {
		return io.Discard
	}
	return stdout
}

func runContext(timeout time.Duration) (context.Context, func()) {
	if timeout <= 0 {
		return context.Background(), func() {}
	}
	return context.WithTimeout(context.Background(), timeout)
}

func validateRunConfig[Inputs RunInputs, Artifacts any](cfg RunConfig[Inputs, Artifacts]) error {
	switch {
	case cfg.Load == nil:
		return fmt.Errorf("%w: Load", errNilRunConfigField)
	case cfg.Collect == nil:
		return fmt.Errorf("%w: Collect", errNilRunConfigField)
	case cfg.Write == nil:
		return fmt.Errorf("%w: Write", errNilRunConfigField)
	default:
		return nil
	}
}
