#!/bin/bash
#make install
# Builds all vega related binaries in one go ######
echo -e " __      __  ______    _____             "
echo -e " \ \    / / |  ____|  / ____|     /\     "
echo -e "  \ \  / /  | |__    | |  __     /  \    "
echo -e "   \ \ \/   |  __|   | | |_ |   / /\ \   "
echo -e "    \ \     | |____  | |__| |  / ____ \  "
echo -e "     \/     |______|  \_____| /_/    \_\ "
echo -e "\n"
echo -e "Building vega"
go build -o vega "$GOPATH/src/vega/cmd/vega"
echo -e "Building vegabench"
go build -o vegabench "$GOPATH/src/vega/cmd/vegabench"
echo -e "Done."
