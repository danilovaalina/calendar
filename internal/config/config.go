package config

import (
	"github.com/cockroachdb/errors"
	"github.com/spf13/viper"
)

type Config struct {
	Addr    string `mapstructure:"addr"`
	LogFile string `mapstructure:"log_file"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		if !errors.Is(err, viper.ConfigFileNotFoundError{}) {
			return nil, errors.Wrap(err, "error reading config file")
		}
	}

	viper.AutomaticEnv()
	err := viper.BindEnv("log_file", "LOG_FILE")
	if err != nil {
		return nil, errors.Wrap(err, "failed to bind env var")
	}

	err = viper.BindEnv("addr", "ADDR")
	if err != nil {
		return nil, errors.Wrap(err, "failed to bind env var")
	}

	var cfg Config
	if err = viper.Unmarshal(&cfg); err != nil {
		return nil, errors.Wrap(err, "unable to decode into config struct")
	}

	return &cfg, nil
}
