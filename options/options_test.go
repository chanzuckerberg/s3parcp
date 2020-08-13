package options

import (
	"runtime"
	"testing"
)

func TestDefaults(t *testing.T) {
	source := "source"
	opts, err := ParseArgs([]string{source})

	if err != nil {
		t.Errorf("encountered error while parsing args %s", err)
		t.FailNow()
	}

	if opts.Positional.Source != source {
		t.Errorf("expected opts.Positional.Source: %s to equal %s", opts.Positional.Source, source)
	}

	if opts.Positional.Destination != source {
		t.Errorf("expected opts.Positional.Destination: %s to equal %s", opts.Positional.Destination, source)
	}

	if opts.Concurrency != runtime.NumCPU() {
		t.Errorf("expected opts.Concurrency: %d to equal runtime.NumCPU(): %d", opts.Concurrency, runtime.NumCPU())
	}
}
