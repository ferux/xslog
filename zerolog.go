package xlog

import (
	"context"
	"errors"
	"log/slog"

	"github.com/ferux/collections"
	"github.com/rs/zerolog"
)

var _ slog.Handler = (*ZerologHandler)(nil)

// HandlerOptions allowes to adjust behaviour of the zerolog handler.
type HandlerOptions struct {
	// Does not print timestamp even if it set by slog.
	SkipTime bool
}

// NewZerologHandler creates a wraper over Zerolog to be used as slog.Handler.
func NewZerologHandler(log zerolog.Logger, opts *HandlerOptions) *ZerologHandler {
	if opts == nil {
		opts = &HandlerOptions{}
	}

	copied := *opts
	return &ZerologHandler{
		log:    log,
		groups: make([]group, 1),
		opts:   copied,
	}
}

type group struct {
	name  string
	attrs []slog.Attr
}

type ZerologHandler struct {
	log    zerolog.Logger
	groups []group
	opts   HandlerOptions
}

// Enabled reports whether the handler handles records at the given level.
// The handler ignores records whose level is lower.
// It is called early, before any arguments are processed,
// to save effort if the log event should be discarded.
// If called from a Logger method, the first argument is the context
// passed to that method, or context.Background() if nil was passed
// or the method does not take a context.
// The context is passed so Enabled can use its values
// to make a decision.
//
//	Enabled implements slog.Handler interface.
func (h *ZerologHandler) Enabled(ctx context.Context, level slog.Level) (enabled bool) {
	switch level {
	case slog.LevelDebug:
		enabled = h.log.Debug().Enabled()
	case slog.LevelInfo:
		enabled = h.log.Info().Enabled()
	case slog.LevelWarn:
		enabled = h.log.Warn().Enabled()
	case slog.LevelError:
		enabled = h.log.Error().Enabled()
	default:

	}

	return enabled
}

// Handle handles the Record.
// It will only be called when Enabled returns true.
// The Context argument is as for Enabled.
// It is present solely to provide Handlers access to the context's values.
// Canceling the context should not affect record processing.
// (Among other things, log messages may be necessary to debug a
// cancellation-related problem.)
//
// Handle methods that produce output should observe the following rules:
//
//   - If r.Time is the zero time, ignore the time.
//
//   - If r.PC is zero, ignore it.
//
//   - Attr's values should be resolved.
//
//   - If an Attr's key and value are both the zero value, ignore the Attr.
//     This can be tested with attr.Equal(Attr{}).
//
//   - If a group's key is empty, inline the group's Attrs.
//
//   - If a group has no Attrs (even if it has a non-empty key),
//     ignore it.
//
// .
//
//	Handle implements slog.Handler interface.
func (h *ZerologHandler) Handle(ctx context.Context, record slog.Record) error {
	var event *zerolog.Event
	switch record.Level {
	case slog.LevelDebug:
		event = h.log.Debug()
	case slog.LevelInfo:
		event = h.log.Info()
	case slog.LevelWarn:
		event = h.log.Warn()
	case slog.LevelError:
		event = h.log.Error()
	default:
		return errors.New("unsupported log level " + record.Level.String())
	}

	if !event.Enabled() {
		return nil
	}

	if !h.opts.SkipTime && !record.Time.IsZero() {
		event = event.Time(zerolog.TimestampFieldName, record.Time.UTC())
	}

	if record.PC != 0 {
		event = event.CallerSkipFrame(int(record.PC))
	}

	var prev *zerolog.Event
	var prevName string

	lastIDx := len(h.groups) - 1
	for i := lastIDx; i >= 0; i-- {
		current := h.groups[i]
		if current.name == "" {
			collections.ForEach(current.attrs, func(a slog.Attr) {
				event = appendAttrToEvent(a, event)
			})
			if i == lastIDx {
				record.Attrs(func(attr slog.Attr) bool {
					event = appendAttrToEvent(attr, event)
					return true
				})
			}

			store := slogFieldsFromContext(ctx)
			if store != nil && len(store.fields) > 0 {
				collections.ForEach(store.fields, func(a slog.Attr) {
					event = appendAttrToEvent(a, event)
				})
			}

			break
		}

		groupAttrs := zerolog.Dict()
		collections.ForEach(current.attrs, func(attr slog.Attr) {
			groupAttrs = appendAttrToEvent(attr, groupAttrs)
		})

		if i == lastIDx {
			record.Attrs(func(attr slog.Attr) bool {
				groupAttrs = appendAttrToEvent(attr, groupAttrs)
				return true
			})
		}

		if prev != nil {
			groupAttrs.Dict(prevName, prev)
		}

		prevName = current.name
		prev = groupAttrs
	}

	if prev != nil && prevName != "" {
		event.Dict(prevName, prev)
	}

	if record.Message == "" {
		event.Send()
	} else {
		event.Msg(record.Message)
	}

	return nil
}

