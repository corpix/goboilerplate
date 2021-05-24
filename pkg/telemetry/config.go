package telemetry

import (
	"time"

	"git.backbone/corpix/goboilerplate/pkg/bus"
	"git.backbone/corpix/goboilerplate/pkg/errors"
)

type Config struct {
	Enable  bool
	Addr    string
	Path    string
	Timeout *TimeoutConfig
}

func (c *Config) Default() {
loop:
	for {
		switch {
		case c.Addr == "":
			c.Addr = "127.0.0.1:8877"
		case c.Path == "":
			c.Path = "/"
		case c.Timeout == nil:
			c.Timeout = &TimeoutConfig{}
		default:
			break loop
		}
	}
}

func (c *Config) Validate() error {
	if !c.Enable {
		return nil
	}

	if c.Addr == "" {
		return errors.New("addr should not be empty")
	}
	if c.Path == "" {
		return errors.New("path should not be empty")
	}

	return nil
}

func (c *Config) Update(cc interface{}) error {
	bus.Config <- bus.ConfigUpdate{
		Subsystem: Subsystem,
		Config:    cc,
	}
	return nil
}

//

type TimeoutConfig struct {
	Read  time.Duration
	Write time.Duration
}

func (c *TimeoutConfig) Default() {
loop:
	for {
		switch {
		case c.Read <= 0:
			c.Read = 5 * time.Second
		case c.Write <= 0:
			c.Write = 5 * time.Second
		default:
			break loop
		}
	}
}
