package snapshot_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/cmd/data-node/commands/start"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/datanode/broker"
	"code.vegaprotocol.io/vega/datanode/candlesv2"
	"code.vegaprotocol.io/vega/datanode/config/encoding"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/service"
	"code.vegaprotocol.io/vega/datanode/snapshot"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/utils/databasetest"
	"code.vegaprotocol.io/vega/logging"

	"github.com/jackc/pgx/v4/pgxpool"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	snapshotInterval     = int64(1000)
	chainID              = "test-chain-7cevW2"
	compressedEventsFile = "testdata/smoketest_to_block_5000.evts.gz"
	numSnapshots         = 5
)

var (
	sqlConfig sqlstore.Config

	fromEventsSnapshotHashes    []string
	fromEventsDatabaseSummaries []databaseSummary

	fromEventsIntervalToHistoryHashes     []string
	fromEventsIntervalToHistoryTableDelta []map[string]tableDataSummary

	snapshotsBackupDir string
	eventsFile         string

	postgresLog *bytes.Buffer
)

func TestMain(t *testing.M) {
	var err error
	testID := uuid.NewV4().String()
	snapshotsBackupDir, err = ioutil.TempDir("", testID)
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(snapshotsBackupDir)

	testID = uuid.NewV4().String()
	eventsDir, err := ioutil.TempDir("", testID)
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(eventsDir)

	log := logging.NewTestLogger()

	eventsFile = filepath.Join(eventsDir, "smoketest_to_block_5000_or_above.evts")
	decompressEventFile()

	var snapshotsDir string
	exitCode := databasetest.TestMain(t, func(config sqlstore.Config, source *sqlstore.ConnectionSource, dir string, pgLog *bytes.Buffer) {
		sqlConfig = config
		snapshotsDir = dir
		postgresLog = pgLog

		emptyDatabase()

		// Do initial run to get the expected state of the datanode from just event playback
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		snapshotService := setupSnapshotService(sqlConfig, snapshotsDir)

		var snapshots []snapshot.Meta
		sqlBroker, err := setupSQLBroker(ctx, eventsFile, sqlConfig, snapshotService,
			func(ctx context.Context, service *snapshot.Service, chainId string, lastCommittedBlockHeight int64) bool {
				if lastCommittedBlockHeight > 0 && lastCommittedBlockHeight%snapshotInterval == 0 {
					lastSnapshot, err := service.CreateSnapshotAsync(ctx, chainId, snapshot.GetFromHeight(lastCommittedBlockHeight, snapshotInterval), lastCommittedBlockHeight)
					if err != nil {
						panic(fmt.Errorf("failed to create snapshot:%w", err))
					}

					waitForSnapshotToCompleteUseMeta(lastSnapshot, snapshotsDir)
					snapshots = append(snapshots, lastSnapshot)
					md5Hash, err := snapshot.GetSnapshotMd5Hash(log, lastSnapshot.CurrentStateSnapshotFile, lastSnapshot.HistorySnapshotFile)
					if err != nil {
						panic(fmt.Errorf("failed to get snapshot hash:%w", err))
					}

					fromEventsSnapshotHashes = append(fromEventsSnapshotHashes, md5Hash)

					historyMd5Hash, err := snapshot.GetHistoryMd5Hash(log, lastSnapshot)
					if err != nil {
						panic(fmt.Errorf("failed to get history hash:%w", err))
					}

					fromEventsIntervalToHistoryHashes = append(fromEventsIntervalToHistoryHashes, historyMd5Hash)

					summary := getDatabaseDataSummary(ctx, sqlConfig.ConnectionConfig)

					fromEventsDatabaseSummaries = append(fromEventsDatabaseSummaries, summary)

					if lastCommittedBlockHeight == numSnapshots*snapshotInterval {
						return true
					}
				}

				return false
			})
		if err != nil {
			panic(fmt.Errorf("failed to get setup sqlbroker:%w", err))
		}

		err = sqlBroker.Receive(ctx)
		if err != nil {
			panic(fmt.Errorf("failed to process events:%w", err))
		}

		if len(fromEventsSnapshotHashes) != numSnapshots {
			panic(fmt.Errorf("expected 5 snapshots, got %d", len(fromEventsSnapshotHashes)))
		}
		// For the same events file and block height this hash should be the same across all OS/Arch
		// If the events file is updated, schema changes, or snapshot height changed this will need updating
		// Easiest way to update is to put a breakpoint here and inspect fromEventsSnapshotHashes
		panicIfSnapshotHashNotEqual(fromEventsSnapshotHashes[0], "ecc54630bbc68d536c63231f07a96b28", snapshots)
		panicIfSnapshotHashNotEqual(fromEventsSnapshotHashes[1], "ba7eca255540b83293178d9a14b79457", snapshots)
		panicIfSnapshotHashNotEqual(fromEventsSnapshotHashes[2], "6c67a5f3d3c719edae61d218c69c479a", snapshots)
		panicIfSnapshotHashNotEqual(fromEventsSnapshotHashes[3], "511a145aeca30ec64881fa8c67f3b3ba", snapshots)
		panicIfSnapshotHashNotEqual(fromEventsSnapshotHashes[4], "5afc2c2aa109572672daa691fa691b7b", snapshots)

		if len(fromEventsDatabaseSummaries) != numSnapshots {
			panic(fmt.Errorf("expected %d database summaries, got %d", numSnapshots, len(fromEventsSnapshotHashes)))
		}

		fromEventsIntervalToHistoryTableDelta = getSnapshotIntervalToHistoryTableDeltaSummary(ctx, sqlConfig.ConnectionConfig)

		// Copy all snapshots to the backup directory
		files, err := os.ReadDir(snapshotsDir)
		if err != nil {
			panic(err)
		}

		for _, file := range files {
			copyFile(filepath.Join(snapshotsDir, file.Name()), filepath.Join(snapshotsBackupDir, file.Name()))
		}
	})

	if exitCode != 0 {
		log.Errorf("One or more tests failed, dumping postgres log:\n%s", postgresLog.String())
	}
}

