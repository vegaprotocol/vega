#!/bin/bash

if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        SNAPSHOTS_COPY_TO_PATH=~/.local/state/vega/data-node/dehistory/snapshotsCopyTo
        SNAPSHOTS_COPY_FROM_PATH=~/.local/state/vega/data-node/dehistory/snapshotsCopyFrom
elif [[ "$OSTYPE" == "darwin"* ]]; then
        SNAPSHOTS_COPY_TO_PATH="$HOME/Library/Application Support/vega/data-node/dehistory/snapshotsCopyTo"
        SNAPSHOTS_COPY_FROM_PATH="$HOME/Library/Application Support/vega/data-node/dehistory/snapshotsCopyFrom"
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
           -v "$SNAPSHOTS_COPY_TO_PATH":/snapshotsCopyTo:z \
           -v "$SNAPSHOTS_COPY_FROM_PATH":/snapshotsCopyFrom:z \
           timescale/timescaledb:2.7.1-pg14
