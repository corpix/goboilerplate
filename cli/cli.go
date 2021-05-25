package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	watchdog "github.com/cloudflare/tableflip"
	"github.com/corpix/revip"
	cli "github.com/urfave/cli/v2"
	di "go.uber.org/dig"

	"git.backbone/corpix/goboilerplate/pkg/bus"
	"git.backbone/corpix/goboilerplate/pkg/config"
	"git.backbone/corpix/goboilerplate/pkg/errors"
	"git.backbone/corpix/goboilerplate/pkg/log"
	"git.backbone/corpix/goboilerplate/pkg/meta"
	"git.backbone/corpix/goboilerplate/pkg/telemetry"
)

var (
	Stdout = os.Stdout
	Stderr = os.Stderr

	Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "pid-file",
			Aliases: []string{"p"},
			EnvVars: []string{config.EnvironPrefix + "_PID_FILE"},
			Usage:   "path to pid file to report into",
			Value:   meta.Name + ".pid",
		},
		&cli.StringFlag{
			Name:    "log-level",
			Aliases: []string{"l"},
			Usage:   "logging level (debug, info, error)",
			Value:   "info",
		},
		&cli.StringSliceFlag{
			Name:    "config",
			Aliases: []string{"c"},
			EnvVars: []string{config.EnvironPrefix + "_CONFIG"},
			Usage:   "path to application configuration file/files (separate multiple files with comma)",
			Value:   cli.NewStringSlice("config.yml"),
		},

		//

		&cli.DurationFlag{
			Name:  "duration",
			Usage: "exit after duration",
		},
		&cli.BoolFlag{
			Name:  "profile",
			Usage: "write profile information for debugging (cpu.prof, heap.prof)",
		},
		&cli.BoolFlag{
			Name:  "trace",
			Usage: "write trace information for debugging (trace.prof)",
		},
	}
	Commands = []*cli.Command{
		&cli.Command{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Configuration tools",
			Subcommands: []*cli.Command{
				&cli.Command{
					Name:    "show-default",
					Aliases: []string{"sd"},
					Usage:   "Show default configuration",
					Action:  ConfigShowDefaultAction,
				},
				&cli.Command{
					Name:    "show",
					Aliases: []string{"s"},
					Usage:   "Show default configuration",
					Action:  ConfigShowAction,
				},
				&cli.Command{
					Name:    "validate",
					Aliases: []string{"v"},
					Usage:   "Validate configuration and exit",
					Action:  ConfigValidateAction,
				},
				&cli.Command{
					Name:      "push",
					Aliases:   []string{"p"},
					Usage:     "Push configuration to specified destination",
					Action:    ConfigPushAction,
					ArgsUsage: "<destination>[,...]",
				},
			},
		},
	}

	c *di.Container
)