func TestGetHistoryIncludingDatanodeStateWhenDatanodeHasData(t *testing.T) {
	var datanodeOldestHistoryBlock *entities.Block
	var datanodeLastBlock *entities.Block

	var historySnapshots []snapshot.HistorySnapshot

	datanodeOldestHistoryBlock = &entities.Block{Height: 0}
	datanodeLastBlock = &entities.Block{Height: 5000}

	currentStateSnapshots := map[int64]snapshot.CurrentStateSnapshot{4000: {Height: 4000}}
	historySnapshots = []snapshot.HistorySnapshot{{HeightFrom: 2001, HeightTo: 3000}, {HeightFrom: 3001, HeightTo: 4000}}

	currentStateSnapshot, histories, _ := snapshot.GetHistoryIncludingDatanodeState(datanodeOldestHistoryBlock, datanodeLastBlock, "", currentStateSnapshots, historySnapshots)
	assert.Equal(t, int64(5000), currentStateSnapshot.Height)
	assert.Equal(t, len(histories), 1)
	assert.Equal(t, int64(0), histories[0].HeightFrom)
	assert.Equal(t, int64(5000), histories[0].HeightTo)

	currentStateSnapshots = map[int64]snapshot.CurrentStateSnapshot{6000: {Height: 6000}}
	historySnapshots = []snapshot.HistorySnapshot{
		{HeightFrom: 2001, HeightTo: 3000},
		{HeightFrom: 3001, HeightTo: 4000},
		{HeightFrom: 4001, HeightTo: 5000},
		{HeightFrom: 5001, HeightTo: 6000},
	}

	currentStateSnapshot, histories, _ = snapshot.GetHistoryIncludingDatanodeState(datanodeOldestHistoryBlock, datanodeLastBlock, "", currentStateSnapshots, historySnapshots)
	assert.Equal(t, int64(5000), currentStateSnapshot.Height)
	assert.Equal(t, len(histories), 1)
	assert.Equal(t, int64(0), histories[0].HeightFrom)
	assert.Equal(t, int64(5000), histories[0].HeightTo)

	datanodeOldestHistoryBlock = &entities.Block{Height: 2001}
	datanodeLastBlock = &entities.Block{Height: 5000}

	currentStateSnapshots = map[int64]snapshot.CurrentStateSnapshot{6000: {Height: 6000}}
	historySnapshots = []snapshot.HistorySnapshot{
		{HeightFrom: 1001, HeightTo: 2000},
		{HeightFrom: 2001, HeightTo: 3000},
		{HeightFrom: 3001, HeightTo: 4000},
		{HeightFrom: 4001, HeightTo: 5000},
		{HeightFrom: 5001, HeightTo: 6000},
	}

	currentStateSnapshot, histories, _ = snapshot.GetHistoryIncludingDatanodeState(datanodeOldestHistoryBlock, datanodeLastBlock, "", currentStateSnapshots, historySnapshots)
	assert.Equal(t, int64(5000), currentStateSnapshot.Height)
	assert.Equal(t, len(histories), 2)
	assert.Equal(t, int64(1001), histories[0].HeightFrom)
	assert.Equal(t, int64(2000), histories[0].HeightTo)
	assert.Equal(t, int64(2001), histories[1].HeightFrom)
	assert.Equal(t, int64(5000), histories[1].HeightTo)

	datanodeOldestHistoryBlock = &entities.Block{Height: 4001}
	datanodeLastBlock = &entities.Block{Height: 5000}

	currentStateSnapshots = map[int64]snapshot.CurrentStateSnapshot{6000: {Height: 6000}}
	historySnapshots = []snapshot.HistorySnapshot{
		{HeightFrom: 1001, HeightTo: 2000},
		{HeightFrom: 2001, HeightTo: 3000},
		{HeightFrom: 3001, HeightTo: 4000},
		{HeightFrom: 4001, HeightTo: 5000},
		{HeightFrom: 5001, HeightTo: 6000},
	}

	currentStateSnapshot, histories, _ = snapshot.GetHistoryIncludingDatanodeState(datanodeOldestHistoryBlock, datanodeLastBlock, "", currentStateSnapshots, historySnapshots)
	assert.Equal(t, int64(5000), currentStateSnapshot.Height)
	assert.Equal(t, len(histories), 4)
	assert.Equal(t, int64(1001), histories[0].HeightFrom)
	assert.Equal(t, int64(2000), histories[0].HeightTo)
	assert.Equal(t, int64(2001), histories[1].HeightFrom)
	assert.Equal(t, int64(3000), histories[1].HeightTo)
	assert.Equal(t, int64(3001), histories[2].HeightFrom)
	assert.Equal(t, int64(4000), histories[2].HeightTo)
	assert.Equal(t, int64(4001), histories[3].HeightFrom)
	assert.Equal(t, int64(5000), histories[3].HeightTo)

	datanodeOldestHistoryBlock = &entities.Block{Height: 4001}
	datanodeLastBlock = &entities.Block{Height: 5050}

	currentStateSnapshots = map[int64]snapshot.CurrentStateSnapshot{6000: {Height: 6000}}
	historySnapshots = []snapshot.HistorySnapshot{
		{HeightFrom: 1001, HeightTo: 2000},
		{HeightFrom: 2001, HeightTo: 3000},
		{HeightFrom: 3001, HeightTo: 4000},
		{HeightFrom: 4001, HeightTo: 5000},
		{HeightFrom: 5001, HeightTo: 6000},
	}

	currentStateSnapshot, histories, _ = snapshot.GetHistoryIncludingDatanodeState(datanodeOldestHistoryBlock, datanodeLastBlock, "", currentStateSnapshots, historySnapshots)
	assert.Equal(t, int64(5050), currentStateSnapshot.Height)
	assert.Equal(t, len(histories), 4)
	assert.Equal(t, int64(1001), histories[0].HeightFrom)
	assert.Equal(t, int64(2000), histories[0].HeightTo)
	assert.Equal(t, int64(2001), histories[1].HeightFrom)
	assert.Equal(t, int64(3000), histories[1].HeightTo)
	assert.Equal(t, int64(3001), histories[2].HeightFrom)
	assert.Equal(t, int64(4000), histories[2].HeightTo)
	assert.Equal(t, int64(4001), histories[3].HeightFrom)
	assert.Equal(t, int64(5050), histories[3].HeightTo)

	datanodeOldestHistoryBlock = &entities.Block{Height: 0}
	datanodeLastBlock = &entities.Block{Height: 5050}

	currentStateSnapshots = map[int64]snapshot.CurrentStateSnapshot{5000: {Height: 5000}, 6000: {Height: 6000}}
	historySnapshots = []snapshot.HistorySnapshot{
		{HeightFrom: 1001, HeightTo: 2000},
		{HeightFrom: 2001, HeightTo: 3000},
		{HeightFrom: 3001, HeightTo: 4000},
		{HeightFrom: 4001, HeightTo: 5000},
		{HeightFrom: 5001, HeightTo: 6000},
	}

	currentStateSnapshot, histories, _ = snapshot.GetHistoryIncludingDatanodeState(datanodeOldestHistoryBlock, datanodeLastBlock, "", currentStateSnapshots, historySnapshots)
	assert.Equal(t, int64(5050), currentStateSnapshot.Height)
	assert.Equal(t, len(histories), 1)
	assert.Equal(t, int64(0), histories[0].HeightFrom)
	assert.Equal(t, int64(5050), histories[0].HeightTo)
}

