package xslog

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"

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

func TestZerologHandlerEnabled(t *testing.T) {
	levels := []zerolog.Level{
		zerolog.DebugLevel,
		zerolog.InfoLevel,
		zerolog.WarnLevel,
		zerolog.ErrorLevel,
	}

	sLevels := []slog.Level{
		slog.LevelDebug,
		slog.LevelInfo,
		slog.LevelWarn,
		slog.LevelError,
	}

	availableLevels := make(map[slog.Level]struct{}, len(levels))
	for _, level := range sLevels {
		availableLevels[level] = struct{}{}
	}

	ctx := context.Background()
	for i, sLevel := range sLevels {
		zLevel := levels[i]

		log := zerolog.New(io.Discard).Level(zLevel)
		handler := NewZerologHandler(log, nil)

		_, enabled := availableLevels[sLevel]
		if handler.Enabled(ctx, sLevel) != enabled {
			t.Errorf("level should be enabled: %t but got opposite", enabled)
		}
		delete(availableLevels, sLevel)
	}
}
