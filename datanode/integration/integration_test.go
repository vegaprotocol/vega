// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package integration_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/cmd/data-node/commands/start"
	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/datanode/config/encoding"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/utils"
	"code.vegaprotocol.io/vega/datanode/utils/databasetest"
	vgfs "code.vegaprotocol.io/vega/libs/fs"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	"github.com/machinebox/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	lastEpoch            = 110
	playbackTimeout      = 5 * time.Minute
	chainID              = "testnet-001"
	compressedTestdata   = "testdata/system_tests.evt.gz"
	eventsDir            = "testdata/events"
	decompressedTestdata = "testdata/events/system_tests.evt"
)

var (
	client        *graphql.Client
	blockWhenDone = flag.Bool("block", false, "leave services running after tests are complete NOTE: EMBEDDED POSGRESQL WILL NOT SHUT DOWN PROPERLY")
	writeGolden   = flag.Bool("golden", false, "write query results to 'golden' files for comparison")
	goldenDir     string
)

func TestMain(m *testing.M) {
	flag.Parse()
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()

	if testing.Short() {
		log.Print("Skipping datanode integration tests, go test run with -short")
		return
	}

	vegaHome, postgresRuntimePath, err := setupDirs()
	if err != nil {
		log.Fatalf("couldn't setup directories: %s", err)
	}
	defer func() { _ = os.RemoveAll(postgresRuntimePath) }()

	testDBSocketDir := filepath.Join(postgresRuntimePath)
	cfg, err := newTestConfig(testDBSocketDir)
	if err != nil {
		log.Fatal("couldn't set up config: ", err)
	}

	err = os.MkdirAll(eventsDir, os.ModePerm)
	if err != nil {
		log.Fatal("failed to make events dir: ", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal("failed to get working dir: ", err)
	}

	decompressedTestDataPath := filepath.Join(cwd, decompressedTestdata)
	if err = utils.DecompressFile(filepath.Join(cwd, compressedTestdata), decompressedTestDataPath); err != nil {
		log.Fatal("couldn't decompress event file ", err)
	}

	defer func() {
		if err := os.RemoveAll(decompressedTestDataPath); err != nil {
			log.Printf("failed to remove event file: %s", err)
		}
	}()

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		if err := runTestNode(ctx, cfg, vegaHome); err != nil {
			cfunc()
			log.Fatal("running test node: ", err)
		}
	}()

	client = graphql.NewClient(fmt.Sprintf("http://localhost:%v/graphql", cfg.Gateway.Port))
	if err = waitForEpoch(client, lastEpoch, playbackTimeout); err != nil {
		cfunc()
		log.Fatal("problem piping event stream: ", err)
	}
	// normal run - services should be terminated properly
	if blockWhenDone == nil || !*blockWhenDone {
		go handleSignal(ctx, cfunc)
	}

	// Cheesy sleep to give everything chance to percolate
	time.Sleep(5 * time.Second)

	select {
	case <-ctx.Done():
		return
	default:
		m.Run()
	}

	log.Printf("Integration tests completed")

	// When you're debugging tests, it's helpful to stop here so you can go in and poke around
	// sending queries via the graphql playground etc..
	if blockWhenDone != nil && *blockWhenDone {
		log.Print("Blocking now to allow debugging")
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM) // nolint
		<-c
		os.Exit(0)
	}

	cfunc()
	wg.Wait()
}

func handleSignal(ctx context.Context, cfunc func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	for {
		select {
		case sig := <-c:
			log.Printf("Received %+v signal", sig)
			cfunc() // cancel context
			return
		case <-ctx.Done():
			// context was cancelled for some reason, close stopper channel
			log.Printf("Context cancelled")
			return
		}
	}
}

func setupDirs() (string, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("couldn't get working directory: %w", err)
	}

	goldenDir = filepath.Join(cwd, "testdata", "golden")
	if err = vgfs.EnsureDir(goldenDir); err != nil {
		return "", "", fmt.Errorf("couldn't create golden dir: %w", err)
	}

	vegaHome, err := os.MkdirTemp("", "datanode_test")
	if err != nil {
		return "", "", fmt.Errorf("couldn't create temp dir: %w", err)
	}

	postgresRuntimePath := filepath.Join(vegaHome, "pgdb")

	if err = os.Mkdir(postgresRuntimePath, fs.ModePerm); err != nil {
		return "", "", fmt.Errorf("couldn't create postgres runtime dir: %w", err)
	}

	return vegaHome, postgresRuntimePath, nil
}

