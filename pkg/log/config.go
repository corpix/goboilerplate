package log

import (
	"git.backbone/corpix/goboilerplate/pkg/errors"
)

var (
	DefaultConfig = Config{
		Level: "info",
	}
)

type Config struct {
	Level string
}

func (c *Config) SetDefaults() {
loop:
	for {
		switch {
		case c.Level == "":
			c.Level = DefaultConfig.Level
		default:
			break loop
		}
	}
}

func (c Config) Validate() error {
	if c.Level == "" {
		return errors.New("level should not be empty")
	}
	return nil
}
