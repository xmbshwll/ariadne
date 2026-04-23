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

type RunConfig[Inputs RunInputs, Artifacts any] struct {
	Args    []string
	Stdout  io.Writer
	Timeout time.Duration
	Load    func([]string) (Inputs, error)
	Collect func(context.Context, Inputs) (Artifacts, error)
	Write   func(string, Artifacts) error
}

func Run[Inputs RunInputs, Artifacts any](cfg RunConfig[Inputs, Artifacts]) error {
	inputs, err := cfg.Load(cfg.Args)
	if err != nil {
		return err
	}

	stdout := cfg.Stdout
	if stdout == nil {
		stdout = io.Discard
	}

	ctx := context.Background()
	cancel := func() {}
	if cfg.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
	}
	defer cancel()

	artifacts, err := cfg.Collect(ctx, inputs)
	if err != nil {
		return err
	}
	if err := cfg.Write(inputs.OutputDir(), artifacts); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(stdout, inputs.SuccessMessage()); err != nil {
		return fmt.Errorf("write success message: %w", err)
	}
	return nil
}
