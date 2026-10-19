package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	exampleagent "github.com/defos-ai/commander-example-agent"
	agents "github.com/defos-ai/commander/framework/agent"
	"github.com/defos-ai/commander/framework/host"
	"github.com/defos-ai/commander/framework/id"
)

type config struct {
	busURL            string
	storageURL        string
	logLevel          string
	logFormat         string
	agentID           string
	heartbeatInterval time.Duration
}

func parseConfig(args []string, stderr io.Writer) (config, error) {
	var result config
	var agentIDSuffix string
	flags := flag.NewFlagSet(serviceName, flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&agentIDSuffix, "agent-id-suffix", "", "8 lowercase hex characters; overrides COMMANDER_AGENT_ID_SUFFIX; empty generates a fresh ID")
	flags.StringVar(&result.busURL, "bus-url", "", "Bus URL or path; overrides BUS_URL; default ./state/bus")
	flags.StringVar(&result.storageURL, "storage-url", "", "Artifact storage URL or path; overrides STORAGE_URL; default ./state/storage")
	flags.StringVar(&result.logLevel, "log-level", "info", "debug, info, warn or error; overrides LOG_LEVEL")
	flags.StringVar(&result.logFormat, "log-format", "json", "json or text; overrides LOG_FORMAT; logs go to stderr")
	flags.DurationVar(&result.heartbeatInterval, "heartbeat-interval", agents.DefaultHeartbeatInterval, "Agent registration heartbeat interval")
	flags.Usage = func() {
		fmt.Fprintf(stderr, "Usage: %s [flags]\n", serviceName)
		fmt.Fprintln(stderr, "Connect to the controller's bus and artifact storage. A fresh agent ID is created unless an agent ID suffix is supplied.")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return config{}, err
	}
	if flags.NArg() != 0 {
		return config{}, errors.New("unexpected positional arguments; run with --help")
	}
	provided := make(map[string]bool)
	flags.Visit(func(f *flag.Flag) { provided[f.Name] = true })
	result.busURL = host.Value(result.busURL, provided["bus-url"], "BUS_URL", filepath.Join("state", "bus"))
	result.storageURL = host.Value(result.storageURL, provided["storage-url"], "STORAGE_URL", filepath.Join("state", "storage"))
	result.logLevel = host.Value(result.logLevel, provided["log-level"], "LOG_LEVEL", "info")
	result.logFormat = host.Value(result.logFormat, provided["log-format"], "LOG_FORMAT", "json")
	suffix := host.Value(agentIDSuffix, provided["agent-id-suffix"], "COMMANDER_AGENT_ID_SUFFIX", "")
	if suffix != "" {
		idType := strings.TrimSuffix(string(exampleagent.AgentType), "-agent")
		result.agentID = idType + "-agent-" + suffix
		if kind, ok := id.RuntimeType(result.agentID, id.Agent); len(suffix) != 8 || !ok || kind != idType {
			return config{}, errors.New("agent-id-suffix / COMMANDER_AGENT_ID_SUFFIX must be exactly 8 lowercase hex characters")
		}
	}
	if result.busURL == "" || result.storageURL == "" {
		return config{}, errors.New("bus-url and storage-url must not be empty")
	}
	if result.heartbeatInterval <= 0 {
		return config{}, errors.New("heartbeat-interval must be positive so controllers can discover this agent")
	}
	return result, nil
}
