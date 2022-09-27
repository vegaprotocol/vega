package snapshot

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/logging"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Meta struct {
	ChainID                  string
	HeightFrom               int64
	HeightTo                 int64
	CurrentStateSnapshotFile string
	HistorySnapshotFile      string
	DatabaseVersion          int64
}

func (m Meta) String() string {
	return fmt.Sprintf("{Chain Id:%s, From Height:%d, To Height:%d, Current State Snapshot File:%s, History Snapshot File:%s}",
		m.ChainID, m.HeightFrom, m.HeightTo, m.CurrentStateSnapshotFile, m.HistorySnapshotFile)
}

func (b *Service) CreateSnapshotAsync(ctx context.Context, chainID string, fromHeight int64, toHeight int64) (Meta, error) {
	return b.createSnapshot(ctx, chainID, fromHeight, toHeight, true)
}

func (b *Service) createSnapshot(ctx context.Context, chainID string, fromHeight int64, toHeight int64, async bool) (Meta, error) {
	var err error
	if len(chainID) == 0 {
		return Meta{}, fmt.Errorf("chain id is required")
	}

	b.log.Infof("creating snapshot for chain %s, from height %d, to height %d", chainID, fromHeight, toHeight)

	conn, err := pgxpool.Connect(ctx, b.connConfig.GetConnectionString())
	if err != nil {
		return Meta{}, fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close()

	dbMetaData, err := NewDatabaseMetaData(ctx, b.connConfig)
	if err != nil {
		return Meta{}, fmt.Errorf("failed to get data dump metadata: %w", err)
	}

	currentSnapshot := NewCurrentSnapshot(chainID, toHeight)
	historySnapshot := NewHistorySnapshot(chainID, fromHeight, toHeight)

	if !b.snapshotInProgress.CompareAndSwap(false, true) {
		return Meta{}, fmt.Errorf("failed to create snapshot at height %d, a snapshot is already in the process of being created",
			toHeight)
	}

	copyDataTx, err := conn.Begin(ctx)
	if err != nil {
		return Meta{}, fmt.Errorf("failed to begin copy table data transaction: %w", err)
	}

	if _, err = copyDataTx.Exec(ctx, "SET TIME ZONE 0"); err != nil {
		copyDataTx.Rollback(ctx)
		return Meta{}, fmt.Errorf("failed to set timezone to UTC:%w", err)
	}

	if _, err = copyDataTx.Exec(ctx, "SET TRANSACTION ISOLATION LEVEL SERIALIZABLE"); err != nil {
		copyDataTx.Rollback(ctx)
		return Meta{}, fmt.Errorf("failed to set transaction isolation level to serilizable:%w", err)
	}

	snapshotInProgressFile := filepath.Join(b.snapshotsPath, InProgressFileName(chainID, toHeight))
	if _, err = os.Create(snapshotInProgressFile); err != nil {
		copyDataTx.Rollback(ctx)
		return Meta{}, fmt.Errorf("failed to create write lock file:%w", err)
	}

	if async {
		go func() {
			err = b.snapshotData(ctx, copyDataTx, dbMetaData, currentSnapshot, historySnapshot, chainID, toHeight)
			if err != nil {
				b.log.Panic("failed to snapshot data", logging.Error(err))
			}
			b.snapshotInProgress.Store(false)

			if err = os.Remove(snapshotInProgressFile); err != nil {
				b.log.Errorf("failed to remove snapshot in progress file:%s", err)
			}
		}()
	} else {
		err = b.snapshotData(ctx, copyDataTx, dbMetaData, currentSnapshot, historySnapshot, chainID, toHeight)
		if err != nil {
			b.log.Panic("failed to snapshot data", logging.Error(err))
		}
		b.snapshotInProgress.Store(false)

		if err = os.Remove(snapshotInProgressFile); err != nil {
			return Meta{}, fmt.Errorf("failed to remove snapshot in progress file:%w", err)
		}
	}

	meta := Meta{
		ChainID:                  chainID,
		HeightFrom:               fromHeight,
		HeightTo:                 toHeight,
		CurrentStateSnapshotFile: filepath.Join(b.snapshotsPath, currentSnapshot.CompressedFileName()),
		HistorySnapshotFile:      filepath.Join(b.snapshotsPath, historySnapshot.CompressedFileName()),
		DatabaseVersion:          dbMetaData.DatabaseVersion,
	}

	return meta, nil
}

func (b *Service) snapshotData(ctx context.Context, copyDataTx pgx.Tx, dbMetaData DatabaseMetadata,
	currentSnapshot CurrentStateSnapshot,
	historySnapshot HistorySnapshot, chainID string, toHeight int64,
) error {
	uncompressedCurrentDataDir, err := b.createUncompressedDataDirectory(currentSnapshot.UncompressedDataDir())
	if err != nil {
		copyDataTx.Rollback(ctx)
		return fmt.Errorf("failed to create uncompressed data directory for current snapshot:%w", err)
	}
	defer func() {
		if err := os.RemoveAll(uncompressedCurrentDataDir); err != nil {
			b.log.Errorf("failed to remove uncompressed current data directory  %s :%w", uncompressedCurrentDataDir, err)
		}
	}()

	uncompressedHistoryDataDir, err := b.createUncompressedDataDirectory(historySnapshot.UncompressedDataDir())
	if err != nil {
		copyDataTx.Rollback(ctx)
		return fmt.Errorf("failed to create uncompressed data directory for history snapshot:%w", err)
	}
	defer func() {
		if err := os.RemoveAll(uncompressedHistoryDataDir); err != nil {
			b.log.Errorf("failed to remove uncompressed history data directory  %s :%w", uncompressedHistoryDataDir, err)
		}
	}()

	start := time.Now()
	b.log.Infof("copying all table data....")
	allCopySQL := append(currentSnapshot.GetCopySQL(dbMetaData, b.config.DatabaseSnapshotsPath),
		historySnapshot.GetCopySQL(dbMetaData, b.config.DatabaseSnapshotsPath)...)
	rowsCopied, err := copyTableData(ctx, copyDataTx, allCopySQL)
	if err != nil {
		rollbackErr := copyDataTx.Rollback(ctx)
		if rollbackErr != nil {
			return fmt.Errorf("failed to rollback snapshot transaction:%s, rollback caused by:%w", rollbackErr, err)
		}
		return fmt.Errorf("failed to copy table data:%w", err)
	}

	err = copyDataTx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit snapshot transaction:%w", err)
	}

	b.log.Infof("compressing current state snapshot data")
	compressedCurrentStateByteCount, err := tarAndCompressSnapshot(b.snapshotsPath, currentSnapshot, dbMetaData.DatabaseVersion)
	if err != nil {
		return fmt.Errorf("failed to compress snapshot:%w", err)
	}
	b.log.Infof("compressed current state snapshot data size: %dB", compressedCurrentStateByteCount)

	b.log.Infof("compressing history data")
	compressedHistoryByteCount, err := tarAndCompressSnapshot(b.snapshotsPath, historySnapshot, dbMetaData.DatabaseVersion)
	if err != nil {
		return fmt.Errorf("failed to compress history snapshot:%w", err)
	}
	b.log.Infof("compressed history snapshot data size: %dB", compressedHistoryByteCount)

	snapshotHash, err := GetSnapshotMd5Hash(b.log,
		filepath.Join(b.snapshotsPath, currentSnapshot.CompressedFileName()),
		filepath.Join(b.snapshotsPath, historySnapshot.CompressedFileName()))
	if err != nil {
		b.log.Errorf("failed to get md5 hash of snapshot:%w", err)
	}

	metrics.SetLastSnapshotRowcount(float64(rowsCopied))
	metrics.SetLastSnapshotCurrentStateBytes(float64(compressedCurrentStateByteCount))
	metrics.SetLastSnapshotHistoryBytes(float64(compressedHistoryByteCount))
	metrics.SetLastSnapshotSeconds(time.Since(start).Seconds())

	b.log.Info("finished creating snapshot for chain", logging.String("chain", chainID),
		logging.Int64("height", toHeight), logging.Duration("time taken", time.Since(start)),
		logging.Int64("rows copied", rowsCopied),
		logging.Int64("compressed current state data size", compressedCurrentStateByteCount),
		logging.Int64("compressed history data size", compressedHistoryByteCount),
		logging.String("md5 hash", snapshotHash))

	return nil
}