func Before(ctx *cli.Context) error {
	var err error

	c = di.New()

	//

	err = c.Provide(func() *cli.Context { return ctx })
	if err != nil {
		return err
	}

	err = c.Provide(func(ctx *cli.Context) (*config.Config, error) {
		c, err := config.Load(ctx.StringSlice("config"))
		if err != nil {
			return nil, err
		}

		return c, nil
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
	err = c.Provide(func() *telemetry.Registry {
		return telemetry.DefaultRegistry
	})
	if err != nil {
		return err
	}

	//

	duration := ctx.Duration("duration")
	if duration == 0 {
		err = c.Provide(func(ctx *cli.Context) context.Context {
			return context.Background()
		})
	} else {
		err = c.Provide(func(ctx *cli.Context) context.Context {
			c, _ := context.WithTimeout(context.Background(), duration)
			return c
		})
	}
	if err != nil {
		return err
	}

	//

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

//

func ConfigShowDefaultAction(ctx *cli.Context) error {
	c, err := config.Default()
	if err != nil {
		return err
	}

	write := revip.ToWriter(os.Stdout, config.Marshaler)

	return write(c)
}

func ConfigShowAction(ctx *cli.Context) error {
	return c.Invoke(func(c *config.Config) error {
		write := revip.ToWriter(os.Stdout, config.Marshaler)

		return write(c)
	})
}

func ConfigValidateAction(ctx *cli.Context) error {
	return c.Invoke(func(l log.Logger) error {
		configs := ctx.StringSlice("config")
		c, err := config.Load(
			configs,
			config.InitPostprocessors...,
		)
		if err != nil {
			return err
		}

		err = config.Validate(c)
		if err != nil {
			return err
		}

		l.Info().
			Strs("configs", configs).
			Msg("configuration validation is ok")

		return nil
	})
}

func ConfigPushAction(ctx *cli.Context) error {
	return c.Invoke(func(l log.Logger) error {
		configs := ctx.StringSlice("config")
		c, err := config.Load(
			configs,
			config.LocalPostprocessors...,
		)
		if err != nil {
			return err
		}

		args := ctx.Args().Slice()
		if len(args) < 1 {
			return errors.New("subcommand requires an argument, example: etcd://127.0.0.1:2379/prefix,file://./config.out.yml")
		}

		destinations := args
		for _, destination := range destinations {
			push, err := revip.ToURL(destination, config.Marshaler)
			if err != nil {
				return err
			}

			err = push(c)
			if err != nil {
				return err
			}
		}

		l.Info().
			Strs("configs", configs).
			Strs("destinations", destinations).
			Msg("configuration pushed")

		return nil
	})
}

//

func RootAction(ctx *cli.Context) error {
	var err error

	err = c.Invoke(func(l log.Logger) {
		l.Info().Msg("running")
	})
	if err != nil {
		return err
	}

	return c.Invoke(func(ctx *cli.Context, cfg *config.Config) error {
		var (
			err  error
			errc = make(chan error, 1)
			w    *watchdog.Upgrader
		)

		w, err = watchdog.New(watchdog.Options{
			UpgradeTimeout: cfg.ShutdownGraceTime,
			PIDFile:        ctx.String("pid-file"),
		})
		if err != nil {
			return err
		}

		//

		return c.Invoke(func(ctx context.Context, cfg *config.Config, l log.Logger, r *telemetry.Registry) error {
			var t *telemetry.Server

			if cfg.Telemetry.Enable {
				lr, err := w.Listen("tcp", cfg.Telemetry.Addr)
				if err != nil {
					return err
				}
				t = telemetry.New(*cfg.Telemetry, l, r, lr)

				go func() {
					errc <- errors.Wrap(t.ListenAndServe(), "failed while listen and serve telemetry server")
				}()
			}

			//

			var (
				sig = make(chan os.Signal, 1)
				err error
			)

			signal.Notify(
				sig,
				syscall.SIGINT,
				syscall.SIGQUIT,
				syscall.SIGTERM,
				syscall.SIGUSR2,
				syscall.SIGHUP,
			)

			err = w.Ready()
			if err != nil {
				return err
			}

			//

		loop:
			for {
				select {
				case <-w.Exit():
					break loop
				case <-ctx.Done():
					break loop

				case err := <-errc:
					if err != nil {
						return err
					}
				case si := <-sig:
					l.Info().Str("signal", si.String()).Msg("received signal")
					switch si {
					case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
						w.Stop()
					case syscall.SIGUSR2, syscall.SIGHUP:
						err = w.Upgrade()
						if err != nil {
							return err
						}
					}
				case <-bus.Config:
					err = w.Upgrade()
					if err != nil {
						return err
					}
				}
			}

			//

			defer os.Exit(0)
			l.Info().Msg("shutdown watchdog")

			time.AfterFunc(cfg.ShutdownGraceTime, func() {
				l.Warn().
					Dur("graceTime", cfg.ShutdownGraceTime).
					Msg("Graceful shutdown timed out")
				os.Exit(1)
			})

			//

			if cfg.Telemetry.Enable {
				err = t.Shutdown(context.Background())
				if err != nil {
					return errors.Wrap(err, "telemetry shutdown failed")
				}
			}

			return nil
		})
	})
}

//

func NewApp() *cli.App {
	app := &cli.App{}

	app.Before = Before
	app.Flags = Flags
	app.Action = RootAction
	app.Commands = Commands
	app.Version = meta.Version

	return app
}

func Run() {
	err := NewApp().Run(os.Args)
	if err != nil {
		errors.Fatal(errors.Wrap(
			err, fmt.Sprintf(
				"pid: %d, ppid: %d",
				os.Getpid(), os.Getppid(),
			),
		))
	}
}
