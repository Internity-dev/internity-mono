// Package logger configures the process-wide structured logger (zerolog).
// One line per request is emitted by the request-logging middleware, with
// request_id/user_id/method/path/status/latency fields for grep-ability.
package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Init(env string) {
	zerolog.TimeFieldFormat = time.RFC3339
	var out zerolog.Logger
	if env == "development" {
		out = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.Kitchen}).With().Timestamp().Logger()
	} else {
		out = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}
	log.Logger = out
}
