package logging

import (
	"context"
	"log/slog"
)

// Ensure dedupeHandler implements the slog.Handler interface.
//
// This declaration ensures that the dedupeHandler type satisfies the slog.Handler
// interface at compile time. If dedupeHandler does not implement all the methods
// required by the slog.Handler interface, a compile-time error will occur.
var _ slog.Handler = new(dedupeHandler)

// dedupeHandler is a custom slog.Handler that deduplicates attributes.
type dedupeHandler struct {
	// base is the underlying slog.Handler to which log records are delegated.
	base slog.Handler

	// attrs is a precomputed, immutable list of deduplicated attributes.
	attrs []slog.Attr
}

// NewDedupeHandler creates a new dedupeHandler instance.
func NewDedupeHandler(base slog.Handler) slog.Handler {
	return &dedupeHandler{
		base:  base,
		attrs: make([]slog.Attr, 0),
	}
}

// Enabled checks if the specified log level is enabled.
func (d *dedupeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return d.base.Enabled(ctx, level)
}

// Handle processes a log record.
func (d *dedupeHandler) Handle(ctx context.Context, record slog.Record) error { // nolint:gocritic // Part of an interface
	// Clone the record to avoid modifying the original.
	record = record.Clone()
	// Add deduplicated attributes temporarily.
	record.AddAttrs(d.attrs...)
	return d.base.Handle(ctx, record)
}

// WithAttrs returns a new handler with additional attributes.
func (d *dedupeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	combined := make(map[string]slog.Attr)

	// Start with existing deduplicated attributes.
	for _, attr := range d.attrs {
		combined[attr.Key] = attr
	}

	// Add new attributes, overriding duplicates.
	for _, attr := range attrs {
		combined[attr.Key] = attr
	}

	// Rebuild the slice of attributes.
	newAttrs := make([]slog.Attr, 0, len(combined))
	for _, attr := range combined {
		newAttrs = append(newAttrs, attr)
	}

	return &dedupeHandler{
		base:  d.base,
		attrs: newAttrs,
	}
}

// WithGroup returns a new handler with a group name.
func (d *dedupeHandler) WithGroup(name string) slog.Handler {
	return &dedupeHandler{
		base:  d.base.WithGroup(name),
		attrs: d.attrs,
	}
}