func GetHistoryMd5Hash(log *logging.Logger, snapshot Meta) (string, error) {
	snapshotHash := md5.New()

	err := hashFile(log, snapshotHash, snapshot.HistorySnapshotFile)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(snapshotHash.Sum(nil)), nil
}

func GetSnapshotMd5Hash(log *logging.Logger, currentStateSnapshotFile string, historySnapshotFile string) (string, error) {
	snapshotHash := md5.New()
	err := hashFile(log, snapshotHash, currentStateSnapshotFile)
	if err != nil {
		return "", err
	}

	err = hashFile(log, snapshotHash, historySnapshotFile)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(snapshotHash.Sum(nil)), nil
}

func hashFile(log *logging.Logger, hash hash.Hash, filepath string) error {
	filePath := filepath
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Errorf("failed to close file %s after hashing:%w", filePath, err)
		}
	}()
	_, err = io.Copy(hash, file)
	if err != nil {
		return err
	}

	return nil
}

func InProgressFileName(chainID string, height int64) string {
	return fmt.Sprintf("%s-%d.snapshotinprogress", chainID, height)
}

func (b *Service) createUncompressedDataDirectory(uncompressedDataDir string) (string, error) {
	dir := filepath.Join(b.snapshotsPath, uncompressedDataDir)

	err := os.Mkdir(dir, fs.ModePerm)
	if err != nil {
		return "", fmt.Errorf("failed to make uncompressed data directory: %s, error: %w", dir, err)
	}

	err = os.Chmod(dir, fs.ModePerm)
	if err != nil {
		return "", fmt.Errorf("failed to modify permissions of uncompressed data directory: %s, error: %w", dir, err)
	}

	return dir, nil
}

func copyTableData(ctx context.Context, tx pgx.Tx, copySQL []string) (int64, error) {
	var numRowsCopied int64
	for _, sql := range copySQL {
		tag, err := tx.Exec(ctx, sql)
		numRowsCopied += tag.RowsAffected()

		if err != nil {
			return 0, fmt.Errorf("failed to execute copy %s: %w", sql, err)
		}
	}

	return numRowsCopied, nil
}