// WithAttrs returns a new Handler whose attributes consist of
// both the receiver's attributes and the arguments.
// The Handler owns the slice: it may retain, modify or discard it.
//
//	WithAttrs implements slog.Handler interface
func (h *ZerologHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newh := h.clone(0, uint(len(attrs)))
	lastGroupIDx := len(newh.groups) - 1

	newh.groups[lastGroupIDx].attrs = append(newh.groups[lastGroupIDx].attrs, attrs...)

	return newh
}

// WithGroup returns a new Handler with the given group appended to
// the receiver's existing groups.
// The keys of all subsequent attributes, whether added by With or in a
// Record, should be qualified by the sequence of group names.
//
// How this qualification happens is up to the Handler, so long as
// this Handler's attribute keys differ from those of another Handler
// with a different sequence of group names.
//
// A Handler should treat WithGroup as starting a Group of Attrs that ends
// at the end of the log event. That is,
//
//	logger.WithGroup("s").LogAttrs(ctx, level, msg, slog.Int("a", 1), slog.Int("b", 2))
//
// should behave like
//
//	logger.LogAttrs(ctx, level, msg, slog.Group("s", slog.Int("a", 1), slog.Int("b", 2)))
//
// If the name is empty, WithGroup returns the receiver.
//
//	WithGroup implements slog.Handler interface.
func (h *ZerologHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	newh := h.clone(1, 0)
	newh.groups = append(newh.groups, group{
		name: name,
	})

	return newh
}

func (h *ZerologHandler) clone(addGroupsCap uint, addSizeCap uint) *ZerologHandler {
	newh := &ZerologHandler{
		log:    h.log,
		groups: make([]group, len(h.groups), len(h.groups)+int(addGroupsCap)),
		opts:   h.opts,
	}

	lastID := len(h.groups) - 1
	for i, g := range h.groups {
		capValue := len(g.attrs)
		if lastID == i {
			capValue += int(addSizeCap)
		}
		newh.groups[i] = group{
			name:  g.name,
			attrs: make([]slog.Attr, len(g.attrs), capValue),
		}

		copy(newh.groups[i].attrs, g.attrs)
	}

	return newh
}

func appendAttrToEvent(attr slog.Attr, event *zerolog.Event) *zerolog.Event {
	switch attr.Value.Kind() {
	case slog.KindGroup:
		group := attr.Value.Group()
		dict := zerolog.Dict()

		for _, groupAttr := range group {
			dict = appendAttrToEvent(groupAttr, dict)
		}

		return event.Dict(attr.Key, dict)
	case slog.KindLogValuer:
		v := attr.Value.LogValuer()
		out := slog.Attr{
			Key:   attr.Key,
			Value: v.LogValue(),
		}
		return appendAttrToEvent(out, event)
	case slog.KindBool:
		return event.Bool(attr.Key, attr.Value.Bool())
	case slog.KindInt64:
		return event.Int64(attr.Key, attr.Value.Int64())
	case slog.KindUint64:
		return event.Uint64(attr.Key, attr.Value.Uint64())
	case slog.KindFloat64:
		return event.Float64(attr.Key, attr.Value.Float64())
	case slog.KindDuration:
		return event.Dur(attr.Key, attr.Value.Duration())
	case slog.KindTime:
		return event.Time(attr.Key, attr.Value.Time())
	case slog.KindString:
		return event.Str(attr.Key, attr.Value.String())
	case slog.KindAny:
		if terr, ok := attr.Value.Any().(error); ok {
			return event.AnErr(attr.Key, terr)
		}

		return event.Any(attr.Key, attr.Value.Any())
	default:
		return event
	}
}
