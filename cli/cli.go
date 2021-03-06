package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	watchdog "github.com/cloudflare/tableflip"
	revip "github.com/corpix/revip"
	spew "github.com/davecgh/go-spew/spew"
	cli "github.com/urfave/cli/v2"
	di "go.uber.org/dig"

	"git.backbone/corpix/goboilerplate/pkg/bus"
	"git.backbone/corpix/goboilerplate/pkg/config"
	"git.backbone/corpix/goboilerplate/pkg/crypto"
	"git.backbone/corpix/goboilerplate/pkg/crypto/container"
	"git.backbone/corpix/goboilerplate/pkg/errors"
	"git.backbone/corpix/goboilerplate/pkg/log"
	"git.backbone/corpix/goboilerplate/pkg/meta"
	"git.backbone/corpix/goboilerplate/pkg/reflect"
	"git.backbone/corpix/goboilerplate/pkg/server/csrf"
	"git.backbone/corpix/goboilerplate/pkg/server/session"
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
			Usage:   "logging level (debug, info, warn, error)",
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
		{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Configuration tools",
			Subcommands: []*cli.Command{
				{
					Name:    "show-default",
					Aliases: []string{"sd"},
					Usage:   "Show default configuration",
					Action:  ConfigShowDefaultAction,
				},
				{
					Name:    "show",
					Aliases: []string{"s"},
					Usage:   "Show default configuration",
					Action:  ConfigShowAction,
				},
				{
					Name:    "validate",
					Aliases: []string{"v"},
					Usage:   "Validate configuration and exit",
					Action:  ConfigValidateAction,
				},
				{
					Name:      "push",
					Aliases:   []string{"p"},
					Usage:     "Push configuration to specified destination",
					Action:    ConfigPushAction,
					ArgsUsage: "<destination>[,...]",
				},
			},
		},
		{
			Name:    "server",
			Aliases: []string{"s"},
			Usage:   "Server tools",
			Subcommands: []*cli.Command{
				{
					Name:    "session",
					Aliases: []string{"s"},
					Usage:   "Session tools",
					Subcommands: []*cli.Command{
						{
							Name:      "show",
							Aliases:   []string{"s"},
							Usage:     "Show session passed as argument (if empty will read from stdin)",
							ArgsUsage: "[session]",
							Action:    ServerSessionShowAction,
							Flags: []cli.Flag{
								&cli.StringFlag{
									Name:    "container",
									Aliases: []string{"c"},
									Usage:   fmt.Sprintf("container type, one of %v", reflect.IndirectValue(reflect.ValueOf(container.Types)).MapKeys()),
									Value:   string(container.SecretBoxType),
								},
								&cli.StringFlag{
									Name:    "serializer",
									Aliases: []string{"s"},
									Usage:   fmt.Sprintf("serializer type, one of %v", reflect.IndirectValue(reflect.ValueOf(container.SerializerTypes)).MapKeys()),
								},
								&cli.StringFlag{
									Name:    "sealer",
									Aliases: []string{"e"},
									Usage:   fmt.Sprintf("sealer type, one of %v", reflect.IndirectValue(reflect.ValueOf(container.SealerTypes)).MapKeys()),
								},
								&cli.StringFlag{
									Name:    "compressor",
									Aliases: []string{"d"},
									Usage:   fmt.Sprintf("compressor type, one of %v", reflect.IndirectValue(reflect.ValueOf(container.CompressorTypes)).MapKeys()),
								},
								&cli.StringFlag{
									Name:    "representer",
									Aliases: []string{"p"},
									Usage:   fmt.Sprintf("representer type, one of %v", reflect.IndirectValue(reflect.ValueOf(container.RepresenterTypes)).MapKeys()),
								},
								&cli.StringFlag{
									Name:     "key",
									Aliases:  []string{"k"},
									Usage:    "encryption key",
									Required: true,
								},
								&cli.BoolFlag{
									Name:    "json",
									Aliases: []string{"j"},
									Usage:   "use json format",
								},
								&cli.StringFlag{
									Name:  "jwt-algo",
									Usage: "jwt algorithm to use",
								},
							},
						},
						{
							Name:      "validate",
							Aliases:   []string{"v"},
							Usage:     "Validate session passed as argument (if empty will read from stdin)",
							ArgsUsage: "[session]",
							Action:    ServerSessionValidateAction,
							Flags: []cli.Flag{
								&cli.StringFlag{
									Name:    "container",
									Aliases: []string{"c"},
									Usage:   fmt.Sprintf("container type, one of %v", reflect.IndirectValue(reflect.ValueOf(container.Types)).MapKeys()),
									Value:   string(container.SecretBoxType),
								},
								&cli.StringFlag{
									Name:    "serializer",
									Aliases: []string{"s"},
									Usage:   fmt.Sprintf("serializer type, one of %v", reflect.IndirectValue(reflect.ValueOf(container.SerializerTypes)).MapKeys()),
								},
								&cli.StringFlag{
									Name:    "sealer",
									Aliases: []string{"e"},
									Usage:   fmt.Sprintf("sealer type, one of %v", reflect.IndirectValue(reflect.ValueOf(container.SealerTypes)).MapKeys()),
								},
								&cli.StringFlag{
									Name:    "compressor",
									Aliases: []string{"d"},
									Usage:   fmt.Sprintf("compressor type, one of %v", reflect.IndirectValue(reflect.ValueOf(container.CompressorTypes)).MapKeys()),
								},
								&cli.StringFlag{
									Name:    "representer",
									Aliases: []string{"p"},
									Usage:   fmt.Sprintf("representer type, one of %v", reflect.IndirectValue(reflect.ValueOf(container.RepresenterTypes)).MapKeys()),
								},
								&cli.StringFlag{
									Name:     "key",
									Aliases:  []string{"k"},
									Usage:    "encryption key",
									Required: true,
								},
								&cli.StringFlag{
									Name:  "jwt-algo",
									Usage: "jwt algorithm to use",
								},
							},
						},
					},
				},
				{
					Name:    "csrf",
					Aliases: []string{"t"},
					Usage:   "CSRF token tools",
					Subcommands: []*cli.Command{
						{
							Name:    "issue",
							Aliases: []string{"s"},
							Usage:   "Issue CSRF token",
							Action:  ServerCSRFIssueAction,
							Flags: []cli.Flag{
								&cli.StringFlag{
									Name:     "key",
									Aliases:  []string{"k"},
									EnvVars:  []string{config.EnvironPrefix + "_SERVER_CSRF_ISSUE_KEY"},
									Usage:    "encryption key",
									Required: true,
								},
								&cli.StringFlag{
									Name:    "source",
									Aliases: []string{"i"},
									EnvVars: []string{config.EnvironPrefix + "_SERVER_CSRF_ISSUE_SOURCE"},
									Usage:   "token source (initiator)",
								},
								&cli.StringFlag{
									Name:    "subject",
									Aliases: []string{"r"},
									EnvVars: []string{config.EnvironPrefix + "_SERVER_CSRF_ISSUE_SUBJECT"},
									Usage:   "token subject (receiver)",
								},
								&cli.DurationFlag{
									Name:    "ttl",
									Aliases: []string{"t"},
									EnvVars: []string{config.EnvironPrefix + "_SERVER_CSRF_ISSUE_TTL"},
									Value:   1 * time.Hour,
									Usage:   "token time to live",
								},
							},
						},
						{
							Name:      "show",
							Aliases:   []string{"s"},
							Usage:     "Show CSRF token passed as argument (if empty will read from stdin)",
							ArgsUsage: "[csrf]",
							Action:    ServerCSRFShowAction,
							Flags: []cli.Flag{
								&cli.StringFlag{
									Name:    "key",
									Aliases: []string{"k"},
									EnvVars: []string{config.EnvironPrefix + "_SERVER_CSRF_SHOW_KEY"},
									Usage:   "encryption key",
								},
								&cli.BoolFlag{
									Name:    "json",
									Aliases: []string{"j"},
									Usage:   "use json format",
								},
							},
						},
						{
							Name:      "validate",
							Aliases:   []string{"s"},
							Usage:     "Validate CSRF token passed as argument (if empty will read from stdin)",
							ArgsUsage: "[csrf]",
							Action:    ServerCSRFValidateAction,
							Flags: []cli.Flag{
								&cli.StringFlag{
									Name:    "key",
									Aliases: []string{"k"},
									EnvVars: []string{config.EnvironPrefix + "_SERVER_CSRF_VALIDATE_KEY"},
									Usage:   "encryption key",
								},
								&cli.StringFlag{
									Name:    "source",
									Aliases: []string{"i"},
									EnvVars: []string{config.EnvironPrefix + "_SERVER_CSRF_VALIDATE_SOURCE"},
									Usage:   "token source (initiator)",
								},
								&cli.StringFlag{
									Name:    "subject",
									Aliases: []string{"r"},
									EnvVars: []string{config.EnvironPrefix + "_SERVER_CSRF_VALIDATE_SUBJECT"},
									Usage:   "token subject (receiver)",
								},
							},
						},
					},
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

	err = c.Provide(func() *spew.ConfigState {
		return &spew.ConfigState{
			DisableMethods:          false,
			DisableCapacities:       true,
			DisablePointerAddresses: true,
			Indent:                  "  ",
			SortKeys:                true,
			SpewKeys:                false,
		}
	})
	if err != nil {
		return err
	}

	err = c.Provide(func() *json.Encoder {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc
	})
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

	err = c.Provide(func(ctx *cli.Context, c *config.Config) (log.Logger, error) {
		lc := *c.Log
		level := ctx.String("log-level")
		if level != "" {
			lc.Level = level
		}

		return log.Create(lc)
	})
	if err != nil {
		return err
	}

	err = c.Provide(func() crypto.Rand { return crypto.DefaultRand })
	if err != nil {
		return err
	}
	err = c.Provide(func() *telemetry.Registry { return telemetry.DefaultRegistry })
	if err != nil {
		return err
	}

	//

	err = c.Provide(func(
		c *config.Config,
		l log.Logger,
		r *telemetry.Registry,
		w *watchdog.Upgrader,
		running *sync.WaitGroup,
		errc chan error,
	) (*telemetry.Server, error) {
		start := func(t *telemetry.Server) {
			errc <- errors.Wrap(
				t.ListenAndServe(),
				"failed while listen and serve telemetry server",
			)
		}

		finalize := func(t *telemetry.Server) {
			defer running.Done()
			<-w.Exit()

			err = t.Shutdown(context.Background())
			if err != nil {
				panic(errors.Wrap(err, "telemetry shutdown failed"))
			}
		}

		if c.Telemetry.Enable {
			lr, err := w.Listen("tcp", c.Telemetry.Addr)
			if err != nil {
				return nil, err
			}
			t, err := telemetry.New(*c.Telemetry, l, r, lr)
			if err != nil {
				return nil, err
			}

			running.Add(1)

			go start(t)
			go finalize(t)

			return t, nil
		}

		return nil, nil
	})
	if err != nil {
		return err
	}

	//

	err = c.Provide(func(ctx *cli.Context, c *config.Config) (*watchdog.Upgrader, error) {
		return watchdog.New(watchdog.Options{
			UpgradeTimeout: c.ShutdownGraceTime,
			PIDFile:        ctx.String("pid-file"),
		})
	})
	if err != nil {
		return err
	}

	err = c.Provide(func() *sync.WaitGroup { return &sync.WaitGroup{} })
	if err != nil {
		return err
	}

	err = c.Provide(func() chan error { return make(chan error, 1) })
	if err != nil {
		return err
	}

	err = c.Provide(func() chan os.Signal {
		sig := make(chan os.Signal, 1)
		signal.Notify(
			sig,
			syscall.SIGQUIT,
			syscall.SIGTERM,
			syscall.SIGINT,
			syscall.SIGUSR1,
			syscall.SIGUSR2,
			syscall.SIGHUP,
		)
		return sig
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
			c, cancel := context.WithTimeout(context.Background(), duration)
			go func() {
				<-c.Done()
				cancel()
			}()
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
			return errors.New("subcommand requires an argument, example: ./config.out.yml")
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

func ServerSessionShowAction(ctx *cli.Context) error {
	return c.Invoke(func(rand crypto.Rand, enc *json.Encoder, debug *spew.ConfigState) error {
		key := ctx.String("key")

		c := &container.Config{
			Type:        ctx.String("container"),
			Serializer:  ctx.String("serializer"),
			Sealer:      ctx.String("sealer"),
			Compressor:  ctx.String("compressor"),
			Representer: ctx.String("representer"),
			Key:         key,
			Jwt:         &container.JwtConfig{Algo: ctx.String("jwt-algo")},
		}
		sc := &session.Config{Container: c}
		err := config.Postprocess(sc)
		if err != nil {
			return err
		}
		s, err := session.New(*sc, rand)
		if err != nil {
			return err
		}

		buf := []byte(ctx.Args().First())
		if len(buf) == 0 {
			buf, err = ioutil.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
		}

		err = s.Load(buf)
		if err != nil {
			return err
		}

		if ctx.Bool("json") {
			err = enc.Encode(s.Data())
			if err != nil {
				return err
			}
			os.Stdout.Write([]byte("\n"))
		} else {
			debug.Dump(s.Data())
		}

		return nil
	})
}

func ServerSessionValidateAction(ctx *cli.Context) error {
	return c.Invoke(func(rand crypto.Rand, enc *json.Encoder, debug *spew.ConfigState) error {
		key := ctx.String("key")

		c := &container.Config{
			Type:        ctx.String("container"),
			Serializer:  ctx.String("serializer"),
			Sealer:      ctx.String("sealer"),
			Compressor:  ctx.String("compressor"),
			Representer: ctx.String("representer"),
			Key:         key,
			Jwt:         &container.JwtConfig{Algo: ctx.String("jwt-algo")},
		}
		err := config.Postprocess(c)
		if err != nil {
			return err
		}

		sc := session.Config{Container: c}
		sc.Container.Default()
		s, err := session.New(sc, rand)
		if err != nil {
			return err
		}

		buf := []byte(ctx.Args().First())
		if len(buf) == 0 {
			buf, err = ioutil.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
		}

		err = s.Load(buf)
		if err != nil {
			return err
		}

		err = s.Validate()
		if err == nil {
			fmt.Println("session validation succeeded")
		} else {
			fmt.Println("session validation failed:", err)
			os.Exit(1)
		}

		return nil
	})
}

func ServerCSRFIssueAction(ctx *cli.Context) error {
	return c.Invoke(func(rand crypto.Rand) error {
		key := ctx.String("key")
		source := ctx.String("source")
		subject := ctx.String("subject")
		ttl := ctx.Duration("ttl")

		c := &csrf.Config{
			Key: key,
			TTL: ttl,
		}
		err := config.Postprocess(c)
		if err != nil {
			return err
		}

		t, err := csrf.New(*c, rand)
		if err != nil {
			return err
		}

		token, err := t.Sign(source, subject)
		if err != nil {
			return err
		}

		fmt.Println(token)

		return nil
	})
}

func ServerCSRFShowAction(ctx *cli.Context) error {
	return c.Invoke(func(rand crypto.Rand, enc *json.Encoder, debug *spew.ConfigState) error {
		key := ctx.String("key")

		c := &csrf.Config{Key: key}
		err := config.Postprocess(c)
		if err != nil {
			return err
		}

		t, err := csrf.New(*c, rand)
		if err != nil {
			return err
		}

		buf := []byte(ctx.Args().First())
		if len(buf) == 0 {
			buf, err = ioutil.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
		}

		token, err := t.Unpack(csrf.Token(buf))
		if err != nil {
			return err
		}

		if ctx.Bool("json") {
			err = enc.Encode(token.Data())
			if err != nil {
				return err
			}
			os.Stdout.Write([]byte("\n"))
		} else {
			debug.Dump(token.Data())
		}

		return nil
	})
}

func ServerCSRFValidateAction(ctx *cli.Context) error {
	return c.Invoke(func(rand crypto.Rand, enc *json.Encoder, debug *spew.ConfigState) error {
		key := ctx.String("key")
		source := ctx.String("source")
		subject := ctx.String("subject")

		c := &csrf.Config{Key: key}
		err := config.Postprocess(c)
		if err != nil {
			return err
		}

		t, err := csrf.New(*c, rand)
		if err != nil {
			return err
		}

		buf := []byte(ctx.Args().First())
		if len(buf) == 0 {
			buf, err = ioutil.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
		}

		err = t.Validate(source, subject, string(buf))
		if err == nil {
			fmt.Println("token validation succeeded")
		} else {
			fmt.Println("token validation failed", err)
			os.Exit(1)
		}

		return nil
	})
}

//

func RootAction(ctx *cli.Context) error {
	components := c.String()
	_ = c.Invoke(func(l log.Logger) {
		l.Trace().Msgf(
			"component graph: %s",
			strings.TrimSpace(components),
		)
	})

	return c.Invoke(func(
		ctx context.Context,
		cfg *config.Config,
		w *watchdog.Upgrader,
		l log.Logger,
		t *telemetry.Server,
		running *sync.WaitGroup,
		errc chan error,
		sig chan os.Signal,
	) error {
		l.Info().Msg("running")

		err := w.Ready()
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
				w.Stop()
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
				case syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGHUP:
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
				Msg("graceful shutdown timed out")
			os.Exit(1)
		})

		running.Wait() // wait for other running components to finish

		return nil
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
