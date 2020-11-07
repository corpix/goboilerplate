package config

import (
	"bytes"
	"os"

	"github.com/corpix/revip"
	yaml "gopkg.in/yaml.v2"

	"git.backbone/corpix/goboilerplate/pkg/log"
)

const (
	EnvironPrefix string = "GOBOILERPLATE"
)

var NewEncoder = yaml.NewEncoder

type Config struct {
	Log *log.Config
}

func (c *Config) Default() {
loop:
	for {
		switch {
		default:
			break loop
		}
	}
}

func (c *Config) Validate() error {
	return nil
}

//

func Default() (*Config, error) {
	c := &Config{}
	err := revip.Postprocess(
		c,
		revip.WithDefaults(),
	)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func Load(path string) (*Config, error) {
	c := &Config{}

	fd, err := os.Open(path)
	if nil != err {
		return nil, err
	}
	defer fd.Close()

	_, err = revip.Load(
		c,
		revip.FromReader(fd, revip.YamlUnmarshaler),
		revip.FromEnviron(EnvironPrefix),
	)
	if err != nil {
		return nil, err
	}

	err = revip.Postprocess(
		c,
		revip.WithDefaults(),
		revip.WithValidation(),
	)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func Encode(c *Config) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := NewEncoder(buf)
	defer enc.Close()

	err := enc.Encode(c)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func Show(c *Config) error {
	buf, err := Encode(c)
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(buf)
	return err
}
