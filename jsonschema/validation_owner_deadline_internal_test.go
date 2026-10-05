package jsonschema

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestValidationOwnerRegexpDeadlineTimeoutUsesInclusivePositiveRemainingTime(t *testing.T) {
	for _, test := range []struct {
		name      string
		remaining time.Duration
		want      time.Duration
		err       error
	}{
		{name: "negative", remaining: -time.Nanosecond, err: context.DeadlineExceeded},
		{name: "zero", err: context.DeadlineExceeded},
		{name: "shorter deadline", remaining: time.Nanosecond, want: time.Nanosecond},
		{name: "equal", remaining: time.Second, want: time.Second},
		{name: "longer deadline", remaining: 2 * time.Second, want: time.Second},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := regexpDeadlineTimeout(test.remaining, time.Second)
			if got != test.want || !errors.Is(err, test.err) {
				t.Fatalf("deadline timeout = %v, %v; want %v, %v", got, err, test.want, test.err)
			}
		})
	}
}