type queryDetails struct {
	TestName string
	Query    string
	Result   json.RawMessage
	Duration time.Duration
}

func assertGraphQLQueriesReturnSame(t *testing.T, query string) {
	t.Helper()

	req := graphql.NewRequest(query)
	var resp map[string]interface{}
	s := time.Now()
	err := client.Run(context.Background(), req, &resp)
	require.NoError(t, err, "failed to run query: '%s'; %s", query, err)
	elapsed := time.Since(s)

	var respJsn json.RawMessage
	respJsn, err = json.MarshalIndent(resp, "", "\t")
	require.NoError(t, err)

	niceName := strings.ReplaceAll(t.Name(), "/", "_")
	goldenFile := filepath.Join(goldenDir, niceName)

	if *writeGolden {
		details := queryDetails{
			TestName: niceName,
			Query:    query,
			Result:   respJsn,
			Duration: elapsed,
		}
		jsonBytes, err := json.MarshalIndent(details, "", "\t")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(goldenFile, jsonBytes, 0o644))
	} else {
		jsonBytes, err := os.ReadFile(goldenFile)
		require.NoError(t, err, "No golden file for this test, generate one by running 'go test' with the -golden flag")
		details := queryDetails{}
		require.NoError(t, json.Unmarshal(jsonBytes, &details), "Unable to unmarshal golden file")
		assert.Equal(t, details.Query, query, "GraphQL query string differs from recorded in the golden file, regenerate by running 'go test' with the -golden flag")
		assert.JSONEq(t, string(respJsn), string(details.Result))
	}
}

func newTestConfig(postgresRuntimePath string) (*config.Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("couldn't get working directory: %w", err)
	}

	cfg := config.NewDefaultConfig()
	// cfg.API.RateLimit.TrustedProxies = []string{}
	cfg.Broker.UseEventFile = true
	cfg.Broker.PanicOnError = true
	cfg.Broker.FileEventSourceConfig.Directory = filepath.Join(cwd, eventsDir)
	cfg.Broker.FileEventSourceConfig.TimeBetweenBlocks = encoding.Duration{Duration: 0}
	cfg.API.WebUIEnabled = true
	cfg.API.Reflection = true
	cfg.ChainID = chainID
	cfg.SQLStore = databasetest.NewTestConfig(5432, "", postgresRuntimePath)
	cfg.NetworkHistory.Enabled = false
	cfg.SQLStore.RetentionPeriod = sqlstore.RetentionPeriodArchive

	return &cfg, nil
}

func runTestNode(ctx context.Context, cfg *config.Config, vegaHome string) error {
	vegaPaths := paths.New(vegaHome)

	loader, err := config.InitialiseLoader(vegaPaths)
	if err != nil {
		return fmt.Errorf("couldn't create config loader: %w", err)
	}

	if err = loader.Save(cfg); err != nil {
		return fmt.Errorf("couldn't save config: %w", err)
	}

	logger := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	configWatcher, err := config.NewWatcher(context.Background(), logger, vegaPaths)
	if err != nil {
		return fmt.Errorf("couldn't create config watcher: %w", err)
	}

	cmd := start.NodeCommand{
		Log:         logger,
		Version:     "test",
		VersionHash: "",
	}

	if err = cmd.Run(ctx, configWatcher, vegaPaths, []string{}); err != nil {
		return fmt.Errorf("couldn't run node: %w", err)
	}
	return nil
}

func waitForEpoch(client *graphql.Client, epoch int, timeout time.Duration) error {
	giveUpAt := time.Now().Add(timeout)
	for {
		currentEpoch, err := getCurrentEpoch(client)
		if err == nil && currentEpoch >= epoch {
			return nil
		}

		log.Printf("Current epoch is %d, waiting for %d", currentEpoch, epoch)

		if time.Now().After(giveUpAt) {
			return fmt.Errorf("didn't reach epoch %v within %v", epoch, timeout)
		}
		time.Sleep(time.Second)
	}
}

func getCurrentEpoch(client *graphql.Client) (int, error) {
	req := graphql.NewRequest("{ epoch{id} }")
	resp := struct{ Epoch struct{ ID string } }{}

	if err := client.Run(context.Background(), req, &resp); err != nil {
		return 0, err
	}
	if resp.Epoch.ID == "" {
		return 0, fmt.Errorf("empty epoch id")
	}

	return strconv.Atoi(resp.Epoch.ID)
}