func TestGetHistoryIncludingDatanodeStatWhenDatanodeIsEmpty(t *testing.T) {
	var datanodeOldestHistoryBlock *entities.Block
	var datanodeLastBlock *entities.Block

	currentStateSnapshots := map[int64]snapshot.CurrentStateSnapshot{}
	var historySnapshots []snapshot.HistorySnapshot

	currentStateSnapshot, histories, _ := snapshot.GetHistoryIncludingDatanodeState(datanodeOldestHistoryBlock, datanodeLastBlock, "", currentStateSnapshots, historySnapshots)
	assert.Nil(t, currentStateSnapshot)
	assert.Nil(t, histories)

	currentStateSnapshots = map[int64]snapshot.CurrentStateSnapshot{3000: {Height: 3000}}

	currentStateSnapshot, histories, _ = snapshot.GetHistoryIncludingDatanodeState(datanodeOldestHistoryBlock, datanodeLastBlock, "", currentStateSnapshots, historySnapshots)
	assert.Equal(t, int64(3000), currentStateSnapshot.Height)
	assert.Nil(t, histories)

	currentStateSnapshots = map[int64]snapshot.CurrentStateSnapshot{3000: {Height: 3000}}
	historySnapshots = []snapshot.HistorySnapshot{{HeightFrom: 0, HeightTo: 1000}, {HeightFrom: 1001, HeightTo: 2000}, {HeightFrom: 2001, HeightTo: 3000}}
	currentStateSnapshot, histories, _ = snapshot.GetHistoryIncludingDatanodeState(datanodeOldestHistoryBlock, datanodeLastBlock, "", currentStateSnapshots, historySnapshots)
	assert.Equal(t, int64(3000), currentStateSnapshot.Height)
	assert.Equal(t, 3, len(histories))
	assert.Equal(t, int64(3000), histories[2].HeightTo)
	assert.Equal(t, int64(2001), histories[2].HeightFrom)
	assert.Equal(t, int64(2000), histories[1].HeightTo)
	assert.Equal(t, int64(1001), histories[1].HeightFrom)
	assert.Equal(t, int64(1000), histories[0].HeightTo)
	assert.Equal(t, int64(0), histories[0].HeightFrom)

	currentStateSnapshots = map[int64]snapshot.CurrentStateSnapshot{3000: {Height: 3000}}
	historySnapshots = []snapshot.HistorySnapshot{{HeightFrom: 1001, HeightTo: 2000}, {HeightFrom: 2001, HeightTo: 3000}}
	currentStateSnapshot, histories, _ = snapshot.GetHistoryIncludingDatanodeState(datanodeOldestHistoryBlock, datanodeLastBlock, "", currentStateSnapshots, historySnapshots)
	assert.Equal(t, int64(3000), currentStateSnapshot.Height)
	assert.Equal(t, 2, len(histories))
	assert.Equal(t, int64(3000), histories[1].HeightTo)
	assert.Equal(t, int64(2001), histories[1].HeightFrom)
	assert.Equal(t, int64(2000), histories[0].HeightTo)
	assert.Equal(t, int64(1001), histories[0].HeightFrom)

	currentStateSnapshots = map[int64]snapshot.CurrentStateSnapshot{3000: {Height: 3000}}
	historySnapshots = []snapshot.HistorySnapshot{{HeightFrom: 1001, HeightTo: 2000}, {HeightFrom: 2001, HeightTo: 3000}, {HeightFrom: 2001, HeightTo: 4000}}
	currentStateSnapshot, histories, _ = snapshot.GetHistoryIncludingDatanodeState(datanodeOldestHistoryBlock, datanodeLastBlock, "", currentStateSnapshots, historySnapshots)
	assert.Equal(t, int64(3000), currentStateSnapshot.Height)
	assert.Equal(t, 2, len(histories))
	assert.Equal(t, int64(3000), histories[1].HeightTo)
	assert.Equal(t, int64(2001), histories[1].HeightFrom)
	assert.Equal(t, int64(2000), histories[0].HeightTo)
	assert.Equal(t, int64(1001), histories[0].HeightFrom)

	currentStateSnapshots = map[int64]snapshot.CurrentStateSnapshot{3000: {Height: 3000}, 4000: {Height: 4000}}
	historySnapshots = []snapshot.HistorySnapshot{{HeightFrom: 1001, HeightTo: 2000}, {HeightFrom: 2001, HeightTo: 3000}, {HeightFrom: 3001, HeightTo: 4000}}
	currentStateSnapshot, histories, _ = snapshot.GetHistoryIncludingDatanodeState(datanodeOldestHistoryBlock, datanodeLastBlock, "", currentStateSnapshots, historySnapshots)
	assert.Equal(t, int64(4000), currentStateSnapshot.Height)
	assert.Equal(t, 3, len(histories))
	assert.Equal(t, int64(4000), histories[2].HeightTo)
	assert.Equal(t, int64(3001), histories[2].HeightFrom)
	assert.Equal(t, int64(3000), histories[1].HeightTo)
	assert.Equal(t, int64(2001), histories[1].HeightFrom)
	assert.Equal(t, int64(2000), histories[0].HeightTo)
	assert.Equal(t, int64(1001), histories[0].HeightFrom)

	currentStateSnapshots = map[int64]snapshot.CurrentStateSnapshot{6000: {Height: 6000}, 7000: {Height: 7000}}
	historySnapshots = []snapshot.HistorySnapshot{{HeightFrom: 1001, HeightTo: 2000}, {HeightFrom: 2001, HeightTo: 3000}, {HeightFrom: 3001, HeightTo: 4000}}
	currentStateSnapshot, histories, _ = snapshot.GetHistoryIncludingDatanodeState(datanodeOldestHistoryBlock, datanodeLastBlock, "", currentStateSnapshots, historySnapshots)
	assert.Equal(t, int64(7000), currentStateSnapshot.Height)
	assert.Nil(t, histories)

	currentStateSnapshots = map[int64]snapshot.CurrentStateSnapshot{}
	historySnapshots = []snapshot.HistorySnapshot{{HeightFrom: 1001, HeightTo: 2000}, {HeightFrom: 2001, HeightTo: 3000}, {HeightFrom: 3001, HeightTo: 4000}}
	currentStateSnapshot, histories, _ = snapshot.GetHistoryIncludingDatanodeState(datanodeOldestHistoryBlock, datanodeLastBlock, "", currentStateSnapshots, historySnapshots)
	assert.Nil(t, currentStateSnapshot)
	assert.Nil(t, histories)

	var err error
	datanodeOldestHistoryBlock = &entities.Block{}
	_, _, err = snapshot.GetHistoryIncludingDatanodeState(datanodeOldestHistoryBlock, datanodeLastBlock, "", currentStateSnapshots, historySnapshots)
	assert.NotNil(t, err)
}

