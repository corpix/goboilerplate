package csrf

import (
	"io/ioutil"
	"time"

	"git.backbone/corpix/goboilerplate/pkg/errors"
)

type Config struct {
	Key     string `yaml:"key"`
	KeyFile string `yaml:"key-file"`
	key     []byte

	TTL           time.Duration `yaml:"ttl"`
	ParameterName string        `yaml:"parameter-name"`
}

func (c *Config) Default() {
loop:
	for {
		switch {
		case c.TTL == 0:
			c.TTL = 6 * time.Hour
		case c.ParameterName == "":
			c.ParameterName = ParameterName
		default:
			break loop
		}
	}
}

func (c *Config) Expand() error {
	var err error

	if c.KeyFile != "" {
		c.key, err = ioutil.ReadFile(c.KeyFile)
		if err != nil {
			return errors.Wrapf(
				err, "failed to load key-file: %q",
				c.KeyFile,
			)
		}
	} else {
		c.key = []byte(c.Key)
	}

	return nil
}

func (c *Config) Validate() error {
	if c.Key != "" && c.KeyFile != "" {
		return errors.New("either key or key-file should be defined, not both")
	}
	if c.Key == "" && c.KeyFile == "" {
		return errors.New("either key or key-file should be defined")
	}
	if len(c.key) == 0 {
		return errors.New("key length should be greater than zero")
	}
	return nil
}
