#!/bin/bash
# We need to have access to the vegacapsule repo locally for this to run
VEGACAPSULEPATH="../../vegacapsule"

# This script much be run from the visor folder

# Build visor-config
go run $VEGACAPSULEPATH/cmd/docs/main.go -type-names config.VisorConfigFile -description-path ./config/description.md -dir-path .. -tag-name toml > visor-config.md

# Build run-config
go run $VEGACAPSULEPATH/cmd/docs/main.go -type-names config.RunConfig -description-path ./config/description.md -dir-path .. -tag-name toml > run-config.md