func TestAlteringSnapshotIntervalBelowMinIntervalWithFileSource(t *testing.T) {
	brokerCfg := broker.NewDefaultConfig()
	brokerCfg.UseEventFile = true
	brokerCfg.FileEventSourceConfig.TimeBetweenBlocks = encoding.Duration{Duration: 0}

	snapshotsDir := t.TempDir()

	callcount := 0
	inputSnapshotService := setupSnapshotServiceWithNetworkParamFunc(sqlConfig, snapshotsDir,
		func(ctx context.Context, key string) (entities.NetworkParameter, error) {
			callcount++
			if callcount <= 1001 {
				return entities.NetworkParameter{
					Key:      netparams.SnapshotIntervalLength,
					Value:    "1000",
					TxHash:   "",
					VegaTime: time.Time{},
				}, nil
			}
			return entities.NetworkParameter{
				Key:      netparams.SnapshotIntervalLength,
				Value:    "300",
				TxHash:   "",
				VegaTime: time.Time{},
			}, nil
		}, brokerCfg)

	for i := 0; i <= 2000; i++ {
		inputSnapshotService.OnBlockCommitted(context.Background(), chainID, int64(i))
		if i == 1000 {
			heightTo := int64(1000)
			heightFrom := int64(0)
			waitForSnapshotToCompleteForHeights(heightTo, heightFrom, snapshotsDir)
		}

		if i == 2000 {
			heightTo := int64(2000)
			heightFrom := int64(1001)
			waitForSnapshotToCompleteForHeights(heightTo, heightFrom, snapshotsDir)
			break
		}
	}
}

func TestAlteringSnapshotInterval(t *testing.T) {
	emptyDatabase()

	brokerCfg := broker.NewDefaultConfig()
	brokerCfg.UseEventFile = false
	brokerCfg.FileEventSourceConfig.TimeBetweenBlocks = encoding.Duration{Duration: 0}

	snapshotsDir := t.TempDir()

	callcount := 0
	inputSnapshotService := setupSnapshotServiceWithNetworkParamFunc(sqlConfig, snapshotsDir,
		func(ctx context.Context, key string) (entities.NetworkParameter, error) {
			callcount++
			if callcount <= 1001 {
				return entities.NetworkParameter{
					Key:      netparams.SnapshotIntervalLength,
					Value:    "1000",
					TxHash:   "",
					VegaTime: time.Time{},
				}, nil
			}
			return entities.NetworkParameter{
				Key:      netparams.SnapshotIntervalLength,
				Value:    "500",
				TxHash:   "",
				VegaTime: time.Time{},
			}, nil
		}, brokerCfg)

	for i := 0; i <= 1500; i++ {
		inputSnapshotService.OnBlockCommitted(context.Background(), chainID, int64(i))
		if i == 1000 {
			heightTo := int64(1000)
			heightFrom := int64(0)
			waitForSnapshotToCompleteForHeights(heightTo, heightFrom, snapshotsDir)
		}
		if i == 1500 {
			heightTo := int64(1500)
			heightFrom := int64(1001)
			waitForSnapshotToCompleteForHeights(heightTo, heightFrom, snapshotsDir)
		}
	}
}

func waitForSnapshotToCompleteForHeights(heightTo int64, heightFrom int64, snapshotsDir string) {
	cs := snapshot.NewCurrentSnapshot(chainID, heightTo)
	hist := snapshot.NewHistorySnapshot(chainID, heightFrom, heightTo)
	csFile := filepath.Join(snapshotsDir, cs.CompressedFileName())
	histFile := filepath.Join(snapshotsDir, hist.CompressedFileName())
	progressFile := filepath.Join(snapshotsDir, snapshot.InProgressFileName(chainID, heightTo))
	waitForSnapshotToComplete(csFile, histFile, progressFile)
}

