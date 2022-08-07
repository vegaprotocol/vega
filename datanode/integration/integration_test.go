// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package integration_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/cmd/data-node/node"
	"code.vegaprotocol.io/vega/datanode/config"
	"code.vegaprotocol.io/vega/datanode/config/encoding"
	"code.vegaprotocol.io/vega/datanode/fsutil"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/machinebox/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	LastEpoch       = 2090
	PlaybackTimeout = 3 * time.Minute
)

var (
	client                  *graphql.Client
	integrationTestsEnabled *bool = flag.Bool("integration", false, "run integration tests")
	blockWhenDone                 = flag.Bool("block", false, "leave services running after tests are complete")
	writeGolden             *bool = flag.Bool("golden", false, "write query results to 'golden' files for comparison")
	goldenDir               string
)

func TestMain(m *testing.M) {
	flag.Parse()

	if !*integrationTestsEnabled {
		log.Print("Skipping integration tests. To enable pass -integration flag to 'go test'")
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		panic("couldn't get working directory")
	}

	goldenDir = filepath.Join(cwd, "testdata", "golden")
	err = fsutil.EnsureDir(goldenDir)
	if err != nil {
		panic("couldn't ensure golden data dir")
	}

	cfg, err := newTestConfig()
	if err != nil {
		log.Fatal("couldn't set up config: ", err)
	}

	if err := runTestNode(cfg); err != nil {
		log.Fatal("running test node: ", err)
	}

	client = graphql.NewClient(fmt.Sprintf("http://localhost:%v/query", cfg.Gateway.GraphQL.Port))
	if err := waitForEpoch(client, LastEpoch, PlaybackTimeout); err != nil {
		log.Fatal("problem piping event stream: ", err)
	}

	// Cheesy sleep to give everything chance to percolate
	time.Sleep(5 * time.Second)

	m.Run()

	log.Printf("Integration tests completed")

	// When you're debugging tests, it's helpful to stop here so you can go in and poke around
	// sending queries via the graphql playground etc..
	if blockWhenDone != nil && *blockWhenDone {
		log.Print("Blocking now to allow debugging")
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

func compareResponses(t *testing.T, oldResp, newResp interface{}) {
	t.Helper()
	require.NotEmpty(t, oldResp)
	require.NotEmpty(t, newResp)

	sortAccounts := cmpopts.SortSlices(func(a Account, b Account) bool {
		if a.Type != b.Type {
			return a.Type < b.Type
		}
		if a.Asset.Id != b.Asset.Id {
			return a.Asset.Id < b.Asset.Id
		}
		if a.Market.Id != b.Market.Id {
			return a.Market.Id < b.Market.Id
		}
		return a.Balance < b.Balance
	})
	sortTrades := cmpopts.SortSlices(func(a Trade, b Trade) bool { return a.Id < b.Id })
	sortMarkets := cmpopts.SortSlices(func(a Market, b Market) bool { return a.Id < b.Id })
	sortProposals := cmpopts.SortSlices(func(a Proposal, b Proposal) bool { return a.Id < b.Id })
	sortNetParams := cmpopts.SortSlices(func(a NetworkParameter, b NetworkParameter) bool { return a.Key < b.Key })
	sortParties := cmpopts.SortSlices(func(a Party, b Party) bool { return a.Id < b.Id })
	sortDeposits := cmpopts.SortSlices(func(a Deposit, b Deposit) bool { return a.ID < b.ID })
	sortSpecs := cmpopts.SortSlices(func(a, b OracleSpec) bool { return a.ID < b.ID })
	sortPositions := cmpopts.SortSlices(func(a, b Position) bool {
		if a.Party.Id != b.Party.Id {
			return a.Party.Id < b.Party.Id
		}
		return a.Market.Id < b.Market.Id
	})
	sortTransfers := cmpopts.SortSlices(func(a Transfer, b Transfer) bool { return a.Id < b.Id })
	sortWithdrawals := cmpopts.SortSlices(func(a, b Withdrawal) bool { return a.ID < b.ID })
	sortOrders := cmpopts.SortSlices(func(a, b Order) bool { return a.Id < b.Id })
	sortNodes := cmpopts.SortSlices(func(a, b Node) bool { return a.Id < b.Id })
	sortDelegations := cmpopts.SortSlices(func(a, b Delegation) bool { return a.Party.Id < b.Party.Id })

	// The old API has nulls for the 'UpdatedAt' field in positions
	ignorePositionTimestamps := cmpopts.IgnoreFields(Position{}, "UpdatedAt")
	truncateOrderNanoseconds := cmp.Transformer("truncateOrderNanoseconds", func(input Order) Order {
		if input.UpdatedAt == "" {
			return input
		}

		updatedAt, err := time.Parse(time.RFC3339Nano, input.UpdatedAt)
		if err != nil {
			t.Logf("could not conver order Update At timestamp: %v", err)
			return input
		}

		input.UpdatedAt = updatedAt.Truncate(time.Microsecond).Format(time.RFC3339Nano)
		return input
	})
	normaliseEthereumAddress := cmp.Transformer("normaliseEthereumAddress", func(input Node) Node {
		input.EthereumAddress = strings.ToLower(input.EthereumAddress)
		return input
	})

	diff := cmp.Diff(oldResp, newResp, removeDupVotes(), normaliseEthereumAddress, truncateOrderNanoseconds,
		sortTrades, sortAccounts, sortMarkets, sortProposals, sortNetParams, sortParties, sortDeposits,
		sortSpecs, sortTransfers, sortWithdrawals, sortOrders, sortNodes, sortPositions, ignorePositionTimestamps,
		sortDelegations)

	assert.Empty(t, diff)
}

func removeDupVotes() cmp.Option {
	// This is a bit grim; in the old API you get repeated entries for votes when they are updated,
	// which is a bug not present in the new API - so remove duplicates when comparing (and sort)
	return cmp.Transformer("DuplicateVotes", func(in []Vote) []Vote {
		m := make(map[string]Vote)
		for _, vote := range in {
			m[fmt.Sprintf("%v-%v", vote.ProposalId, vote.Party.Id)] = vote
		}

		keys := make([]string, len(m))
		sort.Strings(keys)

		out := make([]Vote, len(m))
		for i, key := range keys {
			out[i] = m[key]
		}
		return out
	})
}

type queryDetails[T any] struct {
	TestName string
	Query    string
	Result   T
	Duration time.Duration
}

func assertGraphQLQueriesReturnSame[T any](t *testing.T, query string) {
	t.Helper()

	req := graphql.NewRequest(query)
	var resp T
	start := time.Now()
	err := client.Run(context.Background(), req, &resp)
	require.NoError(t, err)
	elapsed := time.Since(start)

	niceName := strings.Replace(t.Name(), "/", "_", -1)
	goldenFile := filepath.Join(goldenDir, niceName)

	if *writeGolden {
		details := queryDetails[T]{
			TestName: niceName,
			Query:    query,
			Result:   resp,
			Duration: elapsed,
		}
		jsonBytes, err := json.Marshal(details)
		require.NoError(t, err)

		os.WriteFile(goldenFile, jsonBytes, 0o644)
	} else {
		jsonBytes, err := os.ReadFile(goldenFile)
		require.NoError(t, err, "No golden file for this test, generate one by running 'go test' with the -golden flag")

		details := queryDetails[T]{}
		err = json.Unmarshal(jsonBytes, &details)
		require.NoError(t, err, "Unable to unmarshal golden file")

		assert.Equal(t, details.Query, query, "GraphQL query string differs from recorded in the golden file, regenerate by running 'go test' with the -golden flag")
		compareResponses(t, resp, details.Result)
	}
}

func newTestConfig() (*config.Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("couldn't get working directory: %w", err)
	}

	cfg := config.NewDefaultConfig()
	cfg.Broker.UseEventFile = true
	cfg.Broker.FileEventSourceConfig.File = filepath.Join(cwd, "testdata", "system_tests.evt")
	cfg.Broker.FileEventSourceConfig.TimeBetweenBlocks = encoding.Duration{Duration: 0}
	cfg.API.WebUIEnabled = encoding.Bool(true)
	cfg.API.Reflection = encoding.Bool(true)

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
