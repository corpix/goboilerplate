package log

import (
	"io"
	"os"

	console "github.com/mattn/go-isatty"
	"github.com/rs/zerolog"

	"git.backbone/corpix/goboilerplate/pkg/errors"
)

type (
	Level  = zerolog.Level
	Logger = zerolog.Logger
	Event  = zerolog.Event
)

func Create(c Config) (Logger, error) {
	var (
		output = os.Stdout

		log   Logger
		level Level
		err   error
		w     io.Writer
	)

	if console.IsTerminal(output.Fd()) {
		w = zerolog.ConsoleWriter{Out: output}
	} else {
		w = output
	}

	level, err = zerolog.ParseLevel(c.Level)
	if err != nil {
		return log, errors.Wrap(err, "failed to parse logging level from config")
	}

	log = zerolog.New(w).With().
		Timestamp().Logger().
		Level(level)

	return log, nil
}
