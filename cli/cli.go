package cli

import (
	"os"

	cli "github.com/urfave/cli/v2"
	di "go.uber.org/dig"

	"git.backbone/corpix/goboilerplate/pkg/config"
	"git.backbone/corpix/goboilerplate/pkg/errors"
	"git.backbone/corpix/goboilerplate/pkg/log"
)

var (
	Version = "development"

	Stdout = os.Stdout
	Stderr = os.Stderr

	Flags = []cli.Flag{
		&cli.StringFlag{
			Name: "log-level",
			Aliases: []string{
				"l",
			},
			Usage: "logging level (debug, info, error)",
			Value: "info",
		},
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			EnvVars: []string{config.EnvironPrefix + "_CONFIG"},
			Usage:   "path to application configuration file",
			Value:   "config.yaml",
		},
	}
	Commands = []*cli.Command{
		&cli.Command{
			Name: "config",
			Aliases: []string{
				"c",
			},
			Usage: "Configuration Tools",
			Subcommands: []*cli.Command{
				&cli.Command{
					Name: "show-default",
					Aliases: []string{
						"sd",
					},
					Usage:  "Show default configuration",
					Action: ConfigShowDefaultAction,
				},
				&cli.Command{
					Name: "show",
					Aliases: []string{
						"s",
					},
					Usage:  "Show default configuration",
					Action: ConfigShowAction,
				},
			},
		},
	}
)

func ConfigShowDefaultAction(ctx *cli.Context) error {
	enc := config.NewYamlEncoder(Stdout)
	defer enc.Close()
	return enc.Encode(config.Default)
}
func ConfigShowAction(ctx *cli.Context) error {
	c, err := config.Load(ctx.String("config"))
	if nil != err {
		return err
	}
	enc := config.NewYamlEncoder(Stdout)
	defer enc.Close()
	return enc.Encode(c)
}
func RootAction(ctx *cli.Context) error {
	c := di.New()

	c.Provide(func() (*config.Config, error) {
		return config.Load(ctx.String("config"))
	})
	c.Provide(func(c *config.Config) (log.Logger, error) {
		return log.Create(c.Log)
	})

	return c.Invoke(func(l log.Logger) {
		l.Info().Msg("running")

		// FIXME: start your app here
		select {}
	})
}

func NewApp() *cli.App {
	app := &cli.App{}
	app.Flags = Flags
	app.Action = RootAction
	app.Commands = Commands
	return app
}

func Run() {
	err := NewApp().Run(os.Args)
	if err != nil {
		errors.Fatal(err)
	}
}
