package xslog

import (
	"context"
	"log/slog"
)

type slogFieldsStore struct {
	fields []slog.Attr
}

type logFieldsCtxKey struct{}

// WithSlogFields will append to internal store (if exists, otherwise will create a new one) new attributes.
// NOTE: appended attributes will be visible to the upper callers as well, i.e. if function A adds log field
// and then calls function B which adds log field, after exiting from function B and logging in function A,
// there will be both added log fields.
func WithSlogFields(ctx context.Context, attrs ...slog.Attr) context.Context {
	existingStore := slogFieldsFromContext(ctx)
	if existingStore == nil {
		existingStore = &slogFieldsStore{
			fields: attrs,
		}

		return context.WithValue(ctx, logFieldsCtxKey{}, existingStore)
	}

	existingStore.fields = append(existingStore.fields, attrs...)

	return ctx
}

func slogFieldsFromContext(ctx context.Context) *slogFieldsStore {
	store, _ := ctx.Value(slogFieldsStore{}).(*slogFieldsStore)

	return store
}
