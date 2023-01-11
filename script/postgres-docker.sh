#!/bin/bash

# It is important that snapshotscopy{to|from} path is accessible at the same location in
# the container and outside of it. If you're using a custom vega home, you must call this script
# with VEGA_HOME set to your custom vega home when starting the database.

if [ -n "$VEGA_HOME" ]; then
        VEGA_STATE=${VEGA_HOME}/state
else
        if [[ "$OSTYPE" == "linux-gnu"* ]]; then
                VEGA_STATE=~/.local/state/vega
        elif [[ "$OSTYPE" == "darwin"* ]]; then
                VEGA_STATE="${HOME}/Library/Application Support/vega"
        else
                 echo "$OSTYPE" not supported
        fi
fi

SNAPSHOTS_COPY_TO_PATH=${VEGA_STATE}/data-node/networkhistory/snapshotscopyto
SNAPSHOTS_COPY_FROM_PATH=${VEGA_STATE}/data-node/networkhistory/snapshotscopyfrom

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
           timescale/timescaledb:2.8.0-pg14
