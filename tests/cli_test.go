package exampleagent_test

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/defos-ai/commander-example-agent/cli"
)

func TestCLIHelpDoesNotExposeEnvironment(t *testing.T) {
	t.Setenv("BUS_URL", "amqp://user:bus-secret@broker")
	t.Setenv("STORAGE_URL", "s3://private-bucket")
	var output bytes.Buffer
	if err := cli.Run(t.Context(), []string{"--help"}, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "--") && !strings.Contains(output.String(), "-bus-url") {
		t.Fatal("missing usage")
	}
	if strings.Contains(output.String(), "bus-secret") || strings.Contains(output.String(), "private-bucket") {
		t.Fatal("help exposed provider configuration")
	}
}

func TestCLIRejectsInvalidConfiguration(t *testing.T) {
	for _, test := range []struct {
		name, flag, value, want string
	}{
		{"unsupported bus", "bus-url", "nats://user:provider-secret@broker", "unsupported bus scheme"},
		{"unsupported storage", "storage-url", "https://user:provider-secret@storage", "unsupported artifact file store scheme"},
		{"empty bus", "bus-url", "", "must not be empty"},
		{"disabled heartbeat", "heartbeat-interval", "0", "must be positive"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			args := []string{
				"--bus-url=" + filepath.Join(root, "bus"),
				"--storage-url=" + filepath.Join(root, "storage"),
				"--log-level=info", "--log-format=json",
				"--" + test.flag + "=" + test.value,
			}
			var output bytes.Buffer
			err := cli.Run(t.Context(), args, &output)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
			if strings.Contains(err.Error()+output.String(), "provider-secret") {
				t.Fatal("startup error exposed provider credentials")
			}
		})
	}
}

func TestCLICancelledBeforeStartup(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := cli.Run(ctx, nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}
