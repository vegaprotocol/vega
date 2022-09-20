#!/bin/bash

if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        SNAPSHOTS_PATH=~/.local/state/vega/data-node/snapshots
elif [[ "$OSTYPE" == "darwin"* ]]; then
        SNAPSHOTS_PATH="$HOME/Library/Application Support/vega/data-node/snapshots"
else
        echo "$OSTYPE" not supported
fi


mkdir -p "$SNAPSHOTS_PATH"
chmod 777 "$SNAPSHOTS_PATH"
docker run --rm \
           -e POSTGRES_USER=vega \
           -e POSTGRES_PASSWORD=vega \
           -e POSTGRES_DB=vega \
           -p 5432:5432 \
           -v "$SNAPSHOTS_PATH":/snapshots:z \
           timescale/timescaledb:2.7.1-pg14
