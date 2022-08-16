package snapshot

import (
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	pg_commands "github.com/habx/pg-commands"
	"github.com/jackc/pgx/v4"
)

type Service struct {
	log        *logging.Logger
	config     Config
	connConfig sqlstore.ConnectionConfig
	paths      paths.Paths
}

const (
	snapshotFileExtension    = "datanode-snapshot"
	PgDumpCustomOutputFormat = "c"
)

func NewSnapshotService(log *logging.Logger, config Config, connectionConfig sqlstore.ConnectionConfig,
	paths paths.Paths,
) *Service {
	service := &Service{
		log:        log,
		config:     config,
		connConfig: connectionConfig,
		paths:      paths,
	}

	return service
}

func (b *Service) LoadSnapshot(ctx context.Context) (bool, error) {
	if b.config.StartHeight <= 0 {
		return false, nil
	}

	postgresDbConn, err := pgx.Connect(context.Background(), b.connConfig.GetConnectionStringForPostgresDatabase())
	if err != nil {
		return false, fmt.Errorf("unable to connect to database:%w", err)
	}

	defer func() {
		err := postgresDbConn.Close(context.Background())
		if err != nil {
			b.log.Errorf("error closing database connection after loading snapshot:%v", err)
		}
	}()

	_, err = postgresDbConn.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH ( FORCE )", b.connConfig.Database))
	if err != nil {
		return false, fmt.Errorf("unable to drop existing database:%w", err)
	}

	_, err = postgresDbConn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", b.connConfig.Database))
	if err != nil {
		return false, fmt.Errorf("unable to create database:%w", err)
	}

	vegaDbConn, err := pgx.Connect(context.Background(), b.connConfig.GetConnectionString())
	if err != nil {
		return false, fmt.Errorf("unable to connect to vega database:%w", err)
	}

	_, err = vegaDbConn.Exec(ctx, "select timescaledb_pre_restore();")
	if err != nil {
		return false, fmt.Errorf("problem running pre-restore:%w", err)
	}

	defer func() {
		_, err := vegaDbConn.Exec(ctx, "select timescaledb_post_restore();")
		if err != nil {
			b.log.Errorf("error running timescale post restore after loading snapshot:%v", err)
		}
	}()

	snapshotPath := b.paths.StatePathFor(paths.DataNodeSnapshotHome)

	filename, err := getSnapshot(snapshotPath, b.config.ChainId, b.config.StartHeight)
	if err != nil {
		return false, fmt.Errorf("failed to get snapshot filename:%w", err)
	}

	restore := &pg_commands.Restore{Options: []string{"-F" + PgDumpCustomOutputFormat}, Postgres: &pg_commands.Postgres{
		Host:     b.connConfig.Host,
		Port:     b.connConfig.Port,
		DB:       b.connConfig.Database,
		Username: b.connConfig.Username,
		Password: b.connConfig.Password,
	}, Schemas: []string{}}

	restore.SetPath(snapshotPath + string(os.PathSeparator))
	b.log.Info("restoring data node from snapshot:" + filename)

	start := time.Now()
	result := restore.Exec(filename, pg_commands.ExecOptions{StreamPrint: true})
	if result.Error == nil {
		b.log.Info(fmt.Sprintf("restored data node from snapshot: %s, time taken %s", filename, time.Now().Sub(start)))
	} else {
		return false, fmt.Errorf("failed to restore snapshot %s, error: %v", filename, result)
	}

	return true, nil
}

func (b *Service) OnBlockCommitted(chainId string, blockHeight int64) {
	if b.snapshotRequiredAtBlockHeight(blockHeight) {
		go b.createSnapshot(chainId, blockHeight)
	}
}

func (b *Service) snapshotRequiredAtBlockHeight(lastCommittedBlockHeight int64) bool {
	if b.config.BlockInterval > 0 {
		return lastCommittedBlockHeight > 0 && lastCommittedBlockHeight%b.config.BlockInterval == 0
	}

	return false
}

func (b *Service) createSnapshot(chainId string, lastCommittedBlockHeight int64) {
	b.log.Infof("creating snapshot for chain %s, height %d", chainId, lastCommittedBlockHeight)

	dump := pg_commands.NewDump(&pg_commands.Postgres{
		Host:     b.connConfig.Host,
		Port:     b.connConfig.Port,
		DB:       b.connConfig.Database,
		Username: b.connConfig.Username,
		Password: b.connConfig.Password,
	})

	dump.SetupFormat(PgDumpCustomOutputFormat)

	snapshotPath, err := b.paths.CreateStatePathFor(paths.StatePath(paths.DataNodeSnapshotHome.String()))
	if err != nil {
		b.log.Errorf("failed to create snapshot at height %d, unable to create snapshot file: %v", lastCommittedBlockHeight, err)
		return
	}

	dump.SetPath(snapshotPath + string(os.PathSeparator))

	snapshotInfo := newInfo(chainId, lastCommittedBlockHeight)
	dump.SetFileName(snapshotInfo.FileName())
	start := time.Now()
	result := dump.Exec(pg_commands.ExecOptions{StreamPrint: false})

	if result.Error == nil {
		b.log.Infof("finished creating snapshot for chain %s, height %d, time taken %s", chainId, lastCommittedBlockHeight,
			time.Now().Sub(start))
	} else {
		b.log.Errorf("failed to create snapshot for chain %s, height %d,  error: %v", chainId, lastCommittedBlockHeight,
			*result.Error)
	}

	b.removeOldSnapshotFiles()
}

