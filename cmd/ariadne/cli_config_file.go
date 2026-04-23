package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/xmbshwll/ariadne"
)

func configPathFromArgs(args []string) string {
	if value, ok := namedFlagValueFromArgs(args, "--config"); ok {
		return value
	}
	return defaultConfigPath
}

func loadCLIConfigWithLogger(configPath string, logger *cliLogger) (ariadne.Config, error) {
	cfg := ariadne.DefaultConfig()
	v := viper.New()
	v.AutomaticEnv()

	trimmedConfigPath := strings.TrimSpace(configPath)
	if trimmedConfigPath == "" {
		logger.Debugf("cli config file loading disabled")
	} else {
		v.SetConfigFile(trimmedConfigPath)
		if looksLikeEnvFile(trimmedConfigPath) {
			v.SetConfigType("env")
		}
		if err := v.ReadInConfig(); err != nil {
			var notFound viper.ConfigFileNotFoundError
			if !errors.As(err, &notFound) && !errors.Is(err, os.ErrNotExist) {
				return ariadne.Config{}, fmt.Errorf("load config %q: %w", trimmedConfigPath, err)
			}
			logger.Debugf("cli config file not found path=%q", trimmedConfigPath)
		} else {
			logger.Debugf("config file loaded path=%q", v.ConfigFileUsed())
		}
	}

	trimmedValue := func(key string) string {
		return strings.TrimSpace(v.GetString(key))
	}
	logRawCLIConfigValues(logger, trimmedValue)

	httpTimeout := trimmedValue("ARIADNE_HTTP_TIMEOUT")
	if httpTimeout != "" {
		parsedTimeout, err := time.ParseDuration(httpTimeout)
		if err != nil {
			return ariadne.Config{}, fmt.Errorf("parse ARIADNE_HTTP_TIMEOUT %q: %w", httpTimeout, err)
		}
		if parsedTimeout <= 0 {
			return ariadne.Config{}, fmt.Errorf("invalid ARIADNE_HTTP_TIMEOUT %q: %w", httpTimeout, errNonPositiveCLIHTTPTimeout)
		}
		cfg.HTTPTimeout = parsedTimeout
	}

	loaded := ariadne.LoadConfigFromEnv(trimmedValue)
	loaded.HTTPTimeout = cfg.HTTPTimeout
	logNormalizedCLIConfig(logger, loaded)
	return loaded, nil
}

func looksLikeEnvFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasPrefix(base, ".env") || strings.EqualFold(filepath.Ext(base), ".env")
}
