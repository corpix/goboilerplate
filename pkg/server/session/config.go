package session

import (
	"sort"
	"time"

	"git.backbone/corpix/goboilerplate/pkg/errors"
)

type Config struct {
	Container *ContainerConfig `yaml:"container"`
	MaxAge    time.Duration    `yaml:"max-age"`
	Refresh   time.Duration    `yaml:"refresh"`
	Cookie    *CookieConfig    `yaml:"cookie"`
}

func (c *Config) Default() {
loop:
	for {
		switch {
		case c.Container == nil:
			c.Container = &ContainerConfig{}
		case c.MaxAge <= 0:
			c.MaxAge = 7 * 24 * time.Hour
		case c.Refresh <= 0:
			c.Refresh = 3 * time.Hour
		case c.Cookie == nil:
			c.Cookie = &CookieConfig{}
		default:
			break loop
		}
	}
}

//

type CookieConfig struct {
	Name     string `yaml:"name"`
	Path     string `yaml:"path"`
	Domain   string `yaml:"domain"`
	Secure   *bool  `yaml:"secure"`
	HTTPOnly *bool  `yaml:"httponly"`
	SameSite string `yaml:"same-site"`
}

func (c *CookieConfig) Default() {
loop:
	for {
		switch {
		case c.Name == "":
			c.Name = Name
		case c.Path == "":
			c.Path = "/"
		case c.Secure == nil:
			b := true
			c.Secure = &b
		case c.HTTPOnly == nil:
			b := true
			c.HTTPOnly = &b
		case c.SameSite == "":
			c.SameSite = SameSiteDefault
		default:
			break loop
		}
	}
}

func (c *CookieConfig) Validate() error {
	if _, ok := SameSite[c.SameSite]; !ok {
		available := make([]string, len(SameSite))
		n := 0
		for k := range SameSite {
			available[n] = k
			n++
		}
		sort.Strings(available)

		return errors.Errorf(
			"unexpected same-site value %q, expected one of: %q",
			c.SameSite, available,
		)
	}
	return nil
}
