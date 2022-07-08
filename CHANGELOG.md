# Changelog

## Unreleased (0.53.0)

### 🚨 Breaking changes
- [](https://github.com/vegaprotocol/data-node/issues/xxx) -

### 🗑️  Deprecation
- [](https://github.com/vegaprotocol/data-node/issues/xxx) -

### 🛠  Improvements
- [572](https://github.com/vegaprotocol/data-node/issues/572) - Add cursor based pagination for votes requests
- [561](https://github.com/vegaprotocol/data-node/issues/561) - Add cursor based pagination for positions requests
- [565](https://github.com/vegaprotocol/data-node/issues/565) - Add cursor based pagination for candles data requests
- [568](https://github.com/vegaprotocol/data-node/issues/568) - Add cursor based pagination for deposits requests
- [569](https://github.com/vegaprotocol/data-node/issues/569) - Add cursor based pagination for withdrawal requests
- [723](https://github.com/vegaprotocol/data-node/issues/723) - Update contributor information
- [576](https://github.com/vegaprotocol/data-node/issues/576) - Add cursor based pagination for assets requests
- [571](https://github.com/vegaprotocol/data-node/issues/571) - Add cursor based pagination for Oracle Spec and Data requests
- [733](https://github.com/vegaprotocol/data-node/issues/733) - Store chain info in database when using `SQL`
- [748](https://github.com/vegaprotocol/data-node/issues/748) - Add REST endpoint to list `OracleData`
- [761](https://github.com/vegaprotocol/data-node/issues/761) - Delete all badger stores, `SQL` stores only from now
- [566](https://github.com/vegaprotocol/data-node/issues/566) - Liquidity provision pagination 
- [779](https://github.com/vegaprotocol/data-node/issues/779) - Ordering of paginated query results from newest to oldest
- [781](https://github.com/vegaprotocol/data-node/issues/781) - Add a summary table of current balances 

### 🐛 Fixes
- [705](https://github.com/vegaprotocol/data-node/issues/705) - Market Depth returning incorrect book state
- [730](https://github.com/vegaprotocol/data-node/issues/730) - Event bus subscriptions with party and market filter not working
- [678](https://github.com/vegaprotocol/data-node/issues/678) - Add new trading mode variant
- [776](https://github.com/vegaprotocol/data-node/issues/776) - Add support for missing proposal errors

## 0.52.0

### 🛠  Improvements
- [624](https://github.com/vegaprotocol/data-node/issues/624) - Support subscriptions in new `API`
- [666](https://github.com/vegaprotocol/data-node/issues/666) - Cache latest market data
- [564](https://github.com/vegaprotocol/data-node/issues/564) - Add cursor based pagination to market data requests
- [619](https://github.com/vegaprotocol/data-node/issues/619) - Cache markets
- [439](https://github.com/vegaprotocol/data-node/issues/439) - Add new subscription endpoint for batched market data updates
- [675](https://github.com/vegaprotocol/data-node/issues/675) - Monitoring of subscriber count
- [618](https://github.com/vegaprotocol/data-node/issues/618) - Fix positions cache
- [567](https://github.com/vegaprotocol/data-node/issues/567) - Add cursor based pagination to rewards data requests
- [708](https://github.com/vegaprotocol/data-node/issues/708) - Fix problem where querying `nodeData` would fail if there are no delegations

### 🐛 Fixes
- [657](https://github.com/vegaprotocol/data-node/issues/657) - Add missing creation field in `ERC20` withdrawal bundle
- [668](https://github.com/vegaprotocol/data-node/issues/668) - Ensure entity wrappers always hold timestamps to microsecond resolution
- [662](https://github.com/vegaprotocol/data-node/issues/662) - Fix auction trigger enum lookup
- [682](https://github.com/vegaprotocol/data-node/issues/682) - Allow multiple checkpoints per block
- [690](https://github.com/vegaprotocol/data-node/issues/690) - Fix deadlock in market data subscription, close subscriptions when data can't be written rather than silently dropping events.
- [698](https://github.com/vegaprotocol/data-node/issues/698) - Fix bug that was preventing correct translation of reward type in `GraphQL`
- [697](https://github.com/vegaprotocol/data-node/issues/697) - Fix bug that was causing misreporting of delegations in node queries
- [697](https://github.com/vegaprotocol/data-node/issues/697) - Actually fix bug that was causing misreporting of delegations in node queries

## 0.51.1

### 🛠  Improvements
- [615](https://github.com/vegaprotocol/data-node/issues/615) - Ensure pool size less than maximum number of `postgres` connections
- [632](https://github.com/vegaprotocol/data-node/issues/632) - Rename method for listing asset bundle
- [609](https://github.com/vegaprotocol/data-node/issues/609) - Add fields related to network limit for ERC20 asset
- [609](https://github.com/vegaprotocol/data-node/issues/384) - Migrate node data to V2
- [613](https://github.com/vegaprotocol/data-node/issues/613) - Option to regularly dump `pprof` data
- [635](https://github.com/vegaprotocol/data-node/issues/635) - V2 code path returns only nodes that exist in a particular epoch
- [590](https://github.com/vegaprotocol/data-node/pull/590) - Implement pagination for `Data-Node V2 APIs` for Trades, Parties and Markets
- [560](https://github.com/vegaprotocol/data-node/issues/560) - Implement pagination for `Data-Node V2 APIs` for Orders
- [562](https://github.com/vegaprotocol/data-node/issues/560) - Implement pagination for `Data-Node V2 APIs` for Margin Levels
- [621](https://github.com/vegaprotocol/data-node/issues/621) - Reserve database connection for data ingestion
- [625](https://github.com/vegaprotocol/data-node/issues/625) - Make socket server buffer size configurable
- [630](https://github.com/vegaprotocol/data-node/issues/630) - Data retention across all historical data tables
- [645](https://github.com/vegaprotocol/data-node/issues/645) - Make `rewardType` an `enum` in `GraphQL API`
- [427](https://github.com/vegaprotocol/data-node/issues/427) - Handle recovery from snapshots

### 🐛 Fixes
- [616](https://github.com/vegaprotocol/data-node/pull/616) - Don't return multiple delegations per epoch/party/node
- [627](https://github.com/vegaprotocol/data-node/issues/627) - User `from_epoch` in update event to determine if node exists
- [651](https://github.com/vegaprotocol/data-node/issues/651) - Hide the `V2 grpc API` if V2 not enabled

## 0.51.0

### 🚨 Breaking changes
- [518](https://github.com/vegaprotocol/data-node/issues/518) - Free-form properties are moved to rationale.


### 🛠  Improvements
- [491](https://github.com/vegaprotocol/data-node/issues/491) - Expose bundle for asset
- [414](https://github.com/vegaprotocol/data-node/issues/414) - Migrate market depth to retrieve data from `Postgres`
- [495](https://github.com/vegaprotocol/data-node/issues/495) - Remove deprecated `PositionState` event handling, general fixes to `SettlePosition` event handling
- [498](https://github.com/vegaprotocol/data-node/issues/498) - Transaction event broker
- [521](https://github.com/vegaprotocol/data-node/issues/521) - Refactor margin levels to use account id
- [518](https://github.com/vegaprotocol/data-node/issues/518) - Add rationale to proposals
- [526](https://github.com/vegaprotocol/data-node/issues/526) - Add market id and reward type to reward and market to transfer
- [540](https://github.com/vegaprotocol/data-node/issues/540) - CI: trigger Devnet deployment on merges to develop branch
- [546](https://github.com/vegaprotocol/data-node/issues/546) - Data retention for margin levels
- [553](https://github.com/vegaprotocol/data-node/issues/553) - Update transfers API to expose dispatch strategy
- [578](https://github.com/vegaprotocol/data-node/issues/578) - Add metrics for `SQL` queries
- [582](https://github.com/vegaprotocol/data-node/issues/582) - Add a cache for assets
- [548](https://github.com/vegaprotocol/data-node/issues/548) - Remove foreign key constraints on hyper tables
- [596](https://github.com/vegaprotocol/data-node/issues/596) - Speed up querying of orders
- [591](https://github.com/vegaprotocol/data-node/issues/591) - Optimise liquidity provision and margin levels data retention and storage
- [588](https://github.com/vegaprotocol/data-node/issues/588) - Return correct error code when proposal not found
- [556](https://github.com/vegaprotocol/data-node/issues/556) - Expose an endpoint to list oracle data

### 🐛 Fixes
- [524](https://github.com/vegaprotocol/data-node/issues/524) - Fix for incorrect balances
- [600](https://github.com/vegaprotocol/data-node/issues/600) - Node lists filter based on whether a node exists for the given epoch
- [520](https://github.com/vegaprotocol/data-node/issues/520) - Fix event race where a ranking event can come in before the new node event
- [519](https://github.com/vegaprotocol/data-node/issues/519) - Fix market depth update subscriptions streaming events for all markets.
- [551](https://github.com/vegaprotocol/data-node/issues/551) - Shut down cleanly on `SIGINT` or `SIGTERM`
- [585](https://github.com/vegaprotocol/data-node/issues/585) - Fix issue which was stopping asset cache from working properly


## 0.50.0

### 🛠  Improvements
- [386](https://github.com/vegaprotocol/data-node/pull/386) - Migrate withdrawal API to retrieve data from `Postgres`
- [378](https://github.com/vegaprotocol/data-node/issues/378) - Migrate existing Oracles API to new `Postgres` database.
- [461](https://github.com/vegaprotocol/data-node/pull/461) - Migrate market data time series to consistent format
- [375](https://github.com/vegaprotocol/data-node/issues/375) - Migrate existing Liquidity Provisions API to new `Postgres` database.
- [381](https://github.com/vegaprotocol/data-node/issues/381) - Migrate existing Positions API to new `Postgres` database.
- [467](https://github.com/vegaprotocol/data-node/pull/467) - Migrate transfers API to retrieve data from `Postgres`
- [469](https://github.com/vegaprotocol/data-node/issues/469) - Migrate existing stake linking API to new `Postgres` database.
- [496](https://github.com/vegaprotocol/data-node/issues/496) - Migrate `ERC20WithdrawlApproval` and `NodeSignaturesAggregate` API to new `Postgres` database.
- [496](https://github.com/vegaprotocol/data-node/issues/496) - Add API to get `multisig` signer bundles.
- [474](https://github.com/vegaprotocol/data-node/pull/474) - Clean up error handling in subscribers and make action on error configurable
- [487](https://github.com/vegaprotocol/data-node/pull/487) - Trade data retention
- [495](https://github.com/vegaprotocol/data-node/pull/495) - Account for `SettlePosition` events reaching the positions plug-in before the `PositionState` event.
- [495](https://github.com/vegaprotocol/data-node/pull/495) - Make sure `SettlePosition` does not result in a division by zero panic.
- [495](https://github.com/vegaprotocol/data-node/pull/495) - Fix panic caused by incorrect/missing initialisation of `AverageEntryPrice` field.
- [](https://github.com/vegaprotocol/data-node/pull/xxx) -

### 🐛 Fixes
- [451](https://github.com/vegaprotocol/data-node/issues/451) - Correct conversion of pending validator status
- [391](https://github.com/vegaprotocol/data-node/issues/391) - Fix `OracleSpecs GraphQL` query returns error and null when there is no data.
- [477](https://github.com/vegaprotocol/data-node/issues/477) - Fix position open volume calculation.
- [281](https://github.com/vegaprotocol/data-node/issues/281) - Fix Estimate Margin calculates incorrectly for Limit Orders
- [482](https://github.com/vegaprotocol/data-node/issues/482) - Fan out event broker should only call listen once on source broker

## 0.49.3

### 🛠  Improvements
- [426](https://github.com/vegaprotocol/data-node/pull/426) - Add bindings for party less liquidity provision requests
- [430](https://github.com/vegaprotocol/data-node/pull/430) - Migrate deposit API to retrieve data from `Postgres`
- [435](https://github.com/vegaprotocol/data-node/pull/435) - Migrate governance API to retrieve data from `Postgres`
- [438](https://github.com/vegaprotocol/data-node/pull/438) - Migrate candles to retrieve data from `Postgres`
- [447](https://github.com/vegaprotocol/data-node/issue/447) - Use `PositionState` event to update position
- [442](https://github.com/vegaprotocol/data-node/pull/442) - Migrate estimator API to retrieve data from `Postgres`
- [449](https://github.com/vegaprotocol/data-node/issue/449) - Refactor identifiers to use `ID` types instead of `[]byte`
- [373](https://github.com/vegaprotocol/data-node/issue/373) - Migrate `GetVegaTime`, `Checkpoints` and `NetworkParameters` to `Postgres`

### 🐛 Fixes
- [256](https://github.com/vegaprotocol/data-node/pull/256) - Market Risk Factors missing from Market `GraphQL` API

## 0.49.2

### 🛠  Improvements
- [404](https://github.com/vegaprotocol/data-node/pull/404) - Migrate market data API to retrieve data from `Postgres`
- [406](https://github.com/vegaprotocol/data-node/pull/406) - Add a basic integration test
- [412](https://github.com/vegaprotocol/data-node/pull/412) - Migrate markets API to retrieve data from `Postgres`
- [407](https://github.com/vegaprotocol/data-node/pull/407) - Add `positionDecimalPlaces` to market `graphQL`
- [429](https://github.com/vegaprotocol/data-node/issues/429) - Add environment variable to getting started document
- [420](https://github.com/vegaprotocol/data-node/pull/420) - Migrate rewards, delegations and epochs to `Postgres`
- [380](https://github.com/vegaprotocol/data-node/pull/420) - Migrate party API to `postgres`

### 🐛 Fixes
- [411](https://github.com/vegaprotocol/data-node/pull/411) - Fix a couple of incompatibilities in `data-node v2`
- [417](https://github.com/vegaprotocol/data-node/pull/411) - Report correct total tokens for a vote in `graphql`

## 0.49.1

### 🚨 Breaking changes
- [333](https://github.com/vegaprotocol/data-node/issues/333) - extend node model with additional information about reward scores and ranking scores + validator statuses

### 🛠  Improvements
- [362](https://github.com/vegaprotocol/data-node/pull/362) - Added support using TLS for `GraphQL` connections
- [393](https://github.com/vegaprotocol/data-node/pull/393) - Data store migration
- [395](https://github.com/vegaprotocol/data-node/pull/395) - Migrate Asset API to retrieve data from `Postgres`
- [399](https://github.com/vegaprotocol/data-node/pull/399) - Migrate Accounts API to retrieve data from `Postgres`

### 🐛 Fixes
- [387](https://github.com/vegaprotocol/data-node/pull/387) - Fixes incorrect data types in the `MarketData` proto message
- [390](https://github.com/vegaprotocol/data-node/pull/390) - Cache `ChainInfo` data

## 0.49.0

### 🛠  Improvements
- [322](https://github.com/vegaprotocol/data-node/pull/322) - Update the definition of done and issue templates
- [351](https://github.com/vegaprotocol/data-node/pull/351) - Update to latest Vega and downgrade to Tendermint `v.34.15`
- [352](https://github.com/vegaprotocol/data-node/pull/352) - Update to latest Vega
- [356](https://github.com/vegaprotocol/data-node/pull/356) - Added support for fractional positions
- [251](https://github.com/vegaprotocol/data-node/pull/251) - Updated proto and core and added support for the new events (state var and network limits)
- [285](https://github.com/vegaprotocol/data-node/pull/285) - Update changelog for `47.1`
- [244](https://github.com/vegaprotocol/data-node/pull/244) - Constrain the number of epochs for which we keep delegations in memory
- [250](https://github.com/vegaprotocol/data-node/pull/250) - Update go requirement to 1.17
- [251](https://github.com/vegaprotocol/data-node/pull/251) - Updated proto and core and added support for the new events (state var and network limits)
- [289](https://github.com/vegaprotocol/data-node/pull/289) - Add support for pagination of delegations
- [254](https://github.com/vegaprotocol/data-node/pull/254) - Move to `ghcr.io` container registry
- [290](https://github.com/vegaprotocol/data-node/pull/290) - Update pegged orders offset
- [296](https://github.com/vegaprotocol/data-node/pull/296) - Expose validator performance score attributes on Node object
- [298](https://github.com/vegaprotocol/data-node/pull/298) - Remove creation of vendor directory
- [304](https://github.com/vegaprotocol/data-node/pull/304) - Added endpoint to support multiple versions of transaction request
- [316](https://github.com/vegaprotocol/data-node/pull/316) - Add basic framework for connecting to `postgres` database
- [323](https://github.com/vegaprotocol/data-node/pull/323) - Add initial `sql` storage package
- [324](https://github.com/vegaprotocol/data-node/pull/324) - Embed the facility to run a file based event store into the datanode
- [326](https://github.com/vegaprotocol/data-node/pull/326) - Add `BlockNr()` methods to implementers of event interface
- [331](https://github.com/vegaprotocol/data-node/pull/331) - Add support for running an embedded version of `Postgresql`
- [336](https://github.com/vegaprotocol/data-node/pull/336) - Remove trading mode and future maturity
- [338](https://github.com/vegaprotocol/data-node/pull/336) - Add `grpcui` web user interface
- [340](https://github.com/vegaprotocol/data-node/pull/340) - Add brokers for the new data stores to support sequential and concurrent event processing
- [327](https://github.com/vegaprotocol/data-node/pull/327) - Add balances `sql` store and upgrade `gqlgen`
- [329](https://github.com/vegaprotocol/data-node/pull/327) - Add orders `sql` store
- [354](https://github.com/vegaprotocol/data-node/pull/354) - Add network limits store and API
- [338](https://github.com/vegaprotocol/data-node/pull/338) - Fix compatibility with new `protoc-gen-xxx` tools used in `protos` repository
- [330](https://github.com/vegaprotocol/data-node/pull/330) - Add support for storing market data events in the SQL store
- [355](https://github.com/vegaprotocol/data-node/pull/355) - Persist trade data to SQL store

### 🐛 Fixes
- [277](https://github.com/vegaprotocol/data-node/pull/277) - Now returns not-found error instead of internal error when proposal not found
- [274](https://github.com/vegaprotocol/data-node/issues/274) - Bug fix for proposal NO vote showing incorrect weight and tokens
- [288](https://github.com/vegaprotocol/data-node/pull/288) - Add back `assetId` GraphQL resolver for `RewardPerAssetDetail`, change `RiskFactor` fields to strings.
- [317](https://github.com/vegaprotocol/data-node/pull/317) - Fix `graphql` support for free-form governance proposals
- [345](https://github.com/vegaprotocol/data-node/issues/345) - Add the missing events conversion to data node
- [360](https://github.com/vegaprotocol/data-node/pull/360) - Market data record should be using the sequence number from the event

## 0.47.1
*`2021-12-20`*

### 🐛 Fixes
- [244](https://github.com/vegaprotocol/data-node/pull/244) - Constrain the number of epochs for which we keep delegations in memory


## 0.47.0
*`2021-12-10`*

### 🛠 Improvements
- [232](https://github.com/vegaprotocol/data-node/pull/232) - Tidy up repo to align with team processes and workflows
- [235](https://github.com/vegaprotocol/data-node/pull/235) - Add key rotation support
- [246](https://github.com/vegaprotocol/data-node/pull/246) - Add statistics to GraphQL API

### 🐛 Fixes
- [233](https://github.com/vegaprotocol/data-node/pull/233) - Don't return API error when no rewards for party
- [240](https://github.com/vegaprotocol/data-node/pull/240) - Allow risk factor events to be streamed via GraphQL subscription



## 0.46.0
*`2021-11-22`*

### 🛠 Improvements
- [238](https://github.com/vegaprotocol/data-node/pull/230) - Add filtering/pagination GraphQL schema for rewards
- [230](https://github.com/vegaprotocol/data-node/pull/230) - Release Version `0.46.0`
- [229](https://github.com/vegaprotocol/data-node/pull/229) - Add handling for checking/storing Chain ID
- [226](https://github.com/vegaprotocol/data-node/pull/226) - Added subscriptions for delegations & rewards
- [228](https://github.com/vegaprotocol/data-node/pull/228) - Add changelog and project board Github actions and update linked PR action version
- [208](https://github.com/vegaprotocol/data-node/pull/208) - Turn off `api_tests` when run on the CI
- [197](https://github.com/vegaprotocol/data-node/pull/197) - Set time limit for system-tests, and also do not ignore failures for pull requests
- [162](https://github.com/vegaprotocol/data-node/pull/162) - Move to XDG file structure
- [212](https://github.com/vegaprotocol/data-node/pull/212) - Stabilise api tests
- [221](https://github.com/vegaprotocol/data-node/pull/221) - Populate target address for `erc20WithdrawalApprovals`
- [225](https://github.com/vegaprotocol/data-node/pull/225) - Remove SubmitTransaction GraphQL endpoint

### 🐛 Fixes
- [207](https://github.com/vegaprotocol/data-node/pull/207) - Fix rewards schema and update vega dependencies to have reward event fixes
- [239](https://github.com/vegaprotocol/data-node/pull/238) - Update GraphQL schema to not require every asset has a global reward account.

## 0.45.1
*2021-10-23*

### 🛠 Improvements
- [202](https://github.com/vegaprotocol/data-node/pull/202) - Updates after vegawallet name change
- [203](https://github.com/vegaprotocol/data-node/pull/203) - Release version `v0.45.1`
- [205](https://github.com/vegaprotocol/data-node/pull/205) - Release version `v0.45.1`

### 🐛 Fixes
- [199](https://github.com/vegaprotocol/data-node/pull/199) - Add timestamp to reward payload


## 0.45.0
*2021-10-18*

### 🛠 Improvements
- [190](https://github.com/vegaprotocol/data-node/pull/190) - Run golangci-lint as part of CI
- [186](https://github.com/vegaprotocol/data-node/pull/186) - Add system-tests

## 0.44.0
*2021-10-07*

### 🛠 Improvements
- [168](https://github.com/vegaprotocol/data-node/pull/168) - De-duplicate stake linkings
- [182](https://github.com/vegaprotocol/data-node/pull/182) - Update to latest proto, go mod tidy and set pendingStake to 0 in nodes
- [181](https://github.com/vegaprotocol/data-node/pull/181) - add gRPC endpoint for GlobalRewardPool
- [175](https://github.com/vegaprotocol/data-node/pull/175) - Add fields to validators genesis
- [169](https://github.com/vegaprotocol/data-node/pull/169) - Port code to use last version of proto (layout change)
- [163](https://github.com/vegaprotocol/data-node/pull/163) - Release 0.43.0

### 🐛 Fixes
- [180](https://github.com/vegaprotocol/data-node/pull/180) - Update GraphQL schema (rewards)
- [170](https://github.com/vegaprotocol/data-node/pull/170) - Fix setting current epoch


## 0.43.0
*2021-09-24*

### 🛠 Improvements
- [159](https://github.com/vegaprotocol/data-node/pull/159) - Remove the trading proxy to implement the TradingService
- [154](https://github.com/vegaprotocol/data-node/pull/154) - Update to the last version of the proto repository

### 🐛 Fixes
- [148](https://github.com/vegaprotocol/data-node/pull/148) - Remove required party filter for TxErr events
- [147](https://github.com/vegaprotocol/data-node/pull/147) - Update the vega and proto repository dependencies to use the last version of the withdraw and deposits


## 0.42.0
*2021-09-10*

### 🛠 Improvements
- [144](https://github.com/vegaprotocol/data-node/pull/144) - Release 0.42.0
- [142](https://github.com/vegaprotocol/data-node/pull/142) - point to latest proto
- [139](https://github.com/vegaprotocol/data-node/pull/139) - Check version and add new event
- [132](https://github.com/vegaprotocol/data-node/pull/132) - Add block height
- [131](https://github.com/vegaprotocol/data-node/pull/131) - Update readme
- [129](https://github.com/vegaprotocol/data-node/pull/129) - Use vega pub key
- [127](https://github.com/vegaprotocol/data-node/pull/127) - Added expiryTime to epoch queries
- [123](https://github.com/vegaprotocol/data-node/pull/123) - Add validator score
- [120](https://github.com/vegaprotocol/data-node/pull/120) - Update proto version
- [115](https://github.com/vegaprotocol/data-node/pull/115) - Add target address to ERC20 Approval withdrawal
- [113](https://github.com/vegaprotocol/data-node/pull/113) - Return proper types for Node and Party in GraphQL
- [112](https://github.com/vegaprotocol/data-node/pull/112) - Run formatter on the GraphQL schema and regenerate
- [100](https://github.com/vegaprotocol/data-node/pull/100) - Add a subscriber for the vega time service so the datanode can serve the blockchain time
- [99](https://github.com/vegaprotocol/data-node/pull/99) - Add checkpoints API
- [97](https://github.com/vegaprotocol/data-node/pull/97) - Add delegations to GraphQL
- [94](https://github.com/vegaprotocol/data-node/pull/94) - Implemented delegation gRPC API
- [93](https://github.com/vegaprotocol/data-node/pull/93) - Update vega dependencies
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
- [52](https://github.com/vegaprotocol/data-node/pull/52) - Use the proto repo
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

### 🐛 Fixes
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
- [61](https://github.com/vegaprotocol/data-node/pull/61) - Separate build for Docker
- [60](https://github.com/vegaprotocol/data-node/pull/60) - Fix the Jenkins file
- [55](https://github.com/vegaprotocol/data-node/pull/55) - Fix brackets
- [49](https://github.com/vegaprotocol/data-node/pull/49) - Fix Jenkins tag issue
- [9](https://github.com/vegaprotocol/data-node/pull/9) - Fix mock paths
- [7](https://github.com/vegaprotocol/data-node/pull/7) - Fix api tests
