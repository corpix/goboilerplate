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
		&cli.BoolFlag{
			Name:  "profile",
			Usage: "write profile information for debugging(cpu.prof, heap.prof)",
		},
		&cli.BoolFlag{
			Name:  "trace",
			Usage: "write trace information for debugging(trace.prof)",
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

	c *di.Container
)

func Before(ctx *cli.Context) error {
	var err error

	c = di.New()

	err = c.Provide(func() *cli.Context { return ctx })
	if err != nil {
		return err
	}

	err = c.Provide(func(ctx *cli.Context) (*config.Config, error) {
		return config.Load(ctx.String("config"))
	})
	if err != nil {
		return err
	}

	err = c.Provide(func(c *config.Config) (log.Logger, error) {
		return log.Create(*c.Log)
	})
	if err != nil {
		return err
	}

	if ctx.Bool("profile") {
		err = c.Invoke(writeProfile)
		if err != nil {
			return err
		}
	}

	if ctx.Bool("trace") {
		err = c.Invoke(writeTrace)
		if err != nil {
			return err
		}
	}

	return nil
}

func ConfigShowDefaultAction(ctx *cli.Context) error {
	c, err := config.Default()
	if err != nil {
		return err
	}

	return config.Show(c)
}

func ConfigShowAction(ctx *cli.Context) error {
	c, err := config.Load(ctx.String("config"))
	if nil != err {
		return err
	}

	return config.Show(c)
}

func RootAction(ctx *cli.Context) error {
	return c.Invoke(func(c *config.Config, l log.Logger) {
		l.Info().Msg("running")

		// FIXME: start your app here
		select {}
	})
}

func NewApp() *cli.App {
	app := &cli.App{}
	app.Before = Before
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
