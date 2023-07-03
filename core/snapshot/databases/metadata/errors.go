package metadata

import "fmt"

func noMetadataForSnapshotVersion(version int64) error {
	return fmt.Errorf("no metadata found for snapshot version %d", version)
}
