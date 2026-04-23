package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/xmbshwll/ariadne"
)

func run(args []string, stdout io.Writer, stderr io.Writer) error {
	configPath := configPathFromArgs(args)
	helpConfig := ariadne.DefaultConfig()
	commandArgs := argsWithoutPersistentFlags(args)

	switch {
	case len(commandArgs) == 0:
		if err := renderRootHelp(stderr, helpConfig, configPath); err != nil {
			return fmt.Errorf("print usage: %w", err)
		}
		return errMissingCommand
	case isHelpArg(commandArgs[0]):
		if len(commandArgs) == 1 {
			return renderRootHelp(stdout, helpConfig, configPath)
		}
		return executeRootCommand(stdout, stderr, helpConfig, configPath, nil, args)
	case containsHelpArg(commandArgs[1:]):
		return executeRootCommand(stdout, stderr, helpConfig, configPath, nil, args)
	}

	logger, err := newCLILoggerFromArgs(args, stderr)
	if err != nil {
		return err
	}

	baseConfig, err := loadCLIConfigWithLogger(configPath, logger)
	if err != nil {
		return err
	}

	unknownCommand := firstCommandArg(commandArgs, args)
	if err := executeRootCommand(stdout, stderr, baseConfig, configPath, logger, args); err != nil {
		if isUnknownCommandError(err) {
			if helpErr := renderRootHelp(stderr, baseConfig, configPath); helpErr != nil {
				return fmt.Errorf("print usage: %w", helpErr)
			}
			return fmt.Errorf("%w: %s", errUnknownCommand, unknownCommand)
		}
		return err
	}
	return nil
}

func executeRootCommand(stdout io.Writer, stderr io.Writer, baseConfig ariadne.Config, configPath string, logger *cliLogger, args []string) error {
	root := newRootCmd(stdout, stderr, baseConfig, configPath, logger)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		return fmt.Errorf("execute root command: %w", err)
	}
	return nil
}

func newRootCmd(stdout io.Writer, stderr io.Writer, baseConfig ariadne.Config, configPath string, logger *cliLogger) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ariadne",
		Short:         "Resolve music URLs across services.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.PersistentFlags().String("config", configPath, "configuration source (values: empty string to disable file loading, or a path to an .env, yaml, yml, json, or toml file)")
	cmd.PersistentFlags().String("log-level", defaultCLILogLevel.String(), "CLI log level (values: error, warn, info, debug; debug prints effective config including secrets)")
	cmd.AddCommand(newResolveCmd(baseConfig, configPath, logger))
	return cmd
}

func newResolveCmd(baseConfig ariadne.Config, configPath string, logger *cliLogger) *cobra.Command {
	config := defaultResolveConfig(baseConfig)

	cmd := &cobra.Command{
		Use:   defaultResolveCommandUse,
		Short: "Resolve one music URL into likely equivalents on other services.",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errResolveUsage
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			config.inputURL = args[0]
			normalized, err := normalizeAndValidateResolveConfig(config)
			if err != nil {
				return err
			}
			return executeResolve(normalized, cmd.OutOrStdout(), logger, resolveModeFromConfig(normalized))
		},
	}

	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		_, _ = io.WriteString(cmd.OutOrStdout(), resolveHelpTextFor(baseConfig, configPath))
	})

	bindResolveFlags(cmd.Flags(), &config)
	return cmd
}