func (b *Service) removeOldSnapshotFiles() {
	snapshotFiles, err := getSnapshotFiles(b.paths.StatePathFor(paths.DataNodeSnapshotHome))
	if err != nil {
		b.log.Errorf("failed to find snapshot filenames:%w", err)
	}

	toRemove := len(snapshotFiles) - b.config.HistorySize

	if toRemove > 0 {
		sort.Slice(snapshotFiles, func(i int, j int) bool {
			return snapshotFiles[i].ModTime().Before(snapshotFiles[j].ModTime())
		})

		for i := 0; i < toRemove; i++ {
			fileToRemove := snapshotFiles[i]
			err := os.Remove(filepath.Join(b.paths.StatePathFor(paths.DataNodeSnapshotHome), fileToRemove.Name()))
			if err != nil {
				b.log.Errorf("failed to remove old snapshot file:%w", err)
			}
		}
	}
}

func ListSnapshots(vegaPaths paths.Paths) ([]Info, error) {
	snapshotFiles, err := getSnapshotFiles(vegaPaths.StatePathFor(paths.DataNodeSnapshotHome))
	if err != nil {
		return nil, fmt.Errorf("failed to find snapshot filenames:%w", err)
	}

	snapshots := make([]Info, 0, 5)
	for _, file := range snapshotFiles {
		info, err := InfoFromFileName(file.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to get snapshot information from filename:%w", err)
		}

		snapshots = append(snapshots, info)
	}

	return snapshots, nil
}

func getSnapshot(snapshotPath string, chainId string, height int64) (string, error) {
	if len(chainId) == 0 {
		snapshotFilenamePattern := fmt.Sprintf(".*-%d.%s", height, snapshotFileExtension)
		matchingFiles, err := getFilesMatchingPattern(snapshotPath, snapshotFilenamePattern)
		if err != nil {
			return "", fmt.Errorf("failed to find matching filenames:%w", err)
		}

		if len(matchingFiles) == 0 {
			return "", fmt.Errorf("snapshot for blockheight %d not found", height)
		} else if len(matchingFiles) > 1 {
			return "", fmt.Errorf("found more than 1 snapshot for blockheight %d, try specifying the chain id", height)
		}

		return matchingFiles[0].Name(), nil
	} else {
		return fmt.Sprintf("%s-%d.%s", chainId, height, snapshotFileExtension), nil
	}
}

func getFilesMatchingPattern(path string, filenamePattern string) ([]fs.FileInfo, error) {
	regEx, err := regexp.Compile(filenamePattern)
	if err != nil {
		return nil, fmt.Errorf("unable to create filename matcher:%w", err)
	}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get files in snapshot directory:%w", err)
	}

	var matchingFiles []fs.FileInfo
	for _, file := range files {
		if !file.IsDir() {
			if regEx.MatchString(file.Name()) {
				matchingFiles = append(matchingFiles, file)
			}
		}
	}

	return matchingFiles, nil
}

func getSnapshotFiles(snapshotPath string) ([]fs.FileInfo, error) {
	snapshotFilenamePattern := fmt.Sprintf(".*.%s", snapshotFileExtension)
	snapshotFiles, err := getFilesMatchingPattern(snapshotPath, snapshotFilenamePattern)
	return snapshotFiles, err
}

type Info struct {
	ChainId string
	Height  int64
}

func newInfo(chainId string, height int64) Info {
	return Info{
		ChainId: chainId,
		Height:  height,
	}
}

func InfoFromFileName(filename string) (Info, error) {
	re := regexp.MustCompile("(.*)-(\\d+)." + snapshotFileExtension)
	matches := re.FindStringSubmatch(filename)
	if len(matches) != 3 {
		return Info{}, fmt.Errorf("invalid snapshot file name, unable to determine snapshot information")
	}

	height, err := strconv.ParseInt(matches[2], 10, 64)
	if err != nil {
		return Info{}, fmt.Errorf("invalid snapshot blockheight in file name:%w", err)
	}

	return Info{
		ChainId: matches[1],
		Height:  height,
	}, nil
}

func (i Info) FileName() string {
	return fmt.Sprintf("%s-%d.%s", i.ChainId, i.Height, snapshotFileExtension)
}
