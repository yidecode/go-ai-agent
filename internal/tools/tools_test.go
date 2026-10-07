package tools

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestEval(t *testing.T) {
	cases := []struct {
		expr string
		want float64
	}{
		{"2+3", 5},
		{"2+3*4", 14},
		{"(2+3)*4", 20},
		{"10/4", 2.5},
		{"2^10", 1024},
		{"-3+5", 2},
		{"10%3", 1},
		{"(12.5*3 - 4) / 2", 16.75},
		{"2 * -3", -6},
	}
	for _, c := range cases {
		got, err := Eval(c.expr)
		if err != nil {
			t.Errorf("Eval(%q): %v", c.expr, err)
			continue
		}
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("Eval(%q) = %v, want %v", c.expr, got, c.want)
		}
	}
}

func TestEvalErrors(t *testing.T) {
	for _, expr := range []string{"", "2+", "2/0", "(2+3", "abc", "2++2x"} {
		if _, err := Eval(expr); err == nil {
			t.Errorf("Eval(%q): expected error, got nil", expr)
		}
	}
}

func TestFetchURL(t *testing.T) {
	// spin up a local server instead of hitting the network
	srv := newTestServer(t)
	defs := All()
	var fetch Definition
	for _, d := range defs {
		if d.Name == "fetch_url" {
			fetch = d
		}
	}
	out := fetch.Execute(map[string]any{"url": srv.URL + "/hello"})
	if !strings.Contains(out, "HTTP 200") || !strings.Contains(out, "hello world") {
		t.Fatalf("unexpected fetch output: %q", out)
	}
	if got := fetch.Execute(map[string]any{"url": "ftp://x"}); !strings.HasPrefix(got, "error:") {
		t.Fatalf("expected scheme error, got %q", got)
	}
}

func TestUTCTime(t *testing.T) {
	defs := All()
	for _, d := range defs {
		if d.Name == "utc_time" {
			out := d.Execute(nil)
			if _, err := time.Parse(time.RFC3339, out); err != nil {
				t.Fatalf("utc_time output %q is not RFC3339: %v", out, err)
			}
		}
	}
}

func TestExecuteByNameUnknown(t *testing.T) {
	if got := ExecuteByName(All(), "nope", "{}"); !strings.HasPrefix(got, "error:") {
		t.Fatalf("expected error, got %q", got)
	}
}