func TestLoadingAllAvailableHistoryWithNonEmptyDatanode(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := logging.NewTestLogger()

	emptyDatabase()
	snapshotsDir := t.TempDir()
	copySnapshotDataIntoSnapshotDirectory(2001, 4000, snapshotsDir)

	inputSnapshotService := setupSnapshotService(sqlConfig, snapshotsDir)

	from, to, err := inputSnapshotService.LoadAllAvailableHistory(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2001), from)
	assert.Equal(t, int64(4000), to)

	copySnapshotDataIntoSnapshotDirectory(0, 3000, snapshotsDir)
	inputSnapshotService = setupSnapshotService(sqlConfig, snapshotsDir)
	from, to, err = inputSnapshotService.LoadAllAvailableHistory(ctx)
	if err != nil {
		log.Errorf("failed to load available history:%s", err)
		fmt.Printf("failed to load available history:%s", err)
	}
	require.NoError(t, err)
	assert.Equal(t, int64(0), from)
	assert.Equal(t, int64(4000), to)

	summary := getDatabaseDataSummary(ctx, sqlConfig.ConnectionConfig)
	assertTableSummariesAreEqual(t, fromEventsDatabaseSummaries[3].currentTableSummaries, summary.currentTableSummaries)
	assertTableSummariesAreEqual(t, fromEventsDatabaseSummaries[3].historyTableSummaries, summary.historyTableSummaries)

	// Play events to 5000

	var snapshotAt5000 string
	outSnapshotDir := t.TempDir()
	outputSnapshotService := setupSnapshotService(sqlConfig, outSnapshotDir)
	sqlBroker, err := setupSQLBroker(ctx, eventsFile, sqlConfig, outputSnapshotService,
		func(ctx context.Context, service *snapshot.Service, chainId string, lastCommittedBlockHeight int64) bool {
			if lastCommittedBlockHeight > 0 && lastCommittedBlockHeight%snapshotInterval == 0 {
				if lastCommittedBlockHeight == 5000 {
					ss, err := service.CreateSnapshotAsync(ctx, chainId, snapshot.GetFromHeight(lastCommittedBlockHeight, snapshotInterval), lastCommittedBlockHeight)
					require.NoError(t, err)
					waitForSnapshotToCompleteUseMeta(ss, outSnapshotDir)
					snapshotAt5000, err = snapshot.GetSnapshotMd5Hash(logging.NewTestLogger(), ss.CurrentStateSnapshotFile,
						ss.HistorySnapshotFile)
					require.NoError(t, err)
					return true
				}
			}

			return false
		})
	require.NoError(t, err)

	err = sqlBroker.Receive(ctx)
	require.NoError(t, err)

	require.Equal(t, fromEventsSnapshotHashes[4], snapshotAt5000)

	summaryAt5000 := getDatabaseDataSummary(ctx, sqlConfig.ConnectionConfig)

	assertTableSummariesAreEqual(t, fromEventsDatabaseSummaries[4].currentTableSummaries, summaryAt5000.currentTableSummaries)
	assertTableSummariesAreEqual(t, fromEventsDatabaseSummaries[4].historyTableSummaries, summaryAt5000.historyTableSummaries)
}

func TestLoadingAllAvailableHistoryWithJustCurrentStateSnapshot(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var err error

	snapshotsDir := t.TempDir()
	emptyDatabase()

	// Load database just from current snapshot state at height 3000
	copySnapshotDataIntoSnapshotDirectory(3000, 3000, snapshotsDir)

	inputSnapshotService := setupSnapshotService(sqlConfig, snapshotsDir)

	from, to, err := inputSnapshotService.LoadAllAvailableHistory(ctx)
	assert.Equal(t, int64(3000), from)
	assert.Equal(t, int64(3000), to)

	require.NoError(t, err)

	databaseSummaryAfterReloadAtHeight3000 := getDatabaseDataSummary(ctx, sqlConfig.ConnectionConfig)

	assertTableSummariesAreEqual(t, fromEventsDatabaseSummaries[2].currentTableSummaries, databaseSummaryAfterReloadAtHeight3000.currentTableSummaries)
	for tableName, historyTableSummary := range databaseSummaryAfterReloadAtHeight3000.historyTableSummaries {
		assert.Equal(t, 0, historyTableSummary.rowCount, "history table %s should be empty", tableName)
	}

	// Play events from 3001 to 5000
	var snapshotMeta snapshot.Meta
	var newSnapshotFileHashAt5000 string
	outSnapshotDir := t.TempDir()
	outputSnapshotService := setupSnapshotService(sqlConfig, outSnapshotDir)
	sqlBroker, err := setupSQLBroker(ctx, eventsFile, sqlConfig, outputSnapshotService,
		func(ctx context.Context, service *snapshot.Service, chainId string, lastCommittedBlockHeight int64) bool {
			if lastCommittedBlockHeight > 0 && lastCommittedBlockHeight%snapshotInterval == 0 {
				snapshotMeta, err = service.CreateSnapshotAsync(ctx, chainId, snapshot.GetFromHeight(lastCommittedBlockHeight, snapshotInterval), lastCommittedBlockHeight)
				require.NoError(t, err)

				waitForSnapshotToCompleteUseMeta(snapshotMeta, outSnapshotDir)

				if lastCommittedBlockHeight == 5000 {
					newSnapshotFileHashAt5000, err = snapshot.GetSnapshotMd5Hash(logging.NewTestLogger(), snapshotMeta.CurrentStateSnapshotFile,
						snapshotMeta.HistorySnapshotFile)
					require.NoError(t, err)
					return true
				}
			}

			return false
		})
	require.NoError(t, err)

	err = sqlBroker.Receive(ctx)
	require.NoError(t, err)

	assert.Equal(t, fromEventsSnapshotHashes[4], newSnapshotFileHashAt5000)

	databaseSummaryAfterLoadAndReplayToBlock5000 := getDatabaseDataSummary(ctx, sqlConfig.ConnectionConfig)

	assertTableSummariesAreEqual(t, fromEventsDatabaseSummaries[4].currentTableSummaries, databaseSummaryAfterLoadAndReplayToBlock5000.currentTableSummaries)

	historyTableDelta := getSnapshotIntervalToHistoryTableDeltaSummary(ctx, sqlConfig.ConnectionConfig)

	for i := 3; i < numSnapshots; i++ {
		assertTableSummariesAreEqual(t, fromEventsIntervalToHistoryTableDelta[i], historyTableDelta[i])
	}

	assertIntervalHistoryIsEmpty(t, historyTableDelta, 0)
	assertIntervalHistoryIsEmpty(t, historyTableDelta, 1)
	assertIntervalHistoryIsEmpty(t, historyTableDelta, 2)
}

