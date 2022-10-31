#!/bin/bash

if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        SNAPSHOTS_COPY_TO_PATH=~/.local/state/vega/data-node/dehistory/snapshotscopyto
        SNAPSHOTS_COPY_FROM_PATH=~/.local/state/vega/data-node/dehistory/snapshotscopyfrom
elif [[ "$OSTYPE" == "darwin"* ]]; then
        SNAPSHOTS_COPY_TO_PATH="$HOME/Library/Application Support/vega/data-node/dehistory/snapshotscopyto"
        SNAPSHOTS_COPY_FROM_PATH="$HOME/Library/Application Support/vega/data-node/dehistory/snapshotscopyfrom"
else
        echo "$OSTYPE" not supported
fi


mkdir -p "$SNAPSHOTS_COPY_TO_PATH"
chmod 777 "$SNAPSHOTS_COPY_TO_PATH"

mkdir -p "$SNAPSHOTS_COPY_FROM_PATH"
chmod 777 "$SNAPSHOTS_COPY_FROM_PATH"

docker run --rm \
           -e POSTGRES_USER=vega \
           -e POSTGRES_PASSWORD=vega \
           -e POSTGRES_DB=vega \
           -p 5432:5432 \
           -v "$SNAPSHOTS_COPY_TO_PATH":"$SNAPSHOTS_COPY_TO_PATH":z \
           -v "$SNAPSHOTS_COPY_FROM_PATH":"$SNAPSHOTS_COPY_FROM_PATH":z \
           timescale/timescaledb:2.7.1-pg14
