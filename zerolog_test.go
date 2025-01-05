package xlog

import (
	"log/slog"
	"os"

	"github.com/rs/zerolog"
)

func ExampleNewZerologHandler() {
	logger := zerolog.New(os.Stdout)
	handler := NewZerologHandler(logger, &HandlerOptions{SkipTime: true})
	log := slog.New(handler)

	log.
		With(slog.Int("id", 1)).
		WithGroup("bro").
		With(slog.Int("bro_id", 2)).
		WithGroup("group_1").
		With(slog.Int("group_id", 3)).
		Warn("run", slog.String("who", "forest"))
	// Output: {"level":"warn","id":1,"bro":{"bro_id":2,"group_1":{"group_id":3,"who":"forest"}},"message":"run"}
}
