# Changelog

## 0.44.0
*2021-10-07*

### Improvements
- [168](https://github.com/vegaprotocol/data-node/pull/168) - De-dupuplicate stake linkings
- [182](https://github.com/vegaprotocol/data-node/pull/182) - Update to latest proto, go mod tidy and set pendingStake to 0 in nodes
- [181](https://github.com/vegaprotocol/data-node/pull/181) - add gRPC endpoint for GlobalRewardPool
- [175](https://github.com/vegaprotocol/data-node/pull/175) - Add fields to validators genesis
- [169](https://github.com/vegaprotocol/data-node/pull/169) - Port code to use last version of proto (layout change)
- [163](https://github.com/vegaprotocol/data-node/pull/163) - Release/v0.43.0 

### Fixes
- [180](https://github.com/vegaprotocol/data-node/pull/180) - Update GraphQL schema (rewards)
- [170](https://github.com/vegaprotocol/data-node/pull/170) - Fix setting current epoch


## 0.43.0
*2021-09-24*

### Improvements
- [159](https://github.com/vegaprotocol/data-node/pull/159) - Remove the trading proxy to implement the TradingService
- [154](https://github.com/vegaprotocol/data-node/pull/154) - Update to the last version of the proto repository

### Fixes
- [148](https://github.com/vegaprotocol/data-node/pull/148) - Remove required party filter for TxErr events
- [147](https://github.com/vegaprotocol/data-node/pull/147) - Update the vega and proto repository dependencies to use the last version of the withdraw and deposits


## 0.42.0
*2021-09-10*

### Improvements
- [144](https://github.com/vegaprotocol/data-node/pull/144) - Release v0.42.0
- [142](https://github.com/vegaprotocol/data-node/pull/142) - point to latest proto
- [139](https://github.com/vegaprotocol/data-node/pull/139) - Check version and add new event
- [132](https://github.com/vegaprotocol/data-node/pull/132) - Add block height
- [131](https://github.com/vegaprotocol/data-node/pull/131) - Update readme
- [129](https://github.com/vegaprotocol/data-node/pull/129) - Use vega pub key
- [127](https://github.com/vegaprotocol/data-node/pull/127) - Added expiryTime to epoch queries
- [123](https://github.com/vegaprotocol/data-node/pull/123) - Add validator score
- [120](https://github.com/vegaprotocol/data-node/pull/120) - Bump protos
- [115](https://github.com/vegaprotocol/data-node/pull/115) - Add target address to ERC20 Approval withdrawal
- [113](https://github.com/vegaprotocol/data-node/pull/113) - Return proper types for Node and Party in GraphQL
- [112](https://github.com/vegaprotocol/data-node/pull/112) - Run formatter on the GraphQL schema and regen
- [100](https://github.com/vegaprotocol/data-node/pull/100) - Add a subcriber for the vega time service so the datanode can serve the blockchain time
- [99](https://github.com/vegaprotocol/data-node/pull/99) - Add checkpoints API
- [97](https://github.com/vegaprotocol/data-node/pull/97) - Add delegations to GraphQL
- [94](https://github.com/vegaprotocol/data-node/pull/94) - Implemented delegation gRPC API
- [93](https://github.com/vegaprotocol/data-node/pull/93) - Update vegaprotocol dependencies
- [92](https://github.com/vegaprotocol/data-node/pull/92) - Validator
- [91](https://github.com/vegaprotocol/data-node/pull/91) - Command line
- [90](https://github.com/vegaprotocol/data-node/pull/90) - Staking API
- [89](https://github.com/vegaprotocol/data-node/pull/89) - Add placeholder call
- [84](https://github.com/vegaprotocol/data-node/pull/84) - Remove all GraphQL Prepare and inputs
- [82](https://github.com/vegaprotocol/data-node/pull/82) - uint64 to string
- [78](https://github.com/vegaprotocol/data-node/pull/78) - Adding API support for rewards
- [71](https://github.com/vegaprotocol/data-node/pull/71) - Remove Drone
- [70](https://github.com/vegaprotocol/data-node/pull/70) - More CI testing
- [67](https://github.com/vegaprotocol/data-node/pull/67) - Better describe compilation steps
- [66](https://github.com/vegaprotocol/data-node/pull/66) - Improve and clean up the Jenkins file
- [62](https://github.com/vegaprotocol/data-node/pull/62) - Upload artefacts on release
- [59](https://github.com/vegaprotocol/data-node/pull/59) - Remove the if statement for the Jenkins file
- [58](https://github.com/vegaprotocol/data-node/pull/58) - Remove unused files
- [57](https://github.com/vegaprotocol/data-node/pull/57) - Add brackets
- [56](https://github.com/vegaprotocol/data-node/pull/56) - Remove brackets
- [54](https://github.com/vegaprotocol/data-node/pull/54) - Tidy the go packages
- [53](https://github.com/vegaprotocol/data-node/pull/53) - Change docker tag from develop to edge
- [52](https://github.com/vegaprotocol/data-node/pull/52) - Use the protos repo
- [51](https://github.com/vegaprotocol/data-node/pull/51) - Add init command
- [50](https://github.com/vegaprotocol/data-node/pull/50) - Remove unused password and update docker image
- [48](https://github.com/vegaprotocol/data-node/pull/48) - Build docker image
- [47](https://github.com/vegaprotocol/data-node/pull/47) - CI: Post messages to Slack
- [46](https://github.com/vegaprotocol/data-node/pull/46) - Add SubmitTransaction endpoint for rest and GraphQL
- [41](https://github.com/vegaprotocol/data-node/pull/41) - Add capability to receive events from a socket stream
- [40](https://github.com/vegaprotocol/data-node/pull/40) - CI: Checkout repo and compile
- [8](https://github.com/vegaprotocol/data-node/pull/8) - Merge api update
- [6](https://github.com/vegaprotocol/data-node/pull/6) - Remove core functionality
- [5](https://github.com/vegaprotocol/data-node/pull/5) - Add api tests 
- [2](https://github.com/vegaprotocol/data-node/pull/2) - Remove tendermint integration
- [1](https://github.com/vegaprotocol/data-node/pull/1) - Rename module from vega to data-node

### Fixes
- [138](https://github.com/vegaprotocol/data-node/pull/138) - Fix delegation balance to be string
- [136](https://github.com/vegaprotocol/data-node/pull/136) - Fix API tests
- [134](https://github.com/vegaprotocol/data-node/pull/134) - Fix bad reference copy of iterator
- [121](https://github.com/vegaprotocol/data-node/pull/121) - fix node ids & fix nodes storage tests
- [118](https://github.com/vegaprotocol/data-node/pull/118) - Fix data formatting
- [116](https://github.com/vegaprotocol/data-node/pull/116) - Fix staking event in convert switch
- [111](https://github.com/vegaprotocol/data-node/pull/111) - Fix ID, PubKey and Status for Node
- [110](https://github.com/vegaprotocol/data-node/pull/110) - Instantiate broker first
- [108](https://github.com/vegaprotocol/data-node/pull/108) - Add datanode component
- [106](https://github.com/vegaprotocol/data-node/pull/106) - Instantiate node service
- [81](https://github.com/vegaprotocol/data-node/pull/81) - Remove types and events
- [75](https://github.com/vegaprotocol/data-node/pull/75) - Jenkins file various improvements and fixes
- [69](https://github.com/vegaprotocol/data-node/pull/69) - Fix static check
- [61](https://github.com/vegaprotocol/data-node/pull/61) - Seperate build for docker
- [60](https://github.com/vegaprotocol/data-node/pull/60) - Fix the Jenkins file
- [55](https://github.com/vegaprotocol/data-node/pull/55) - Fix brackets
- [49](https://github.com/vegaprotocol/data-node/pull/49) - CI: Fix jenkins tag issue
- [9](https://github.com/vegaprotocol/data-node/pull/9) - Fix mock paths
- [7](https://github.com/vegaprotocol/data-node/pull/7) - Fix api tests