func TestRestoreFromPartialHistoryAndProcessEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var err error

	emptyDatabase()
	snapshotDir := t.TempDir()
	copySnapshotDataIntoSnapshotDirectory(2001, 3000, snapshotDir)

	inputSnapshotService := setupSnapshotService(sqlConfig, snapshotDir)

	// Load database just from history 2001 to 3000
	from, to, err := inputSnapshotService.LoadAllAvailableHistory(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2001), from)
	assert.Equal(t, int64(3000), to)

	// Play events from 3001 to 4000
	var snapshotMeta snapshot.Meta
	var newSnapshotFileHashAt4000 string
	outSnapshotDir := t.TempDir()
	outputSnapshotService := setupSnapshotService(sqlConfig, outSnapshotDir)
	sqlBroker, err := setupSQLBroker(ctx, eventsFile, sqlConfig, outputSnapshotService,
		func(ctx context.Context, service *snapshot.Service, chainId string, lastCommittedBlockHeight int64) bool {
			if lastCommittedBlockHeight > 0 && lastCommittedBlockHeight%snapshotInterval == 0 {
				snapshotMeta, err = service.CreateSnapshotAsync(ctx, chainId, snapshot.GetFromHeight(lastCommittedBlockHeight, snapshotInterval), lastCommittedBlockHeight)
				require.NoError(t, err)
				waitForSnapshotToCompleteUseMeta(snapshotMeta, outSnapshotDir)

				if lastCommittedBlockHeight == 4000 {
					newSnapshotFileHashAt4000, err = snapshot.GetSnapshotMd5Hash(logging.NewTestLogger(), snapshotMeta.CurrentStateSnapshotFile,
						snapshotMeta.HistorySnapshotFile)
					require.NoError(t, err)
				}

				if lastCommittedBlockHeight == 5000 {
					return true
				}
			}

			return false
		})
	require.NoError(t, err)

	err = sqlBroker.Receive(ctx)
	require.NoError(t, err)

	assert.Equal(t, fromEventsSnapshotHashes[3], newSnapshotFileHashAt4000)

	historyTableDelta := getSnapshotIntervalToHistoryTableDeltaSummary(ctx, sqlConfig.ConnectionConfig)

	for i := 2; i < 4; i++ {
		assertTableSummariesAreEqual(t, fromEventsIntervalToHistoryTableDelta[i], historyTableDelta[i])
	}

	assertIntervalHistoryIsEmpty(t, historyTableDelta, 0)
	assertIntervalHistoryIsEmpty(t, historyTableDelta, 1)
}

func TestRestoreFromFullHistorySnapshotAndProcessEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	emptyDatabase()
	snapshotDir := t.TempDir()
	copySnapshotDataIntoSnapshotDirectory(0, 2000, snapshotDir)

	inputSnapshotService := setupSnapshotService(sqlConfig, snapshotDir)

	from, to, err := inputSnapshotService.LoadAllAvailableHistory(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), from)
	assert.Equal(t, int64(2000), to)

	var snapshotFileHashAfterReloadAt2000AndEventReplayTo3000 string
	outSnapshotDir := t.TempDir()
	outputSnapshotService := setupSnapshotService(sqlConfig, outSnapshotDir)
	sqlBroker, err := setupSQLBroker(ctx, eventsFile, sqlConfig, outputSnapshotService,
		func(ctx context.Context, service *snapshot.Service, chainId string, lastCommittedBlockHeight int64) bool {
			if lastCommittedBlockHeight > 0 && lastCommittedBlockHeight%snapshotInterval == 0 {
				if lastCommittedBlockHeight == 3000 {
					ss, err := service.CreateSnapshotAsync(ctx, chainId, snapshot.GetFromHeight(lastCommittedBlockHeight, snapshotInterval), lastCommittedBlockHeight)
					require.NoError(t, err)
					waitForSnapshotToCompleteUseMeta(ss, outSnapshotDir)
					snapshotFileHashAfterReloadAt2000AndEventReplayTo3000, err = snapshot.GetSnapshotMd5Hash(logging.NewTestLogger(), ss.CurrentStateSnapshotFile, ss.HistorySnapshotFile)
					require.NoError(t, err)
					return true
				}
			}

			return false
		})
	require.NoError(t, err)

	err = sqlBroker.Receive(ctx)
	require.NoError(t, err)

	require.Equal(t, fromEventsSnapshotHashes[2], snapshotFileHashAfterReloadAt2000AndEventReplayTo3000)

	databaseSummaryAtBlock3000AfterSnapshotReloadFromBlock2000 := getDatabaseDataSummary(ctx, sqlConfig.ConnectionConfig)

	assertTableSummariesAreEqual(t, fromEventsDatabaseSummaries[2].currentTableSummaries, databaseSummaryAtBlock3000AfterSnapshotReloadFromBlock2000.currentTableSummaries)
	assertTableSummariesAreEqual(t, fromEventsDatabaseSummaries[2].historyTableSummaries, databaseSummaryAtBlock3000AfterSnapshotReloadFromBlock2000.historyTableSummaries)
}

func emptyDatabase() {
	databasetest.DeleteEverything()
}

func TestRestoringFromDifferentHeightsWithFullHistory(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	snapshotDir := t.TempDir()

	for i := int64(0); i < numSnapshots; i++ {
		emptyDatabase()
		copySnapshotDataIntoSnapshotDirectory(0, snapshotInterval*(i+1), snapshotDir)

		inputSnapshotService := setupSnapshotService(sqlConfig, snapshotDir)
		_, _, err := inputSnapshotService.LoadAllAvailableHistory(ctx)
		require.NoError(t, err)

		dbSummary := getDatabaseDataSummary(ctx, sqlConfig.ConnectionConfig)
		assertTableSummariesAreEqual(t, fromEventsDatabaseSummaries[i].currentTableSummaries, dbSummary.currentTableSummaries)
		assertTableSummariesAreEqual(t, fromEventsDatabaseSummaries[i].historyTableSummaries, dbSummary.historyTableSummaries)
	}
}

