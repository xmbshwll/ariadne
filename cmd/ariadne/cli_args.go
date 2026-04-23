package main

import (
	"slices"
	"strings"
)

func argsWithoutPersistentFlags(args []string) []string {
	return argsWithoutNamedFlags(args, "--config", "--log-level")
}

func argsWithoutNamedFlags(args []string, flags ...string) []string {
	filtered := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if matchesNamedFlag(arg, flags...) {
			if i+1 < len(args) {
				value := args[i+1]
				if value == "" || !strings.HasPrefix(value, "-") {
					i++
				}
			}
			continue
		}
		if matchesNamedFlagAssignment(arg, flags...) {
			continue
		}
		filtered = append(filtered, arg)
	}
	return filtered
}

func matchesNamedFlag(arg string, flags ...string) bool {
	return slices.Contains(flags, arg)
}

func matchesNamedFlagAssignment(arg string, flags ...string) bool {
	return slices.ContainsFunc(flags, func(flag string) bool {
		return strings.HasPrefix(arg, flag+"=")
	})
}

func firstCommandArg(commandArgs []string, args []string) string {
	if len(commandArgs) > 0 {
		return commandArgs[0]
	}
	if len(args) > 0 {
		return args[0]
	}
	return "<unknown>"
}

func containsHelpArg(args []string) bool {
	return slices.ContainsFunc(args, isHelpArg)
}

func isHelpArg(arg string) bool {
	return arg == "help" || arg == "--help" || arg == "-h"
}

func isUnknownCommandError(err error) bool {
	return strings.Contains(err.Error(), "unknown command ")
}
