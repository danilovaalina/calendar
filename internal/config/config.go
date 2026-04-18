package config

import (
	"time"

	"github.com/cockroachdb/errors"
	"github.com/spf13/viper"
)

type Config struct {
	Addr        string        `mapstructure:"addr"`
	LogFile     string        `mapstructure:"log_file"`
	BufferSize  int           `mapstructure:"buffer_size"`
	Interval    time.Duration `mapstructure:"interval"`
	NotifierURL string        `mapstructure:"notifier_url"`
}

func Load() (Config, error) {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	v.SetDefault("addr", ":8080")
	v.SetDefault("buffer_size", 100)

	if err := v.ReadInConfig(); err != nil {
		if !errors.As(err, &viper.ConfigFileNotFoundError{}) {
			return Config{}, errors.WithStack(err)
		}
	}

	_ = v.BindEnv("log_file", "LOG_FILE")
	_ = v.BindEnv("addr", "ADDR")
	_ = v.BindEnv("buffer_size", "BUFFER_SIZE")
	_ = v.BindEnv("interval", "INTERVAL")
	_ = v.BindEnv("notifier_url", "NOTIFIER_URL")

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, errors.Wrap(err, "unable to decode into config struct")
	}

	return cfg, nil
}
