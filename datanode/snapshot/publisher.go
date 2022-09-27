package snapshot

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"code.vegaprotocol.io/vega/logging"
)

type Publisher struct {
	log           *logging.Logger
	snapshotsPath string
}

func NewPublisher(ctx context.Context, log *logging.Logger, config Config, snapshotsPath string) (*Publisher, error) {
	p := &Publisher{
		log:           log.Named("snapshot-publisher"),
		snapshotsPath: snapshotsPath,
	}

	if config.Publish {
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					publishAndRemoveSnapshots(snapshotsPath, p)
				}
			}
		}()
	}

	return p, nil
}

func publishAndRemoveSnapshots(snapshotsPath string, p *Publisher) {
	_, histories, err := GetHistorySnapshots(snapshotsPath)
	if err != nil {
		p.log.Errorf("failed to get history snapshots:%w", err)
	}

	for _, history := range histories {
		err = p.removeSnapshots(history)
		if err != nil {
			p.log.Errorf("failed to publish and remove history snapshot:%w", err)
		}
	}

	_, snapshots, err := GetCurrentStateSnapshots(snapshotsPath)
	if err != nil {
		p.log.Errorf("failed to get current state snapshots:%w", err)
	}

	for _, snapshot := range snapshots {
		err = p.removeSnapshots(snapshot)
		if err != nil {
			p.log.Errorf("failed to publish and remove history snapshot:%w", err)
		}
	}
}

func (p *Publisher) removeSnapshots(sn snapshot) error {
	err := os.RemoveAll(filepath.Join(p.snapshotsPath, sn.CompressedFileName()))
	if err != nil {
		return fmt.Errorf("failed to remove snapshot:%w", err)
	}
	return nil
}
