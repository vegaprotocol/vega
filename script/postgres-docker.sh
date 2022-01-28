#!/bin/bash
docker run --rm \
           -e POSTGRES_USER=vega \
           -e POSTGRES_PASSWORD=vega \
           -e POSTGRES_DB=vega \
           -p 5432:5432 \
           timescale/timescaledb:latest-pg14