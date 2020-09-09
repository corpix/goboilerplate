package config

import (
	"os"

	"github.com/corpix/revip"
	yaml "gopkg.in/yaml.v2"

	"git.backbone/corpix/goboilerplate/pkg/errors"
	"git.backbone/corpix/goboilerplate/pkg/log"
)

const (
	EnvironPrefix string = "GOBOILERPLATE"
)

var (
	Default = &Config{}

	YamlUnmarshaler = revip.YamlUnmarshaler
	Unmarshal       = revip.Unmarshal
	FromReader      = revip.FromReader
	FromEnviron     = revip.FromEnviron

	NewYamlEncoder = yaml.NewEncoder
)

func init() { Default.SetDefaults() }

type Config struct {
	Log log.Config
}

func (c *Config) SetDefaults() {
	c.Log.SetDefaults()
}

func (c *Config) Validate() error {
	var err error

	err = c.Log.Validate()
	if err != nil {
		return errors.Wrap(err, "failed to validate log config")
	}

	return nil
}

//

func Load(path string) (*Config, error) {
	c := &Config{}
	err := Parse(c, path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse configuration")
	}
	c.SetDefaults()

	err = c.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to validate configuration")
	}

	return c, nil
}

func Parse(ptr *Config, path string) error {
	fd, err := os.Open(path)
	if nil != err {
		return err
	}
	defer fd.Close()
	_, err = Unmarshal(
		ptr,
		FromReader(fd, YamlUnmarshaler),
		FromEnviron(EnvironPrefix),
	)
	return err
}
