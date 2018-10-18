#!/bin/bash
PATH=/usr/local/bin:/usr/bin:/bin
mkdir -p ./log/tendermint
chown -R vega:vega ./log/tendermint
exec ./tendermint node 2>&1 | multilog t s10485760 n200 ./log/tendermint &
