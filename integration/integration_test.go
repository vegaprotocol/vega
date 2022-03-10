package integration_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/cmd/data-node/node"
	"code.vegaprotocol.io/data-node/config"
	"code.vegaprotocol.io/data-node/config/encoding"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/shared/paths"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/machinebox/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const LastEpoch = 210

const PlaybackTimeout = 30 * time.Second

var (
	newClient               *graphql.Client
	oldClient               *graphql.Client
	integrationTestsEnabled bool = false
	blockWhenDone           bool = false
)

func TestMain(m *testing.M) {
	if !integrationTestsEnabled {
		return
	}

	cfg, err := newTestConfig()
	if err != nil {
		log.Fatal("couldn't set up config: ", err)
	}

	if err := runTestNode(cfg); err != nil {
		log.Fatal("running test node: ", err)
	}

	newClient = graphql.NewClient(fmt.Sprintf("http://localhost:%v/query", cfg.Gateway.GraphQL.Port))
	oldClient = graphql.NewClient(fmt.Sprintf("http://localhost:%v/query", cfg.Gateway.GraphQL.Port+cfg.API.LegacyAPIPortOffset))
	if err := waitForEpoch(newClient, LastEpoch, PlaybackTimeout); err != nil {
		log.Fatal("problem piping event stream: ", err)

	}

	// Cheesy sleep to give everything chance to percolate
	time.Sleep(5 * time.Second)

	m.Run()

	// When you're debugging tests, it's helpful to stop here so you can go in and poke around
	// sending queries via the graphql playground etc..
	if blockWhenDone {
		waitForSIGTERM()
	}
}

func waitForSIGTERM() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM) // nolint
	go func() {
		<-c
		os.Exit(1)
	}()

	for {
		time.Sleep(1 * time.Second)
	}
}

func assertGraphQLQueriesReturnSame(t *testing.T, query string, oldResp, newResp interface{}) {
	t.Helper()
	req := graphql.NewRequest(query)

	err := oldClient.Run(context.Background(), req, &oldResp)
	require.NoError(t, err)

	err = newClient.Run(context.Background(), req, &newResp)
	require.NoError(t, err)

	sortTrades := cmpopts.SortSlices(func(a Trade, b Trade) bool { return a.Id < b.Id })
	sortVotes := cmpopts.SortSlices(func(a Vote, b Vote) bool { return a.Party.Id < b.Party.Id })
	diff := cmp.Diff(oldResp, newResp, sortTrades, sortVotes)
	assert.Empty(t, diff)
}

func newTestConfig() (*config.Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("couldn't get working directory: %w", err)
	}

	cfg := config.NewDefaultConfig()
	cfg.SQLStore.Enabled = true
	cfg.Broker.UseEventFile = true
	cfg.Broker.FileEventSourceConfig.File = filepath.Join(cwd, "testdata", "system_tests.evt")
	cfg.Broker.FileEventSourceConfig.TimeBetweenBlocks = encoding.Duration{Duration: 0}
	cfg.API.ExposeLegacyAPI = encoding.Bool(true)
	cfg.API.LegacyAPIPortOffset = 10
	return &cfg, nil
}

func runTestNode(cfg *config.Config) error {
	log := logging.NewLoggerFromConfig(logging.NewDefaultConfig())

	vegaHome, err := ioutil.TempDir("", "datanode_integration_test")
	if err != nil {
		return fmt.Errorf("Couldn't create temporary vega home: %w", err)
	}

	vegaPaths := paths.New(vegaHome)

	loader, err := config.InitialiseLoader(vegaPaths)
	if err != nil {
		return fmt.Errorf("Couldn't create config loader: %w", err)
	}

	loader.Save(cfg)

	configWatcher, err := config.NewWatcher(context.Background(), log, vegaPaths)
	if err != nil {
		log.Fatal("Couldn't set up config", logging.Error(err))
	}

	cmd := node.NodeCommand{
		Log:         log,
		Version:     "test",
		VersionHash: "",
	}

	go cmd.Run(configWatcher, vegaPaths, []string{})
	return nil
}

func waitForEpoch(client *graphql.Client, epoch int, timeout time.Duration) error {
	giveUpAt := time.Now().Add(timeout)
	for {
		currentEpoch, err := getCurrentEpoch(client)
		if err == nil && currentEpoch >= epoch {
			return nil
		}
		if time.Now().After(giveUpAt) {
			return fmt.Errorf("Didn't reach epoch %v within %v", epoch, timeout)
		}
		time.Sleep(time.Second)
	}
}

func getCurrentEpoch(client *graphql.Client) (int, error) {
	req := graphql.NewRequest("{ epoch{id} }")
	resp := struct{ Epoch struct{ Id string } }{}

	if err := client.Run(context.Background(), req, &resp); err != nil {
		return 0, err
	}
	if resp.Epoch.Id == "" {
		return 0, fmt.Errorf("Empty epoch id")
	}

	return strconv.Atoi(resp.Epoch.Id)
}