type sqlStoreBroker interface {
	Receive(ctx context.Context) error
}

func copySnapshotDataIntoSnapshotDirectory(fromHeight int64, toHeight int64, snapshotsDir string) {
	_, histories, err := snapshot.GetHistorySnapshots(snapshotsBackupDir)
	if err != nil {
		panic(err)
	}

	_, csSnapshots, err := snapshot.GetCurrentStateSnapshots(snapshotsBackupDir)
	if err != nil {
		panic(err)
	}

	for _, history := range histories {
		if history.HeightFrom >= fromHeight && history.HeightTo <= toHeight {
			copyFile(filepath.Join(snapshotsBackupDir, history.CompressedFileName()), filepath.Join(snapshotsDir, history.CompressedFileName()))
		}
	}

	for _, csSnapshot := range csSnapshots {
		if csSnapshot.Height >= fromHeight && csSnapshot.Height <= toHeight {
			copyFile(filepath.Join(snapshotsBackupDir, csSnapshot.CompressedFileName()), filepath.Join(snapshotsDir, csSnapshot.CompressedFileName()))
		}
	}
}

func copyFile(src string, dst string) {
	data, err := os.ReadFile(src)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(dst, data, 0o644)
	if err != nil {
		panic(err)
	}
}

func panicIfSnapshotHashNotEqual(expected string, actual string, snapshots []snapshot.Meta) {
	if expected != actual {
		snapshotPaths := ""
		for _, sn := range snapshots {
			snapshotPaths += "," + sn.CurrentStateSnapshotFile + "," + sn.HistorySnapshotFile
		}

		panic(fmt.Errorf("snapshot hashes are not equal, expected: %s  actual: %s\n"+
			"If the database schema has changed or event file been updated the snapshot hashes "+
			"will need updating.  Snapshot files: %s", expected, actual, snapshotPaths))
	}
}

func assertIntervalHistoryIsEmpty(t *testing.T, historyTableDelta []map[string]tableDataSummary, interval int) {
	t.Helper()
	totalRowCount := 0
	for _, summary := range historyTableDelta[interval] {
		totalRowCount += summary.rowCount
	}
	assert.Equal(t, 0, totalRowCount, "expected interval history to be empty but found %d rows", totalRowCount)
}

func setupSnapshotService(testDbConfig sqlstore.Config, snapshotsDirectory string) *snapshot.Service {
	getNetworkParam := func(ctx context.Context, key string) (entities.NetworkParameter, error) {
		return entities.NetworkParameter{
			Key:      netparams.SnapshotIntervalLength,
			Value:    "1000",
			TxHash:   "",
			VegaTime: time.Time{},
		}, nil
	}

	brokerCfg := broker.NewDefaultConfig()
	brokerCfg.UseEventFile = true
	brokerCfg.FileEventSourceConfig.TimeBetweenBlocks = encoding.Duration{Duration: 0}

	return setupSnapshotServiceWithNetworkParamFunc(testDbConfig, snapshotsDirectory, getNetworkParam, brokerCfg)
}

func setupSnapshotServiceWithNetworkParamFunc(testDbConfig sqlstore.Config, snapshotsDirectory string,
	getNetworkParam func(ctx context.Context, key string) (entities.NetworkParameter, error),
	brokerConfig broker.Config,
) *snapshot.Service {
	snapshotServiceCfg := snapshot.NewDefaultConfig()
	snapshotServiceCfg.DatabaseSnapshotsPath = snapshotsDirectory
	snapshotServiceCfg.Enabled = true

	connSource, err := sqlstore.NewTransactionalConnectionSource(logging.NewTestLogger(), testDbConfig.ConnectionConfig)
	if err != nil {
		panic(err)
	}

	blockStore := sqlstore.NewBlocks(connSource)
	chainService := service.NewChain(sqlstore.NewChain(connSource), logging.NewTestLogger())

	snapshotService, err := snapshot.NewSnapshotService(logging.NewTestLogger(), snapshotServiceCfg, brokerConfig,
		blockStore, getNetworkParam, chainService, testDbConfig.ConnectionConfig, snapshotsDirectory)
	if err != nil {
		panic(err)
	}

	return snapshotService
}

func setupSQLBroker(ctx context.Context, eventsFile string, testDbConfig sqlstore.Config, snapshotService *snapshot.Service,
	onBlockCommitted func(ctx context.Context, service *snapshot.Service, chainId string, lastCommittedBlockHeight int64) bool,
) (sqlStoreBroker, error) {
	transactionalConnectionSource, err := sqlstore.NewTransactionalConnectionSource(logging.NewTestLogger(), testDbConfig.ConnectionConfig)
	if err != nil {
		return nil, err
	}

	candlesV2Config := candlesv2.NewDefaultConfig()
	subscribers := start.SQLSubscribers{}
	subscribers.CreateAllStores(ctx, logging.NewTestLogger(), transactionalConnectionSource, candlesV2Config.CandleStore)
	err = subscribers.SetupServices(ctx, logging.NewTestLogger(), candlesV2Config)
	if err != nil {
		return nil, err
	}

	subscribers.SetupSQLSubscribers(ctx, logging.NewTestLogger())

	evtSource, err := broker.NewFileEventSource(eventsFile, 0, 0)
	if err != nil {
		return nil, err
	}

	blockStore := sqlstore.NewBlocks(transactionalConnectionSource)

	config := broker.NewDefaultConfig()

	sqlBroker := broker.NewSQLStoreBroker(logging.NewTestLogger(), config, testChainInfo{chainID: chainID}, evtSource,
		transactionalConnectionSource, blockStore, func(ctx context.Context, chainId string, lastCommittedBlockHeight int64) bool {
			return onBlockCommitted(ctx, snapshotService, chainId, lastCommittedBlockHeight)
		}, subscribers.GetSQLSubscribers(),
	)
	return sqlBroker, nil
}

type testChainInfo struct {
	chainID string
}

func (t testChainInfo) SetChainID(s string) error {
	panic("implement me")
}

