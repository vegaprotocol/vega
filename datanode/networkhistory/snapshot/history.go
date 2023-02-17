package snapshot

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"code.vegaprotocol.io/vega/libs/fs"
)

type History struct {
	ChainID    string
	HeightFrom int64
	HeightTo   int64
}

const historySnapshotTypeIdentifier = "historysnapshot"

func NewHistorySnapshot(chainID string, heightFrom int64, heightTo int64) History {
	return History{
		ChainID:    chainID,
		HeightFrom: heightFrom,
		HeightTo:   heightTo,
	}
}

func (h History) String() string {
	return fmt.Sprintf("{History Snapshot for Chain ID:%s Height From:%d Height To:%d}", h.ChainID, h.HeightFrom, h.HeightTo)
}

func (h History) UncompressedDataDir() string {
	return fmt.Sprintf("%s-%d-%d-%s", h.ChainID, h.HeightFrom, h.HeightTo, historySnapshotTypeIdentifier)
}

func (h History) CompressedFileName() string {
	return fmt.Sprintf("%s-%d-%d-%s.tar.gz", h.ChainID, h.HeightFrom, h.HeightTo, historySnapshotTypeIdentifier)
}

func (h History) GetCopySQL(dbMetaData DatabaseMetadata, databaseSnapshotsPath string) []string {
	var copySQL []string
	for tableName, meta := range dbMetaData.TableNameToMetaData {
		if dbMetaData.TableNameToMetaData[tableName].Hypertable {
			if len(meta.SortOrder) == 0 {
				meta.SortOrder = "vega_time, seq_num" // force sorting by time and sequence number
			}
			partitionColumn := dbMetaData.TableNameToMetaData[tableName].PartitionColumn
			snapshotFile := filepath.Join(databaseSnapshotsPath, h.UncompressedDataDir(), tableName)
			hyperTableCopySQL := fmt.Sprintf(`copy (select * from %s where %s >= (SELECT vega_time from blocks where height = %d) order by %s) to '%s'`,
				tableName,
				partitionColumn,
				h.HeightFrom,
				meta.SortOrder, snapshotFile)
			copySQL = append(copySQL, hyperTableCopySQL)
		}
	}

	return copySQL
}

func GetHistorySnapshots(snapshotsDir string) (string, []History, error) {
	files, err := os.ReadDir(snapshotsDir)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get files in snapshot directory:%w", err)
	}

	chainID := ""
	var histories []History
	for _, file := range files {
		if !file.IsDir() {
			history, err := fromHistoryFileName(file.Name())
			if err != nil {
				return "", nil, fmt.Errorf("error whilst getting history from filename")
			}

			if history == nil {
				continue
			}

			if len(chainID) == 0 {
				chainID = history.ChainID
			}

			if history.ChainID != chainID {
				return "", nil, fmt.Errorf("history snapshots for multiple chain ids exist in snapshot directory %s", snapshotsDir)
			}

			lockFileExists, err := fs.FileExists(filepath.Join(snapshotsDir, InProgressFileName(chainID, history.HeightTo)))
			if err != nil {
				return "", nil, fmt.Errorf("failed to check for lockfile:%w", err)
			}

			if lockFileExists {
				continue
			}

			histories = append(histories, *history)
		}
	}

	return chainID, histories, nil
}

func fromHistoryFileName(fileName string) (*History, error) {
	re, err := regexp.Compile("(.*)-(\\d+)-(\\d+)-" + historySnapshotTypeIdentifier + ".tar.gz")
	if err != nil {
		return nil, fmt.Errorf("failed to compile reg exp:%w", err)
	}

	matches := re.FindStringSubmatch(fileName)
	if len(matches) != 4 {
		return nil, nil
	}

	heightFrom, err := strconv.ParseInt(matches[2], 10, 64)
	if err != nil {
		return nil, err
	}

	heightTo, err := strconv.ParseInt(matches[3], 10, 64)
	if err != nil {
		return nil, err
	}

	result := NewHistorySnapshot(matches[1], heightFrom, heightTo)
	return &result, nil
}
