// Package errors is a replacement for the golang standard library "errors". This replacement
// adds errors to the Open Telemetry spans. The signatures only differs in that
// New() now takes a context.Context object and fmt.Errorf() has been moved here and also takes a Context.Context.
package errors

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// New creates a new error and writes the error to a span if it exists in the context.
func New(ctx context.Context, text string) error {
	span := trace.SpanFromContext(ctx)

	err := errors.New(text)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	return err
}

// Errorf implements fmt.Errorf with the addition of a Context that if it contains a span
// will have the error added to the span.
func Errorf(ctx context.Context, s string, i ...interface{}) error {
	span := trace.SpanFromContext(ctx)

	err := fmt.Errorf(s, i...)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	return err
}

// As implements errors.As().
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// Is implements errors.Is().
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// Unwrap implemements errors.Unwrap().
func Unwrap(err error) error {
	return errors.Unwrap(err)
}
