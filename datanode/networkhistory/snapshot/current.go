package snapshot

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"code.vegaprotocol.io/vega/libs/fs"
)

type CurrentState struct {
	ChainID string
	Height  int64
}

type TableCopySql struct {
	metaData TableMetadata
	copySql  string
}

const currentStateSnapshotIdentifier = "currentstatesnapshot"

func NewCurrentSnapshot(chainID string, height int64) CurrentState {
	return CurrentState{
		ChainID: chainID,
		Height:  height,
	}
}

func (s CurrentState) UncompressedDataDir() string {
	return fmt.Sprintf("%s-%d-%s", s.ChainID, s.Height, currentStateSnapshotIdentifier)
}

func (s CurrentState) CompressedFileName() string {
	return fmt.Sprintf("%s-%d-%s.tar.gz", s.ChainID, s.Height, currentStateSnapshotIdentifier)
}

func (s CurrentState) String() string {
	return fmt.Sprintf("{Current State Snapshot Chain ID:%s Height:%d}", s.ChainID, s.Height)
}

func (s CurrentState) GetCopySQL(dbMetaData DatabaseMetadata, databaseSnapshotsPath string) []TableCopySql {
	var copySQL []TableCopySql
	for tableName, meta := range dbMetaData.TableNameToMetaData {
		if !dbMetaData.TableNameToMetaData[tableName].Hypertable {
			snapshotFile := filepath.Join(databaseSnapshotsPath, s.UncompressedDataDir(), tableName)
			tableCopySQL := fmt.Sprintf(`copy (select * from %s order by %s) to '%s'`, tableName,
				meta.SortOrder, snapshotFile)
			copySQL = append(copySQL, TableCopySql{meta, tableCopySQL})
		}
	}

	return copySQL
}

func GetCurrentStateSnapshots(dir string) (string, map[int64]CurrentState, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get files in snapshot directory:%w", err)
	}

	currentStateSnapshots := map[int64]CurrentState{}
	chainID := ""
	for _, file := range files {
		if !file.IsDir() {
			csSnapshot, err := fromCurrentStateSnapshotFileName(file.Name())
			if err != nil {
				return "", nil, fmt.Errorf("error whilst getting snapshot from filename")
			}

			if csSnapshot == nil {
				continue
			}

			if len(chainID) == 0 {
				chainID = csSnapshot.ChainID
			}

			if csSnapshot.ChainID != chainID {
				return "", nil, fmt.Errorf("current state snapshots for multiple chain ids exist in snapshots directory %s", dir)
			}

			lockFileExists, err := fs.FileExists(filepath.Join(dir,
				InProgressFileName(csSnapshot.ChainID, csSnapshot.Height)))
			if err != nil {
				return "", nil, fmt.Errorf("failed to check for lock file:%w", err)
			}

			if lockFileExists {
				continue
			}
			currentStateSnapshots[csSnapshot.Height] = *csSnapshot
		}
	}

	return chainID, currentStateSnapshots, nil
}

func fromCurrentStateSnapshotFileName(fileName string) (*CurrentState, error) {
	re, err := regexp.Compile("(.*)-(\\d+)-" + currentStateSnapshotIdentifier + ".tar.gz")
	if err != nil {
		return nil, fmt.Errorf("failed to compile reg exp:%w", err)
	}

	matches := re.FindStringSubmatch(fileName)
	if len(matches) != 3 {
		return nil, nil
	}

	height, err := strconv.ParseInt(matches[2], 10, 64)
	if err != nil {
		return nil, err
	}

	return &CurrentState{
		ChainID: matches[1],
		Height:  height,
	}, nil
}

func snapshotExists(snapshotsDir string, snapshot CurrentState) (bool, error) {
	lockFileExists, err := fs.FileExists(filepath.Join(snapshotsDir, InProgressFileName(snapshot.ChainID, snapshot.Height)))
	if err != nil {
		return false, fmt.Errorf("failed to check if lock file exists:%w", err)
	}

	if lockFileExists {
		return false, nil
	}

	files, err := os.ReadDir(snapshotsDir)
	if err != nil {
		return false, fmt.Errorf("failed to get files in snapshot directory:%w", err)
	}

	for _, file := range files {
		if !file.IsDir() {
			if file.Name() == snapshot.CompressedFileName() {
				return true, nil
			}
		}
	}

	return false, nil
}