func (t testChainInfo) GetChainID() (string, error) {
	return t.chainID, nil
}

type tableDataSummary struct {
	tableName string
	rowCount  int
	dataHash  string
}

func assertTableSummariesAreEqual(t *testing.T, expected map[string]tableDataSummary, actual map[string]tableDataSummary) {
	t.Helper()
	if len(expected) != len(actual) {
		require.Equalf(t, len(expected), len(actual), "expected table count: %d actual: %d", len(expected), len(actual))
	}

	for k, v := range expected {
		if v.rowCount != actual[k].rowCount {
			assert.Equalf(t, v.rowCount, actual[k].rowCount, "expected table row count for %s: %d actual row count: %d", k, v.rowCount, actual[k].rowCount)
		}

		if v.dataHash != actual[k].dataHash {
			assert.Equalf(t, v.dataHash, actual[k].dataHash, "expected data hash for %s: %s actual data hash: %s", k, v.dataHash, actual[k].dataHash)
		}
	}
}

type databaseSummary struct {
	currentTableSummaries map[string]tableDataSummary
	historyTableSummaries map[string]tableDataSummary
	dbMetaData            snapshot.DatabaseMetadata
}

func getDatabaseDataSummary(ctx context.Context, connConfig sqlstore.ConnectionConfig) databaseSummary {
	conn, err := pgxpool.Connect(ctx, connConfig.GetConnectionString())
	if err != nil {
		panic(err)
	}

	currentStateDataSummaries := map[string]tableDataSummary{}
	historyStateDataSummaries := map[string]tableDataSummary{}
	dbMetaData, err := snapshot.NewDatabaseMetaData(ctx, connConfig)
	if err != nil {
		panic(err)
	}

	for table, meta := range dbMetaData.TableNameToMetaData {
		summary := tableDataSummary{tableName: table}
		err = conn.QueryRow(ctx, fmt.Sprintf("select count(*) from %s", table)).Scan(&summary.rowCount)
		if err != nil {
			panic(err)
		}

		if summary.rowCount > 0 {
			err = conn.QueryRow(ctx, fmt.Sprintf("SELECT md5(CAST((array_agg(f.* order by %s))AS text)) FROM %s f; ",
				meta.SortOrder, table)).Scan(&summary.dataHash)
			if err != nil {
				panic(err)
			}
		}

		if meta.Hypertable {
			historyStateDataSummaries[table] = summary
		} else {
			currentStateDataSummaries[table] = summary
		}
	}

	return databaseSummary{
		historyTableSummaries: historyStateDataSummaries, currentTableSummaries: currentStateDataSummaries,
		dbMetaData: dbMetaData,
	}
}

func getSnapshotIntervalToHistoryTableDeltaSummary(ctx context.Context,
	connConfig sqlstore.ConnectionConfig,
) []map[string]tableDataSummary {
	conn, err := pgxpool.Connect(ctx, connConfig.GetConnectionString())
	if err != nil {
		panic(err)
	}

	dbMetaData, err := snapshot.NewDatabaseMetaData(ctx, connConfig)
	if err != nil {
		panic(err)
	}

	var snapshotNumToHistoryTableSummary []map[string]tableDataSummary

	for i := 0; i < numSnapshots; i++ {
		toHeight := int64(i+1) * snapshotInterval
		fromHeight := snapshot.GetFromHeight(toHeight, snapshotInterval)

		whereClause := fmt.Sprintf("Where vega_time >= (SELECT vega_time from blocks where height = %d) and  vega_time <= (SELECT vega_time from blocks where height = %d)",
			fromHeight, toHeight)

		intervalHistoryTableSummary := map[string]tableDataSummary{}
		for table, meta := range dbMetaData.TableNameToMetaData {
			if meta.Hypertable {
				summary := tableDataSummary{tableName: table}
				err := conn.QueryRow(ctx, fmt.Sprintf("select count(*) from %s %s", table, whereClause)).Scan(&summary.rowCount)
				if err != nil {
					panic(err)
				}

				if summary.rowCount > 0 {
					err = conn.QueryRow(ctx, fmt.Sprintf("SELECT md5(CAST((array_agg(f.* order by %s))AS text)) FROM %s f %s; ",
						meta.SortOrder, table, whereClause)).Scan(&summary.dataHash)
					if err != nil {
						panic(err)
					}
				}

				intervalHistoryTableSummary[table] = summary
			}
		}
		snapshotNumToHistoryTableSummary = append(snapshotNumToHistoryTableSummary, intervalHistoryTableSummary)
	}
	return snapshotNumToHistoryTableSummary
}

func waitForSnapshotToCompleteUseMeta(sn snapshot.Meta, snapshotDir string) {
	currentSnapshotFileName := sn.CurrentStateSnapshotFile
	historySnapshotFileName := sn.HistorySnapshotFile
	snapshotInProgressFileName := filepath.Join(snapshotDir, snapshot.InProgressFileName(sn.ChainID, sn.HeightTo))

	waitForSnapshotToComplete(currentSnapshotFileName, historySnapshotFileName, snapshotInProgressFileName)
}

func waitForSnapshotToComplete(currentSnapshotFileName string, historySnapshotFileName string, snapshotInProgressFileName string) {
	for {
		time.Sleep(10 * time.Millisecond)
		// wait for snapshot current  file
		_, err := os.Stat(currentSnapshotFileName)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			} else {
				panic(err)
			}
		}

		// wait for snapshot history file
		_, err = os.Stat(historySnapshotFileName)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			} else {
				panic(err)
			}
		}

		// wait for snapshot data dump in progress file to be removed

		_, err = os.Stat(snapshotInProgressFileName)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				break
			} else {
				panic(err)
			}
		} else {
			continue
		}
	}
}

func decompressEventFile() {
	sourceFile, err := os.Open(compressedEventsFile)
	if err != nil {
		panic(err)
	}

	zr, err := gzip.NewReader(sourceFile)
	if err != nil {
		panic(err)
	}

	fileToWrite, err := os.Create(eventsFile)
	if err != nil {
		panic(err)
	}
	if _, err := io.Copy(fileToWrite, zr); err != nil {
		panic(err)
	}
}
