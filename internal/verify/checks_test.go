package verify

import (
	"context"
	"fmt"
	"testing"
)

func TestRunChecksReportsFailureAndWarning(t *testing.T) {
	results := RunChecks(context.Background(), []Check{
		{ID: "ok", Description: "passes", Run: func(context.Context) error { return nil }},
		{ID: "warn", Description: "warns", Soft: true, Run: func(context.Context) error { return fmt.Errorf("warning") }},
		{ID: "fail", Description: "fails", Run: func(context.Context) error { return fmt.Errorf("failure") }},
	})
	if len(results) != 3 {
		t.Fatalf("unexpected results len: %d", len(results))
	}
	if results[1].Status != CheckStatusWarning || results[2].Status != CheckStatusFailed {
		t.Fatalf("unexpected statuses: %#v", results)
	}
}
