# Changelog

## Unreleased 0.75.0

### 🚨 Breaking changes

- [](https://github.com/vegaprotocol/vega/issues/xxx) -

### 🗑️ Deprecation

- [](https://github.com/vegaprotocol/vega/issues/xxx) -

### 🛠 Improvements

- [](https://github.com/vegaprotocol/vega/issues/xxx) -
- [971](https://github.com/vegaprotocol/core-test-coverage/issues/971) - Add `AMM` support to the integration test framework.

### 🐛 Fixes

- [10631](https://github.com/vegaprotocol/vega/issues/10631) - Fix snapshot for `ethCallEvents`

## 0.74.1

### 🚨 Breaking changes

- [](https://github.com/vegaprotocol/vega/issues/xxx) -

### 🗑️ Deprecation

- [](https://github.com/vegaprotocol/vega/issues/xxx) -

### 🛠 Improvements

- [10647](https://github.com/vegaprotocol/vega/issues/10647) - Add filter by game ID to transfers API.

### 🐛 Fixes

- [10611](https://github.com/vegaprotocol/vega/issues/10611) - Added internal config price to update `perps`.
- [10615](https://github.com/vegaprotocol/vega/issues/10615) - Fix oracle scaling function in internal composite price.
- [10621](https://github.com/vegaprotocol/vega/issues/10621) - Fix market activity tracker storing incorrect data for previous `epochMakerFeesPaid`.
- [10643](https://github.com/vegaprotocol/vega/issues/10643) - Games `API` not showing quantum values and added filter for team and party.

## 0.74.0

### 🚨 Breaking changes

- [9945](https://github.com/vegaprotocol/vega/issues/9945) - Add liquidation strategy.
- [10215](https://github.com/vegaprotocol/vega/issues/10215) - Listing transactions on block explorer does not support the field `limit` any more.
- [8056](https://github.com/vegaprotocol/vega/issues/8056) - Getting a transfer by ID now returns a `TransferNode`.
- [10597](https://github.com/vegaprotocol/vega/pull/10597) - Migrate the `IPFS` store for the network history to version 15.

### 🗑️ Deprecation

- [9881](https://github.com/vegaprotocol/vega/issues/9881) - Liquidity monitoring auctions to be removed.
- [10000](https://github.com/vegaprotocol/vega/issues/9881) - Commands `tm` and `tendermint` are deprecated in favour of `cometbft`.

### 🛠 Improvements

- [9930](https://github.com/vegaprotocol/vega/issues/9930) - `LiquidityFeeSettings` can now be used in market proposals to choose how liquidity fees are calculated.
- [9985](https://github.com/vegaprotocol/vega/issues/9985) - Add update margin mode transaction.
- [9936](https://github.com/vegaprotocol/vega/issues/9936) - Time spent in auction no longer contributes to a perpetual market's funding payment.
- [9982](https://github.com/vegaprotocol/vega/issues/9982) - Remove fees and minimal transfer amount from vested account
- [9955](https://github.com/vegaprotocol/vega/issues/9955) - Add data node subscription for transaction results.
- [10004](https://github.com/vegaprotocol/vega/issues/10004) Track average entry price in position engine
- [9825](https://github.com/vegaprotocol/vega/issues/9825) - Remove quadratic slippage.
- [9516](https://github.com/vegaprotocol/vega/issues/9516) - Add filter by transfer ID for ledger entries API.
- [9943](https://github.com/vegaprotocol/vega/issues/9943) - Support amending the order size by defining the target size.
- [9231](https://github.com/vegaprotocol/vega/issues/9231) - Add a `JoinTeam API`
- [10095](https://github.com/vegaprotocol/vega/issues/10095) - Add isolated margin support.
- [10222](https://github.com/vegaprotocol/vega/issues/10222) - Supply bootstrap peers after starting the `IPFS` node to increase reliability.
- [10097](https://github.com/vegaprotocol/vega/issues/10097) - Add funding rate modifiers to perpetual product definition.
- [9981](https://github.com/vegaprotocol/vega/issues/9981) - Support filtering on epoch range on transfers.
- [9981](https://github.com/vegaprotocol/vega/issues/9981) - Support filtering on status on transfers.
- [10104](https://github.com/vegaprotocol/vega/issues/10104) - Add network position tracking.
- [9981](https://github.com/vegaprotocol/vega/issues/9981) - Support filtering on scope on transfers.
- [9983](https://github.com/vegaprotocol/vega/issues/9983) - Implement cap and discount for transfer fees.
- [9980](https://github.com/vegaprotocol/vega/issues/9980) - Add teams statistics API.
- [9257](https://github.com/vegaprotocol/vega/issues/9257) - Add games details API
- [9992](https://github.com/vegaprotocol/vega/issues/9992) - Add configuration to control the number of blocks worth of Ethereum events to read.
- [9260](https://github.com/vegaprotocol/vega/issues/9260) - Enhance rewards API for competitions
- [10180](https://github.com/vegaprotocol/vega/issues/10180) - Additional candle intervals
- [10218](https://github.com/vegaprotocol/vega/issues/10218) - Volume discount stats shows volumes even if party doesn't qualify for a discount tier.
- [9880](https://github.com/vegaprotocol/vega/issues/9880) - Add support for batch proposals.
- [10159](https://github.com/vegaprotocol/vega/issues/10159) - Add additional funding period data to market data API to allow streaming funding period data.
- [10143](https://github.com/vegaprotocol/vega/issues/10143) - Allow changing the market name in update market governance proposal.
- [10154](https://github.com/vegaprotocol/vega/issues/10154) - Move remaining insurance pool balance into the network treasury rather than splitting between other markets and global insurance.
- [10154](https://github.com/vegaprotocol/vega/issues/10154) - Move remaining insurance pool balance into the network treasury rather than splitting between other markets and global insurance.
- [10154](https://github.com/vegaprotocol/vega/issues/10154) - Move remaining insurance pool balance into the network treasury rather than splitting between other markets and global insurance.
- [10154](https://github.com/vegaprotocol/vega/issues/10154) - Move remaining insurance pool balance into the network treasury rather than splitting between other markets and global insurance.
- [10266](https://github.com/vegaprotocol/vega/issues/10266) - Deprecated `marketID` and populate `gameID` in reward API
- [10154](https://github.com/vegaprotocol/vega/issues/10154) - Move remaining insurance pool balance into the network treasury rather than splitting between other markets and global insurance.
- [10155](https://github.com/vegaprotocol/vega/issues/10155) - Add next network close-out timestamp to market data.
- [10000](https://github.com/vegaprotocol/vega/issues/10000) - Introduce `cometbtf` command to replace `tendermint`.
- [10294](https://github.com/vegaprotocol/vega/issues/10294) - New mark price methodology
- [9948](https://github.com/vegaprotocol/vega/issues/9948) - Add support for linked stop orders.
- [9849](https://github.com/vegaprotocol/vega/issues/9849) - Add database support for `num.Uint`.
- [10275](https://github.com/vegaprotocol/vega/issues/10275) - Add API to list party's margin mode.
- [459](https://github.com/vegaprotocol/core-test-coverage/issues/459) - Add liquidation test coverage with market updates.
- [462](https://github.com/vegaprotocol/core-test-coverage/issues/462) - Cover `0012-POSR-011` explicitly
- [595](https://github.com/vegaprotocol/core-test-coverage/issues/595) - Ensure the full size of iceberg orders is considered when creating a network order.
- [10308](https://github.com/vegaprotocol/vega/issues/10308) - Support joining to closed teams based on an allow list.
- [10349](https://github.com/vegaprotocol/vega/issues/10349) - Add oracle support to mark price configuration.
- [10350](https://github.com/vegaprotocol/vega/issues/10350) - Set mark price to uncrossing price if at the end of opening auction no price was yielded by the mark price methodology.
- [521](https://github.com/vegaprotocol/core-test-coverage/issues/521) - Add tests for allow list functionality when joining teams.
- [10375](https://github.com/vegaprotocol/vega/issues/10375) - Expose party's profile in parties API.
- [10340](https://github.com/vegaprotocol/core-test-coverage/issues/10340) - Support updating profile for a party.
- [551](https://github.com/vegaprotocol/core-test-coverage/issues/551) - Cover `0053-PERP-033` explicitly.
- [646](https://github.com/vegaprotocol/core-test-coverage/issues/646) - Add integration test for team rewards `0009-MRKP-130`
- [647](https://github.com/vegaprotocol/core-test-coverage/issues/647) - Add integration test for team rewards `0009-MRKP-131`
- [648](https://github.com/vegaprotocol/core-test-coverage/issues/648) - Add integration test for team rewards `0009-MRKP-132`
- [533](https://github.com/vegaprotocol/core-test-coverage/issues/533) - Add integration test for team rewards `0009-MRKP-016`
- [534](https://github.com/vegaprotocol/core-test-coverage/issues/534) - Add integration test for team rewards `0009-MRKP-017`
- [536](https://github.com/vegaprotocol/core-test-coverage/issues/536) - Add integration test for team rewards `0009-MRKP-019`
- [10317](https://github.com/vegaprotocol/vega/issues/10317) - Add support in Ethereum oracles to data source from `L2s`
- [10429](https://github.com/vegaprotocol/vega/issues/10429) - Restrict the mark price by book to prices from the last mark price duration.
- [552](https://github.com/vegaprotocol/core-test-coverage/issues/552) - Add integration test for mark price `0053-PERP-034`
- [10459](https://github.com/vegaprotocol/vega/issues/10459) - Update `pMedian` to consider staleness of the inputs.
- [10429](https://github.com/vegaprotocol/vega/issues/10439) - Account for isolated margin mode in `EstimatePosition` endpoint.
- [10441](https://github.com/vegaprotocol/vega/issues/10441) - Remove active restore check in collateral snapshot loading, snapshot order change removes the need for it.
- [10286](https://github.com/vegaprotocol/vega/issues/10286) - If probability of trading is less than or equal to the minimum, assign it weight of zero for liquidity score calculation and change the validation of the tau scaling network parameter.
- [10376](https://github.com/vegaprotocol/vega/issues/10376) - Add spam protection for update profile.
- [10502](https://github.com/vegaprotocol/vega/issues/10502) - Rename index price to `internalCompositePrice`
- [10464](https://github.com/vegaprotocol/vega/issues/10464) - Add total of members in teams API.
- [10464](https://github.com/vegaprotocol/vega/issues/10464) - Add total of members in referral sets API.
- [10508](https://github.com/vegaprotocol/vega/issues/10508) - Change the behaviour of aggregation epochs for teams statistics API.
- [10502](https://github.com/vegaprotocol/vega/issues/10502) - Add `underlyingIndexPrice` to perpetual data.
- [10523](https://github.com/vegaprotocol/vega/issues/10523) - Fix repeated games statistics for multiple recurring transfers.
- [10527](https://github.com/vegaprotocol/vega/issues/10527) - Add support for `byte32` type in market proposal oracle definition.
- [10563](https://github.com/vegaprotocol/vega/issues/10563) - Spam protection for create/update referral program.
- [10246](https://github.com/vegaprotocol/vega/issues/10246) - Add quantum volumes to teams statistics API.
- [10550](https://github.com/vegaprotocol/vega/issues/10550) - Update network parameters with default values.
- [10543](https://github.com/vegaprotocol/vega/issues/10543) - Add `ELS` for `vAMM`.
- [10541](https://github.com/vegaprotocol/vega/issues/10541) - Add `SLA` for `vAMM`.

### 🐛 Fixes

- [9941](https://github.com/vegaprotocol/vega/issues/9941) - Add data node mapping for `WasEligible` field in referral set.
- [9940](https://github.com/vegaprotocol/vega/issues/9940) - Truncate fee stats in quantum down to the 6 decimal.
- [9940](https://github.com/vegaprotocol/vega/issues/9940) - Do not assume stop order is valid when generating ids up front.
- [9998](https://github.com/vegaprotocol/vega/issues/9998) - Slippage factors can now me updated in a market.
- [10036](https://github.com/vegaprotocol/vega/issues/10036) - Average entry price no longer flickers after a trade.
- [9956](https://github.com/vegaprotocol/vega/issues/9956) - Prevent validator node from starting if they do not have a Ethereum `RPCAddress` set.
- [9952](https://github.com/vegaprotocol/vega/issues/9952) - `PnL` flickering fix.
- [9977](https://github.com/vegaprotocol/vega/issues/9977) - Transfer infra fees directly to general account without going through vesting.
- [10041](https://github.com/vegaprotocol/vega/issues/10041) - List ledger entries `API` errors when using pagination.
- [10050](https://github.com/vegaprotocol/vega/issues/10050) - Cleanup `mempool` cache on commit.
- [10052](https://github.com/vegaprotocol/vega/issues/10052) - Some recent stats tables should have been `hypertables` with retention periods.
- [10103](https://github.com/vegaprotocol/vega/issues/10103) - List ledgers `API` returns bad error when filtering by transfer type only.
- [10120](https://github.com/vegaprotocol/vega/issues/10120) - Assure theoretical and actual funding payment calculations are consistent.
- [10164](https://github.com/vegaprotocol/vega/issues/10164) - Properly handle edge case where an external data point is received out of order.
- [10121](https://github.com/vegaprotocol/vega/issues/10121) - Assure `EstimatePosition` API works correctly with sparse perps data
- [10126](https://github.com/vegaprotocol/vega/issues/10126) - Account for invalid stop orders in batch, charge default gas.
- [10123](https://github.com/vegaprotocol/vega/issues/10123) - Ledger exports contain account types of "UNKNOWN" type
- [10132](https://github.com/vegaprotocol/vega/issues/10132) - Add mapping in `GraphQL` for update perps market proposal.
- [10125](https://github.com/vegaprotocol/vega/issues/10125) - Wire the `JoinTeam` command in the wallet.
- [10177](https://github.com/vegaprotocol/vega/issues/10177) - Add validation that order sizes are not strangely large.
- [10189](https://github.com/vegaprotocol/vega/issues/10189) - Votes for assets while proposal is waiting for node votes are included in the snapshot state.
- [10166](https://github.com/vegaprotocol/vega/issues/10166) - Closed markets should not be subscribed to data sources when restored from a snapshot.
- [10127](https://github.com/vegaprotocol/vega/issues/10127) - Untangle `ApplyReferralCode` and `JoinTeam` command verification.
- [10520](https://github.com/vegaprotocol/vega/issues/10520) - Fix `TWAP` calculations due to differences in time precision.
- [10153](https://github.com/vegaprotocol/vega/issues/10153) - Add metrics and reduce amount of request sent to the Ethereum `RPC`.
- [10147](https://github.com/vegaprotocol/vega/issues/10147) - Add network transfer largest share to the transfers if needed.
- [10158](https://github.com/vegaprotocol/vega/issues/10158) - Add the network as the zero-share default party in settlement engine.
- [10183](https://github.com/vegaprotocol/vega/issues/10183) - Fix transfer fees registration and decay.
- [9840](https://github.com/vegaprotocol/vega/issues/9840) - Team API inconsistency in joined at timestamps.
- [10205](https://github.com/vegaprotocol/vega/issues/10205) - Fix for transfer discount fees.
- [10211](https://github.com/vegaprotocol/vega/issues/10211) - Ensure infra fees don't get counted for vesting.
- [10217](https://github.com/vegaprotocol/vega/issues/10217) - Game ID for reward entity should be optional
- [10238](https://github.com/vegaprotocol/vega/issues/10238) - Fix logic when a user firsts requests spam information
- [10227](https://github.com/vegaprotocol/vega/issues/10227) - Make the wallet errors on spam stats meaningful.
- [10193](https://github.com/vegaprotocol/vega/issues/10193) - Denormalize `tx_results` to avoid joins with blocks when queried.
- [10233](https://github.com/vegaprotocol/vega/issues/10233) - Fix expiring stop orders panic.
- [10215](https://github.com/vegaprotocol/vega/issues/10215) - Rework pagination to align with the natural reverse-chronological order of the block explorer.
- [10241](https://github.com/vegaprotocol/vega/issues/10241) - Do not include start epoch on referees set statistics.
- [10219](https://github.com/vegaprotocol/vega/issues/10219) - Candles API should fill gaps when there are periods with no trades.
- [10050](https://github.com/vegaprotocol/vega/issues/10050) - Cleanup `mempool` cache on commit.
- [9882](https://github.com/vegaprotocol/vega/issues/9882) - Fix `net params` sent on closed channel.
- [10257](https://github.com/vegaprotocol/vega/issues/10257) - Fix equity like share votes count on update market proposal.
- [10260](https://github.com/vegaprotocol/vega/issues/10260) - `ListCandleData` errors when interval is `block`
- [9677](https://github.com/vegaprotocol/vega/issues/9677) - Removing snapshots and checkpoints do not fail on missing or corrupt state.
- [10267](https://github.com/vegaprotocol/vega/issues/10267) - `ListCandleData` errors when market is in opening auction
- [10276](https://github.com/vegaprotocol/vega/issues/10276) - `ListCandleData` should return empty strings instead of zero for prices when being gap filled.
- [10278](https://github.com/vegaprotocol/vega/issues/10278) - Transfers connection filtering by `isReward` and party causes error
- [10251](https://github.com/vegaprotocol/vega/issues/10251) - Add batch proposal API and fix batch proposal submission.
- [10285](https://github.com/vegaprotocol/vega/issues/10285) - Fix `MTM` settlement panic.
- [10321](https://github.com/vegaprotocol/vega/issues/10321) - Fix `ListPartyMarginModes` panic.
- [10324](https://github.com/vegaprotocol/vega/issues/10324) - Fix `GetMarketHistoryByID` to only return the most recent market data information when no dates are provided.
- [10318](https://github.com/vegaprotocol/vega/issues/10318) - Prevent stop orders being placed during opening auction.
- [476](https://github.com/vegaprotocol/core-test-coverage/issues/476) - Add tests for markets expiring in opening auction, fix a bug for future markets.
- [10354](https://github.com/vegaprotocol/vega/issues/10354) - Renumbered SQL migration scripts 0055-0067 due to 0068 being released as part of a patch without renumbering.
- [476](https://github.com/vegaprotocol/core-test-coverage/issues/476) - Add tests for markets expiring in opening auction, fix a bug for future markets.
- [10362](https://github.com/vegaprotocol/vega/issues/10362) - Include notional volumes in `SLA` snapshot.
- [10339](https://github.com/vegaprotocol/vega/issues/10339) - Fix for GraphQL batch proposals support.
- [10326](https://github.com/vegaprotocol/vega/issues/10326) - Fix intermittent test failure.
- [10382](https://github.com/vegaprotocol/vega/issues/10382) - Fix switch to isolated margin with parked pegged orders.
- [10390](https://github.com/vegaprotocol/vega/pull/10390) - Fix the data node startup when an external `hypertable` exist outside of the public schema
- [10393](https://github.com/vegaprotocol/vega/issues/10393) - Fixed source weight validation.
- [10396](https://github.com/vegaprotocol/vega/issues/10396) - `ListTransfers` returns error.
- [10299](https://github.com/vegaprotocol/vega/issues/10299) - `ListTransfers` does not return staking rewards.
- [10496](https://github.com/vegaprotocol/vega/issues/10496) - `MarketData` API now always report correct funding payment when a perpetual market is terminated.
- [10407](https://github.com/vegaprotocol/vega/issues/10407) - Workaround to allow running feature test with invalid 0 mark price frequency.
- [10378](https://github.com/vegaprotocol/vega/issues/10378) - Ensure network position has price set at all times.
- [10409](https://github.com/vegaprotocol/vega/issues/10499) - Block explorer `API` failing in release `0.74.0`.
- [10417](https://github.com/vegaprotocol/vega/issues/10417) - Party margin modes `API` always errors.
- [10431](https://github.com/vegaprotocol/vega/issues/10431) - Fix source staleness validation.
- [10436](https://github.com/vegaprotocol/vega/issues/10436) - Fix source staleness validation when oracles are not defined.
- [10434](https://github.com/vegaprotocol/vega/issues/10434) - Unsubscribe oracles when market is closed.
- [10454](https://github.com/vegaprotocol/vega/issues/10454) - Fix account resolver validation to include order margin account.
- [10451](https://github.com/vegaprotocol/vega/issues/10451) - Fix get update asset bundle.
- [10480](https://github.com/vegaprotocol/vega/issues/10480) - Fix migration of position average entry price.
- [10419](https://github.com/vegaprotocol/vega/issues/10419) - Block explorer database migration is slow.
- [10431](https://github.com/vegaprotocol/vega/issues/10431) - Fix source staleness validation.
- [10419](https://github.com/vegaprotocol/vega/issues/10419) - Block explorer database migration is slow.
- [10510](https://github.com/vegaprotocol/vega/issues/10510) - Removing distressed position who hasn't traded does not populate trade map for network.
- [10470](https://github.com/vegaprotocol/vega/issues/10470) - Mark non-optional parameters as required and update documentation strings.
- [10456](https://github.com/vegaprotocol/vega/issues/10456) - Expose proper enum for `GraphQL` dispatch metric.
- [10301](https://github.com/vegaprotocol/vega/issues/10301) - Fix get epoch by block.
- [10343](https://github.com/vegaprotocol/vega/issues/10343) - Remove auction trigger extension and triggering ratio from liquidity monitoring parameters.
- [10493](https://github.com/vegaprotocol/vega/issues/10493) - Fix isolated margin panic on amend order.
- [10490](https://github.com/vegaprotocol/vega/issues/10490) - Handle the `quant` library returning `NaN`
- [10504](https://github.com/vegaprotocol/vega/issues/10504) - Make sure the referral sets API accounts for referees switch.
- [10525](https://github.com/vegaprotocol/vega/issues/10525) - Move `batchTerms` to parent object in batch proposals API.
- [10530](https://github.com/vegaprotocol/vega/issues/10530) - Game API returns errors.
- [10531](https://github.com/vegaprotocol/vega/issues/10531) - Team Members Statistics API doesn't return data.
- [10533](https://github.com/vegaprotocol/vega/issues/10533) - Fix `EstiamtePosition` returning an error when trying to access the party id field via GraphQL.
- [10546](https://github.com/vegaprotocol/vega/issues/10546) - `EstimateTransferFee` API errors when there is no discount.
- [10532](https://github.com/vegaprotocol/vega/issues/10532) - Fix for games statistics.
- [10568](https://github.com/vegaprotocol/vega/issues/10568) - Fix for `PnL` underflow.
- [10299](https://github.com/vegaprotocol/vega/issues/10299) - Fix for rewards transfers list.
- [10567](https://github.com/vegaprotocol/vega/issues/10567) - Rewards/Teams/Games API disagree on numbers.
- [10578](https://github.com/vegaprotocol/vega/issues/10578) - Game API reward amounts should display quantum values and asset ID.
- [10558](https://github.com/vegaprotocol/vega/issues/10558) - Return current market observable is collateral available is below maintenance margin for specified position.
- [10604](https://github.com/vegaprotocol/vega/issues/10604) - Register margin modes API subscriber.
- [10595](https://github.com/vegaprotocol/vega/issues/10595) - Fix failed amends for isolated margin orders causing negative spread in console.
- [10606](https://github.com/vegaprotocol/vega/issues/10606) - Party profiles `API` was not returning results.
- [10625](https://github.com/vegaprotocol/vega/issues/10625) - Fix panic in update spot market.

## 0.73.0

### 🚨 Breaking changes

- [8679](https://github.com/vegaprotocol/vega/issues/8679) - Snapshot configuration `load-from-block-height` no longer accept `-1` as value. To reload from the latest snapshot, it should be valued `0`.
- [8679](https://github.com/vegaprotocol/vega/issues/8679) - Snapshot configuration `snapshot-keep-recent` only accept values from `1` (included) to `10` (included) .
- [8944](https://github.com/vegaprotocol/vega/issues/8944) - Asset ID field on the `ExportLedgerEntriesRequest gRPC API` for exporting ledger entries has changed type to make it optional.
- [9562](https://github.com/vegaprotocol/vega/issues/9562) - `--lite` and `--archive` options to data node have been replaced with `--retention-profile=[archive|conservative|minimal]` with default mode as archive.
- [9258](https://github.com/vegaprotocol/vega/issues/9258) - Rework `GetReferralSetStats` endpoint.
- [9258](https://github.com/vegaprotocol/vega/issues/9258) - Change HTTP endpoint from `/volume-discount-stats` to `/volume-discount-program/stats`.
- [9258](https://github.com/vegaprotocol/vega/issues/9258) - Change HTTP endpoint from `/referral-sets/stats/{id}` to `/referral-sets/{id}/stats`.
- [9719](https://github.com/vegaprotocol/vega/issues/9719) - Remove unnecessary fields from referral and volume discount program proposals.
- [9733](https://github.com/vegaprotocol/vega/issues/9733) - Making `set_id` optional in `referral set stats` endpoint
- [9743](https://github.com/vegaprotocol/vega/issues/9743) - Rename `ReferralFeeStats` endpoints to `FeesStats`, and `FeeStats` event to `FeesStats`.
- [9408](https://github.com/vegaprotocol/vega/issues/9408) - Enforce pagination range.
- [9757](https://github.com/vegaprotocol/vega/issues/9757) - Liquidity provisions `API` shows the pending `LP` instead of the current when an update is accepted by the network.

### 🛠 Improvements

- [8051](https://github.com/vegaprotocol/vega/issues/8051) - Upgrade to comet `0.38.0`
- [9484](https://github.com/vegaprotocol/vega/issues/9484) - Remove network parameters that only provide defaults for market liquidity settings.
- [8718](https://github.com/vegaprotocol/vega/issues/8718) - Emit market data event after setting mark price prior to final settlement.
- [8590](https://github.com/vegaprotocol/vega/issues/8590) - Improved Ethereum oracle support.
- [8754](https://github.com/vegaprotocol/vega/issues/8754) - Introduce Perpetuals and their funding payment calculations.
- [8731](https://github.com/vegaprotocol/vega/issues/8731) - Add liquidity provision `SLA` to governance proposals for spot market.
- [8741](https://github.com/vegaprotocol/vega/issues/8741) - Add a network parameter for disabling `Ethereum` oracles.
- [8600](https://github.com/vegaprotocol/vega/issues/8600) - Clean and refactor data source packages.
- [8845](https://github.com/vegaprotocol/vega/issues/8845) - Add support for network treasury and global insurance accounts.
- [9545](https://github.com/vegaprotocol/vega/issues/9545) - Auto load new segments after load finishes
- [8661](https://github.com/vegaprotocol/vega/issues/8661) - Refactor the snapshot engine to make it testable.
- [8680](https://github.com/vegaprotocol/vega/issues/8680) - Move loading the local snapshot in the initialization steps.
- [8682](https://github.com/vegaprotocol/vega/issues/8682) - Share snapshot by search the metadata database instead of loading the tree.
- [8846](https://github.com/vegaprotocol/vega/issues/8846) - Add support to transfer recurring transfers to metric based reward
- [9549](https://github.com/vegaprotocol/vega/issues/9549) - Update config defaults to better support archive nodes
- [8857](https://github.com/vegaprotocol/vega/issues/8857) - Add a step for getting the balance of the liquidity provider liquidity fee account
- [9483](https://github.com/vegaprotocol/vega/issues/9483) - Zip history segments only once
- [8847](https://github.com/vegaprotocol/vega/issues/8847) - Implement internal time trigger data source.
- [8895](https://github.com/vegaprotocol/vega/issues/8895) - Allow to set runtime parameters in the SQL Store connection structure
- [9678](https://github.com/vegaprotocol/vega/issues/9678) - Cache and forward referral rewards multiplier and factor multiplier
- [8779](https://github.com/vegaprotocol/vega/issues/8779) - Query all details of liquidity providers via an API.
- [8924](https://github.com/vegaprotocol/vega/issues/8924) - Refactor slightly to remove need to deep clone `proto` types
- [8782](https://github.com/vegaprotocol/vega/issues/8782) - List all active liquidity providers for a market via API.
- [8753](https://github.com/vegaprotocol/vega/issues/8753) - Governance for new market proposal.
- [8752](https://github.com/vegaprotocol/vega/issues/8752) - Add `PERPS` network parameter.
- [8759](https://github.com/vegaprotocol/vega/issues/8759) - Add update market support for `PERPS`.
- [8758](https://github.com/vegaprotocol/vega/issues/8758) - Internal recurring time trigger for `PERPS`.
- [8913](https://github.com/vegaprotocol/vega/issues/8913) - Add business logic for team management.
- [8765](https://github.com/vegaprotocol/vega/issues/8765) - Implement snapshots state for `PERPS`.
- [8918](https://github.com/vegaprotocol/vega/issues/8918) - Implement commands for team management.
- [9401](https://github.com/vegaprotocol/vega/issues/9401) - Use boot strap peers when fetching initial network history segment
- [9670](https://github.com/vegaprotocol/vega/issues/9670) - Do not check destination account exists when validating transfer proposal
- [8960](https://github.com/vegaprotocol/vega/issues/8960) - Improve wiring perpetual markets through governance.
- [8969](https://github.com/vegaprotocol/vega/issues/8969) - Improve wiring of internal time triggers for perpetual markets.
- [9001](https://github.com/vegaprotocol/vega/issues/9001) - Improve wiring of perpetual markets into the data node.
- [8985](https://github.com/vegaprotocol/vega/issues/8985) - Improve snapshot restore of internal time triggers for perpetual markets.
- [9146](https://github.com/vegaprotocol/vega/issues/9146) - Improve `TWAP` for perpetual markets to do calculations incrementally.
- [8817](https://github.com/vegaprotocol/vega/issues/8817) - Add interest term to perpetual funding payment calculation.
- [8755](https://github.com/vegaprotocol/vega/issues/8755) - Improve testing for perpetual settlement and collateral transfers.
- [9319](https://github.com/vegaprotocol/vega/issues/9319) - Introduce product data field in market data for product specific information.
- [8756](https://github.com/vegaprotocol/vega/issues/8756) - Settlement and margin implementation for `PERPS`.
- [8932](https://github.com/vegaprotocol/vega/issues/8932) - Fix range validation of `performanceHysteresisEpochs`
- [8887](https://github.com/vegaprotocol/vega/pull/8887) - Remove differences for snapshot loading when the `nullchain` is used instead of `tendermint`
- [8973](https://github.com/vegaprotocol/vega/issues/8973) - Do some more validation on Ethereum call specifications, add explicit error types to improve reporting
- [8957](https://github.com/vegaprotocol/vega/issues/8957) - Oracle bindings for `PERPS`.
- [8770](https://github.com/vegaprotocol/vega/issues/8770) - Add `PERPS` to integration tests.
- [8763](https://github.com/vegaprotocol/vega/issues/8763) - Periodic settlement data endpoint.
- [8920](https://github.com/vegaprotocol/vega/issues/8920) - Emit events when something happens to teams.
- [8917](https://github.com/vegaprotocol/vega/issues/8917) - Support teams engine snapshots
- [9007](https://github.com/vegaprotocol/vega/issues/9007) - Add reward vesting mechanisms.
- [8914](https://github.com/vegaprotocol/vega/issues/8914) - Add referral network parameters.
- [9023](https://github.com/vegaprotocol/vega/issues/9023) - Add transaction comparison tool.
- [8120](https://github.com/vegaprotocol/vega/issues/8120) - Data node `API` support for spot markets, data and governance.
- [8762](https://github.com/vegaprotocol/vega/issues/8762) - Data node `API` support for perpetual markets, data and governance.
- [8761](https://github.com/vegaprotocol/vega/issues/8761) - Add terminating `PERPS` via governance.
- [9021](https://github.com/vegaprotocol/vega/issues/9021) - Add rejection reason to stop orders.
- [9012](https://github.com/vegaprotocol/vega/issues/9012) - Add governance to update the referral program.
- [9077](https://github.com/vegaprotocol/vega/issues/9077) - Ensure liquidity `SLA` parameters are exposed on markets and proposals
- [9046](https://github.com/vegaprotocol/vega/issues/9046) - Send event on referral engine state change.
- [9045](https://github.com/vegaprotocol/vega/issues/9045) - Snapshot the referral engine.
- [9136](https://github.com/vegaprotocol/vega/issues/9136) - Referral engine returns reward factor for a given party
- [9076](https://github.com/vegaprotocol/vega/issues/9076) - Add current funding rate to market data.
- [1932](https://github.com/vegaprotocol/devops-infra/issues/1932) - Allow configuring an `SQL` statement timeout.
- [9162](https://github.com/vegaprotocol/vega/issues/9162) - Refactor transfers to support new metric based rewards and distribution strategies
- [9163](https://github.com/vegaprotocol/vega/issues/9163) - Refactor reward engine to support the new metric based reward distribution
- [9164](https://github.com/vegaprotocol/vega/issues/9164) - Refactor market activity tracker to support more metrics and history
- [9219](https://github.com/vegaprotocol/vega/issues/9219) - Don't do `IPFS` garbage collection every segment
- [9208](https://github.com/vegaprotocol/vega/issues/9208) - Refactor referral set and teams state
- [9204](https://github.com/vegaprotocol/vega/issues/9204) - Ensure teams names are not duplicates
- [9080](https://github.com/vegaprotocol/vega/issues/9080) - Add support for vested and vesting account in GraphQL
- [9147](https://github.com/vegaprotocol/vega/issues/9147) - Add reward multiplier to vesting engine
- [9593](https://github.com/vegaprotocol/vega/issues/9593) - Version and whether it was taken for a protocol upgrade added to snapshot.
- [9234](https://github.com/vegaprotocol/vega/issues/9234) - Add support for transfers out of the vested account
- [9199](https://github.com/vegaprotocol/vega/issues/9199) - Consider running notional volume to determine referrers and referees tier
- [9214](https://github.com/vegaprotocol/vega/issues/9214) - Add staking tier on referral program
- [9205](https://github.com/vegaprotocol/vega/issues/9205) - Ensure staking requirement when creating / joining referral sets
- [9032](https://github.com/vegaprotocol/vega/issues/9032) - Implement activity streaks.
- [9133](https://github.com/vegaprotocol/vega/issues/9133) - Apply discounts/rewards in fee calculation
- [9281](https://github.com/vegaprotocol/vega/issues/9281) - Add ability to filter funding period data points by the sequence they land in.
- [9254](https://github.com/vegaprotocol/vega/issues/9254) - Add fee discounts to trade API
- [9246](https://github.com/vegaprotocol/vega/issues/9246) - Add rewards multiplier support in the referral engine.
- [9063](https://github.com/vegaprotocol/vega/issues/9063) - Make `calculationTimeStep` a network parameter
- [9167](https://github.com/vegaprotocol/vega/issues/9167) - Rename liquidity network parameters
- [9302](https://github.com/vegaprotocol/vega/issues/9302) - Volume discount program
- [9288](https://github.com/vegaprotocol/vega/issues/9288) - Add check for current epoch to integration tests.
- [9288](https://github.com/vegaprotocol/vega/issues/9288) - Allow integration test time to progress by epoch.
- [9078](https://github.com/vegaprotocol/vega/issues/9078) - Add activity streak `API`.
- [9351](https://github.com/vegaprotocol/vega/issues/9351) - Avoid using strings in market activity tracker snapshot and checkpoint
- [9079](https://github.com/vegaprotocol/vega/issues/9079) - Add API to get the current referral program
- [8916](https://github.com/vegaprotocol/vega/issues/8916) - Add data node `API` for teams.
- [7461](https://github.com/vegaprotocol/vega/issues/7461) - Add endpoint to get transfer by ID.
- [9417](https://github.com/vegaprotocol/vega/issues/9417) - Add additional filters for referral set `APIs`.
- [9375](https://github.com/vegaprotocol/vega/issues/9375) - Apply SLA parameters at epoch boundary.
- [9279](https://github.com/vegaprotocol/vega/issues/9279) - Remove checks for best bid/ask when leaving opening auction.
- [9456](https://github.com/vegaprotocol/vega/issues/9456) - Add liquidity `SLA` parameters to `NewMarket` and `UpdateMarketConfiguration` proposals in `GraphQL`.
- [9408](https://github.com/vegaprotocol/vega/issues/9408) - Check for integer overflow in pagination.
- [5176](https://github.com/vegaprotocol/vega/issues/5176) - Check for duplicate asset registration in integration tests.
- [9496](https://github.com/vegaprotocol/vega/issues/9496) - Added support for new dispatch strategy fields in feature tests
- [9536](https://github.com/vegaprotocol/vega/issues/9536) - Feature tests for average position metric transfers and reward
- [9547](https://github.com/vegaprotocol/vega/issues/9547) - Less disk space is now needed to initialise a data node from network history.
- [8764](https://github.com/vegaprotocol/vega/issues/8764) - Include funding payment in margin and liquidation price estimates for `PERPS`.
- [9519](https://github.com/vegaprotocol/vega/issues/9519) - Fix `oracle_specs` data in the `database` that was inadvertently removed during an earlier database migration
- [9475](https://github.com/vegaprotocol/vega/issues/9475) - Make `oracle_data` and `oracle_data_oracle_specs` into `hypertables`
- [9478](https://github.com/vegaprotocol/vega/issues/8764) - Add SLA statistics to market data and liquidity provision APIs.
- [9558](https://github.com/vegaprotocol/vega/issues/9558) - Feature tests for relative return metric transfers and reward
- [9559](https://github.com/vegaprotocol/vega/issues/9559) - Feature tests for return volatility metric transfers and reward
- [9564](https://github.com/vegaprotocol/vega/issues/9564) - Fix error message for too many staking tiers.
- [8421](https://github.com/vegaprotocol/vega/issues/8421) - Markets that spend too long in opening auction should be cancelled.
- [9575](https://github.com/vegaprotocol/vega/issues/9575) - Expand SLA statistics by required liquidity and notional volumes.
- [9578](https://github.com/vegaprotocol/vega/issues/9578) - Add spam protection for referral transactions
- [9590](https://github.com/vegaprotocol/vega/issues/9590) - Restore positions for market activity tracker on migration from old version.
- [9589](https://github.com/vegaprotocol/vega/issues/9589) - Add event for funding payments.
- [9460](https://github.com/vegaprotocol/vega/issues/9460) - Add APIs for volume discount program.
- [9628](https://github.com/vegaprotocol/vega/issues/9628) - Upgrade `CometBFT`.
- [9577](https://github.com/vegaprotocol/vega/issues/9577) - Feature test coverage for distribution strategy
- [9542](https://github.com/vegaprotocol/vega/issues/9542) - Ensure all per asset accounts exist after migration
- [8475](https://github.com/vegaprotocol/vega/issues/8475) - Emit margin levels event when margin balance is cleared.
- [9664](https://github.com/vegaprotocol/vega/issues/9664) - Handle pagination of release request with github.
- [9681](https://github.com/vegaprotocol/vega/issues/9681) - Move referral set reward factor to the referral set stats events
- [9708](https://github.com/vegaprotocol/vega/issues/9708) - Use the correct transaction hash when submitting orders through the `nullblockchain`
- [9755](https://github.com/vegaprotocol/vega/issues/9755) - Add referral program spam statistics to `GetSpamStatistics` core API.
- [409](https://github.com/vegaprotocol/OctoberACs/issues/409) - Add integration test for team rewards `0056-REWA-106`
- [410](https://github.com/vegaprotocol/OctoberACs/issues/410) - Add integration test for team rewards `0056-REWA-107`
- [411](https://github.com/vegaprotocol/OctoberACs/issues/411) - Add integration test for team rewards `0056-REWA-108`
- [408](https://github.com/vegaprotocol/OctoberACs/issues/408) - Add integration test for team rewards `0056-REWA-105`
- [407](https://github.com/vegaprotocol/OctoberACs/issues/407) - Add integration test for team rewards `0056-REWA-104`
- [406](https://github.com/vegaprotocol/OctoberACs/issues/406) - Add integration test for team rewards `0056-REWA-103`
- [388](https://github.com/vegaprotocol/OctoberACs/issues/388) - Add integration test for `0029-FEES-031`
- [387](https://github.com/vegaprotocol/OctoberACs/issues/387) - Add integration test for `0029-FEES-028`
- [9710](https://github.com/vegaprotocol/vega/issues/9710) - Add config to keep spam and `POW` enabled when using null chain.
- [403](https://github.com/vegaprotocol/OctoberACs/issues/403) - Add integration test for team rewards `0056-REWA-100`
- [402](https://github.com/vegaprotocol/OctoberACs/issues/402) - Add integration test for team rewards `0056-REWA-099`
- [401](https://github.com/vegaprotocol/OctoberACs/issues/401) - Add integration test for team rewards `0056-REWA-098`
- [400](https://github.com/vegaprotocol/OctoberACs/issues/400) - Add integration test for team rewards `0056-REWA-097`
- [2971](https://github.com/vegaprotocol/system-tests/issues/2971) - Fix margin calculation for `PERPS`.
- [2948](https://github.com/vegaprotocol/system-tests/issues/2948) - Add integration test for failing system test.
- [9541](https://github.com/vegaprotocol/vega/issues/9731) - Add filtering for party to the referral fees API.
- [9541](https://github.com/vegaprotocol/vega/issues/9731) - Add X day aggregate totals for referral set referees.
- [2985](https://github.com/vegaprotocol/system-tests/issues/2985) - Coverage for insurance pool transfers, fix deadlock when terminating pending market through governance.
- [9770](https://github.com/vegaprotocol/vega/issues/9770) - Fix `PnL` flickering bug.
- [9785](https://github.com/vegaprotocol/vega/issues/9785) - Support transfers to network treasury.
- [9743](https://github.com/vegaprotocol/system-tests/issues/9743) - Expose maker fees in fees stats API.
- [9750](https://github.com/vegaprotocol/vega/issues/9750) - Add paid liquidity fees statistics to API.
- [8919](https://github.com/vegaprotocol/vega/issues/8919) - Add `CreatedAt` field to transaction type in the block explorer API.
- [7248](https://github.com/vegaprotocol/vega/issues/7248) - Add transaction proof of work and version in block explorer API.
- [9815](https://github.com/vegaprotocol/vega/issues/9815) - Introduce API to get fees statistics for a given party.
- [9861](https://github.com/vegaprotocol/vega/issues/9861) - Add `lockedUntilEpoch` to rewards API
- [9846](https://github.com/vegaprotocol/vega/issues/9846) - Add API for vesting and locked balances.
- [9876](https://github.com/vegaprotocol/vega/issues/9876) - Disable teams support in transfers and referral sets.
- [8056](https://github.com/vegaprotocol/vega/issues/8056) - Add transfer fees event.
- [9883](https://github.com/vegaprotocol/vega/issues/9883) - Add reward filter to transfer API.
- [9886](https://github.com/vegaprotocol/vega/issues/9886) - Add party vesting statistics endpoint.
- [9770](https://github.com/vegaprotocol/vega/issues/9770) - Fix `PnL` flickering bug on an already open position.
- [9827](https://github.com/vegaprotocol/vega/issues/9827) - Expose `was_eligible` in the referral set statistics.
- [9918](https://github.com/vegaprotocol/vega/issues/9918) - Expose vesting quantum balance.
- [9923](https://github.com/vegaprotocol/vega/issues/9923) - Add referrer volume to referral set stats.

### 🐛 Fixes

- [8417](https://github.com/vegaprotocol/vega/issues/8417) - Fix normalisation of Ethereum calls that return structures
- [8719](https://github.com/vegaprotocol/vega/issues/8719) - Do not try to resolve iceberg order if it's not set
- [8721](https://github.com/vegaprotocol/vega/issues/8721) - Fix panic with triggered OCO expiring
- [8751](https://github.com/vegaprotocol/vega/issues/8751) - Fix Ethereum oracle data error event sent with incorrect sequence number
- [8906](https://github.com/vegaprotocol/vega/issues/8906) - Fix Ethereum oracle confirmation height not be used
- [8729](https://github.com/vegaprotocol/vega/issues/8729) - Stop order direction not set in datanode
- [8545](https://github.com/vegaprotocol/vega/issues/8545) - Block Explorer pagination does not order correctly.
- [8748](https://github.com/vegaprotocol/vega/issues/8748) - `ListSuccessorMarkets` does not return results.
- [9784](https://github.com/vegaprotocol/vega/issues/9784) - Referral timestamp consistency
- [8749](https://github.com/vegaprotocol/vega/issues/8749) - Ensure stop order expiry is in the future
- [9337](https://github.com/vegaprotocol/vega/issues/9337) - Non deterministic ordering of vesting ledger events
- [8773](https://github.com/vegaprotocol/vega/issues/8773) - Fix panic with stop orders
- [9343](https://github.com/vegaprotocol/vega/issues/9343) - Prevent malicious validator submitting Ethereum oracle chain event prior to initial start time
- [8792](https://github.com/vegaprotocol/vega/issues/8792) - Fix panic when starting null block chain node.
- [8739](https://github.com/vegaprotocol/vega/issues/8739) - Cancel orders for rejected markets.
- [9831](https://github.com/vegaprotocol/vega/issues/9831) - Disable auto vacuum during network history load
- [9685](https://github.com/vegaprotocol/vega/issues/9685) - All events for core are sent out in a stable order.
- [9594](https://github.com/vegaprotocol/vega/issues/9594) - non deterministic order events on market termination
- [9350](https://github.com/vegaprotocol/vega/issues/9350) - Clear account ledger events causing segment divergence
- [9118](https://github.com/vegaprotocol/vega/issues/9118) - Improve list stop orders error message
- [9406](https://github.com/vegaprotocol/vega/issues/9105) - Fix Ethereum oracle validation failing unexpectedly when using go 1.19
- [9105](https://github.com/vegaprotocol/vega/issues/9105) - Truncate virtual stake decimal places
- [9517](https://github.com/vegaprotocol/vega/issues/9517) - Start and end time stamps in `GetMarketDataHistoryByID` now work as expected.
- [9348](https://github.com/vegaprotocol/vega/issues/9348) - Nil pointer error in Ethereum call engine when running with null block chain
- [8800](https://github.com/vegaprotocol/vega/issues/8800) - `expiresAt` is always null in the Stop Orders `API`.
- [8796](https://github.com/vegaprotocol/vega/issues/8796) - Avoid updating active proposals slice while iterating over it.
- [8631](https://github.com/vegaprotocol/vega/issues/8631) - Prevent duplicate Ethereum call chain event after snapshot start
- [9528](https://github.com/vegaprotocol/vega/issues/9528) - Allow order submission when in governance suspended auction.
- [8679](https://github.com/vegaprotocol/vega/issues/8679) - Disallow snapshot state-sync if local snapshots exist
- [8364](https://github.com/vegaprotocol/vega/issues/8364) - Initialising from network history not working after database wipe
- [8827](https://github.com/vegaprotocol/vega/issues/8827) - Add block height validation to validator initiated transactions and pruning to the `pow` engine cache
- [8836](https://github.com/vegaprotocol/vega/issues/8836) - Fix enactment of market update state
- [9760](https://github.com/vegaprotocol/vega/issues/9760) - Validate that a recurring transfer with markets matches the asset in the transfer.
- [8848](https://github.com/vegaprotocol/vega/issues/8848) - Handle the case where the market is terminated and the epoch ends at the same block.
- [8853](https://github.com/vegaprotocol/vega/issues/8853) - Liquidity provision amendment bug fixes
- [8862](https://github.com/vegaprotocol/vega/issues/8862) - Fix settlement via governance
- [9088](https://github.com/vegaprotocol/vega/issues/9088) - Include error field in `contractCall` snapshot.
- [8854](https://github.com/vegaprotocol/vega/issues/8854) - Add liquidity `v2` snapshots to the list of providers
- [8772](https://github.com/vegaprotocol/vega/issues/8772) - Checkpoint panic on successor markets.
- [8962](https://github.com/vegaprotocol/vega/issues/8962) - Refreshed pegged iceberg orders remain tracked as pegged orders.
- [8837](https://github.com/vegaprotocol/vega/issues/8837) - Remove successor entries from snapshot if they will be removed next tick.
- [8868](https://github.com/vegaprotocol/vega/issues/8868) - Fix `oracle_specs` table null value error.
- [9038](https://github.com/vegaprotocol/vega/issues/9038) - New market proposals are no longer in both the active and enacted slices to prevent pointer sharing.
- [8878](https://github.com/vegaprotocol/vega/issues/8878) - Fix amend to consider market decimals when checking for sufficient funds.
- [8698](https://github.com/vegaprotocol/vega/issues/8698) - Final settlement rounding can be off by a 1 factor instead of just one.
- [9113](https://github.com/vegaprotocol/vega/issues/9113) - Only add pending signatures to the retry list when the notary engine snapshot restores.
- [8861](https://github.com/vegaprotocol/vega/issues/8861) - Fix successor proposals never leaving proposed state.
- [9086](https://github.com/vegaprotocol/vega/issues/9086) - Set identifier generator during perpetual settlement for closed out orders.
- [8884](https://github.com/vegaprotocol/vega/issues/8884) - Do not assume `\n` is present on the first read chunk of the input
- [8477](https://github.com/vegaprotocol/vega/issues/8477) - Do not allow opening auction duration of 0
- [9272](https://github.com/vegaprotocol/vega/issues/9272) - Fix dead-lock when terminating a perpetual market.
- [8891](https://github.com/vegaprotocol/vega/issues/8891) - Emit market update event when resuming via governance
- [8874](https://github.com/vegaprotocol/vega/issues/8874) - Database migration can fail when rolling back and migrating up again.
- [8855](https://github.com/vegaprotocol/vega/issues/8855) - Preserve reference to parent market when restoring checkpoint data
- [9576](https://github.com/vegaprotocol/vega/issues/9576) - Metric collection during bridge stops no longer causes a panic.
- [8909](https://github.com/vegaprotocol/vega/issues/8909) - initialise id generator for all branches of market state update
- [9004](https://github.com/vegaprotocol/vega/issues/9004) - Clear insurance pools in a deterministic order in successor markets.
- [8908](https://github.com/vegaprotocol/vega/issues/8908) - A rejected parent market should result in all successors getting rejected, too.
- [8953](https://github.com/vegaprotocol/vega/issues/8953) - Fix reward distribution when validator delegate to other validators
- [8898](https://github.com/vegaprotocol/vega/issues/8898) - Fix auction uncrossing handling for spots
- [8968](https://github.com/vegaprotocol/vega/issues/8968) - Fix missing parent reference on checkpoint restore.
- [9009](https://github.com/vegaprotocol/vega/issues/9009) - Fix non determinism in `RollbackParentELS`
- [8945](https://github.com/vegaprotocol/vega/issues/8945) - Expose missing order from the stop order `GraphQL` endpoint.
- [9034](https://github.com/vegaprotocol/vega/issues/9034) - Add missing transfer types to `GraphQL` schema definition.
- [9075](https://github.com/vegaprotocol/vega/issues/9075) - `Oracle Specs API` can fail when oracle data is null.
- [8944](https://github.com/vegaprotocol/vega/issues/8944) - Fix ignoring of asset `ID` in ledger export, and make it optional.
- [8971](https://github.com/vegaprotocol/vega/issues/8971) - Record the epoch start time even in opening auction
- [8992](https://github.com/vegaprotocol/vega/issues/8992) - allow for 0 time step for `SLA` fee calculation
- [9266](https://github.com/vegaprotocol/vega/issues/9266) - Preserve perpetual state when updating product
- [8988](https://github.com/vegaprotocol/vega/issues/8988) - allow amend/cancel of pending liquidity provision
- [8993](https://github.com/vegaprotocol/vega/issues/8993) - handle the case where commitment min time fraction is 1
- [9252](https://github.com/vegaprotocol/vega/issues/9252) - Preserve the order of pegged orders in snapshots.
- [9066](https://github.com/vegaprotocol/vega/issues/9066) - Ensure a party have a liquidity provision before trying to cancel
- [9140](https://github.com/vegaprotocol/vega/issues/9140) - Stop orders table should be a `hypertable` with retention policy.
- [9153](https://github.com/vegaprotocol/vega/issues/9153) - `MTM` win transfers can be less than one.
- [9227](https://github.com/vegaprotocol/vega/issues/9227) - Do not allow market updates that change the product type.
- [9535](https://github.com/vegaprotocol/vega/issues/9535) - Populate referral statistics when loading from a snapshot.
- [9178](https://github.com/vegaprotocol/vega/issues/9178) - Fix LP amendment panic
- [9329](https://github.com/vegaprotocol/vega/issues/9329) - Roll next trigger time forward until after the time that triggered it.
- [9053](https://github.com/vegaprotocol/vega/issues/9053) - Handle settle market events in core positions plug-in.
- [9189](https://github.com/vegaprotocol/vega/issues/9189) - Fix stop orders snapshots
- [9313](https://github.com/vegaprotocol/vega/issues/9313) - `ListLedgerEntries` transfers type filter works as expected with open to and from filters.
- [9198](https://github.com/vegaprotocol/vega/issues/9198) - Fix panic during LP amendment applications
- [9196](https://github.com/vegaprotocol/vega/issues/9196) - `API` documentation should specify the time format.
- [9294](https://github.com/vegaprotocol/vega/issues/9294) - Data node panics when many markets use the same data source
- [9203](https://github.com/vegaprotocol/vega/issues/9203) - Do not remove LPs from the parties map if they are an LP without an open position
- [9202](https://github.com/vegaprotocol/vega/issues/9202) - Fix `ELS` not calculated when leaving opening auction.
- [9276](https://github.com/vegaprotocol/vega/issues/9276) - Properly save stats in the positions snapshot
- [9276](https://github.com/vegaprotocol/vega/issues/9276) - Properly save stats in the positions snapshot
- [9305](https://github.com/vegaprotocol/vega/issues/9305) - Properly propagate `providersFeeCalculationTimeStep` parameter
- [9374](https://github.com/vegaprotocol/vega/issues/9374) - `ListGovernanceData` ignored reference filter.
- [9385](https://github.com/vegaprotocol/vega/issues/9385) - Support deprecated liquidity auction type for compatibility
- [9398](https://github.com/vegaprotocol/vega/issues/9398) - Fix division by zero panic in market liquidity
- [9413](https://github.com/vegaprotocol/vega/issues/9413) - Fix range validation for SLA parameters
- [9332](https://github.com/vegaprotocol/vega/issues/9332) - Ethereum oracles sending data to unintended destinations
- [9447](https://github.com/vegaprotocol/vega/issues/9447) - Store current SLA parameters in the liquidity engine snapshot.
- [9433](https://github.com/vegaprotocol/vega/issues/9433) - fix referral set snapshot
- [9432](https://github.com/vegaprotocol/vega/issues/9432) - fix referral set not saved to database.
- [9449](https://github.com/vegaprotocol/vega/issues/9449) - if expiration is empty, never expire a discount/reward program
- [9263](https://github.com/vegaprotocol/vega/issues/9263) - save dispatch strategy details in the database and allow for its retrieval.
- [9374](https://github.com/vegaprotocol/vega/issues/9374) - `ListGovernanceData` returns an error instead of results.
- [9344](https://github.com/vegaprotocol/vega/issues/9344) - Verify `EthTime` in submitted `ChainEvent` transactions matches the chain.
- [9461](https://github.com/vegaprotocol/vega/issues/9461) - Do not make SLA related transfers for 0 amount.
- [9462](https://github.com/vegaprotocol/vega/issues/9462) - Add missing proposal error `enum` values to the database.
- [9466](https://github.com/vegaprotocol/vega/issues/9466) - `ListReferralSets` returns error when there are no stats available.
- [9341](https://github.com/vegaprotocol/vega/issues/9341) - Fix checkpoint loading
- [9472](https://github.com/vegaprotocol/vega/issues/9472) - `ListTeams` returns error when filtering by referrer or team.
- [9477](https://github.com/vegaprotocol/vega/issues/947y) - Teams data not persisted to the database.
- [9412](https://github.com/vegaprotocol/vega/issues/9412) - Use vote close time for opening auction start time.
- [9487](https://github.com/vegaprotocol/vega/issues/9487) - Reset auction trigger appropriately when market is resumed via governance.
- [9489](https://github.com/vegaprotocol/vega/issues/9489) - A referrer cannot join another team.
- [9441](https://github.com/vegaprotocol/vega/issues/9441) - fix occasional double close of channel in integration tests
- [9074](https://github.com/vegaprotocol/vega/issues/9074) - Fix error response for `CSV` exports.
- [9512](https://github.com/vegaprotocol/vega/issues/9512) - Allow hysteresis period to be set to 0.
- [9526](https://github.com/vegaprotocol/vega/issues/9526) - Rename go enum value `REJECTION_REASON_STOP_ORDER_NOT_CLOSING_THE_POSITION` to match `db` schema
- [8979](https://github.com/vegaprotocol/vega/issues/8979) - Add trusted proxy config and verification for `XFF` header.
- [9530](https://github.com/vegaprotocol/vega/issues/9530) - Referral program end timestamp not correctly displaying in `GraphQL API`.
- [9532](https://github.com/vegaprotocol/vega/issues/9532) - Data node crashes if referral program starts and ends in the same block.
- [9540](https://github.com/vegaprotocol/vega/issues/9540) - Proposals connection errors for `UpdateReferralProgram`.
- [9570](https://github.com/vegaprotocol/vega/issues/9570) - Ledger entry export doesn't work if `dateRange.End` is specified
- [9571](https://github.com/vegaprotocol/vega/issues/9571) - Ledger entry `CSV` export slow even when looking at narrow time window
- [8934](https://github.com/vegaprotocol/vega/issues/8934) - Check for special characters in headers.
- [9609](https://github.com/vegaprotocol/vega/issues/9609) - Do not prompt for funding payments when terminating a perpetual market if we have already processed a trigger.
- [8979](https://github.com/vegaprotocol/vega/issues/8979) - Update `XFF` information in rate limiter documentation.
- [9540](https://github.com/vegaprotocol/vega/issues/9540) - Proposals connection does not resolve `UpdateReferralProgram` proposals correctly.
- [9580](https://github.com/vegaprotocol/vega/issues/9580) - Add program end time validation for referral proposals.
- [9588](https://github.com/vegaprotocol/vega/issues/9588) - Fix snapshot state for referral and volume discount programs.
- [9596](https://github.com/vegaprotocol/vega/issues/9596) - Empty positions and potential positions are excluded from win socialisation.
- [9619](https://github.com/vegaprotocol/vega/issues/9619) - Win socialisation on funding payments should include parties who haven't traded.
- [9624](https://github.com/vegaprotocol/vega/issues/9624) - Fix get returns when window is larger than available returns
- [9634](https://github.com/vegaprotocol/vega/issues/9634) - Do not allow to submit liquidity provision for pending LPs
- [9636](https://github.com/vegaprotocol/vega/issues/9636) - include ersatz validator for validator ranking reward.
- [9655](https://github.com/vegaprotocol/vega/issues/9655) - Make liquidity monitoring parameter required in market proposals validation
- [9280](https://github.com/vegaprotocol/vega/issues/9280) - Fix block height off by one issue.
- [9658](https://github.com/vegaprotocol/vega/issues/9658) - Fix `updateVolumeDiscountProgram` GraphQL resolver.
- [9672](https://github.com/vegaprotocol/vega/issues/9672) - Fix margin being non-zero on `PERPS`, add tests to ensure distressed parties are handled correctly
- [9280](https://github.com/vegaprotocol/vega/issues/9280) - Get block height directly from `blocks` table.
- [9787](https://github.com/vegaprotocol/vega/issues/9787) - Do not sort market input so that tracking of dispatch strategies are not disturbed.
- [9675](https://github.com/vegaprotocol/vega/issues/9675) - Fix snapshot issue with not applying `providersCalculationStep` at epoch start.
- [9693](https://github.com/vegaprotocol/vega/issues/9693) - Add missing validation for general account public key in governance transfer
- [9691](https://github.com/vegaprotocol/vega/issues/9691) - Refactor referral engine snapshot
- [8570](https://github.com/vegaprotocol/vega/issues/8570) - Ensure pagination doesn't trigger a sequential scan on block-explorer transactions table.
- [9704](https://github.com/vegaprotocol/vega/issues/9704) - Fix referral program snapshot
- [9705](https://github.com/vegaprotocol/vega/issues/9705) - Ensure vote events are sent in the same order.
- [9714](https://github.com/vegaprotocol/vega/issues/9714) - Missing data from the fee stats
- [9722](https://github.com/vegaprotocol/vega/issues/9722) - Add missing wiring for activity streak `gRPC`
- [9734](https://github.com/vegaprotocol/vega/issues/9734) - Fix creation of new account types for existing assets during migration
- [9731](https://github.com/vegaprotocol/vega/issues/9731) - Allow team rewards to apply to all teams.
- [9727](https://github.com/vegaprotocol/vega/issues/9727) - Initialize teams earlier to avoid panic.
- [9746](https://github.com/vegaprotocol/vega/issues/9746) - Fix handling of LP fees reward
- [9747](https://github.com/vegaprotocol/vega/issues/9747) - Return correct destination type
- [9541](https://github.com/vegaprotocol/vega/issues/9731) - Add filtering for party to the referral fees API.
- [9751](https://github.com/vegaprotocol/vega/issues/9751) - Make sure that LP fee party accounts exists.
- [9762](https://github.com/vegaprotocol/vega/issues/9762) - Referral fees API not filtering by party correctly.
- [9775](https://github.com/vegaprotocol/vega/issues/9775) - Do not pay discount if set is not eligible
- [9788](https://github.com/vegaprotocol/vega/issues/9788) - Fix transfer account validation.
- [9859](https://github.com/vegaprotocol/vega/issues/9859) - Cache nonces in `pow` engine to prevent reuse.
- [9544](https://github.com/vegaprotocol/vega/issues/9544) - Fix race in visor restart logic.
- [9797](https://github.com/vegaprotocol/vega/issues/9797) - Default pagination limits are not always correctly set.
- [9813](https://github.com/vegaprotocol/vega/issues/9813) - Add validation for team scope vs individual scope
- [8570](https://github.com/vegaprotocol/vega/issues/8570) - Make sure the filtering by `cmd_type` on block explorer doesn't result with a bad query planning.
- [9801](https://github.com/vegaprotocol/vega/issues/9801) - Set right numbers on volume discount in fees stats
- [9835](https://github.com/vegaprotocol/vega/issues/9835) - Fix panic batch market instruction.
- [9841](https://github.com/vegaprotocol/vega/issues/9841) - Release `postgres` connections after ledger export is complete
- [9833](https://github.com/vegaprotocol/vega/issues/9833) - Validate market exists before enacting governance transfer.
- [9725](https://github.com/vegaprotocol/vega/issues/9725) - List ledger entries `API` documentation and functionality.
- [9874](https://github.com/vegaprotocol/vega/issues/9874) - Use correct field reference when paging with `JSONB` fields.
- [9818](https://github.com/vegaprotocol/vega/issues/9818) - Correctly register generated rewards in fees statistics.
- [9850](https://github.com/vegaprotocol/vega/issues/9850) - Do not use default cursor when Before or After is defined.
- [9855](https://github.com/vegaprotocol/vega/issues/9855) - Use max governance transfer amount as a quantum multiplier
- [9853](https://github.com/vegaprotocol/vega/issues/9853) - Data node crashing due to duplicate key violation in paid liquidity fees table.
- [9843](https://github.com/vegaprotocol/vega/issues/9843) - Migration script `0039` down migration drops incorrect column.
- [9858](https://github.com/vegaprotocol/vega/issues/9858) - Fix missing where statement in list referral set referees.
- [9868](https://github.com/vegaprotocol/vega/issues/9868) - Avoid empty ledger entries from governance transfers for metric when there is no activity.
- [9871](https://github.com/vegaprotocol/vega/issues/9871) - Fix panic on snapshot after market termination following protocol upgrade.
- [8769](https://github.com/vegaprotocol/vega/issues/8769) - Support timestamp data from different oracle types.
- [9894](https://github.com/vegaprotocol/vega/issues/9894) - Reset distress party position in market activity tracker.
- [9895](https://github.com/vegaprotocol/vega/issues/9895) - Expose discounts on trades fees.
- [9900](https://github.com/vegaprotocol/vega/issues/9900) - Fix for duplicate liquidity provisions in the `API`.
- [9900](https://github.com/vegaprotocol/vega/issues/9900) - Fix for duplicate liquidity provisions in the `API`.
- [9911](https://github.com/vegaprotocol/vega/issues/9911) - Bind correct property in `GraphQL` resolver.
- [9913](https://github.com/vegaprotocol/vega/issues/9913) - Add missing properties in schema.
- [9916](https://github.com/vegaprotocol/vega/issues/9916) - Fix activity streak locked account until epoch.
- [9924](https://github.com/vegaprotocol/vega/issues/9924) - Referral set referees `API` should aggregate by number of epochs.
- [9922](https://github.com/vegaprotocol/vega/issues/9922) - Ensure the factors in referral program account for set eligibility.
- [10249](https://github.com/vegaprotocol/vega/issues/10249) - Upgrade `cometbft` to 0.38.2

## 0.72.1

### 🐛 Fixes

- [8709](https://github.com/vegaprotocol/vega/issues/8709) - Fix stop order enum in data node.
- [8713](https://github.com/vegaprotocol/vega/issues/8713) - Wallet name with upper-case letter is allowed, again
- [8698](https://github.com/vegaprotocol/vega/issues/8698) - Always set the `ToAccount` field when clearing fees.
- [8711](https://github.com/vegaprotocol/vega/issues/8711) - Fix market lineage trigger.
- [8737](https://github.com/vegaprotocol/vega/issues/8737) - Fix `GetStopOrder` errors when order has a state change.
- [8727](https://github.com/vegaprotocol/vega/issues/8727) - Clear parent market on checkpoint restore if the parent market was already succeeded.
- [8835](https://github.com/vegaprotocol/vega/issues/8835) - Spot snapshot fixes
- [9651](https://github.com/vegaprotocol/vega/issues/9651) - Park pegged orders if they lose their peg
- [9666](https://github.com/vegaprotocol/vega/issues/9666) - Fix balance cache refresh snapshot issue

## 0.72.0

### 🗑️ Deprecation

- [8280](https://github.com/vegaprotocol/vega/issues/8280) - Unused rewards related network parameters are now deprecated and will be removed

### 🛠 Improvements

- [8666](https://github.com/vegaprotocol/vega/issues/8666) - Fix `total notional value` database migrations deadlock.
- [8409](https://github.com/vegaprotocol/vega/issues/8409) - Add `total notional value` to the candles.
- [7684](https://github.com/vegaprotocol/vega/issues/7684) - Add filters for `Block Explorer` transactions `API` for multiple command types (inclusive and exclusive) and multiple parties
- [7592](https://github.com/vegaprotocol/vega/issues/7592) - Add `block` parameter to `epoch` query.
- [7906](https://github.com/vegaprotocol/vega/issues/7906) - Connection tokens on the wallet survive reboot.
- [8264](https://github.com/vegaprotocol/vega/issues/8264) - Add a command line on the wallet to locate the wallet files
- [8026](https://github.com/vegaprotocol/vega/issues/8026) - Update `UPGRADING.md document`
- [8283](https://github.com/vegaprotocol/vega/issues/8283) - Add disclaimer to the wallet `CLI`
- [8296](https://github.com/vegaprotocol/vega/issues/8296) - Improve error handling for invalid proposal validation timestamp
- [8318](https://github.com/vegaprotocol/vega/issues/8318) - Proto definitions for spots
- [8117](https://github.com/vegaprotocol/vega/issues/8117) - Added spots governance implementation
- [8259](https://github.com/vegaprotocol/vega/issues/8259) - Proto definitions for successor markets.
- [8201](https://github.com/vegaprotocol/vega/issues/8201) - Add support for successor markets.
- [8339](https://github.com/vegaprotocol/vega/issues/8339) - Target stake for spots
- [8337](https://github.com/vegaprotocol/vega/issues/8337) - ELS for spots
- [8359](https://github.com/vegaprotocol/vega/issues/8359) - Add proto definitions for iceberg orders
- [8592](https://github.com/vegaprotocol/vega/issues/8592) - Remove oracle data current state table
- [8398](https://github.com/vegaprotocol/vega/issues/8398) - Implement iceberg orders in data node
- [8361](https://github.com/vegaprotocol/vega/issues/8361) - Implement pegged iceberg orders
- [8301](https://github.com/vegaprotocol/vega/issues/8301) - Implement iceberg orders in core
- [8301](https://github.com/vegaprotocol/vega/issues/8301) - Implement iceberg orders in feature tests
- [8429](https://github.com/vegaprotocol/vega/issues/8429) - Implement iceberg orders in `graphQL`
- [8429](https://github.com/vegaprotocol/vega/issues/8429) - Implement iceberg orders during auction uncrossing
- [8524](https://github.com/vegaprotocol/vega/issues/8524) - Rename iceberg fields for clarity
- [8459](https://github.com/vegaprotocol/vega/issues/8459) - Market depth and book volume include iceberg reserves
- [8332](https://github.com/vegaprotocol/vega/issues/8332) - Add support in collateral engine for spots
- [8330](https://github.com/vegaprotocol/vega/issues/8330) - Implement validation on successor market proposals.
- [8247](https://github.com/vegaprotocol/vega/issues/8247) - Initial support for `Ethereum` `oracles`
- [8334](https://github.com/vegaprotocol/vega/issues/8334) - Implement market succession in execution engine.
- [8354](https://github.com/vegaprotocol/vega/issues/8354) - refactor execution package
- [8394](https://github.com/vegaprotocol/vega/issues/8394) - Get rid of spot liquidity provision commands and data structures.
- [8613](https://github.com/vegaprotocol/vega/issues/8613) - Pass context into witness resource check method
- [8402](https://github.com/vegaprotocol/vega/issues/8402) - Avoid division by 0 in market activity tracker
- [8347](https://github.com/vegaprotocol/vega/issues/8347) - Market state (`ELS`) to be included in checkpoint data.
- [8303](https://github.com/vegaprotocol/vega/issues/8303) - Add support for successor markets in datanode.
- [8118](https://github.com/vegaprotocol/vega/issues/8118) - Spot market execution
- [7416](https://github.com/vegaprotocol/vega/issues/7416) - Support for governance transfers
- [7701](https://github.com/vegaprotocol/vega/issues/7701) - Support parallel request on different party on the wallet API
- [8353](https://github.com/vegaprotocol/vega/issues/8353) - Improve ledger entry `CSV` export.
- [8445](https://github.com/vegaprotocol/vega/issues/8445) - Additional feature tests for iceberg orders.
- [8349](https://github.com/vegaprotocol/vega/issues/8349) - Add successor market integration test coverage.
- [8434](https://github.com/vegaprotocol/vega/issues/8434) - Add pagination for `ListSuccessorMarkets`.
- [8439](https://github.com/vegaprotocol/vega/issues/8439) - Include proposals for the `ListSuccessorMarkets API`.
- [8476](https://github.com/vegaprotocol/vega/issues/8476) - Add successor market per `AC`
- [8365](https://github.com/vegaprotocol/vega/issues/8365) - Add new liquidity engine with SLA support.
- [8466](https://github.com/vegaprotocol/vega/issues/8466) - Add stop orders protobufs and domain types
- [8467](https://github.com/vegaprotocol/vega/issues/8467) - Add stop orders data structures
- [8516](https://github.com/vegaprotocol/vega/issues/8516) - Add stop orders network parameter
- [9509](https://github.com/vegaprotocol/vega/issues/9509) - Markets can now be updated when they are in an opening auction.
- [8470](https://github.com/vegaprotocol/vega/issues/8470) - Stop orders snapshots
- [8548](https://github.com/vegaprotocol/vega/issues/8548) - Use default for tendermint `RPC` address and better validation for `semver`
- [8472](https://github.com/vegaprotocol/vega/issues/8472) - Add support for stop orders with batch market instructions
- [8567](https://github.com/vegaprotocol/vega/issues/8567) - Add virtual stake and market growth to market data.
- [8508](https://github.com/vegaprotocol/vega/issues/8508) - Add network parameters for SLA.
- [8468](https://github.com/vegaprotocol/vega/issues/8468) - Wire in stop orders in markets
- [8609](https://github.com/vegaprotocol/vega/issues/8609) - Add `graphQL` support for governance transfers
- [8528](https://github.com/vegaprotocol/vega/issues/8528) - Add support for Stop Orders in the data node.
- [8635](https://github.com/vegaprotocol/vega/issues/8635) - Allow market update proposal with ELS only
- [8675](https://github.com/vegaprotocol/vega/issues/8675) - Fix inconsistent naming for successor markets.
- [8504](https://github.com/vegaprotocol/vega/issues/8504) - Add market liquidity common layer for spot market.
- [8415](https://github.com/vegaprotocol/vega/issues/8415) - Allow to terminate, suspend/resume market via governance
- [8690](https://github.com/vegaprotocol/vega/issues/8690) - Add gas estimation for stop orders.
- [8505](https://github.com/vegaprotocol/vega/issues/8505) - Add snapshots to liquidity engine version 2.

### 🐛 Fixes

- [8236](https://github.com/vegaprotocol/vega/issues/8236) - Fix `orderById` `GraphQL` docs.
- [8208](https://github.com/vegaprotocol/vega/issues/8208) - Fix block explorer API documentation
- [8203](https://github.com/vegaprotocol/vega/issues/8203) - Fix `assetId` parsing for Ledger entries export to `CSV` file.
- [8251](https://github.com/vegaprotocol/vega/issues/8251) - Fix bug in expired orders optimisation resulting in non deterministic order sequence numbers
- [8226](https://github.com/vegaprotocol/vega/issues/8226) - Fix auto initialise failure when initialising empty node
- [8186](https://github.com/vegaprotocol/vega/issues/8186) - Set a close timestamp when closing a market
- [8206](https://github.com/vegaprotocol/vega/issues/8206) - Add number of decimal places to oracle spec.
- [8668](https://github.com/vegaprotocol/vega/issues/8668) - Market data migration scripts fix
- [8225](https://github.com/vegaprotocol/vega/issues/8225) - Better error handling in `ListEntities`
- [8222](https://github.com/vegaprotocol/vega/issues/8222) - `EstimatePositions` does not correctly validate data.
- [8357](https://github.com/vegaprotocol/vega/issues/8357) - Load network history segments into staging area prior to load if not already present
- [8266](https://github.com/vegaprotocol/vega/issues/8266) - Fix HTTPS with `autocert`.
- [8471](https://github.com/vegaprotocol/vega/issues/8471) - Restore network parameters from snapshot without validation to avoid order dependence.
- [8290](https://github.com/vegaprotocol/vega/issues/8290) - Calling network history `API` without network history enabled caused panics in data node.
- [8299](https://github.com/vegaprotocol/vega/issues/8299) - Fix listing of internal data sources in GraphQL.
- [8279](https://github.com/vegaprotocol/vega/issues/8279) - Avoid overriding a map entry while iterating on it, on the wallet connection manager.
- [8341](https://github.com/vegaprotocol/vega/issues/8341) - Remind the user to check his internet connection if the wallet can't connect to a node.
- [8343](https://github.com/vegaprotocol/vega/issues/8343) - Make the service starter easier to instantiate
- [8429](https://github.com/vegaprotocol/vega/issues/8429) - Release margin when decreasing iceberg size like normal orders do
- [8429](https://github.com/vegaprotocol/vega/issues/8429) - Set order status to stopped if an iceberg order instantly causes a wash trade
- [8376](https://github.com/vegaprotocol/vega/issues/8376) - Ensure the structure fields match their JSON counter parts in the wallet API requests and responses.
- [8363](https://github.com/vegaprotocol/vega/issues/8363) - Add missing name property in `admin.describe_key` wallet API example
- [8536](https://github.com/vegaprotocol/vega/issues/8536) - If liquidity fee account is empty do not create 0 amount transfers to insurance pool when clearing market
- [8313](https://github.com/vegaprotocol/vega/issues/8313) - Assure liquidation price estimate works with 0 open volume
- [8412](https://github.com/vegaprotocol/vega/issues/8412) - Fix non deterministic ordering of events emitted on auto delegation
- [8414](https://github.com/vegaprotocol/vega/issues/8414) - Fix corruption on order subscription
- [8453](https://github.com/vegaprotocol/vega/issues/8453) - Do not verify termination timestamp in update market when pre-enacting proposal
- [8418](https://github.com/vegaprotocol/vega/issues/8418) - Fix data node panics when a bad successor market proposal is rejected
- [8358](https://github.com/vegaprotocol/vega/issues/8358) - Fix replay protection
- [8655](https://github.com/vegaprotocol/vega/issues/8655) - Set market close timestamp when market closes
- [8362](https://github.com/vegaprotocol/vega/issues/8362) - Fix `PnL` flickering bug.
- [8565](https://github.com/vegaprotocol/vega/issues/8565) - Unsubscribe all data sources when restoring a settled market from a snapshot
- [8578](https://github.com/vegaprotocol/vega/issues/8578) - Add iceberg option fields to live orders trigger
- [8451](https://github.com/vegaprotocol/vega/issues/8451) - Fix invalid auction duration for new market proposals.
- [8500](https://github.com/vegaprotocol/vega/issues/8500) - Fix liquidity provision `ID` is nullable in `GraphQL API`.
- [8511](https://github.com/vegaprotocol/vega/issues/8511) - Include settled markets in the snapshots
- [8551](https://github.com/vegaprotocol/vega/issues/8551) - Reload market checkpoint data on snapshot loaded.
- [8486](https://github.com/vegaprotocol/vega/issues/8486) - Fix enactment timestamp being lost in checkpoints.
- [8572](https://github.com/vegaprotocol/vega/issues/8572) - Fix governance fraction validation
- [8618](https://github.com/vegaprotocol/vega/issues/8618) - Add iceberg fields to GraphQL `OrderUpdate`
- [8694](https://github.com/vegaprotocol/vega/issues/8694) - Properly decrement by difference when amending iceberg order
- [8580](https://github.com/vegaprotocol/vega/issues/8580) - Fix wallet `CLI` ignoring max-request-duration
- [8583](https://github.com/vegaprotocol/vega/issues/8583) - Fix validation of ineffectual transfer
- [8586](https://github.com/vegaprotocol/vega/issues/8586) - Fix cancel governance transfer proposal validation
- [8597](https://github.com/vegaprotocol/vega/issues/8597) - Enact governance transfer cancellation
- [8428](https://github.com/vegaprotocol/vega/issues/8428) - Add missing `LastTradedPrice` field in market data
- [8335](https://github.com/vegaprotocol/vega/issues/8335) - Validate asset for metrics in transfers to be an actual asset
- [8603](https://github.com/vegaprotocol/vega/issues/8603) - Restore `lastTradedPrice` of `nil` as `nil` in market snapshot
- [8617](https://github.com/vegaprotocol/vega/issues/8617) - Fix panic with order gauge in future market
- [8596](https://github.com/vegaprotocol/vega/issues/8596) - Fix panic when rejecting markets on time update.
- [8545](https://github.com/vegaprotocol/vega/issues/6545) - Block explorer does not page backwards correctly.
- [8654](https://github.com/vegaprotocol/vega/issues/8654) - `GraphQL` query on trades with no filters returns an error.
- [8623](https://github.com/vegaprotocol/vega/issues/8623) - Send market data event when a market is rejected.
- [8642](https://github.com/vegaprotocol/vega/issues/8642) - Restore successor maps from snapshot.
- [8636](https://github.com/vegaprotocol/vega/issues/8636) - Trading mode in market data events should be set to `NO_TRADING` if the market is in a final state.
- [8651](https://github.com/vegaprotocol/vega/issues/8651) - Wallet support for stop orders
- [8630](https://github.com/vegaprotocol/vega/issues/8630) - Fix duplicate stake linking due to `re-org`
- [8664](https://github.com/vegaprotocol/vega/issues/8664) - Stop order invalid expiry
- [8643](https://github.com/vegaprotocol/vega/issues/8643) - Handle vote close for pending successor markets.
- [8665](https://github.com/vegaprotocol/vega/issues/8665) - `Datanode` not persisting stop order events
- [8702](https://github.com/vegaprotocol/vega/issues/8702) - Fix panic on auction exit after stop orders expired in auction
- [8703](https://github.com/vegaprotocol/vega/issues/8703) - Wire stop orders cancellation
- [8698](https://github.com/vegaprotocol/vega/issues/8698) - Always set the `ToAccount` field when clearing fees.
- [9773](https://github.com/vegaprotocol/vega/issues/9773) - Ensure invalid market for transfer doesn't panic
- [9777](https://github.com/vegaprotocol/vega/issues/9777) - Ensure new account types are created for existing assets

## 0.71.0

### 🚨 Breaking changes

- [7859](https://github.com/vegaprotocol/vega/issues/7859) - Fix Ledger entries exporting `CSV` file.
- [8064](https://github.com/vegaprotocol/vega/issues/8064) - Remove `websocket` for rewards
- [8093](https://github.com/vegaprotocol/vega/issues/8093) - Remove offset pagination
- [8111](https://github.com/vegaprotocol/vega/issues/8111) - Unify payload between `admin.update_network` and `admin.describe_network` endpoint in the wallet API.
- [7916](https://github.com/vegaprotocol/vega/issues/7916) - Deprecated `TradesConnection GraphQL sub-queries` in favour of an `un-nested` Trades query with a filter parameter. This requires a change in the underlying `gRPC` request message. Trades subscription takes a `TradesSubscriptionFilter` that allows multiple `MarketID` and `PartyID` filters to be specified.
- [8143](https://github.com/vegaprotocol/vega/issues/8143) - Merge GraphQL and REST servers
- [8111](https://github.com/vegaprotocol/vega/issues/8111) - Reduce passphrase requests for admin endpoints by introducing `admin.unlock_wallet` and removing the `passphrase` field from wallet-related endpoints.

### 🛠 Improvements

- [8030](https://github.com/vegaprotocol/vega/issues/8030) - Add `API` for fetching `CSV` data from network history.
- [7943](https://github.com/vegaprotocol/vega/issues/7943) - Add version to network file to be future-proof.
- [7759](https://github.com/vegaprotocol/vega/issues/7759) - Support for rolling back data node to a previous network history segment
- [8131](https://github.com/vegaprotocol/vega/issues/8131) - Add reset all command to data node and remove wipe on start up flags
- [7505](https://github.com/vegaprotocol/vega/issues/7505) - `Datanode` batcher statistics
- [8045](https://github.com/vegaprotocol/vega/issues/8045) - Fix bug in handling internal sources data.
- [7843](https://github.com/vegaprotocol/vega/issues/7843) - Report partial batch market instruction processing failure
- [7990](https://github.com/vegaprotocol/vega/issues/7990) - Remove reference to `postgres` in the `protobuf` documentation comments
- [7992](https://github.com/vegaprotocol/vega/issues/7992) - Improve Candles related `APIs`
- [7986](https://github.com/vegaprotocol/vega/issues/7986) - Remove cross `protobuf` files documentation references
- [8146](https://github.com/vegaprotocol/vega/issues/8146) - Add fetch retry behaviour to network history fetch command
- [7982](https://github.com/vegaprotocol/vega/issues/7982) - Fix behaviour of endpoints with `marketIds` and `partyIds` filters
- [7846](https://github.com/vegaprotocol/vega/issues/7846) - Add event indicating distressed parties that are still holding an active position.
- [7985](https://github.com/vegaprotocol/vega/issues/7985) - Add full stop on all fields documentation to get it properly generated
- [8024](https://github.com/vegaprotocol/vega/issues/8024) - Unify naming in `rpc` endpoints and add tags
- [7989](https://github.com/vegaprotocol/vega/issues/7989) - Remove reference to cursor based pagination in `rpc` documentations
- [7991](https://github.com/vegaprotocol/vega/issues/7991) - Improve `EstimateFees` documentation
- [7108](https://github.com/vegaprotocol/vega/issues/7108) - Annotate required fields in `API` requests.
- [8039](https://github.com/vegaprotocol/vega/issues/8039) - Write network history segments in the `datanode` process instead of requesting `postgres` to write them.
- [7987](https://github.com/vegaprotocol/vega/issues/7987) - Make terms consistent in `API` documentation.
- [8025](https://github.com/vegaprotocol/vega/issues/8025) - Address inconsistent verb and grammar in the `API` documentation.
- [7999](https://github.com/vegaprotocol/vega/issues/7999) - Review `DateRange API` documentation.
- [7955](https://github.com/vegaprotocol/vega/issues/7955) - Ensure the wallet API documentation matches the Go definitions
- [8023](https://github.com/vegaprotocol/vega/issues/8023) - Made pagination `docstrings` consistent.
- [8105](https://github.com/vegaprotocol/vega/issues/8105) - Make candles return in ascending order when queried from `graphql`.
- [8144](https://github.com/vegaprotocol/vega/issues/8144) - Visor - remove data node asset option from the config. Use only one asset.
- [8000](https://github.com/vegaprotocol/vega/issues/8000) - Add documentation for `Pagination` `protobuf` message.
- [7969](https://github.com/vegaprotocol/vega/issues/7969) - Add `GoodForBlocks` field to transaction input data.
- [8155](https://github.com/vegaprotocol/vega/issues/8155) - Visor - allow restart without snapshot.
- [8129](https://github.com/vegaprotocol/vega/issues/8129) - Keep liquidity fee remainder in fee account.
- [8022](https://github.com/vegaprotocol/vega/issues/8022) - Improve `ListTransfers` API documentation.
- [8231](https://github.com/vegaprotocol/vega/issues/8231) - Fix `GetNetworkHistoryStatus`
- [8154](https://github.com/vegaprotocol/vega/issues/8154) - Visor - added option for delaying stop of binaries.
- [8169](https://github.com/vegaprotocol/vega/issues/8169) - Add `buf` format
- [7997](https://github.com/vegaprotocol/vega/issues/7997) - Clean up `API` comments when returned value is signed/unsigned.
- [7988](https://github.com/vegaprotocol/vega/issues/7988) - Make information about numbers expressed as strings more clear.
- [7998](https://github.com/vegaprotocol/vega/issues/7998) - Clean up `API` documentation for `ListLedgerEntries`.
- [8021](https://github.com/vegaprotocol/vega/issues/8021) - Add better field descriptions in the `API` documentation.
- [8171](https://github.com/vegaprotocol/vega/issues/8171) - Optimise the way offsets are used in probability of trading.
- [8194](https://github.com/vegaprotocol/vega/issues/8194) - Don't include query string as part of `Prometheus` metric labels
- [7847](https://github.com/vegaprotocol/vega/issues/7847) - Add `EstimatePosition` `API` method, mark `EstimateOrder` (GraphQL) and `EstimateMargin` (gRPC) as deprecated.
- [7969](https://github.com/vegaprotocol/vega/issues/7969) - Reverted the `TTL` changes, minor tweak to proof of work verification to ensure validator commands can't be rejected based on age.
- [7926](https://github.com/vegaprotocol/vega/issues/7926) - Squash `SQL` migration scripts into a single script.

### 🐛 Fixes

- [7938](https://github.com/vegaprotocol/vega/issues/7938) - Attempt to fix protocol upgrade failure because of `LevelDB` file lock issue
- [7944](https://github.com/vegaprotocol/vega/issues/7944) - Better error message if we fail to parse the network configuration in wallet
- [7870](https://github.com/vegaprotocol/vega/issues/7870) - Fix `LP` subscription filters
- [8159](https://github.com/vegaprotocol/vega/issues/8159) - Remove corresponding network history segments on rollback
- [7954](https://github.com/vegaprotocol/vega/issues/7954) - Don't error if subscribing to a market/party that has no position yet
- [7899](https://github.com/vegaprotocol/vega/issues/7899) - Fixes inconsistency in the `HTTP` status codes returned when rate limited
- [7968](https://github.com/vegaprotocol/vega/issues/7968) - Ready for protocol upgrade flag set without going through memory barrier
- [7962](https://github.com/vegaprotocol/vega/issues/7962) - Set `isValidator` when loading from a checkpoint
- [7950](https://github.com/vegaprotocol/vega/issues/7950) - Fix the restore of deposits from checkpoint
- [7933](https://github.com/vegaprotocol/vega/issues/7933) - Ensure the wallet store is closed to avoid "too many opened files" error
- [8069](https://github.com/vegaprotocol/vega/issues/8069) - Handle zero return value for memory when setting IPFS resource limits
- [7956](https://github.com/vegaprotocol/vega/issues/7956) - Floor negative slippage per unit at 0
- [7964](https://github.com/vegaprotocol/vega/issues/7964) - Use mark price for all margin calculations
- [8003](https://github.com/vegaprotocol/vega/issues/8003) - Fix `ListGovernanceData` does not honour `TYPE_ALL`
- [8057](https://github.com/vegaprotocol/vega/issues/8057) - Load history and current state in one transaction
- [8058](https://github.com/vegaprotocol/vega/issues/8058) - Continuous aggregates should be updated according to the watermark and span of history loaded
- [8001](https://github.com/vegaprotocol/vega/issues/8001) - Fix issues with order subscriptions
- [7980](https://github.com/vegaprotocol/vega/issues/7980) - Visor - prevent panic when auto install configuration is missing assets
- [7995](https://github.com/vegaprotocol/vega/issues/7995) - Validate order price input to `estimateFee` and `estimateMargin`
- [8011](https://github.com/vegaprotocol/vega/issues/8011) - Return a not found error for an invalid network parameter key for the API
- [8012](https://github.com/vegaprotocol/vega/issues/8012) - Ensure client do not specify both a before and after cursor
- [8017](https://github.com/vegaprotocol/vega/issues/8017) - Return an error when requesting order with negative version
- [8020](https://github.com/vegaprotocol/vega/issues/8020) - Update default `tendermint` home path to `cometbft`
- [7919](https://github.com/vegaprotocol/vega/issues/7919) - Avoid sending empty ledger movements
- [8053](https://github.com/vegaprotocol/vega/issues/8053) - Fix notary vote count
- [8004](https://github.com/vegaprotocol/vega/issues/8004) - Validate signatures exist in announce node command
- [8004](https://github.com/vegaprotocol/vega/issues/8004) - Validate value in state variable bundles
- [8004](https://github.com/vegaprotocol/vega/issues/8004) - Validate Ethereum addresses and add a cap on node vote reference length
- [8046](https://github.com/vegaprotocol/vega/issues/8046) - Update GraphQL schema with new order rejection reason
- [6659](https://github.com/vegaprotocol/vega/issues/6659) - Wallet application configuration is correctly reported on default location
- [8074](https://github.com/vegaprotocol/vega/issues/8074) - Add missing order rejection reason to `graphql` schema
- [8090](https://github.com/vegaprotocol/vega/issues/8090) - Rename network history `APIs` that did not follow the naming convention
- [8060](https://github.com/vegaprotocol/vega/issues/8060) - Allow 0 decimals assets
- [7993](https://github.com/vegaprotocol/vega/issues/7993) - Fix `ListDeposits` endpoint and documentation
- [8072](https://github.com/vegaprotocol/vega/issues/8072) - Fix `panics` in estimate orders
- [8125](https://github.com/vegaprotocol/vega/issues/8125) - Ensure network compatibility can be checked against TLS nodes
- [8103](https://github.com/vegaprotocol/vega/issues/8103) - Fix incorrect rate limiting behaviour on `gRPC` `API`
- [8128](https://github.com/vegaprotocol/vega/issues/8128) - Assure price monitoring engine extends the auction one bound at a time
- [8149](https://github.com/vegaprotocol/vega/issues/8149) - Trigger populating `orders_live` table out of date and does not filter correctly for live orders.
- [8165](https://github.com/vegaprotocol/vega/issues/8165) - Send order events when an `lp` order is cancelled or rejected
- [8173](https://github.com/vegaprotocol/vega/issues/8173) - Trades when leaving auction should should have the aggressor field set to `SideUnspecified`.
- [8184](https://github.com/vegaprotocol/vega/issues/8184) - Handle case for time termination value used with `LessThan` condition.
- [8157](https://github.com/vegaprotocol/vega/issues/8157) - Handle kill/interrupt signals in datanode, and clean up properly.
- [7914](https://github.com/vegaprotocol/vega/issues/7914) - Offer node signatures after snapshot restore
- [8187](https://github.com/vegaprotocol/vega/issues/8187) - Expose Live Only filter to the `GraphQL` Orders filter.
- [9793](https://github.com/vegaprotocol/vega/issues/9793) - Map network owner correctly in creating account from transfer.
- [10516](https://github.com/vegaprotocol/vega/issues/10516) - Fix mapping of estimate position.

## 0.70.0

### 🚨 Breaking changes

- [7794](https://github.com/vegaprotocol/vega/issues/7794) - Rename `marketId` and `partyId` to `marketIds` and `partyIds` in the orders queries' filter.
- [7876](https://github.com/vegaprotocol/vega/issues/7876) - Change `DeliverOn` on one-off transfer to be in nanoseconds as everything else.
- [7326](https://github.com/vegaprotocol/vega/issues/7326) - Rename table `current liquidity provisions` to `live liquiditiy provisions` and add a `live` option

### 🛠 Improvements

- [7862](https://github.com/vegaprotocol/vega/issues/7862) - Add per table statistics for network history segment creation
- [7834](https://github.com/vegaprotocol/vega/issues/7834) - Support TLS connection for gRPC endpoints in wallet when prefixed with `tls://`
- [7851](https://github.com/vegaprotocol/vega/issues/7851) - Implement post only and reduce only orders
- [7768](https://github.com/vegaprotocol/vega/issues/7768) - Set sensible defaults for the IPFS resource manager
- [7863](https://github.com/vegaprotocol/vega/issues/7863) - Rework positions indexes so that snapshot creation does not slow down
- [7829](https://github.com/vegaprotocol/vega/issues/7829) - Get precision for reference price from price monitoring bounds when getting market data
- [7670](https://github.com/vegaprotocol/vega/issues/7670) - Removes the need for the buffered event source to hold a large buffer of sequence numbers
- [7904](https://github.com/vegaprotocol/vega/issues/7904) - Add a default system test template for integration tests
- [7894](https://github.com/vegaprotocol/vega/issues/7894) - Use slippage cap when market is in auction mode
- [7923](https://github.com/vegaprotocol/vega/issues/7923) - Subscription rate limiter is enabled on `gRPC` and `REST` subscriptions

### 🐛 Fixes

- [7910](https://github.com/vegaprotocol/vega/issues/7910) - Store heartbeats in the checkpoint so that validator sets do not reorder unexpectedly after loading
- [7835](https://github.com/vegaprotocol/vega/issues/7835) - Ensure the command errors have the same format on arrays
- [7871](https://github.com/vegaprotocol/vega/issues/7871) - Bad `SQL` generated when paginating reward summaries
- [7908](https://github.com/vegaprotocol/vega/issues/7908) - Expired `heartbeats` no longer invalidate subsequent `heartbeats`
- [7880](https://github.com/vegaprotocol/vega/issues/7880) - Update volume-weighted average price party's of open orders after a trade
- [7883](https://github.com/vegaprotocol/vega/issues/7883) - Fix snapshot issue with witness on accounting
- [7921](https://github.com/vegaprotocol/vega/issues/7921) - Fix streams batches
- [7895](https://github.com/vegaprotocol/vega/issues/7895) - Fix margin calculation during auction
- [7940](https://github.com/vegaprotocol/vega/issues/7940) - Enhance validation of tendermint public keys
- [7930](https://github.com/vegaprotocol/vega/issues/7930) - Fix typo in the `Vegavisor` configuration and improve Visor binaries runner logging
- [7981](https://github.com/vegaprotocol/vega/issues/7981) - Ensure LP order events are not sent when nothing changes

## 0.69.0

### 🚨 Breaking changes

- [7798](https://github.com/vegaprotocol/vega/issues/7798) - Remove redundant headers from the rate limiter response.
- [7710](https://github.com/vegaprotocol/vega/issues/7710) - Rename "token dApp" to "governance"
- [6905](https://github.com/vegaprotocol/vega/issues/6905) - Deprecated `Version` field removed from `admin.import_wallet`
- [6905](https://github.com/vegaprotocol/vega/issues/6905) - References to file paths have been removed from `admin.import_wallet`, `admin.import_network`, `admin.create_wallet` and `admin.isolate_key` API
- [7731](https://github.com/vegaprotocol/vega/issues/7731) - Upgrade the interplanetary file system library to latest release
- [7802](https://github.com/vegaprotocol/vega/issues/7802) - Split liquidity auction trigger into two cases
- [7728](https://github.com/vegaprotocol/vega/issues/7728) - Remove current order flag from table - adds restrictions to how orders can be paged
- [7816](https://github.com/vegaprotocol/vega/issues/7816) - Require slippage factors to always be set

### 🛠 Improvements

- [6942](https://github.com/vegaprotocol/vega/issues/6942) - Add `admin.rename_network` with `vega wallet network rename`
- [7656](https://github.com/vegaprotocol/vega/issues/7656) - Add `vega wallet service config locate` CLI that returns the location of the service configuration file.
- [7656](https://github.com/vegaprotocol/vega/issues/7656) - Add `vega wallet service config describe` CLI that display the service configuration.
- [7656](https://github.com/vegaprotocol/vega/issues/7656) - Add `vega wallet service config reset` CLI that reset the service configuration to its default state.
- [7681](https://github.com/vegaprotocol/vega/issues/7681) - Remove unnecessary `protobuf` marshalling in event pipeline
- [7288](https://github.com/vegaprotocol/vega/issues/7288) - Add `block` interval for trade candles
- [7696](https://github.com/vegaprotocol/vega/issues/7696) - Cache `ListMarket` store queries
- [7532](https://github.com/vegaprotocol/vega/issues/7532) - Load network history in a transaction
- [7413](https://github.com/vegaprotocol/vega/issues/7413) - Add foreign block height to stake linkings in `GraphQL`
- [7675](https://github.com/vegaprotocol/vega/issues/7675) - Migrate to comet `bft`
- [7792](https://github.com/vegaprotocol/vega/issues/7792) - An attempt to import a network when the `url` is to `github` and not the raw file contents is caught early with a suggested `url`
- [7722](https://github.com/vegaprotocol/vega/issues/7722) - Send a reason for a passphrase request through the wallet's `interactor`
- [5967](https://github.com/vegaprotocol/vega/issues/5967) - Do not ask for wallet passphrase if it has already been unlocked.
- [5967](https://github.com/vegaprotocol/vega/issues/5967) - Preselect the wallet during connection is there is only one.
- [7723](https://github.com/vegaprotocol/vega/issues/7723) - Make the `SessionBegan` interaction easy to identify using a `WorkflowType`
- [7724](https://github.com/vegaprotocol/vega/issues/7724) - Add steps number to interactions to convey a progression feeling.
- [7353](https://github.com/vegaprotocol/vega/issues/7353) - Improve query setting current orders to only the most recent row after snapshot restore.
- [7763](https://github.com/vegaprotocol/vega/issues/7763) - Remove separate LP close out code path.
- [7686](https://github.com/vegaprotocol/vega/issues/7686) - Network History load will retry when IPFS cannot connect to peers.
- [7804](https://github.com/vegaprotocol/vega/issues/7804) - Headers include `Retry-After` when banned for exceeding rate limit.
- [7840](https://github.com/vegaprotocol/vega/issues/7840) - Make chunk time interval configurable.

### 🐛 Fixes

- [7688](https://github.com/vegaprotocol/vega/issues/7688) - Fix `BlockExplorer` case insensitive transaction retrieval.
- [7695](https://github.com/vegaprotocol/vega/issues/7695) - Fix `create_hypertable` in migrations.
- [7596](https://github.com/vegaprotocol/vega/issues/7596) - Slippage factors not persisted in database
- [7535](https://github.com/vegaprotocol/vega/issues/7535) - Fix network history load takes an increasingly long time to complete
- [7517](https://github.com/vegaprotocol/vega/issues/7517) - Add buffer files event source
- [7720](https://github.com/vegaprotocol/vega/issues/7720) - Return an empty slice instead of nil when describing a wallet network
- [7517](https://github.com/vegaprotocol/vega/issues/7517) - Add buffer files event source
- [7659](https://github.com/vegaprotocol/vega/issues/7659) - Tidy up REST documentation for consistency
- [7563](https://github.com/vegaprotocol/vega/issues/7563) - Let the wallet work again with null `blockchain`
- [7692](https://github.com/vegaprotocol/vega/issues/7692) - Fix network history load hanging after protocol upgrade
- [7751](https://github.com/vegaprotocol/vega/issues/7751) - Store the block height of the last seen `ERC20` event in the snapshot so deposits are not lost when the network is down
- [7778](https://github.com/vegaprotocol/vega/issues/7778) - Store the block height of the last seen `ERC20` event in the checkpoint so deposits are not lost when the network is down
- [7713](https://github.com/vegaprotocol/vega/issues/7713) - Fix PnL values on trade in the positions API
- [7726](https://github.com/vegaprotocol/vega/issues/7726) - Add market data current state table to ensure node restored from network history has latest market data
- [7673](https://github.com/vegaprotocol/vega/issues/7673) - Accept internal data sources without signers
- [7483](https://github.com/vegaprotocol/vega/issues/7483) - Fix market data history returning 0 values for price monitoring bounds
- [7732](https://github.com/vegaprotocol/vega/issues/7732) - Fix panic when amending orders
- [7588](https://github.com/vegaprotocol/vega/issues/7588) - Fix margin calculations when missing exit price
- [7766](https://github.com/vegaprotocol/vega/issues/7766) - Fix orders from new parties not being included in the nearest MTM
- [7499](https://github.com/vegaprotocol/vega/issues/7499) - Implement transaction check functionality to wallet
- [7745](https://github.com/vegaprotocol/vega/issues/7745) - Use margin after the application of a bond penalty to assess LP solvency
- [7765](https://github.com/vegaprotocol/vega/issues/7765) - Assure pegged order won't get deployed with insufficient margin
- [7786](https://github.com/vegaprotocol/vega/issues/7786) - Fix validation of order amendments (check for negative pegged offset)
- [7750](https://github.com/vegaprotocol/vega/issues/7750) - Fix not all paths cleanly close network history index store.
- [7805](https://github.com/vegaprotocol/vega/issues/7805) - Fix re-announcing node in the same epoch kills data node.
- [7820](https://github.com/vegaprotocol/vega/issues/7820) - Remove the check for past date in limits engine
- [7822](https://github.com/vegaprotocol/vega/issues/7822) - Fix get last epoch query
- [7823](https://github.com/vegaprotocol/vega/issues/7823) - Fix validation of liquidity provisions shapes references

## 0.68.0

### 🚨 Breaking changes

- [7304](https://github.com/vegaprotocol/vega/issues/7304) - In the `datanode` `GraphQL` schema, move `fromEpoch` and `toEpoch` into a new `filter` for `epochRewardSummaries` query. Also add `assetIds` and `marketIds` to the same filter.
- [7419](https://github.com/vegaprotocol/vega/issues/7419) - Remove the deprecated headers with the `Grpc-Metadata-` prefix in `datanode` `API` and `REST` and `GraphQL` gateways.
- [6963](https://github.com/vegaprotocol/vega/issues/6963) - Remove the legacy fields from network API
- [7361](https://github.com/vegaprotocol/vega/issues/7361) - Network history loading and current order set tracking - database requires database to be dropped
- [6963](https://github.com/vegaprotocol/vega/issues/7382) - `IssueSignatures` is no longer a validator command and is now protected by the spam engine
- [7445](https://github.com/vegaprotocol/vega/issues/7445) - Added rate limiting to `GRPC`, `Rest` and `GraphQL` `APIs`
- [7614](https://github.com/vegaprotocol/vega/issues/7614) - Market parties added to snapshot state to ensure proper restoration
- [7542](https://github.com/vegaprotocol/vega/issues/7542) - Add optional slippage factors to market proposal and use them to cap slippage component of maintenance margin

### 🗑️ Deprecation

- [7385](https://github.com/vegaprotocol/vega/issues/7385) - Deprecating the `X-Vega-Connection` HTTP header in `datanode` `API` and `REST` and `GraphQL` gateways.

### 🛠 Improvements

- [7501](https://github.com/vegaprotocol/vega/issues/7501) - Make logs more clear
- [7555](https://github.com/vegaprotocol/vega/issues/7555) - Clean up code, add missing metrics and comments
- [7477](https://github.com/vegaprotocol/vega/issues/7477) - Improve `gRPC` service error handling and formatting
- [7386](https://github.com/vegaprotocol/vega/issues/7386) - Add indexed filtering by command type to block explorer
- [6962](https://github.com/vegaprotocol/vega/issues/6962) - Add a dedicated configuration for the wallet service
- [7434](https://github.com/vegaprotocol/vega/issues/7434) - Update design architecture diagram
- [7517](https://github.com/vegaprotocol/vega/issues/7517) - Archive and roll event buffer files
- [7429](https://github.com/vegaprotocol/vega/issues/7429) - Do not mark wallet and network as incompatible when the patch version doesn't match
- [6650](https://github.com/vegaprotocol/vega/issues/6650) - Add ability to filter rewards with `fromEpoch` and `toEpoch`
- [7429](https://github.com/vegaprotocol/vega/issues/7359) - `vega wallet` will not send in a transaction if it will result in a party becoming banned
- [7289](https://github.com/vegaprotocol/vega/issues/7289) - `positionsConnection` query added to `GraphQL`root query with filter for multiple parties and markets
- [7454](https://github.com/vegaprotocol/vega/issues/7454) - Retention policies for new types do not honour the `lite` or `archive` when added after `init`
- [7469](https://github.com/vegaprotocol/vega/issues/7469) - Sanitize `Prometheus` labels for `HTTP API` requests
- [7495](https://github.com/vegaprotocol/vega/issues/7495) - Upgrade `tendermint` to 0.34.25
- [7496](https://github.com/vegaprotocol/vega/issues/7496) - Enforce using priority `mempool` and max packet size in `tendermint config`
- [5987](https://github.com/vegaprotocol/vega/issues/5987) - Pick up the wallet changes when the service is started
- [7450](https://github.com/vegaprotocol/vega/issues/7450) - Positions API reporting close-out information and loss socialisation data.
- [7538](https://github.com/vegaprotocol/vega/issues/7538) - Add node information to the wallet response when sending the transaction
- [7550](https://github.com/vegaprotocol/vega/issues/7550) - Update feature tests to use specify explicitly linear and quadratic slippage factors
- [7558](https://github.com/vegaprotocol/vega/issues/7558) - Add `hypertable` for rewards
- [7509](https://github.com/vegaprotocol/vega/issues/7509) - Automatically reconcile account balance changes with transfer events after each integration test step
- [7564](https://github.com/vegaprotocol/vega/issues/7564) - Add logging when database migrations are run
- [7546](https://github.com/vegaprotocol/vega/issues/7546) - Visor automatically uses snapshot on core based on latest data node snapshot.
- [7576](https://github.com/vegaprotocol/vega/issues/7576) - include the application version in the block hash
- [7605](https://github.com/vegaprotocol/vega/issues/7605) - Return better error text when the wallet blocks a transaction due to spam rules
- [7591](https://github.com/vegaprotocol/vega/issues/7591) - Add metadata and links to app to the network configuration
- [7632](https://github.com/vegaprotocol/vega/issues/7632) - Make the wallet change events JSON friendly
- [7601](https://github.com/vegaprotocol/vega/issues/7601) - introduce the expired orders event for optimisation.
- [7655](https://github.com/vegaprotocol/vega/issues/7655) - Require initial margin level to be met on new orders

### 🐛 Fixes

- [7422](https://github.com/vegaprotocol/vega/issues/7422) - Fix missing `priceMonitoringParameters` and `liquidityMonitoringParameters` in `GraphQL` schema
- [7462](https://github.com/vegaprotocol/vega/issues/7462) - Fix `BlockExplorer` `API` not returning details on transactions.
- [7407](https://github.com/vegaprotocol/vega/issues/7407) - fix `ethereum` timestamp in stake linking in `graphql`
- [7494](https://github.com/vegaprotocol/vega/issues/7494) - fix memory leak in event bus stream subscriber when consumer is slow
- [7420](https://github.com/vegaprotocol/vega/issues/7420) - `clearFeeActivity` now clears fee activity
- [7420](https://github.com/vegaprotocol/vega/issues/7420) - set seed nonce for joining and leaving signatures during begin block
- [7420](https://github.com/vegaprotocol/vega/issues/7515) - protect `vegawallet` with recovers to shield against file system oddities
- [7399](https://github.com/vegaprotocol/vega/issues/7399) - Fix issue where market cache not working after restoring from network history
- [7410](https://github.com/vegaprotocol/vega/issues/7410) - Return underlying error when parsing a command failed in the wallet API version 2
- [7169](https://github.com/vegaprotocol/vega/issues/7169) - Fix migration, account for existing position data
- [7427](https://github.com/vegaprotocol/vega/issues/7427) - Fix nil pointer panic on settlement of restored markets.
- [7438](https://github.com/vegaprotocol/vega/issues/7438) - Update JSON-RPC documentation with all wallet errors
- [7451](https://github.com/vegaprotocol/vega/issues/7451) - Fix floating point consensus to use voting power rather than node count
- [7399](https://github.com/vegaprotocol/vega/issues/7399) - Revert previous fix
- [7399](https://github.com/vegaprotocol/vega/issues/7399) - Add option to filter out settled markets when listing markets in `API` requests
- [7559](https://github.com/vegaprotocol/vega/issues/7559) - Workaround `leveldb` issue and open `db` in write mode when listing using `vega tools snapshot`
- [7417](https://github.com/vegaprotocol/vega/issues/7417) - Missing entries in default data retention configuration for `datanode`
- [7504](https://github.com/vegaprotocol/vega/issues/7504) - Fixed panic in collateral engine when trying to clear a market
- [7468](https://github.com/vegaprotocol/vega/issues/7468) - `Datanode` network history load command only prompts when run from a terminal
- [7164](https://github.com/vegaprotocol/vega/issues/7164) - The command `vega wallet transaction send` now returns verbose errors
- [7514](https://github.com/vegaprotocol/vega/issues/7514) - Network names cannot contain `/`, `\` or start with a `.`
- [7519](https://github.com/vegaprotocol/vega/issues/7519) - Fix memory leak and increased CPU usage when streaming data.
- [7536](https://github.com/vegaprotocol/vega/issues/7536) - Ensure all errors are displayed when the wallet service cannot bind
- [7540](https://github.com/vegaprotocol/vega/issues/7540) - Prevent the double appending of the `http` scheme when ensuring port binding
- [7549](https://github.com/vegaprotocol/vega/issues/7549) - Switch proof-of-work ban error from an internal error to an application error on the wallet API
- [7543](https://github.com/vegaprotocol/vega/issues/7543) - Initiate post-auction close out only when all the parked orders get redeployed
- [7508](https://github.com/vegaprotocol/vega/issues/7508) - Assure transfer events always sent after margin recheck
- [7492](https://github.com/vegaprotocol/vega/issues/7492) - Send market depth events at the end of each block
- [7582](https://github.com/vegaprotocol/vega/issues/7582) - Validate transfer amount in `checkTx`
- [7582](https://github.com/vegaprotocol/vega/issues/7625) - Add validation to wallet's server configuration
- [7577](https://github.com/vegaprotocol/vega/issues/7577) - Use correct trade size when calculating pending open volume
- [7598](https://github.com/vegaprotocol/vega/issues/7598) - Set up log for rate limiter
- [7629](https://github.com/vegaprotocol/vega/issues/7629) - Handle error from `e.initialiseTree()` in the snapshot engine
- [7607](https://github.com/vegaprotocol/vega/issues/7607) - Fix handling of removed transfers
- [7622](https://github.com/vegaprotocol/vega/issues/7622) - Fix cleaning path for Visor when restarting data node
- [7638](https://github.com/vegaprotocol/vega/issues/7638) - Add missing fields to position update resolver
- [7647](https://github.com/vegaprotocol/vega/issues/7647) - Assure LP orders never trade on entry

## 0.67.2

### 🐛 Fixes

- [7387](https://github.com/vegaprotocol/vega/issues/7387) - Allow authorization headers in wallet service

## 0.67.1

### 🛠 Improvements

- [7374](https://github.com/vegaprotocol/vega/issues/7374) - Add `TLS` support to the `REST` `api`
- [7349](https://github.com/vegaprotocol/vega/issues/7349) - Add `Access-Control-Max-Age` header with configurable value for the in `core`, `datanode` and `blockexplorer` HTTP `APIs`
- [7381](https://github.com/vegaprotocol/vega/pull/7381) - Allow target stake to drop within auction once the time window elapses

### 🐛 Fixes

- [7366](https://github.com/vegaprotocol/vega/issues/7366) - Fix typos in the API descriptions
- [7335](https://github.com/vegaprotocol/vega/issues/7335) - Fix custom http headers not being returned - add configurable `CORS` headers to `core`, `datanode` and `blockexplorer` HTTP `APIs`

## 0.67.0

### 🚨 Breaking changes

- [6895](https://github.com/vegaprotocol/vega/issues/6895) - Move the authentication of wallet API version 2 to the transport layer (HTTP). This brings several breaking changes:
  - A unified HTTP response payload has been introduced for structured response and error handling for data coming from the HTTP layer.
  - the `/api/v2/methods` endpoints now uses the new HTTP response payload.
  - the `/api/v2/requests` endpoint can either return the HTTP or the JSON-RPC response payload depending on the situation.
  - the token has been moved out of the JSON-RPC requests, to HTTP `Authorization` header.
- [7293](https://github.com/vegaprotocol/vega/issues/7293) - Rename restricted keys to allowed keys
- [7211](https://github.com/vegaprotocol/vega/issues/7211) - Add sender and receiver balances in ledger entries
- [7255](https://github.com/vegaprotocol/vega/issues/7255) - Rename `dehistory` to network history

### 🛠 Improvements

- [7317](https://github.com/vegaprotocol/vega/issues/7317) - Add database schema docs
- [7279](https://github.com/vegaprotocol/vega/issues/7279) - Add `--archive` and `--lite` to `datanode init`
- [7302](https://github.com/vegaprotocol/vega/issues/7302) - Add withdrawal minimal amount
- [5487](https://github.com/vegaprotocol/vega/issues/5487) - Add `UPGRADING.md`
- [7358](https://github.com/vegaprotocol/vega/issues/7358) - Improve `datanode init` and `vega init` help text
- [7114](https://github.com/vegaprotocol/vega/issues/7114) - Expose user spam statistics via `API`
- [7316](https://github.com/vegaprotocol/vega/issues/7316) - Add a bunch of database indexes following audit of queries
- [7331](https://github.com/vegaprotocol/vega/issues/7331) - Control the decrease of the number of validators when network parameter is decreased
- [6754](https://github.com/vegaprotocol/vega/issues/6754) - Add `csv` export for ledger entries
- [7093](https://github.com/vegaprotocol/vega/issues/7093) - Pick up the long-living tokens after the wallet service is started
- [7328](https://github.com/vegaprotocol/vega/issues/7328) - Add missing documentation of JSON-RPC methods `admin.update_passphrase`

### 🐛 Fixes

- [7260](https://github.com/vegaprotocol/vega/issues/7260) - Fix bug where pagination `before` or `after` cursors were ignored if `first` or `last` not set
- [7281](https://github.com/vegaprotocol/vega/issues/7281) - Fix formatting of status enum for `dataSourceSpec` in `GraphQL`
- [7283](https://github.com/vegaprotocol/vega/issues/7283) - Fix validation of future product oracles signers
- [7306](https://github.com/vegaprotocol/vega/issues/7306) - Improve speed of querying deposits and withdrawals by party
- [7337](https://github.com/vegaprotocol/vega/issues/7337) - Add `UpdateAsset` change types to proposal terms `GraphQL` resolver
- [7278](https://github.com/vegaprotocol/vega/issues/7278) - Use `Informal systems` fork of Tendermint
- [7294](https://github.com/vegaprotocol/vega/issues/7294) - Submission of `OpenOracle` data is broken
- [7286](https://github.com/vegaprotocol/vega/issues/7286) - Fix serialisation of `oracle specs`
- [7327](https://github.com/vegaprotocol/vega/issues/7327) - Improve and add API info, remove unused `AccountField` enum in `GraphQL`
- [7345](https://github.com/vegaprotocol/vega/issues/7345) - Cache account lookup by id

## 0.66.1

- [7269](https://github.com/vegaprotocol/vega/pull/7269) - Fix wallet release pipeline

## 0.66.0

### 🚨 Breaking changes

- [6957](https://github.com/vegaprotocol/vega/issues/6957) - Remove `client.<get|request>_permissions` endpoints on the wallet.
- [7079](https://github.com/vegaprotocol/vega/issues/7079) - Remove deprecated `version` property from wallet API.
- [7067](https://github.com/vegaprotocol/vega/issues/7067) - Remove legacy technical commands on the wallet command line.
- [7069](https://github.com/vegaprotocol/vega/issues/7069) - Remove deprecated `vegawallet info` command line.
- [7010](https://github.com/vegaprotocol/vega/issues/7010) - Remove the deprecated `encodedTransaction` fields on wallet API endpoints.
- [7232](https://github.com/vegaprotocol/vega/issues/7232) - Rename `stakeToCcySiskas` network parameter to `stakeToCcyVolume`
- [7171](https://github.com/vegaprotocol/vega/issues/7171) - Change liquidity triggering ratio value type from float to string

### 🛠 Improvements

- [7216](https://github.com/vegaprotocol/vega/issues/7216) - Support filtering by market for `ordersConnection` under party queries.
- [7252](https://github.com/vegaprotocol/vega/issues/7252) - Add limits to `MarkPriceUpdateMaximumFrequency` network parameter
- [7169](https://github.com/vegaprotocol/vega/issues/7169) - Handle events to update PnL on trade, instead of waiting for MTM settlements.

### 🐛 Fixes

- [7207](https://github.com/vegaprotocol/vega/issues/7207) - Fix panic, return on error in pool configuration
- [7213](https://github.com/vegaprotocol/vega/issues/7213) - Implement separate `DB` for snapshots `metadata`
- [7220](https://github.com/vegaprotocol/vega/issues/7220) - Fix panic when LP is closed out
- [7235](https://github.com/vegaprotocol/vega/issues/7235) - Do not update existing markets when changing global default `LiquidityMonitoringParameters`
- [7029](https://github.com/vegaprotocol/vega/issues/7029) - Added admin `API` for Data Node to secure some `dehistory` commands
- [7239](https://github.com/vegaprotocol/vega/issues/7239) - Added upper and lower bounds for floating point engine updates
- [7253](https://github.com/vegaprotocol/vega/issues/7253) - improve the adjustment of delegator weight to avoid overflow
- [7075](https://github.com/vegaprotocol/vega/issues/7075) - Remove unused expiry field in withdrawal

## 0.65.1

### 🛠 Improvements

- [6574](https://github.com/vegaprotocol/vega/issues/6574) - Use same default for the probability of trading for floating point consensus as we do for the value between best bid and ask.

### 🐛 Fixes

- [7188](https://github.com/vegaprotocol/vega/issues/7188) - Reset liquidity score even if fees accrued in a period were 0.
- [7189](https://github.com/vegaprotocol/vega/issues/7189) - Include LP orders outside PM price range but within LP price in the liquidity score.
- [7195](https://github.com/vegaprotocol/vega/issues/7195) - Ignore oracle messages while market is in proposed state
- [7198](https://github.com/vegaprotocol/vega/issues/7198) - Reduce `RAM` usage when tendermint calls list snapshot
- [6996](https://github.com/vegaprotocol/vega/issues/6996) - Add Visor docs

## 0.65.0

### 🚨 Breaking changes

- [6955](https://github.com/vegaprotocol/vega/issues/6955) - Market definition extended with the new field for LP price range across the API.
- [6645](https://github.com/vegaprotocol/vega/issues/6645) - Set decimal number value to be used from oracle instead of from tradable instruments

### 🗑️ Deprecation

- [7068](https://github.com/vegaprotocol/vega/issues/7068) - Alias `vegawallet info` to `vegawallet describe`, before definitive renaming.

### 🛠 Improvements

- [7032](https://github.com/vegaprotocol/vega/issues/7032) - Make deposits and withdrawals `hypertables` and change `deposits_current` and `withdrawals_current` into views to improve resource usage
- [7136](https://github.com/vegaprotocol/vega/issues/7136) - Update ban duration to 30 minutes for spam
- [7026](https://github.com/vegaprotocol/vega/issues/7026) - Let decentralised history use the snapshot event from the core as an indication for snapshot rather than doing the calculation based on the interval network parameter.
- [7108](https://github.com/vegaprotocol/vega/issues/7108) - Return `ArgumentError` if candle id not supplied to `ListCandleData`
- [7098](https://github.com/vegaprotocol/vega/issues/7098) - Add an event when the core is taking a snapshot
- [7028](https://github.com/vegaprotocol/vega/issues/7028) - Add `JSON` output for `dehistory` commands; fix `config` override on command line
- [7122](https://github.com/vegaprotocol/vega/issues/7122) - Allow for tolerance in validator performance calculation
- [7104](https://github.com/vegaprotocol/vega/issues/7104) - Provide a better error message when party has insufficient balance of an asset
- [7143](https://github.com/vegaprotocol/vega/issues/7143) - Update `grpc-rest-bindings` for Oracle `API`
- [7027](https://github.com/vegaprotocol/vega/issues/7027) - `Dehistory` store does not clean up resources after a graceful shutdown
- [7157](https://github.com/vegaprotocol/vega/issues/7157) - Core waits for data node and shuts down gracefully during protocol upgrade
- [7113](https://github.com/vegaprotocol/vega/issues/7113) - Added API for epoch summaries of rewards distributed
- [6956](https://github.com/vegaprotocol/vega/issues/6956) - Include liquidity measure of deployed orders in the fees distribution
- [7168](https://github.com/vegaprotocol/vega/issues/7168) - Expose liquidity score on on market data `API`

### 🐛 Fixes

- [7040](https://github.com/vegaprotocol/vega/issues/7040) - Block explorer use different codes than 500 on error
- [7099](https://github.com/vegaprotocol/vega/issues/7099) - Remove undelegate method `IN_ANGER`
- [7021](https://github.com/vegaprotocol/vega/issues/7021) - MTM settlement on trading terminated fix.
- [7102](https://github.com/vegaprotocol/vega/issues/7102) - Ensure the `api-token init -f` wipes the tokens file
- [7106](https://github.com/vegaprotocol/vega/issues/7106) - Properties of oracle data sent in non-deterministic order
- [7000](https://github.com/vegaprotocol/vega/issues/7000) - Wallet honours proof of work difficulty increases
- [7029](https://github.com/vegaprotocol/vega/issues/7029) - Remove unsafe `GRPC` endpoint in data node
- [7116](https://github.com/vegaprotocol/vega/issues/7116) - Fix MTM trade price check when trading is terminated.
- [7173](https://github.com/vegaprotocol/vega/issues/7173) - Fix deterministic order of price bounds on market data events
- [7112](https://github.com/vegaprotocol/vega/issues/7112) - Restore order's original price when restoring from a snapshot
- [6955](https://github.com/vegaprotocol/vega/issues/6955) - Remove scaling by probability when implying LP volumes. Only change the LP order price if it’s outside the new “valid LP price range” - move it to the bound in that case.
- [7132](https://github.com/vegaprotocol/vega/issues/7132) - Make the recovery phrase import white space resistant
- [7150](https://github.com/vegaprotocol/vega/issues/7150) - Avoid taking 2 snapshots upon protocol upgrade block
- [7142](https://github.com/vegaprotocol/vega/issues/7142) - Do not recalculate margins based on potential positions when market is terminated.
- [7172](https://github.com/vegaprotocol/vega/issues/7172) - Make markets table a hyper table and update queries.

## 0.64.0

### 🗑️ Deprecation

- [7065](https://github.com/vegaprotocol/vega/issues/7065) - Scope technical commands on wallet command line
- [7066](https://github.com/vegaprotocol/vega/issues/7066) - Move network compatibility check to a dedicated wallet command line

### 🛠 Improvements

- [7052](https://github.com/vegaprotocol/vega/issues/7052) - Add a specific error message when trying to access administrative endpoints on wallet API
- [7064](https://github.com/vegaprotocol/vega/issues/7064) - Make `SQL` store tests run in temporary transactions instead of truncating all tables for each test
- [7053](https://github.com/vegaprotocol/vega/issues/7053) - Add info endpoint for the block explorer

### 🐛 Fixes

- [7011](https://github.com/vegaprotocol/vega/issues/7011) - Incorrect flagging of live orders when multiple updates in the same block
- [7037](https://github.com/vegaprotocol/vega/issues/7037) - Reinstate permissions endpoints on the wallet API
- [7034](https://github.com/vegaprotocol/vega/issues/7034) - Rename `network` to `name` in `admin.remove_network`
- [7031](https://github.com/vegaprotocol/vega/issues/7031) - `datanode` expects protocol upgrade event in the right sequence
- [7072](https://github.com/vegaprotocol/vega/issues/7072) - Check if event forwarding engine is started before reloading
- [7017](https://github.com/vegaprotocol/vega/issues/7017) - Fix issue with market update during opening auction

## 0.63.1

### 🛠 Improvements

- [7003](https://github.com/vegaprotocol/vega/pull/7003) - Expose bus event stream on the `REST` API
- [7044](https://github.com/vegaprotocol/vega/issues/7044) - Proof of work improvements
- [7041](https://github.com/vegaprotocol/vega/issues/7041) - Change witness vote count to be based on voting power
- [7073](https://github.com/vegaprotocol/vega/issues/7073) - Upgrade `btcd` library

## 0.63.0

### 🚨 Breaking changes

- [6898](https://github.com/vegaprotocol/vega/issues/6795) - allow `-snapshot.load-from-block-height=` to apply to `statesync` snapshots
- [6716](https://github.com/vegaprotocol/vega/issues/6716) - Use timestamp on all times fields
- [6887](https://github.com/vegaprotocol/vega/issues/6716) - `client.get_permissions` and `client.request_permissions` have been removed from Wallet service V2 with permissions now asked during `client.list_keys`
- [6725](https://github.com/vegaprotocol/vega/issues/6725) - Fix inconsistent use of node field on `GraphQL` connection edges
- [6746](https://github.com/vegaprotocol/vega/issues/6746) - The `validating_nodes` has been removed from `NodeData` and replaced with details of each node set

### 🛠 Improvements

- [6898](https://github.com/vegaprotocol/vega/issues/6898) - allow `-snapshot.load-from-block-height=` to apply to `statesync` snapshots
- [6871](https://github.com/vegaprotocol/vega/issues/6871) - Assure integration test framework throws an error when no watchers specified for a network parameter being set/updated
- [6795](https://github.com/vegaprotocol/vega/issues/6795) - max gas implementation
- [6641](https://github.com/vegaprotocol/vega/issues/6641) - network wide limits
- [6731](https://github.com/vegaprotocol/vega/issues/6731) - standardize on 'network' and '' for network party and no market identifiers
- [6792](https://github.com/vegaprotocol/vega/issues/6792) - Better handling of panics when moving time with `nullchain`, add endpoint to query whether `nullchain` is replaying
- [6753](https://github.com/vegaprotocol/vega/issues/6753) - Filter votes per party and/or proposal
- [6959](https://github.com/vegaprotocol/vega/issues/6959) - Fix listing transactions by block height in block explorer back end
- [6832](https://github.com/vegaprotocol/vega/issues/6832) - Add signature to transaction information returned by block explorer API
- [6884](https://github.com/vegaprotocol/vega/issues/6884) - Specify transaction as `JSON` rather than a base64 encoded string in `client_{sign|send}_transaction`
- [6975](https://github.com/vegaprotocol/vega/issues/6975) - Implement `admin.sign_transaction` in the wallet
- [6974](https://github.com/vegaprotocol/vega/issues/6974) - Make names in wallet admin `API` consistent
- [6642](https://github.com/vegaprotocol/vega/issues/6642) - Add methods to manage the wallet service and its connections on wallet API version 2
- [6853](https://github.com/vegaprotocol/vega/issues/6853) - Max gas and priority improvements
- [6782](https://github.com/vegaprotocol/vega/issues/6782) - Bump embedded `postgres` version to hopefully fix `CI` instability
- [6880](https://github.com/vegaprotocol/vega/issues/6880) - Omit transactions we can't decode in block explorer transaction list
- [6640](https://github.com/vegaprotocol/vega/issues/6640) - Mark to market to happen every N seconds.
- [6827](https://github.com/vegaprotocol/vega/issues/6827) - Add `LastTradedPrice` field in market data
- [6871](https://github.com/vegaprotocol/vega/issues/6871) - Assure integration test framework throws an error when no watchers specified for a network parameter being set/updated
- [6908](https://github.com/vegaprotocol/vega/issues/6871) - Update default retention policy
- [6827](https://github.com/vegaprotocol/vega/issues/6615) - Add filters to `ordersConnection`
- [6910](https://github.com/vegaprotocol/vega/issues/6910) - Separate settled position from position
- [6988](https://github.com/vegaprotocol/vega/issues/6988) - Handle 0 timestamps in `graphql` marshaller
- [6910](https://github.com/vegaprotocol/vega/issues/6910) - Separate settled position from position
- [6949](https://github.com/vegaprotocol/vega/issues/6949) - Mark positions to market at the end of the block.
- [6819](https://github.com/vegaprotocol/vega/issues/6819) - Support long-living token in wallet client API
- [6964](https://github.com/vegaprotocol/vega/issues/6964) - Add support for long living tokens with expiry
- [6991](https://github.com/vegaprotocol/vega/issues/6991) - Expose error field in explorer API
- [5769](https://github.com/vegaprotocol/vega/issues/5769) - Automatically resolve the host name in the client wallet API
- [6910](https://github.com/vegaprotocol/vega/issues/6910) - Separate settled position from position

### 🐛 Fixes

- [6758](https://github.com/vegaprotocol/vega/issues/6758) - Fix first and last block not returned on querying epoch
- [6924](https://github.com/vegaprotocol/vega/issues/6924) - Fix deterministic sorting when nodes have equal scores and we have to choose who is in the signer set
- [6812](https://github.com/vegaprotocol/vega/issues/6812) - Network name is derived solely from the filename to cause less confusion if the network `config` is renamed
- [6831](https://github.com/vegaprotocol/vega/issues/6831) - Fix settlement state in snapshots and market settlement.
- [6856](https://github.com/vegaprotocol/vega/issues/6856) - When creating liquidity provision, seed dummy orders in order to prevent broken references when querying the market later
- [6801](https://github.com/vegaprotocol/vega/issues/6801) - Fix internal data source validations
- [6766](https://github.com/vegaprotocol/vega/issues/6766) - Handle relative vega home path being passed in `postgres` snapshots
- [6885](https://github.com/vegaprotocol/vega/issues/6885) - Don't ignore 'bootstrap peers' `IPFS` configuration setting in `datanode`
- [6799](https://github.com/vegaprotocol/vega/issues/6799) - Move LP fees in transit to the network treasury
- [6781](https://github.com/vegaprotocol/vega/issues/6781) - Fix bug where only first 32 characters of the `IPFS` identity seed were used.
- [6824](https://github.com/vegaprotocol/vega/issues/6824) - Respect `VEGA_HOME` for embedded `postgres` log location
- [6843](https://github.com/vegaprotocol/vega/issues/6843) - Fix Visor runner keys
- [6934](https://github.com/vegaprotocol/vega/issues/6934) - from/to accounts for ledger entries in database were reversed
- [6826](https://github.com/vegaprotocol/vega/issues/6826) - Update `spam.pow.numberOfPastBlocks` range values
- [6332](https://github.com/vegaprotocol/vega/issues/6332) - Standardise `graphql` responses
- [6862](https://github.com/vegaprotocol/vega/issues/6862) - Add party in account update
- [6888](https://github.com/vegaprotocol/vega/issues/6888) - Errors on accepted transaction with an invalid state are correctly handled in the wallet API version 2
- [6899](https://github.com/vegaprotocol/vega/issues/6899) - Upgrade to tendermint 0.34.24
- [6894](https://github.com/vegaprotocol/vega/issues/6894) - Finer error code returned to the third-party application
- [6849](https://github.com/vegaprotocol/vega/issues/6849) - Ensure the positions are remove from the positions engine when they are closed
- [6767](https://github.com/vegaprotocol/vega/issues/6767) - Protocol upgrade rejected events fail to write in the database
- [6896](https://github.com/vegaprotocol/vega/issues/6896) - Fix timestamps in proposals (`GQL`)
- [6844](https://github.com/vegaprotocol/vega/issues/6844) - Use proper type in `GQL` for transfer types and some types rename
- [6783](https://github.com/vegaprotocol/vega/issues/6783) - Unstable `CI` tests for `dehistory`
- [6844](https://github.com/vegaprotocol/vega/issues/6844) - Unstable `CI` tests for `dehistory`
- [6844](https://github.com/vegaprotocol/vega/issues/6844) - Add API descriptions, remove unused ledger entries and fix typos
- [6960](https://github.com/vegaprotocol/vega/issues/6960) - Infer has traded from settlement engine rather than from an unsaved-to-snapshot flag
- [6941](https://github.com/vegaprotocol/vega/issues/6941) - Rename `admin.describe_network` parameter to `name`
- [6976](https://github.com/vegaprotocol/vega/issues/6976) - Recalculate margins on MTM anniversary even if there were no trades.
- [6977](https://github.com/vegaprotocol/vega/issues/6977) - Prior to final settlement, perform MTM on unsettled trades.
- [6569](https://github.com/vegaprotocol/vega/issues/6569) - Fix margin calculations during auctions.
- [7001](https://github.com/vegaprotocol/vega/issues/7001) - Set mark price on final settlement.

## 0.62.1

### 🛠 Improvements

- [6726](https://github.com/vegaprotocol/vega/issues/6726) - Talk to embedded `postgres` via a `UNIX` domain socket in tests.

### 🐛 Fixes

- [6759](https://github.com/vegaprotocol/vega/issues/6759) - Send events when liquidity provisions are `undeployed`
- [6764](https://github.com/vegaprotocol/vega/issues/6764) - If a trading terminated oracle changes after trading already terminated do not subscribe to it
- [6775](https://github.com/vegaprotocol/vega/issues/6775) - Fix oracle spec identifiers
- [6762](https://github.com/vegaprotocol/vega/issues/6762) - Fix one off transfer events serialization
- [6747](https://github.com/vegaprotocol/vega/issues/6747) - Ensure proposal with no participation does not get enacted
- [6757](https://github.com/vegaprotocol/vega/issues/6655) - Fix oracle spec resolvers in Gateway
- [6952](https://github.com/vegaprotocol/vega/issues/6757) - Fix signers resolvers in Gateway

## 0.62.0

### 🚨 Breaking changes

- [6598](https://github.com/vegaprotocol/vega/issues/6598) - Rework `vega tools snapshot` command to be more consistent with other CLI options

### 🛠 Improvements

- [6681](https://github.com/vegaprotocol/vega/issues/6681) - Add indexes to improve balance history query
- [6682](https://github.com/vegaprotocol/vega/issues/6682) - Add indexes to orders by reference query
- [6668](https://github.com/vegaprotocol/vega/issues/6668) - Add indexes to trades by buyer/seller
- [6628](https://github.com/vegaprotocol/vega/issues/6628) - Improve node health check in the wallet
- [6711](https://github.com/vegaprotocol/vega/issues/6711) - `Anti-whale ersatz` validators reward stake scores

### 🐛 Fixes

- [6701](https://github.com/vegaprotocol/vega/issues/6701) - Fix `GraphQL` `API` not returning `x-vega-*` headers
- [6563](https://github.com/vegaprotocol/vega/issues/6563) - Liquidity engine reads orders directly from the matching engine
- [6696](https://github.com/vegaprotocol/vega/issues/6696) - New nodes are now visible from the epoch they announced and not epoch they become active
- [6661](https://github.com/vegaprotocol/vega/issues/6661) - Scale price to asset decimal in estimate orders
- [6685](https://github.com/vegaprotocol/vega/issues/6685) - `vega announce_node` now returns a `txHash` when successful or errors from `CheckTx`
- [6687](https://github.com/vegaprotocol/vega/issues/6687) - Expose `admin.update_passphrase` in admin wallet API
- [6686](https://github.com/vegaprotocol/vega/issues/6686) - Expose `admin.rename_wallet` in admin wallet API
- [6496](https://github.com/vegaprotocol/vega/issues/6496) - Fix margin calculation for pegged and liquidity orders
- [6670](https://github.com/vegaprotocol/vega/issues/6670) - Add governance by `ID` endpoint to `REST` bindings
- [6679](https://github.com/vegaprotocol/vega/issues/6679) - Permit `GFN` pegged orders
- [6707](https://github.com/vegaprotocol/vega/issues/6707) - Fix order event for liquidity provisions
- [6699](https://github.com/vegaprotocol/vega/issues/6699) - `orders` and `orders_current` view uses a redundant union causing performance issues
- [6721](https://github.com/vegaprotocol/vega/issues/6721) - Visor fix if condition for `maxNumberOfFirstConnectionRetries`
- [6655](https://github.com/vegaprotocol/vega/issues/6655) - Fix market query by `ID`
- [6656](https://github.com/vegaprotocol/vega/issues/6656) - Fix data sources to handle opening with internal source
- [6722](https://github.com/vegaprotocol/vega/issues/6722) - Fix get market response to contain oracle id

## 0.61.0

### 🚨 Breaking changes

- [5674](https://github.com/vegaprotocol/vega/issues/5674) - Remove `V1` data node `API`
- [5714](https://github.com/vegaprotocol/vega/issues/5714) - Update data sourcing types

### 🛠 Improvements

- [6603](https://github.com/vegaprotocol/vega/issues/6603) - Put embedded `postgres` files in proper state directory
- [6552](https://github.com/vegaprotocol/vega/issues/6552) - Add `datanode` `API` for querying protocol upgrade proposals
- [6613](https://github.com/vegaprotocol/vega/issues/6613) - Add file buffering to datanode
- [6602](https://github.com/vegaprotocol/vega/issues/6602) - Panic if data node receives events in unexpected order
- [6595](https://github.com/vegaprotocol/vega/issues/6595) - Support for cross network parameter dependency and validation
- [6627](https://github.com/vegaprotocol/vega/issues/6627) - Fix order estimates
- [6604](https://github.com/vegaprotocol/vega/issues/6604) - Fix transfer funds documentations in `protos`
- [6463](https://github.com/vegaprotocol/vega/issues/6463) - Implement chain replay and snapshot restore for the `nullblockchain`
- [6652](https://github.com/vegaprotocol/vega/issues/6652) - Change protocol upgrade consensus do be based on voting power

### 🐛 Fixes

- [6356](https://github.com/vegaprotocol/vega/issues/6356) - When querying for proposals from `GQL` return votes.
- [6623](https://github.com/vegaprotocol/vega/issues/6623) - Fix `nil` pointer panic in `datanode` for race condition in `recvEventRequest`
- [6601](https://github.com/vegaprotocol/vega/issues/6601) - Removed resend event when the socket client fails
- [5715](https://github.com/vegaprotocol/vega/issues/5715) - Fix documentation for Oracle Submission elements
- [5770](https://github.com/vegaprotocol/vega/issues/5770) - Fix Nodes data query returns incorrect results

## 0.60.0

### 🚨 Breaking changes

- [6227](https://github.com/vegaprotocol/vega/issues/6227) - Datanode Decentralized History - datanode init command now requires the chain id as a parameter

### 🛠 Improvements

- [6530](https://github.com/vegaprotocol/vega/issues/6530) - Add command to rename a wallet
- [6531](https://github.com/vegaprotocol/vega/issues/6531) - Add command to update the passphrase of a wallet
- [6482](https://github.com/vegaprotocol/vega/issues/6482) - Improve `TransferType` mapping usage
- [6546](https://github.com/vegaprotocol/vega/issues/6546) - Add a separate README for datanode/api gRPC handling principles
- [6582](https://github.com/vegaprotocol/vega/issues/6582) - Match validation to the required ranges
- [6596](https://github.com/vegaprotocol/vega/issues/6596) - Add market risk parameter validation

### 🐛 Fixes

- [6410](https://github.com/vegaprotocol/vega/issues/6410) - Add input validation for the `EstimateFee` endpoint.
- [6556](https://github.com/vegaprotocol/vega/issues/6556) - Limit ledger entries filtering complexity and potential number of items.
- [6539](https://github.com/vegaprotocol/vega/issues/6539) - Fix total fee calculation in estimate order
- [6584](https://github.com/vegaprotocol/vega/issues/6584) - Simplify `ListBalanceChanges`, removing aggregation and forward filling for now
- [6583](https://github.com/vegaprotocol/vega/issues/6583) - Cancel wallet connection request if no wallet

## 0.59.0

### 🚨 Breaking changes

- [6505](https://github.com/vegaprotocol/vega/issues/6505) - Allow negative position decimal places for market
- [6477](https://github.com/vegaprotocol/vega/issues/6477) - Allow the user to specify a different passphrase when isolating a key
- [6549](https://github.com/vegaprotocol/vega/issues/6549) - Output from `nodewallet reload` is now more useful `json`
- [6458](https://github.com/vegaprotocol/vega/issues/6458) - Rename `GetMultiSigSigner...Bundles API` functions to `ListMultiSigSigner...Bundles` to be consistent with `v2 APIs`
- [6506](https://github.com/vegaprotocol/vega/issues/6506) - Swap places of PID and date in log files in the wallet service

### 🛠 Improvements

- [6080](https://github.com/vegaprotocol/vega/issues/6080) - Data-node handles upgrade block and ensures data is persisted before upgrade
- [6527](https://github.com/vegaprotocol/vega/issues/6527) - Add `last-block` sub-command to `datanode CLI`
- [6529](https://github.com/vegaprotocol/vega/issues/6529) - Added reason to transfer to explain why it was stopped or rejected
- [6513](https://github.com/vegaprotocol/vega/issues/6513) - Refactor `datanode` `api` for getting balance history

### 🐛 Fixes

- [6480](https://github.com/vegaprotocol/vega/issues/6480) - Wallet `openrpc.json` is now a valid OpenRPC file
- [6473](https://github.com/vegaprotocol/vega/issues/6473) - Infrastructure Fee Account returns error when asset is pending listing
- [5690](https://github.com/vegaprotocol/vega/issues/5690) - Markets query now excludes rejected markets
- [5479](https://github.com/vegaprotocol/vega/issues/5479) - Fix inconsistent naming in API error
- [6525](https://github.com/vegaprotocol/vega/issues/6525) - Round the right way when restoring the integer representation of cached price ranges from a snapshot
- [6011](https://github.com/vegaprotocol/vega/issues/6011) - Fix data node fails when `Postgres` starts slowly
- [6341](https://github.com/vegaprotocol/vega/issues/6341) - Embedded `Postgres` should only capture logs during testing
- [6511](https://github.com/vegaprotocol/vega/issues/6511) - Do not check writer interface for null when starting embedded `Postgres`
- [6510](https://github.com/vegaprotocol/vega/issues/6510) - Filter parties with 0 reward from reward payout event
- [6471](https://github.com/vegaprotocol/vega/issues/6471) - Fix potential nil reference when owner is system for ledger entries
- [6519](https://github.com/vegaprotocol/vega/issues/6519) - Fix errors in the ledger entries `GraphQL` query.
- [6515](https://github.com/vegaprotocol/vega/issues/6515) - Required properties in OpenRPC documentation are marked as such
- [6234](https://github.com/vegaprotocol/vega/issues/6234) - Fix response in query for oracle data spec by id
- [6294](https://github.com/vegaprotocol/vega/issues/6294) - Fix response for query for non-existing market
- [6508](https://github.com/vegaprotocol/vega/issues/6508) - Fix data node starts slowly when the database is not empty
- [6532](https://github.com/vegaprotocol/vega/issues/6532) - Add current totals to the vote events

## 0.58.0

### 🚨 Breaking changes

- [6271](https://github.com/vegaprotocol/vega/issues/6271) - Require signature from new Ethereum key to validate key rotation submission
- [6364](https://github.com/vegaprotocol/vega/issues/6364) - Rename `oracleSpecForSettlementPrice` to `oracleSpecForSettlementData`
- [6401](https://github.com/vegaprotocol/vega/issues/6401) - Fix estimate fees and margin `APis`
- [6428](https://github.com/vegaprotocol/vega/issues/6428) - Update the wallet connection decision for future work
- [6429](https://github.com/vegaprotocol/vega/issues/6429) - Rename pipeline to interactor for better understanding
- [6430](https://github.com/vegaprotocol/vega/issues/6430) - Split the transaction status interaction depending on success and failure

### 🛠 Improvements

- [6399](https://github.com/vegaprotocol/vega/issues/6399) - Add `init-db` and `unsafe-reset-all` commands to block explorer
- [6348](https://github.com/vegaprotocol/vega/issues/6348) - Reduce pool size to leave more available `Postgres` connections
- [6453](https://github.com/vegaprotocol/vega/issues/6453) - Add ability to write `pprofs` at intervals to core
- [6312](https://github.com/vegaprotocol/vega/issues/6312) - Add back amended balance tests and correct ordering
- [6320](https://github.com/vegaprotocol/vega/issues/6320) - Use `Account` type without internal `id` in `datanode`
- [6461](https://github.com/vegaprotocol/vega/issues/6461) - Occasionally close `postgres` pool connections
- [6435](https://github.com/vegaprotocol/vega/issues/6435) - Add `GetTransaction` `API` call for block explorer
- [6464](https://github.com/vegaprotocol/vega/issues/6464) - Improve block explorer performance when filtering by submitter
- [6211](https://github.com/vegaprotocol/vega/issues/6211) - Handle `BeginBlock` and `EndBlock` events
- [6361](https://github.com/vegaprotocol/vega/issues/6361) - Remove unnecessary logging in market
- [6378](https://github.com/vegaprotocol/vega/issues/6378) - Migrate remaining views of current data to tables with current data
- [6425](https://github.com/vegaprotocol/vega/issues/6425) - Introduce interaction for beginning and ending of request
- [6308](https://github.com/vegaprotocol/vega/issues/6308) - Support parallel requests in wallet API version 2
- [6426](https://github.com/vegaprotocol/vega/issues/6426) - Add a name field on interaction to know what they are when JSON
- [6427](https://github.com/vegaprotocol/vega/issues/6427) - Improve interactions documentation
- [6431](https://github.com/vegaprotocol/vega/issues/6431) - Pass a human-readable input data in Transaction Succeeded and Failed notifications
- [6448](https://github.com/vegaprotocol/vega/issues/6448) - Improve wallet interaction JSON conversion
- [6454](https://github.com/vegaprotocol/vega/issues/6454) - Improve test coverage for setting fees and rewarding LPs
- [6458](https://github.com/vegaprotocol/vega/issues/6458) - Return a context aware message in `RequestSuccessful` interaction
- [6451](https://github.com/vegaprotocol/vega/issues/6451) - Improve interaction error message
- [6432](https://github.com/vegaprotocol/vega/issues/6432) - Use optionals for order error and proposal error
- [6368](https://github.com/vegaprotocol/vega/pull/6368) - Add Ledger Entry API

### 🐛 Fixes

- [6444](https://github.com/vegaprotocol/vega/issues/6444) - Send a transaction error if the same node announces itself twice
- [6388](https://github.com/vegaprotocol/vega/issues/6388) - Do not transfer stake and delegations after a key rotation
- [6266](https://github.com/vegaprotocol/vega/issues/6266) - Do not take a snapshot at block height 1 and handle increase of interval appropriately
- [6338](https://github.com/vegaprotocol/vega/issues/6338) - Fix validation for update an new asset proposals
- [6357](https://github.com/vegaprotocol/vega/issues/6357) - Fix potential panic in `gql` resolvers
- [6391](https://github.com/vegaprotocol/vega/issues/6391) - Fix dropped connection between core and data node when large `(>1mb)` messages are sent.
- [6358](https://github.com/vegaprotocol/vega/issues/6358) - Do not show hidden files nor directories as wallet
- [6374](https://github.com/vegaprotocol/vega/issues/6374) - Fix panic with the metrics
- [6373](https://github.com/vegaprotocol/vega/issues/6373) - Fix panic with the metrics as well
- [6238](https://github.com/vegaprotocol/vega/issues/6238) - Return empty string for `multisig` bundle, not `0x` when asset doesn't have one
- [6236](https://github.com/vegaprotocol/vega/issues/6236) - Make `erc20ListAssetBundle` `nullable` in `GraphQL`
- [6395](https://github.com/vegaprotocol/vega/issues/6395) - Wallet selection doesn't lower case the wallet name during input verification
- [6408](https://github.com/vegaprotocol/vega/issues/6408) - Initialise observer in liquidity provision `sql` store
- [6406](https://github.com/vegaprotocol/vega/issues/6406) - Fix invalid tracking of cumulative volume and price
- [6387](https://github.com/vegaprotocol/vega/issues/6387) - Fix max open interest calculation
- [6416](https://github.com/vegaprotocol/vega/issues/6416) - Prevent submission of `erc20` address already used by another asset
- [6375](https://github.com/vegaprotocol/vega/issues/6375) - If there is one unit left over at the end of final market settlement - transfer it to the network treasury. if there is more than one, log all transfers and panic.
- [6456](https://github.com/vegaprotocol/vega/issues/6456) - Assure liquidity fee gets update when target stake drops (even in the absence of trades)
- [6459](https://github.com/vegaprotocol/vega/issues/6459) - Send lifecycle notifications after parameters validation
- [6420](https://github.com/vegaprotocol/vega/issues/6420) - Support cancellation of a request during a wallet interaction

## 0.57.0

### 🚨 Breaking changes

- [6291](https://github.com/vegaprotocol/vega/issues/6291) - Remove `Nodewallet.ETH` configuration and add flags to supply `clef` addresses when importing or generating accounts
- [6314](https://github.com/vegaprotocol/vega/issues/6314) - Rename session namespace to client in wallet API version 2

### 🛠 Improvements

- [6283](https://github.com/vegaprotocol/vega/issues/6283) - Add commit hash to version if is development version
- [6321](https://github.com/vegaprotocol/vega/issues/6321) - Get rid of the `HasChanged` check in snapshot engines
- [6126](https://github.com/vegaprotocol/vega/issues/6126) - Don't generate market depth subscription messages if nothing has changed

### 🐛 Fixes

- [6287](https://github.com/vegaprotocol/vega/issues/6287) - Fix GraphQL `proposals` API `proposalType` filter
- [6307](https://github.com/vegaprotocol/vega/issues/6307) - Emit an event with status rejected if a protocol upgrade proposal has no validator behind it
- [5305](https://github.com/vegaprotocol/vega/issues/5305) - Handle market updates changing price monitoring parameters correctly.

## 0.56.0

### 🚨 Breaking changes

- [6196](https://github.com/vegaprotocol/vega/pull/6196) - Remove unused network parameters network end of life and market freeze date
- [6155](https://github.com/vegaprotocol/vega/issues/6155) - Rename "Client" to "User" in wallet API version 2
- [5641](https://github.com/vegaprotocol/vega/issues/5641) - Rename `SettlementPriceDecimals` to `SettlementDataDecimals`

### 🛠 Improvements

- [6103](hhttps://github.com/vegaprotocol/vega/issues/6103) - Verify that order amendment has the desired effect on opening auction
- [6170](https://github.com/vegaprotocol/vega/pull/6170) - Order GraphQL schema (query and subscription types) alphabetically
- [6163](https://github.com/vegaprotocol/vega/issues/6163) - Add block explorer back end
- [6153](https://github.com/vegaprotocol/vega/issues/6153) - Display UI friendly logs when calling `session.send_transaction`
- [6063](https://github.com/vegaprotocol/vega/pull/6063) - Update average entry valuation calculation according to spec change.
- [6191](https://github.com/vegaprotocol/vega/pull/6191) - Remove the retry on node health check in the wallet API version 2
- [6221](https://github.com/vegaprotocol/vega/pull/6221) - Add documentation for new `GraphQL endpoints`
- [6498](https://github.com/vegaprotocol/vega/pull/6498) - Fix incorrectly encoded account id
- [5600](https://github.com/vegaprotocol/vega/issues/5600) - Migrate all wallet capabilities to V2 api
- [6077](https://github.com/vegaprotocol/vega/issues/6077) - Add proof-of-work to transaction when using `vegawallet command sign`
- [6203](https://github.com/vegaprotocol/vega/issues/6203) - Support automatic consent for transactions sent through the wallet API version 2
- [6203](https://github.com/vegaprotocol/vega/issues/6203) - Log node selection process on the wallet CLI
- [5925](https://github.com/vegaprotocol/vega/issues/5925) - Clean transfer response API, now ledger movements
- [6254](https://github.com/vegaprotocol/vega/issues/6254) - Reject Ethereum configuration update via proposals
- [5706](https://github.com/vegaprotocol/vega/issues/5076) - Datanode snapshot create and restore support

### 🐛 Fixes

- [6255](https://github.com/vegaprotocol/vega/issues/6255) - Fix `WebSocket` upgrading when setting headers in HTTP middleware.
- [6101](https://github.com/vegaprotocol/vega/issues/6101) - Fix Nodes API not returning new `ethereumAdress` after `EthereumKeyRotation` event.
- [6183](https://github.com/vegaprotocol/vega/issues/6183) - Shutdown blockchain before protocol services
- [6148](https://github.com/vegaprotocol/vega/issues/6148) - Fix API descriptions for typos
- [6187](https://github.com/vegaprotocol/vega/issues/6187) - Not hash message before signing if using clef for validator heartbeats
- [6138](https://github.com/vegaprotocol/vega/issues/6138) - Return more useful information when a transaction submitted to a node contains validation errors
- [6156](https://github.com/vegaprotocol/vega/issues/6156) - Return only delegations for the specific node in `graphql` node delegation query
- [6233](https://github.com/vegaprotocol/vega/issues/6233) - Fix `GetNodeSignatures` GRPC api
- [6175](https://github.com/vegaprotocol/vega/issues/6175) - Fix `datanode` updating node public key on key rotation
- [5948](https://github.com/vegaprotocol/vega/issues/5948) - Shutdown node gracefully when panics or `sigterm` during chain-replay
- [6109](https://github.com/vegaprotocol/vega/issues/6109) - Candle query returns unexpected data.
- [5988](https://github.com/vegaprotocol/vega/issues/5988) - Exclude tainted keys from `session.list_keys` endpoint
- [5164](https://github.com/vegaprotocol/vega/issues/5164) - Distribute LP fees on settlement
- [6212](https://github.com/vegaprotocol/vega/issues/6212) - Change the error for protocol upgrade request for block 0
- [6242](https://github.com/vegaprotocol/vega/issues/6242) - Allow migrate between wallet types during Ethereum key rotation reload
- [6202](https://github.com/vegaprotocol/vega/issues/6202) - Always update margins for parties on amend
- [6228](https://github.com/vegaprotocol/vega/issues/6228) - Reject protocol upgrade downgrades
- [6245](https://github.com/vegaprotocol/vega/issues/6245) - Recalculate equity values when virtual stake changes
- [6260](https://github.com/vegaprotocol/vega/issues/6260) - Prepend `chainID` to input data only when signing the transaction
- [6036](https://github.com/vegaprotocol/vega/issues/6036) - Fix `protobuf<->swagger` generation
- [6248](https://github.com/vegaprotocol/vega/issues/6245) - Candles connection is not returning any candle data
- [6037](https://github.com/vegaprotocol/vega/issues/6037) - Fix auction events.
- [6061](https://github.com/vegaprotocol/vega/issues/6061) - Attempt at stabilizing the tests on the broker in the core
- [6178](https://github.com/vegaprotocol/vega/issues/6178) - Historical balances fails with `scany` error
- [6193](https://github.com/vegaprotocol/vega/issues/6193) - Use Data field from transaction successfully sent but that were rejected
- [6230](https://github.com/vegaprotocol/vega/issues/6230) - Node Signature Connection should return a list or an appropriate error message
- [5998](https://github.com/vegaprotocol/vega/issues/5998) - Positions should be zero when markets are closed and settled
- [6297](https://github.com/vegaprotocol/vega/issues/6297) - Historic Balances fails if `MarketId` is used in `groupBy`

## 0.55.0

### 🚨 Breaking changes

- [5989](https://github.com/vegaprotocol/vega/issues/5989) - Remove liquidity commitment from market proposal
- [6031](https://github.com/vegaprotocol/vega/issues/6031) - Remove market name from `graphql` market type
- [6095](https://github.com/vegaprotocol/vega/issues/6095) - Rename taker fees to maker paid fees
- [5442](https://github.com/vegaprotocol/vega/issues/5442) - Default behaviour when starting to node is to use the latest local snapshot if it exists
- [6139](https://github.com/vegaprotocol/vega/issues/6139) - Return the key on `session.list_keys` endpoint on wallet API version 2

### 🛠 Improvements

- [5971](https://github.com/vegaprotocol/vega/issues/5971) - Add headers `X-Block-Height`, `X-Block-Timestamp` and `X-Vega-Connection` to all API responses
- [5694](https://github.com/vegaprotocol/vega/issues/5694) - Add field `settlementPriceDecimals` to GraphQL `Future` and `FutureProduct` types
- [6048](https://github.com/vegaprotocol/vega/issues/6048) - Upgrade `golangci-lint` to `1.49.0` and implement its suggestions
- [5807](https://github.com/vegaprotocol/vega/issues/5807) - Add Vega tools: `stream`, `snapshot` and `checkpoint`
- [5678](https://github.com/vegaprotocol/vega/issues/5678) - Add GraphQL endpoints for Ethereum bundles: `listAsset`, `updateAsset`, `addSigner` and `removeSigner`
- [5881](https://github.com/vegaprotocol/vega/issues/5881) - Return account subscription as a list
- [5766](https://github.com/vegaprotocol/vega/issues/5766) - Better notification for version update on the wallet
- [5841](https://github.com/vegaprotocol/vega/issues/5841) - Add transaction to request `multisigControl` signatures on demand
- [5937](https://github.com/vegaprotocol/vega/issues/5937) - Add more flexibility to market creation bonus
- [5932](https://github.com/vegaprotocol/vega/issues/5932) - Remove Name and Symbol from update asset proposal
- [5880](https://github.com/vegaprotocol/vega/issues/5880) - Send initial image with subscriptions to positions, orders & accounts
- [5878](https://github.com/vegaprotocol/vega/issues/5878) - Add option to return only live orders in `ListOrders` `API`
- [5937](https://github.com/vegaprotocol/vega/issues/5937) - Add more flexibility to market creation bonus
- [5708](https://github.com/vegaprotocol/vega/issues/5708) - Use market price when reporting average trade price
- [5949](https://github.com/vegaprotocol/vega/issues/5949) - Transfers processed in the order they were received
- [5966](https://github.com/vegaprotocol/vega/issues/5966) - Do not send transaction from wallet if `chainID` is empty
- [5675](https://github.com/vegaprotocol/vega/issues/5675) - Add transaction information to all database tables
- [6004](https://github.com/vegaprotocol/vega/issues/6004) - Probability of trading refactoring
- [5849](https://github.com/vegaprotocol/vega/issues/5849) - Use network parameter from creation time of the proposal for requirements
- [5846](https://github.com/vegaprotocol/vega/issues/5846) - Expose network parameter from creation time of the proposal through `APIs`.
- [5999](https://github.com/vegaprotocol/vega/issues/5999) - Recalculate margins after risk parameters are updated.
- [5682](https://github.com/vegaprotocol/vega/issues/5682) - Expose equity share weight in the API
- [5684](https://github.com/vegaprotocol/vega/issues/5684) - Added date range to a number of historic balances, deposits, withdrawals, orders and trades queries
- [6071](https://github.com/vegaprotocol/vega/issues/6071) - Allow for empty settlement asset in recurring transfer metric definition for market proposer bonus
- [6042](https://github.com/vegaprotocol/vega/issues/6042) - Set GraphQL query complexity limit
- [6106](https://github.com/vegaprotocol/vega/issues/6106) - Returned signed transaction in wallet API version 2 `session.send_transaction`
- [6105](https://github.com/vegaprotocol/vega/issues/6105) - Add `session.sign_transaction` endpoint on wallet API version 2
- [6042](https://github.com/vegaprotocol/vega/issues/5270) - Set GraphQL query complexity limit
- [5888](https://github.com/vegaprotocol/vega/issues/5888) - Add Liquidity Provision subscription to GraphQL
- [5961](https://github.com/vegaprotocol/vega/issues/5961) - Add batch market instructions command
- [5974](https://github.com/vegaprotocol/vega/issues/5974) - Flatten subscription in `Graphql`
- [6146](https://github.com/vegaprotocol/vega/issues/6146) - Add version command to Vega Visor
- [6671](https://github.com/vegaprotocol/vega/issues/6671) - Vega Visor allows to postpone first failure when Core node is slow to startup

### 🐛 Fixes

- [5934](https://github.com/vegaprotocol/vega/issues/5934) - Ensure wallet without permissions can be read
- [5950](https://github.com/vegaprotocol/vega/issues/5934) - Fix documentation for new wallet command
- [5687](https://github.com/vegaprotocol/vega/issues/5934) - Asset cache was returning stale data
- [6032](https://github.com/vegaprotocol/vega/issues/6032) - Risk factors store errors after update to a market
- [5986](https://github.com/vegaprotocol/vega/issues/5986) - Error string on failed transaction is sent in the plain, no need to decode
- [5860](https://github.com/vegaprotocol/vega/issues/5860) - Enacted but unlisted new assets are now included in checkpoints
- [6023](https://github.com/vegaprotocol/vega/issues/6023) - Tell the `datanode` when a genesis validator does not exist in a `checkpoint`
- [5963](https://github.com/vegaprotocol/vega/issues/5963) - Check other nodes during version check if the first one is unavailable
- [6002](https://github.com/vegaprotocol/vega/issues/6002) - Do not emit events for unmatched oracle data and unsubscribe market as soon as oracle data is received
- [6008](https://github.com/vegaprotocol/vega/issues/6008) - Fix equity like share and average trade value calculation with opening auctions
- [6040](https://github.com/vegaprotocol/vega/issues/6040) - Fix protocol upgrade transaction submission and small Visor improvements
- [5977](https://github.com/vegaprotocol/vega/issues/5977) - Fix missing block height and block time on stake linking API
- [6054](https://github.com/vegaprotocol/vega/issues/6054) - Fix panic on settlement
- [6060](https://github.com/vegaprotocol/vega/issues/6060) - Fix connection results should not be declared as mandatory in GQL schema.
- [6097](https://github.com/vegaprotocol/vega/issues/6067) - Fix incorrect asset (metric asset) used for checking market proposer eligibility
- [6099](https://github.com/vegaprotocol/vega/issues/6099) - Allow recurring transfers with the same to and from but with different asset
- [6067](https://github.com/vegaprotocol/vega/issues/6067) - Verify global reward is transferred to party address 0
- [6131](https://github.com/vegaprotocol/vega/issues/6131) - `nullblockchain` should call Tendermint Info `abci` to match real flow
- [6119](https://github.com/vegaprotocol/vega/issues/6119) - Correct order in which market event is emitted
- [5890](https://github.com/vegaprotocol/vega/issues/5890) - Margin breach during amend doesn't cancel order
- [6144](https://github.com/vegaprotocol/vega/issues/6144) - Price and Pegged Offset in orders are Decimals
- [6111](https://github.com/vegaprotocol/vega/issues/5890) - Handle candles transient failure and prevent subscription blocking
- [6204](https://github.com/vegaprotocol/vega/issues/6204) - Data Node add Ethereum Key Rotations subscriber and rest binding

## 0.54.0

### 🚨 Breaking changes

With this release a few breaking changes are introduced.
The Vega application is now a built-in application. This means that Tendermint doesn't need to be started separately any more.
The `vega node` command has been renamed `vega start`.
The `vega tm` command has been renamed `vega tendermint`.
The `Blockchain.Tendermint.ClientAddr` configuration field have been renamed `Blockchain.Tendermint.RPCAddr`.
The init command now also generate the configuration for tendermint, the flags `--no-tendermint`, `--tendermint-home` and `--tendermint-key` have been introduced

- [5579](https://github.com/vegaprotocol/vega/issues/5579) - Make vega a built-in Tendermint application
- [5249](https://github.com/vegaprotocol/vega/issues/5249) - Migrate to Tendermint version 0.35.8
- [5624](https://github.com/vegaprotocol/vega/issues/5624) - Get rid of `updateFrequency` in price monitoring definition
- [5601](https://github.com/vegaprotocol/vega/issues/5601) - Remove support for launching a proxy in front of console and token dApp
- [5872](https://github.com/vegaprotocol/vega/issues/5872) - Remove console and token dApp from networks
- [5802](https://github.com/vegaprotocol/vega/issues/5802) - Remove support for transaction version 1

### 🗑️ Deprecation

- [4655](https://github.com/vegaprotocol/vega/issues/4655) - Move Ethereum `RPC` endpoint configuration from `Nodewallet` section to `Ethereum` section

### 🛠 Improvements

- [5589](https://github.com/vegaprotocol/vega/issues/5589) - Used custom version of Clef
- [5541](https://github.com/vegaprotocol/vega/issues/5541) - Support permissions in wallets
- [5439](https://github.com/vegaprotocol/vega/issues/5439) - `vegwallet` returns better responses when a transaction fails
- [5465](https://github.com/vegaprotocol/vega/issues/5465) - Verify `bytecode` of smart-contracts on startup
- [5608](https://github.com/vegaprotocol/vega/issues/5608) - Ignore stale price monitoring trigger when market is already in auction
- [5673](https://github.com/vegaprotocol/vega/issues/5673) - Add support for `ethereum` key rotations to `datanode`
- [5639](https://github.com/vegaprotocol/vega/issues/5639) - Move all core code in the core directory
- [5613](https://github.com/vegaprotocol/vega/issues/5613) - Import the `datanode` in the vega repo
- [5660](https://github.com/vegaprotocol/vega/issues/5660) - Migrate subscription `apis` from `datanode v1 api` to `datanode v2 api`
- [5636](https://github.com/vegaprotocol/vega/issues/5636) - Assure no false positives in cucumber steps
- [5011](https://github.com/vegaprotocol/vega/issues/5011) - Import the `protos` repo in the vega repo
- [5774](https://github.com/vegaprotocol/vega/issues/5774) - Use `generics` for `ID` types
- [5785](https://github.com/vegaprotocol/vega/issues/5785) - Add support form `ERC20` bridge stopped and resumed events
- [5712](https://github.com/vegaprotocol/vega/issues/5712) - Configurable `graphql` endpoint
- [5689](https://github.com/vegaprotocol/vega/issues/5689) - Support `UpdateAsset` proposal in APIs
- [5685](https://github.com/vegaprotocol/vega/issues/5685) - Migrated `apis` from `datanode v1` to `datanode v2`
- [5760](https://github.com/vegaprotocol/vega/issues/5760) - Map all `GRPC` to `REST`
- [5804](https://github.com/vegaprotocol/vega/issues/5804) - Rollback Tendermint to version `0.34.20`
- [5503](https://github.com/vegaprotocol/vega/issues/5503) - Introduce wallet API version 2 based on JSON-RPC with new authentication workflow
- [5822](https://github.com/vegaprotocol/vega/issues/5822) - Rename `Graphql` enums
- [5618](https://github.com/vegaprotocol/vega/issues/5618) - Add wallet JSON-RPC documentation
- [5776](https://github.com/vegaprotocol/vega/issues/5776) - Add endpoint to get a single network parameter
- [5685](https://github.com/vegaprotocol/vega/issues/5685) - Migrated `apis` from `datanode v1` to `datanode v2`
- [5761](https://github.com/vegaprotocol/vega/issues/5761) - Transfers connection make direction optional
- [5762](https://github.com/vegaprotocol/vega/issues/5762) - Transfers connection add under `party` type
- [5685](https://github.com/vegaprotocol/vega/issues/5685) - Migrated `apis` from `datanode v1` to `datanode v2`
- [5705](https://github.com/vegaprotocol/vega/issues/5705) - Use enum for validator status
- [5685](https://github.com/vegaprotocol/vega/issues/5685) - Migrated `apis` from `datanode v1` to `datanode v2`
- [5834](https://github.com/vegaprotocol/vega/issues/5834) - Avoid saving proposals of terminated/cancelled/rejected/settled markets in checkpoint
- [5619](https://github.com/vegaprotocol/vega/issues/5619) - Add wallet HTTP API version 2 documentation
- [5823](https://github.com/vegaprotocol/vega/issues/5823) - Add endpoint to wallet HTTP API version 2 to list available RPC methods
- [5814](https://github.com/vegaprotocol/vega/issues/5815) - Add proposal validation date time to `graphql`
- [5865](https://github.com/vegaprotocol/vega/issues/5865) - Allow a validator to withdraw their protocol upgrade proposal
- [5803](https://github.com/vegaprotocol/vega/issues/5803) - Update cursor pagination to use new method from [5784](https://github.com/vegaprotocol/vega/pull/5784)
- [5862](https://github.com/vegaprotocol/vega/issues/5862) - Add base `URL` in `swagger`
- [5817](https://github.com/vegaprotocol/vega/issues/5817) - Add validation error on asset proposal when rejected
- [5816](https://github.com/vegaprotocol/vega/issues/5816) - Set proper status to rejected asset proposal
- [5893](https://github.com/vegaprotocol/vega/issues/5893) - Remove total supply from assets
- [5752](https://github.com/vegaprotocol/vega/issues/5752) - Remove URL and Hash from proposal rationale, add Title
- [5802](https://github.com/vegaprotocol/vega/issues/5802) - Introduce transaction version 3 that encode the chain ID in its input data to protect against transaction replay
- [5358](https://github.com/vegaprotocol/vega/issues/5358) - Port equity like shares update to new structure
- [5926](https://github.com/vegaprotocol/vega/issues/5926) - Check for liquidity auction at the end of a block instead of after every trade

### 🐛 Fixes

- [5571](https://github.com/vegaprotocol/vega/issues/5571) - Restore pending assets status correctly after snapshot restore
- [5857](https://github.com/vegaprotocol/vega/issues/5857) - Fix panic when calling `ListAssets` `grpc` end point with no arguments
- [5572](https://github.com/vegaprotocol/vega/issues/5572) - Add validation on `IDs` and public keys
- [5348](https://github.com/vegaprotocol/vega/issues/5348) - Restore markets from checkpoint proposal
- [5279](https://github.com/vegaprotocol/vega/issues/5279) - Fix loading of proposals from checkpoint
- [5598](https://github.com/vegaprotocol/vega/issues/5598) - Remove `currentTime` from topology engine to ease snapshot restoration
- [5836](https://github.com/vegaprotocol/vega/issues/5836) - Add missing `GetMarket` `GRPC` end point
- [5609](https://github.com/vegaprotocol/vega/issues/5609) - Set event forwarder last seen height after snapshot restore
- [5782](https://github.com/vegaprotocol/vega/issues/5782) - `Pagination` with a cursor was returning incorrect results
- [5629](https://github.com/vegaprotocol/vega/issues/5629) - Fixes for loading voting power from checkpoint with non genesis validators
- [5626](https://github.com/vegaprotocol/vega/issues/5626) - Update `protos`, remove optional types
- [5665](https://github.com/vegaprotocol/vega/issues/5665) - Binary version hash always contain `-modified` suffix
- [5633](https://github.com/vegaprotocol/vega/issues/5633) - Allow `minProposerEquityLikeShare` to accept 0
- [5672](https://github.com/vegaprotocol/vega/issues/5672) - Typo fixed in datanode `ethereum` address
- [5863](https://github.com/vegaprotocol/vega/issues/5863) - Fix panic when calling `VegaTime` on `v2 api`
- [5683](https://github.com/vegaprotocol/vega/issues/5683) - Made market mandatory in `GraphQL` for order
- [5789](https://github.com/vegaprotocol/vega/issues/5789) - Fix performance issue with position query
- [5677](https://github.com/vegaprotocol/vega/issues/5677) - Fixed trading mode status
- [5663](https://github.com/vegaprotocol/vega/issues/5663) - Fixed panic with de-registering positions
- [5781](https://github.com/vegaprotocol/vega/issues/5781) - Make enactment timestamp optional in proposal for `graphql`
- [5767](https://github.com/vegaprotocol/vega/issues/5767) - Fix typo in command validation
- [5900](https://github.com/vegaprotocol/vega/issues/5900) - Add missing `/api/v2/parties/{party_id}/stake` `REST` end point
- [5825](https://github.com/vegaprotocol/vega/issues/5825) - Fix panic in pegged orders when going into auction
- [5763](https://github.com/vegaprotocol/vega/issues/5763) - Transfers connection rename `pubkey` to `partyId`
- [5486](https://github.com/vegaprotocol/vega/issues/5486) - Fix amend order expiring
- [5809](https://github.com/vegaprotocol/vega/issues/5809) - Remove state variables when a market proposal is rejected
- [5329](https://github.com/vegaprotocol/vega/issues/5329) - Fix checks for market enactment and termination
- [5837](https://github.com/vegaprotocol/vega/issues/5837) - Allow a promotion due to increased slots and a swap to happen in the same epoch
- [5819](https://github.com/vegaprotocol/vega/issues/5819) - Add new asset proposal validation timestamp validation
- [5897](https://github.com/vegaprotocol/vega/issues/5897) - Return uptime of 0, rather than error, when querying for `NodeData` before end of first epoch
- [5811](https://github.com/vegaprotocol/vega/issues/5811) - Do not overwrite local changes when updating wallet through JSON-RPC API
- [5868](https://github.com/vegaprotocol/vega/issues/5868) - Clarify the error for insufficient token to submit proposal or vote
- [5867](https://github.com/vegaprotocol/vega/issues/5867) - Fix witness check for majority
- [5853](https://github.com/vegaprotocol/vega/issues/5853) - Do not ignore market update proposals when loading from checkpoint
- [5648](https://github.com/vegaprotocol/vega/issues/5648) - Ethereum key rotation - search validators by Vega pub key and listen to rotation events in core API
- [5648](https://github.com/vegaprotocol/vega/issues/5648) - Search validator by vega pub key and update the core validators API

## 0.53.0

### 🗑️ Deprecation

- [5513](https://github.com/vegaprotocol/vega/issues/5513) - Remove all checkpoint restore command

### 🛠 Improvements

- [5428](https://github.com/vegaprotocol/vega/pull/5428) - Update contributor information
- [5519](https://github.com/vegaprotocol/vega/pull/5519) - Add `--genesis-file` option to the `load_checkpoint` command
- [5538](https://github.com/vegaprotocol/vega/issues/5538) - Core side implementation of protocol upgrade
- [5525](https://github.com/vegaprotocol/vega/pull/5525) - Release `vegawallet` from the core
- [5524](https://github.com/vegaprotocol/vega/pull/5524) - Align `vegawallet` and core versions
- [5524](https://github.com/vegaprotocol/vega/pull/5549) - Add endpoint for getting the network's `chain-id`
- [5524](https://github.com/vegaprotocol/vega/pull/5552) - Handle tendermint demotion and `ersatz` slot reduction at the same time

### 🐛 Fixes

- [5476](https://github.com/vegaprotocol/vega/issues/5476) - Include settlement price in snapshot
- [5476](https://github.com/vegaprotocol/vega/issues/5314) - Fix validation of checkpoint file
- [5499](https://github.com/vegaprotocol/vega/issues/5499) - Add error from app specific validation to check transaction response
- [5508](https://github.com/vegaprotocol/vega/issues/5508) - Fix duplicated staking events
- [5514](https://github.com/vegaprotocol/vega/issues/5514) - Emit `rewardScore` event correctly after loading from checkpoint
- [5520](https://github.com/vegaprotocol/vega/issues/5520) - Do not fail silently when wallet fails to start
- [5521](https://github.com/vegaprotocol/vega/issues/5521) - Fix asset bundle and add asset status
- [5546](https://github.com/vegaprotocol/vega/issues/5546) - Fix collateral checkpoint to unlock locked reward account balance
- [5194](https://github.com/vegaprotocol/vega/issues/5194) - Fix market trading mode vs market state
- [5432](https://github.com/vegaprotocol/vega/issues/5431) - Do not accept transaction with unexpected public keys
- [5478](https://github.com/vegaprotocol/vega/issues/5478) - Assure uncross and fake uncross are in line with each other
- [5480](https://github.com/vegaprotocol/vega/issues/5480) - Assure indicative trades are in line with actual uncrossing trades
- [5556](https://github.com/vegaprotocol/vega/issues/5556) - Fix id generation seed
- [5361](https://github.com/vegaprotocol/vega/issues/5361) - Fix limits for proposals
- [5557](https://github.com/vegaprotocol/vega/issues/5427) - Fix oracle status at market settlement

## 0.52.0

### 🛠 Improvements

- [5421](https://github.com/vegaprotocol/vega/issues/5421) - Fix notary snapshot determinism when no signature are generated yet
- [5415](https://github.com/vegaprotocol/vega/issues/5415) - Regenerate smart contracts code
- [5434](https://github.com/vegaprotocol/vega/issues/5434) - Add health check for faucet
- [5412](https://github.com/vegaprotocol/vega/issues/5412) - Proof of work improvement to support history of changes to network parameters
- [5378](https://github.com/vegaprotocol/vega/issues/5278) - Allow new market proposals without LP

### 🐛 Fixes

- [5438](https://github.com/vegaprotocol/vega/issues/5438) - Evaluate all trades resulting from an aggressive orders in one call to price monitoring engine
- [5444](https://github.com/vegaprotocol/vega/issues/5444) - Merge both checkpoints and genesis asset on startup
- [5446](https://github.com/vegaprotocol/vega/issues/5446) - Cover liquidity monitoring acceptance criteria relating to aggressive order removing best bid or ask from the book
- [5457](https://github.com/vegaprotocol/vega/issues/5457) - Fix sorting of validators for demotion check
- [5460](https://github.com/vegaprotocol/vega/issues/5460) - Fix theoretical open interest calculation
- [5477](https://github.com/vegaprotocol/vega/issues/5477) - Pass a clone of the liquidity commitment offset to pegged orders
- [5468](https://github.com/vegaprotocol/vega/issues/5468) - Bring indicative trades inline with actual auction uncrossing trades in presence of wash trades
- [5419](https://github.com/vegaprotocol/vega/issues/5419) - Fix listeners ordering and state updates

## 0.51.1

### 🛠 Improvements

- [5395](https://github.com/vegaprotocol/vega/issues/5395) - Add `burn_nonce` bridge tool
- [5403](https://github.com/vegaprotocol/vega/issues/5403) - Allow spam free / proof of work free running of null blockchain
- [5175](https://github.com/vegaprotocol/vega/issues/5175) - Validation free transactions (including signature verification) for null blockchain
- [5371](https://github.com/vegaprotocol/vega/issues/5371) - Ensure threshold is not breached in ERC20 withdrawal
- [5358](https://github.com/vegaprotocol/vega/issues/5358) - Update equity shares following updated spec.

### 🐛 Fixes

- [5362](https://github.com/vegaprotocol/vega/issues/5362) - Liquidity and order book point to same underlying order after restore
- [5367](https://github.com/vegaprotocol/vega/issues/5367) - better serialisation for party orders in liquidity snapshot
- [5377](https://github.com/vegaprotocol/vega/issues/5377) - Serialise state var internal state
- [5388](https://github.com/vegaprotocol/vega/issues/5388) - State variable snapshot now works as intended
- [5388](https://github.com/vegaprotocol/vega/issues/5388) - Repopulate cached order-book after snapshot restore
- [5203](https://github.com/vegaprotocol/vega/issues/5203) - Market liquidity monitor parameters trump network parameters on market creation
- [5297](https://github.com/vegaprotocol/vega/issues/5297) - Assure min/max price always accurate
- [4223](https://github.com/vegaprotocol/vega/issues/4223) - Use uncrossing price for target stake calculation during auction
- [3047](https://github.com/vegaprotocol/vega/issues/3047) - Improve interaction between liquidity and price monitoring auctions
- [3570](https://github.com/vegaprotocol/vega/issues/3570) - Set extension trigger during opening auction with insufficient liquidity
- [3362](https://github.com/vegaprotocol/vega/issues/3362) - Stop non-persistent orders from triggering auctions
- [5388](https://github.com/vegaprotocol/vega/issues/5388) - Use `UnixNano()` to snapshot price monitor times
- [5237](https://github.com/vegaprotocol/vega/issues/5237) - Trigger state variable calculation first time indicative uncrossing price is available
- [5397](https://github.com/vegaprotocol/vega/issues/5397) - Bring indicative trades price inline with that of actual auction uncrossing trades

## 0.51.0

### 🚨 Breaking changes

- [5192](https://github.com/vegaprotocol/vega/issues/5192) - Require a rationale on proposals

### 🛠 Improvements

- [5318](https://github.com/vegaprotocol/vega/issues/5318) - Automatically dispatch reward pool into markets in recurring transfers
- [5333](https://github.com/vegaprotocol/vega/issues/5333) - Run snapshot generation for all providers in parallel
- [5343](https://github.com/vegaprotocol/vega/issues/5343) - Snapshot optimisation part II - get rid of `getHash`
- [5324](https://github.com/vegaprotocol/vega/issues/5324) - Send event when oracle data doesn't match
- [5140](https://github.com/vegaprotocol/vega/issues/5140) - Move limits (enabled market / assets from) to network parameters
- [5360](https://github.com/vegaprotocol/vega/issues/5360) - rewards test coverage

### 🐛 Fixes

- [5338](https://github.com/vegaprotocol/vega/issues/5338) - Checking a transaction should return proper success code
- [5277](https://github.com/vegaprotocol/vega/issues/5277) - Updating a market should default auction extension to 1
- [5284](https://github.com/vegaprotocol/vega/issues/5284) - price monitoring past prices are now included in the snapshot
- [5294](https://github.com/vegaprotocol/vega/issues/5294) - Parse timestamps oracle in market proposal validation
- [5292](https://github.com/vegaprotocol/vega/issues/5292) - Internal time oracle broadcasts timestamp without nanoseconds
- [5297](https://github.com/vegaprotocol/vega/issues/5297) - Assure min/max price always accurate
- [5286](https://github.com/vegaprotocol/vega/issues/5286) - Ensure liquidity fees are updated when updating the market
- [5322](https://github.com/vegaprotocol/vega/issues/5322) - Change vega pub key hashing in topology to fix key rotation submission.
- [5313](https://github.com/vegaprotocol/vega/issues/5313) - Future update was using oracle spec for settlement price as trading termination spec
- [5304](https://github.com/vegaprotocol/vega/issues/5304) - Fix bug causing trade events at auction end showing the wrong price.
- [5345](https://github.com/vegaprotocol/vega/issues/5345) - Fix issue with state variable transactions assumed gone missing
- [5351](https://github.com/vegaprotocol/vega/issues/5351) - Fix panic when node is interrupted before snapshot engine gets cleared and initialised
- [5972](https://github.com/vegaprotocol/vega/issues/5972) - Allow submitting a market update with termination oracle ticking before enactment of the update

## 0.50.2

### 🛠 Improvements

- [5001](https://github.com/vegaprotocol/vega/issues/5001) - Set and increment LP version field correctly
- [5001](https://github.com/vegaprotocol/vega/issues/5001) - Add integration test for LP versioning
- [3372](https://github.com/vegaprotocol/vega/issues/3372) - Add integration test making sure margin is released when an LP is cancelled.
- [5235](https://github.com/vegaprotocol/vega/issues/5235) - Use `BroadcastTxSync` instead of async for submitting transactions to `tendermint`
- [5268](https://github.com/vegaprotocol/vega/issues/5268) - Make validator heartbeat frequency a function of the epoch duration.
- [5271](https://github.com/vegaprotocol/vega/issues/5271) - Make generated hex IDs lower case
- [5273](https://github.com/vegaprotocol/vega/issues/5273) - Reward / Transfer to allow payout of reward in an arbitrary asset unrelated to the settlement and by market.
- [5207](https://github.com/vegaprotocol/vega/issues/5206) - Add integration tests to ensure price bounds and decimal places work as expected
- [5243](https://github.com/vegaprotocol/vega/issues/5243) - Update equity like share according to spec changes.
- [5249](https://github.com/vegaprotocol/vega/issues/5249) - Upgrade to tendermint 0.35.6

### 🐛 Fixes

- [4798](https://github.com/vegaprotocol/vega/issues/4978) - Set market pending timestamp to the time at which the market is created.
- [5222](https://github.com/vegaprotocol/vega/issues/5222) - Do not panic when admin server stops.
- [5103](https://github.com/vegaprotocol/vega/issues/5103) - Fix invalid http status set in faucet
- [5239](https://github.com/vegaprotocol/vega/issues/5239) - Always call `StartAggregate()` when signing validators joining and leaving even if not a validator
- [5128](https://github.com/vegaprotocol/vega/issues/5128) - Fix wrong http rate limit for faucet
- [5231](https://github.com/vegaprotocol/vega/issues/5231) - Fix pegged orders to be reset to the order pointer after snapshot loading
- [5247](https://github.com/vegaprotocol/vega/issues/5247) - Fix the check for overflow in scaling settlement price
- [5250](https://github.com/vegaprotocol/vega/issues/5250) - Fixed panic in loading validator checkpoint
- [5260](https://github.com/vegaprotocol/vega/issues/5260) - Process recurring transfer before rewards
- [5262](https://github.com/vegaprotocol/vega/issues/5262) - Allow recurring transfers to start during the current epoch
- [5267](https://github.com/vegaprotocol/vega/issues/5267) - Do not check commitment on `UpdateMarket` proposals

## 0.50.1

### 🐛 Fixes

- [5226](https://github.com/vegaprotocol/vega/issues/5226) - Add support for settlement price decimal place in governance

## 0.50.0

### 🚨 Breaking changes

- [5197](https://github.com/vegaprotocol/vega/issues/5197) - Scale settlement price based on oracle definition

### 🛠 Improvements

- [5055](https://github.com/vegaprotocol/vega/issues/5055) - Ensure at most 5 triggers are used in price monitoring settings
- [5100](https://github.com/vegaprotocol/vega/issues/5100) - add a new scenario into feature test, auction folder, leaving auction when liquidity provider provides a limit order
- [4919](https://github.com/vegaprotocol/vega/issues/4919) - Feature tests for 0011 check order allocate margin
- [4922](https://github.com/vegaprotocol/vega/issues/4922) - Feature tests for 0015 market insurance pool collateral
- [4926](https://github.com/vegaprotocol/vega/issues/4926) - Feature tests for 0019 margin calculator scenarios
- [5119](https://github.com/vegaprotocol/vega/issues/5119) - Add Ethereum key rotation support
- [5209](https://github.com/vegaprotocol/vega/issues/5209) - Add retries to floating point consensus engine to work around tendermint missing transactions
- [5219](https://github.com/vegaprotocol/vega/issues/5219) - Remove genesis sign command.

### 🐛 Fixes

- [5078](https://github.com/vegaprotocol/vega/issues/5078) - Unwrap properly position decimal place from payload
- [5076](https://github.com/vegaprotocol/vega/issues/5076) - Set last mark price to settlement price when market is settled
- [5038](https://github.com/vegaprotocol/vega/issues/5038) - Send proof-of-work when when announcing node
- [5034](https://github.com/vegaprotocol/vega/issues/5034) - Ensure to / from in transfers payloads are vega public keys
- [5111](https://github.com/vegaprotocol/vega/issues/5111) - Stop updating the market's initial configuration when an opening auction is extended
- [5066](https://github.com/vegaprotocol/vega/issues/5066) - Return an error if market decimal place > to asset decimal place
- [5095](https://github.com/vegaprotocol/vega/issues/5095) - Stabilise state sync restore and restore block height in the topology engine
- [5204](https://github.com/vegaprotocol/vega/issues/5204) - Mark a snapshot state change when liquidity provision state changes
- [4870](https://github.com/vegaprotocol/vega/issues/5870) - Add missing commands to the `TxError` event
- [5136](https://github.com/vegaprotocol/vega/issues/5136) - Fix banking snapshot for transfers, risk factor restoration, and `statevar` handling of settled markets
- [5088](https://github.com/vegaprotocol/vega/issues/5088) - Fixed MTM bug where settlement balance would not be zero when loss amount was 1.
- [5093](https://github.com/vegaprotocol/vega/issues/5093) - Fixed proof of engine end of block callback never called to clear up state
- [4996](https://github.com/vegaprotocol/vega/issues/4996) - Fix positions engines `vwBuys` and `vwSell` when amending, send events on `Update` and `UpdateNetwork`
- [5016](https://github.com/vegaprotocol/vega/issues/5016) - Target stake in asset decimal place in Market Data
- [5109](https://github.com/vegaprotocol/vega/issues/5109) - Fixed promotion of ersatz to tendermint validator
- [5110](https://github.com/vegaprotocol/vega/issues/5110) - Fixed wrong tick size used for calculating probability of trading
- [5144](https://github.com/vegaprotocol/vega/issues/5144) - Fixed the default voting power in case there is stake in the network
- [5124](https://github.com/vegaprotocol/vega/issues/5124) - Add proto serialization for update market proposal
- [5124](https://github.com/vegaprotocol/vega/issues/5124) - Ensure update market proposal compute a proper auction duration
- [5172](https://github.com/vegaprotocol/vega/issues/5172) - Add replay protection for validator commands
- [5181](https://github.com/vegaprotocol/vega/issues/5181) - Ensure Oracle specs handle numbers using `num.Decimal` and `num.Int`
- [5059](https://github.com/vegaprotocol/vega/issues/5059) - Validators without tendermint status vote in the witness and notary engine but their votes do not count
- [5190](https://github.com/vegaprotocol/vega/issues/5190) - Fix settlement at expiry to scale the settlement price from market decimals to asset decimals
- [5185](https://github.com/vegaprotocol/vega/issues/5185) - Fix MTM settlement where win transfers get truncated resulting in settlement balance not being zero after settlement.
- [4943](https://github.com/vegaprotocol/vega/issues/4943) - Fix bug where amending orders in opening auctions did not work as expected

## 0.49.8

### 🛠 Improvements

- [4814](https://github.com/vegaprotocol/vega/issues/4814) - Review fees tests
- [5067](https://github.com/vegaprotocol/vega/pull/5067) - Adding acceptance codes and tidy up tests
- [5052](https://github.com/vegaprotocol/vega/issues/5052) - Adding acceptance criteria tests for market decimal places
- [5138](https://github.com/vegaprotocol/vega/issues/5038) - Adding feature test for "0032-PRIM-price_monitoring.md"
- [4753](https://github.com/vegaprotocol/vega/issues/4753) - Adding feature test for oracle spec public key validation
- [4559](https://github.com/vegaprotocol/vega/issues/4559) - Small fixes to the amend order flow

### 🐛 Fixes

- [5064](https://github.com/vegaprotocol/vega/issues/5064) - Send order event on settlement
- [5068](https://github.com/vegaprotocol/vega/issues/5068) - Use settlement price if exists when received trading terminated event

## 0.49.7

### 🚨 Breaking changes

- [4985](https://github.com/vegaprotocol/vega/issues/4985) - Proof of work spam protection

### 🛠 Improvements

- [5007](https://github.com/vegaprotocol/vega/issues/5007) - Run approbation as part of the CI pipeline
- [5019](https://github.com/vegaprotocol/vega/issues/5019) - Label Price Monitoring tests
- [5022](https://github.com/vegaprotocol/vega/issues/5022) - CI: Run approbation for main/master/develop branches only
- [5017](https://github.com/vegaprotocol/vega/issues/5017) - Added access functions to `PositionState` type
- [5049](https://github.com/vegaprotocol/vega/issues/5049) - Liquidity Provision test coverage for 0034 spec
- [5022](https://github.com/vegaprotocol/vega/issues/5022) - CI: Run approbation for main/master/develop branches
  only
- [4916](https://github.com/vegaprotocol/vega/issues/4916) - Add acceptance criteria number in the existing feature tests to address acceptance criteria in `0008-TRAD-trading_workflow.md`
- [5061](https://github.com/vegaprotocol/vega/issues/5061) - Add a test scenario using log normal risk model into feature test "insurance-pool-balance-test.feature"

### 🐛 Fixes

- [5025](https://github.com/vegaprotocol/vega/issues/5025) - Witness snapshot breaking consensus
- [5046](https://github.com/vegaprotocol/vega/issues/5046) - Save all events in `ERC20` topology snapshot

## 0.49.4

### 🛠 Improvements

- [2585](https://github.com/vegaprotocol/vega/issues/2585) - Adding position state event to event bus
- [4952](https://github.com/vegaprotocol/vega/issues/4952) - Add checkpoints for staking and `multisig control`
- [4923](https://github.com/vegaprotocol/vega/issues/4923) - Add checkpoint state in the genesis file + add subcommand to do it.

### 🐛 Fixes

- [4983](https://github.com/vegaprotocol/vega/issues/4983) - Set correct event type for positions state event
- [4989](https://github.com/vegaprotocol/vega/issues/4989) - Fixing incorrect overflow logic
- [5036](https://github.com/vegaprotocol/vega/issues/5036) - Fix the `nullblockchain`
- [4981](https://github.com/vegaprotocol/vega/issues/4981) - Fix bug causing LP orders to uncross at auction end.

## 0.49.2

### 🛠 Improvements

- [4951](https://github.com/vegaprotocol/vega/issues/4951) - Add ability to stream events to a file
- [4953](https://github.com/vegaprotocol/vega/issues/4953) - Add block hash to statistics and to block height request
- [4961](https://github.com/vegaprotocol/vega/issues/4961) - Extend auction feature tests
- [4832](https://github.com/vegaprotocol/vega/issues/4832) - Add validation of update market proposals.
- [4971](https://github.com/vegaprotocol/vega/issues/4971) - Add acceptance criteria to auction tests
- [4833](https://github.com/vegaprotocol/vega/issues/4833) - Propagate market update to other engines

### 🐛 Fixes

- [4947](https://github.com/vegaprotocol/vega/issues/4947) - Fix time formatting problem that was breaking consensus on nodes in different time zones
- [4956](https://github.com/vegaprotocol/vega/issues/4956) - Fix concurrent write to price monitoring ref price cache
- [4987](https://github.com/vegaprotocol/vega/issues/4987) - Include the witness engine in snapshots
- [4957](https://github.com/vegaprotocol/vega/issues/4957) - Fix `vega announce_node` to work with `--home` and `--passphrase-file`
- [4964](https://github.com/vegaprotocol/vega/issues/4964) - Fix price monitoring snapshot
- [4974](https://github.com/vegaprotocol/vega/issues/4974) - Fix panic when checkpointing staking accounts if there are no events
- [4888](https://github.com/vegaprotocol/vega/issues/4888) - Fix memory leak when loading snapshots.
- [4993](https://github.com/vegaprotocol/vega/issues/4993) - Stop snapshots thinking we've loaded via `statesync` when we just lost connection to TM
- [4981](https://github.com/vegaprotocol/vega/issues/4981) - Fix bug causing LP orders to uncross at auction end.

## 0.49.1

### 🛠 Improvements

- [4895](https://github.com/vegaprotocol/vega/issues/4895) - Emit validators signature when a validator is added or remove from the set
- [4901](https://github.com/vegaprotocol/vega/issues/4901) - Update the decimal library
- [4906](https://github.com/vegaprotocol/vega/issues/4906) - Get rid of unnecessary `ToDecimal` conversions (no functional change)
- [4838](https://github.com/vegaprotocol/vega/issues/4838) - Implement governance vote based on equity-like share for market update

### 🐛 Fixes

- [4877](https://github.com/vegaprotocol/vega/issues/4877) - Fix topology and `erc20` topology snapshots
- [4890](https://github.com/vegaprotocol/vega/issues/4890) - epoch service now notifies other engines when it has restored from a snapshot
- [4879](https://github.com/vegaprotocol/vega/issues/4879) - Fixes for invalid data types in the `MarketData` proto message.
- [4881](https://github.com/vegaprotocol/vega/issues/4881) - Set tendermint validators' voting power when loading from snapshot
- [4915](https://github.com/vegaprotocol/vega/issues/4915) - Take full snapshot of collateral engine, always read epoch length from network parameters, use back-off on heartbeats
- [4882](https://github.com/vegaprotocol/vega/issues/4882) - Fixed tracking of liquidity fee received and added feature tests for the fee based rewards
- [4898](https://github.com/vegaprotocol/vega/issues/4898) - Add ranking score information to checkpoint and snapshot and emit an event when loaded
- [4932](https://github.com/vegaprotocol/vega/issues/4932) - Fix the string used for resource id of stake total supply to be stable to fix the replay of non validator node locally

## 0.49.0

### 🚨 Breaking changes

- [4900](https://github.com/vegaprotocol/vega/issues/4809) - Review LP fee tests, and move to VEGA repo
- [4844](https://github.com/vegaprotocol/vega/issues/4844) - Add endpoints for checking transactions raw transactions
- [4515](https://github.com/vegaprotocol/vega/issues/4615) - Add snapshot options description and check provided storage method
- [4581](https://github.com/vegaprotocol/vega/issues/4561) - Separate endpoints for liquidity provision submissions, amendment and cancellation
- [4390](https://github.com/vegaprotocol/vega/pull/4390) - Introduce node mode, `vega init` now require a mode: full or validator
- [4383](https://github.com/vegaprotocol/vega/pull/4383) - Rename flag `--tm-root` to `--tm-home`
- [4588](https://github.com/vegaprotocol/vega/pull/4588) - Remove the outdated `--network` flag on `vega genesis generate` and `vega genesis update`
- [4605](https://github.com/vegaprotocol/vega/pull/4605) - Use new format for `EthereumConfig` in network parameters.
- [4508](https://github.com/vegaprotocol/vega/pull/4508) - Disallow negative offset for pegged orders
- [4465](https://github.com/vegaprotocol/vega/pull/4465) - Update to tendermint `v0.35.0`
- [4594](https://github.com/vegaprotocol/vega/issues/4594) - Add support for decimal places specific to markets. This means market price values and position events can have different values. Positions will be expressed in asset decimal places, market specific data events will list prices in market precision.
- [4660](https://github.com/vegaprotocol/vega/pull/4660) - Add tendermint transaction hash to events
- [4670](https://github.com/vegaprotocol/vega/pull/4670) - Rework `freeform proposal` structure so that they align with other proposals
- [4681](https://github.com/vegaprotocol/vega/issues/4681) - Remove tick size from market
- [4698](https://github.com/vegaprotocol/vega/issues/4698) - Remove maturity field from future
- [4699](https://github.com/vegaprotocol/vega/issues/4699) - Remove trading mode one off from market proposal
- [4790](https://github.com/vegaprotocol/vega/issues/4790) - Fix core to work with `protos` generated by newer versions of `protoc-gen-xxx`
- [4856](https://github.com/vegaprotocol/vega/issues/4856) - Fractional orders and positions

### 🗑️ Deprecation

### 🛠 Improvements

- [4793](https://github.com/vegaprotocol/vega/issues/4793) - Add specific insurance pool balance test
- [4633](https://github.com/vegaprotocol/vega/pull/4633) - Add possibility to list snapshots from the vega command line
- [4640](https://github.com/vegaprotocol/vega/pull/4640) - Update feature tests related to liquidity provision following integration of probability of trading with floating point consensus
- [4558](https://github.com/vegaprotocol/vega/pull/4558) - Add MacOS install steps and information required to use `dockerisedvega.sh` script with private docker repository
- [4496](https://github.com/vegaprotocol/vega/pull/4496) - State variable engine for floating point consensus
- [4481](https://github.com/vegaprotocol/vega/pull/4481) - Add an example client application that uses the null-blockchain
- [4514](https://github.com/vegaprotocol/vega/pull/4514) - Add network limits service and events
- [4516](https://github.com/vegaprotocol/vega/pull/4516) - Add a command to cleanup all vega node state
- [4531](https://github.com/vegaprotocol/vega/pull/4531) - Remove Float from network parameters, use `num.Decimal` instead
- [4537](https://github.com/vegaprotocol/vega/pull/4537) - Send staking asset total supply through consensus
- [4540](https://github.com/vegaprotocol/vega/pull/4540) - Require Go minimum version 1.17
- [4530](https://github.com/vegaprotocol/vega/pull/4530) - Integrate risk factors with floating point consensus engine
- [4485](https://github.com/vegaprotocol/vega/pull/4485) - Change snapshot interval default to 1000 blocks
- [4505](https://github.com/vegaprotocol/vega/pull/4505) - Fast forward epochs when loading from checkpoint to trigger payouts for the skipped time
- [4554](https://github.com/vegaprotocol/vega/pull/4554) - Integrate price ranges with floating point consensus engine
- [4544](https://github.com/vegaprotocol/vega/pull/4544) - Ensure validators are started with the right set of keys
- [4569](https://github.com/vegaprotocol/vega/pull/4569) - Move to `ghcr.io` docker container registry
- [4571](https://github.com/vegaprotocol/vega/pull/4571) - Update `CHANGELOG.md` for `0.47.x`
- [4577](https://github.com/vegaprotocol/vega/pull/4577) - Update `CHANGELOG.md` for `0.45.6` patch
- [4573](https://github.com/vegaprotocol/vega/pull/4573) - Remove execution configuration duplication from configuration root
- [4555](https://github.com/vegaprotocol/vega/issues/4555) - Probability of trading integrated into floating point consensus engine
- [4592](https://github.com/vegaprotocol/vega/pull/4592) - Update instructions on how to use docker without `sudo`
- [4491](https://github.com/vegaprotocol/vega/issues/4491) - Measure validator performance and use to penalise rewards
- [4599](https://github.com/vegaprotocol/vega/pull/4599) - Allow raw private keys for bridges functions
- [4588](https://github.com/vegaprotocol/vega/pull/4588) - Add `--update` and `--replace` flags on `vega genesis new validator`
- [4522](https://github.com/vegaprotocol/vega/pull/4522) - Add `--network-url` option to `vega tm`
- [4580](https://github.com/vegaprotocol/vega/pull/4580) - Add transfer command support (one off transfers)
- [4636](https://github.com/vegaprotocol/vega/pull/4636) - Update the Core Team DoD and issue templates
- [4629](https://github.com/vegaprotocol/vega/pull/4629) - Update `CHANGELOG.md` to include `0.47.5` changes
- [4580](https://github.com/vegaprotocol/vega/pull/4580) - Add transfer command support (recurring transfers)
- [4643](https://github.com/vegaprotocol/vega/issues/4643) - Add noise to floating point consensus variables in QA
- [4639](https://github.com/vegaprotocol/vega/pull/4639) - Add cancel transfer command
- [4750](https://github.com/vegaprotocol/vega/pull/4750) - Fix null blockchain by forcing it to always be a non-validator node
- [4754](https://github.com/vegaprotocol/vega/pull/4754) - Fix null blockchain properly this time
- [4754](https://github.com/vegaprotocol/vega/pull/4754) - Remove old id generator fields from execution engine's snapshot
- [4830](https://github.com/vegaprotocol/vega/pull/4830) - Reward refactoring for network treasury
- [4647](https://github.com/vegaprotocol/vega/pull/4647) - Added endpoint `SubmitRawTransaction` to provide support for different transaction request message versions
- [4653](https://github.com/vegaprotocol/vega/issues/4653) - Replace asset insurance pool with network treasury
- [4638](https://github.com/vegaprotocol/vega/pull/4638) - CI add option to specify connected changes in other repos
- [4650](https://github.com/vegaprotocol/vega/pull/4650) - Restore code from rebase and ensure node retries connection with application
- [4570](https://github.com/vegaprotocol/vega/pull/4570) - Internalize Ethereum Event Forwarder
- [4663](https://github.com/vegaprotocol/vega/issues/4663) - CI set `qa` build tag when running system tests
- [4709](https://github.com/vegaprotocol/vega/issues/4709) - Make `BlockNr` part of event interface
- [4657](https://github.com/vegaprotocol/vega/pull/4657) - Rename `min_lp_stake` to quantum + use it in liquidity provisions
- [4672](https://github.com/vegaprotocol/vega/issues/4672) - Update Jenkinsfile
- [4712](https://github.com/vegaprotocol/vega/issues/4712) - Check smart contract hash on startup to ensure the correct version is being used
- [4594](https://github.com/vegaprotocol/vega/issues/4594) - Add integration test ensuring positions plug-in calculates P&L accurately.
- [4689](https://github.com/vegaprotocol/vega/issues/4689) - Validators joining and leaving the network
- [4680](https://github.com/vegaprotocol/vega/issues/4680) - Add `totalTokenSupplyStake` to the snapshots
- [4645](https://github.com/vegaprotocol/vega/pull/4645) - Add transfers snapshots
- [4707](https://github.com/vegaprotocol/vega/pull/4707) - Serialize timestamp in time update message as number of nano seconds instead of seconds
- [4595](https://github.com/vegaprotocol/vega/pull/4595) - Add internal oracle supplying vega time data for time-triggered events
- [4737](https://github.com/vegaprotocol/vega/pull/4737) - Use a deterministic generator for order ids, set new order ids to the transaction hash of the Submit transaction
- [4741](https://github.com/vegaprotocol/vega/pull/4741) - Hash again list of hash from engines
- [4751](https://github.com/vegaprotocol/vega/pull/4751) - Make trade ids unique using the deterministic id generator
- [4766](https://github.com/vegaprotocol/vega/issues/4766) - Added feature tests and integration steps for transfers
- [4771](https://github.com/vegaprotocol/vega/issues/4771) - Small fixes and conformance update for transfers
- [4785](https://github.com/vegaprotocol/vega/issues/4785) - Implement feature tests given the acceptance criteria for transfers
- [4784](https://github.com/vegaprotocol/vega/issues/4784) - Moving feature tests from specs internal to verified folder
- [4797](https://github.com/vegaprotocol/vega/issues/4784) - Update `CODEOWNERS` for research to review verified feature files
- [4801](https://github.com/vegaprotocol/vega/issues/4801) - added acceptance criteria codes to feature tests for Settlement at expiry spec
- [4823](https://github.com/vegaprotocol/vega/issues/4823) - simplified performance score
- [4805](https://github.com/vegaprotocol/vega/issues/4805) - Add command line tool to sign for the asset pool method `set_bridge_address`
- [4839](https://github.com/vegaprotocol/vega/issues/4839) - Send governance events when restoring proposals on checkpoint reload.
- [4829](https://github.com/vegaprotocol/vega/issues/4829) - Fix margins calculations for positions with a size of 0 but with a non zero potential sell or buy
- [4826](https://github.com/vegaprotocol/vega/issues/4826) - Tidying up feature tests in verified folder
- [4843](https://github.com/vegaprotocol/vega/issues/4843) - Make snapshot engine aware of local storage old versions, and manage them accordingly to stop growing disk usage.
- [4863](https://github.com/vegaprotocol/vega/issues/4863) - Improve replay protection
- [4867](https://github.com/vegaprotocol/vega/issues/4867) - Optimise replay protection
- [4865](https://github.com/vegaprotocol/vega/issues/4865) - Fix: issue with project board automation action and update commit checker action
- [4674](https://github.com/vegaprotocol/vega/issues/4674) - Add Ethereum events reconciliation for `multisig control`
- [4886](https://github.com/vegaprotocol/vega/pull/4886) - Add more integration tests around order amends and fees.
- [4885](https://github.com/vegaprotocol/vega/pull/4885) - Update amend orders scenario to have fees transfers in int tests

### 🐛 Fixes

- [4842](https://github.com/vegaprotocol/vega/pull/4842) - Fix margin balance not being released after close-out.
- [4798](https://github.com/vegaprotocol/vega/pull/4798) - Fix panic in loading topology from snapshot
- [4521](https://github.com/vegaprotocol/vega/pull/4521) - Better error when trying to use the null-blockchain with an ERC20 asset
- [4692](https://github.com/vegaprotocol/vega/pull/4692) - Set statistics block height after a snapshot reload
- [4702](https://github.com/vegaprotocol/vega/pull/4702) - User tree importer and exporter to transfer snapshots via `statesync`
- [4516](https://github.com/vegaprotocol/vega/pull/4516) - Fix release number title typo - 0.46.1 > 0.46.2
- [4524](https://github.com/vegaprotocol/vega/pull/4524) - Updated `vega verify genesis` to understand new `app_state` layout
- [4515](https://github.com/vegaprotocol/vega/pull/4515) - Set log level in snapshot engine
- [4721](https://github.com/vegaprotocol/vega/pull/4721) - Save checkpoint with `UnixNano` when taking a snapshot
- [4728](https://github.com/vegaprotocol/vega/pull/4728) - Fix restoring markets from snapshot by handling generated providers properly
- [4742](https://github.com/vegaprotocol/vega/pull/4742) - `corestate` endpoints are now populated after a snapshot restore
- [4847](https://github.com/vegaprotocol/vega/pull/4847) - save state of the `feesplitter` in the execution snapshot
- [4782](https://github.com/vegaprotocol/vega/pull/4782) - Fix restoring markets from snapshot in an auction with orders
- [4522](https://github.com/vegaprotocol/vega/pull/4522) - Set transfer responses event when paying rewards
- [4566](https://github.com/vegaprotocol/vega/pull/4566) - Withdrawal fails should return a status rejected rather than cancelled
- [4582](https://github.com/vegaprotocol/vega/pull/4582) - Deposits stayed in memory indefinitely, and withdrawal keys were not being sorted to ensure determinism.
- [4588](https://github.com/vegaprotocol/vega/pull/4588) - Fail when missing tendermint home and public key in `nodewallet import` command
- [4623](https://github.com/vegaprotocol/vega/pull/4623) - Bug fix for `--snapshot.db-path` parameter not being used if it is set
- [4634](https://github.com/vegaprotocol/vega/pull/4634) - Bug fix for `--snapshot.max-retries` parameter not working correctly
- [4775](https://github.com/vegaprotocol/vega/pull/4775) - Restore all market fields when restoring from a snapshot
- [4845](https://github.com/vegaprotocol/vega/pull/4845) - Fix restoring rejected markets by signalling to the generated providers that their parent is dead
- [4651](https://github.com/vegaprotocol/vega/pull/4651) - An array of fixes in the snapshot code path
- [4658](https://github.com/vegaprotocol/vega/pull/4658) - Allow replaying a chain from zero when old snapshots exist
- [4659](https://github.com/vegaprotocol/vega/pull/4659) - Fix liquidity provision commands decode
- [4665](https://github.com/vegaprotocol/vega/pull/4665) - Remove all references to `TxV2`
- [4686](https://github.com/vegaprotocol/vega/pull/4686) - Fix commit hash problem when checkpoint and snapshot overlap. Ensure the snapshot contains the correct checkpoint state.
- [4691](https://github.com/vegaprotocol/vega/pull/4691) - Handle undelegate stake with no balances gracefully
- [4716](https://github.com/vegaprotocol/vega/pull/4716) - Fix protobuf conversion in orders
- [4861](https://github.com/vegaprotocol/vega/pull/4861) - Set a protocol version and properly send it to `Tendermint` in all cases
- [4732](https://github.com/vegaprotocol/vega/pull/4732) - `TimeUpdate` is now first event sent
- [4714](https://github.com/vegaprotocol/vega/pull/4714) - Ensure EEF doesn't process the current block multiple times
- [4700](https://github.com/vegaprotocol/vega/pull/4700) - Ensure verification of type between oracle spec binding and oracle spec
- [4738](https://github.com/vegaprotocol/vega/pull/4738) - Add vesting contract as part of the Ethereum event forwarder
- [4747](https://github.com/vegaprotocol/vega/pull/4747) - Dispatch network parameter updates at the same block when loaded from checkpoint
- [4732](https://github.com/vegaprotocol/vega/pull/4732) - Revert tendermint to version 0.34.14
- [4756](https://github.com/vegaprotocol/vega/pull/4756) - Fix for markets loaded from snapshot not terminated by their oracle
- [4776](https://github.com/vegaprotocol/vega/pull/4776) - Add testing for auction state changes and remove unnecessary market state change
- [4590](https://github.com/vegaprotocol/vega/pull/4590) - Added verification of uint market data in integration test
- [4749](https://github.com/vegaprotocol/vega/pull/4794) - Fixed issue where LP orders did not get redeployed
- [4820](https://github.com/vegaprotocol/vega/pull/4820) - Snapshot fixes for market + update market tracker on trades
- [4854](https://github.com/vegaprotocol/vega/pull/4854) - Snapshot fixes for the `statevar` engine
- [3919](https://github.com/vegaprotocol/vega/pull/3919) - Fixed panic in `maybeInvalidateDuringAuction`
- [4849](https://github.com/vegaprotocol/vega/pull/4849) - Fixed liquidity auction trigger for certain cancel & replace amends.

## 0.47.6

_2022-02-01_

### 🐛 Fixes

- [4691](https://github.com/vegaprotocol/vega/pull/4691) - Handle undelegate stake with no balances gracefully

## 0.47.5

_2022-01-20_

### 🐛 Fixes

- [4617](https://github.com/vegaprotocol/vega/pull/4617) - Bug fix for incorrectly reporting auto delegation

## 0.47.4

_2022-01-05_

### 🐛 Fixes

- [4563](https://github.com/vegaprotocol/vega/pull/4563) - Send an epoch event when loaded from checkpoint

## 0.47.3

_2021-12-24_

### 🐛 Fixes

- [4529](https://github.com/vegaprotocol/vega/pull/4529) - Non determinism in checkpoint fixed

## 0.47.2

_2021-12-17_

### 🐛 Fixes

- [4500](https://github.com/vegaprotocol/vega/pull/4500) - Set minimum for validator power to avoid accidentally removing them
- [4503](https://github.com/vegaprotocol/vega/pull/4503) - Limit delegation epochs in core API
- [4504](https://github.com/vegaprotocol/vega/pull/4504) - Fix premature ending of epoch when loading from checkpoint

## 0.47.1

_2021-11-24_

### 🐛 Fixes

- [4488](https://github.com/vegaprotocol/vega/pull/4488) - Disable snapshots
- [4536](https://github.com/vegaprotocol/vega/pull/4536) - Fixed non determinism in topology checkpoint
- [4550](https://github.com/vegaprotocol/vega/pull/4550) - Do not validate assets when loading checkpoint from non-validators

## 0.47.0

_2021-11-24_

### 🛠 Improvements

- [4480](https://github.com/vegaprotocol/vega/pull/4480) - Update `CHANGELOG.md` since GH Action implemented
- [4439](https://github.com/vegaprotocol/vega/pull/4439) - Create `release_ticket.md` issue template
- [4456](https://github.com/vegaprotocol/vega/pull/4456) - Return 400 on bad mint amounts sent via the faucet
- [4434](https://github.com/vegaprotocol/vega/pull/4434) - Add free form governance net parameters to `allKeys` map
- [4436](https://github.com/vegaprotocol/vega/pull/4436) - Add ability for the null-blockchain to deliver transactions
- [4455](https://github.com/vegaprotocol/vega/pull/4455) - Introduce API to allow time-forwarding in the null-blockchain
- [4422](https://github.com/vegaprotocol/vega/pull/4422) - Add support for validator key rotation
- [4463](https://github.com/vegaprotocol/vega/pull/4463) - Remove the need for an Ethereum connection when using the null-blockchain
- [4477](https://github.com/vegaprotocol/vega/pull/4477) - Allow reloading of null-blockchain configuration while core is running
- [4468](https://github.com/vegaprotocol/vega/pull/4468) - Change validator weights to be based on validator score
- [4484](https://github.com/vegaprotocol/vega/pull/4484) - Add checkpoint validator key rotation
- [4459](https://github.com/vegaprotocol/vega/pull/4459) - Add network parameters overwrite from checkpoints
- [4070](https://github.com/vegaprotocol/vega/pull/4070) - Add calls to enable state-sync via tendermint
- [4465](https://github.com/vegaprotocol/vega/pull/4465) - Add events tags to the `ResponseDeliverTx`

### 🐛 Fixes

- [4435](https://github.com/vegaprotocol/vega/pull/4435) - Fix non determinism in deposits snapshot
- [4418](https://github.com/vegaprotocol/vega/pull/4418) - Add some logging + height/version handling fixes
- [4461](https://github.com/vegaprotocol/vega/pull/4461) - Fix problem where chain id was not present on event bus during checkpoint loading
- [4475](https://github.com/vegaprotocol/vega/pull/4475) - Fix rewards checkpoint not assigned to its correct place

## 0.46.2

_2021-11-24_

### 🐛 Fixes

- [4445](https://github.com/vegaprotocol/vega/pull/4445) - Limit the number of iterations for reward calculation for delegator and fix for division by zero

## 0.46.1

_2021-11-22_

### 🛠 Improvements

- [4437](https://github.com/vegaprotocol/vega/pull/4437) - Turn snapshots off for `v0.46.1` only

## 0.46.0

_2021-11-22_

### 🛠 Improvements

- [4431](https://github.com/vegaprotocol/vega/pull/4431) - Update Vega wallet to version 0.10.0
- [4406](https://github.com/vegaprotocol/vega/pull/4406) - Add changelog and project board Github actions and update linked PR action version
- [4328](https://github.com/vegaprotocol/vega/pull/4328) - Unwrap the timestamps in reward payout event
- [4330](https://github.com/vegaprotocol/vega/pull/4330) - Remove badger related code from the codebase
- [4336](https://github.com/vegaprotocol/vega/pull/4336) - Add oracle snapshot
- [4299](https://github.com/vegaprotocol/vega/pull/4299) - Add liquidity snapshot
- [4196](https://github.com/vegaprotocol/vega/pull/4196) - Experiment at removing the snapshot details from the engine
- [4338](https://github.com/vegaprotocol/vega/pull/4338) - Adding more error messages
- [4317](https://github.com/vegaprotocol/vega/pull/4317) - Extend integration tests with global check for net deposits
- [3616](https://github.com/vegaprotocol/vega/pull/3616) - Add tests to show margins not being released
- [4171](https://github.com/vegaprotocol/vega/pull/4171) - Add trading fees feature test
- [4348](https://github.com/vegaprotocol/vega/pull/4348) - Updating return codes
- [4346](https://github.com/vegaprotocol/vega/pull/4346) - Implement liquidity supplied snapshot
- [4351](https://github.com/vegaprotocol/vega/pull/4351) - Add target liquidity engine
- [4362](https://github.com/vegaprotocol/vega/pull/4362) - Remove staking of cache at the beginning of the epoch for spam protection
- [4364](https://github.com/vegaprotocol/vega/pull/4364) - Change spam error messages to debug and enabled reloading of configuration
- [4353](https://github.com/vegaprotocol/vega/pull/4353) - remove usage of `vegatime.Now` over the codebase
- [4382](https://github.com/vegaprotocol/vega/pull/4382) - Add Prometheus metrics on snapshots
- [4190](https://github.com/vegaprotocol/vega/pull/4190) - Add markets snapshot
- [4389](https://github.com/vegaprotocol/vega/pull/4389) - Update issue templates #4389
- [4392](https://github.com/vegaprotocol/vega/pull/4392) - Update `GETTING_STARTED.md` documentation
- [4391](https://github.com/vegaprotocol/vega/pull/4391) - Refactor delegation
- [4423](https://github.com/vegaprotocol/vega/pull/4423) - Add CLI options to start node with a null-blockchain
- [4400](https://github.com/vegaprotocol/vega/pull/4400) - Add transaction hash to `SubmitTransactionResponse`
- [4394](https://github.com/vegaprotocol/vega/pull/4394) - Add step to clear all events in integration tests
- [4403](https://github.com/vegaprotocol/vega/pull/4403) - Fully remove expiry from withdrawals #4403
- [4396](https://github.com/vegaprotocol/vega/pull/4396) - Add free form governance proposals
- [4413](https://github.com/vegaprotocol/vega/pull/4413) - Deploy to Devnet with Jenkins and remove drone
- [4429](https://github.com/vegaprotocol/vega/pull/4429) - Release version `v0.46.0`
- [4442](https://github.com/vegaprotocol/vega/pull/4442) - Reduce the number of iterations in reward calculation
- [4409](https://github.com/vegaprotocol/vega/pull/4409) - Include chain id in bus messages
- [4464](https://github.com/vegaprotocol/vega/pull/4466) - Update validator power in tendermint based on their staking

### 🐛 Fixes

- [4325](https://github.com/vegaprotocol/vega/pull/4325) - Remove state from the witness snapshot and infer it from votes
- [4334](https://github.com/vegaprotocol/vega/pull/4334) - Fix notary implementation
- [4343](https://github.com/vegaprotocol/vega/pull/4343) - Fix non deterministic test by using same `idGenerator`
- [4352](https://github.com/vegaprotocol/vega/pull/4352) - Remove usage of `time.Now()` in the auction state
- [4380](https://github.com/vegaprotocol/vega/pull/4380) - Implement Uint for network parameters and use it for monies values
- [4369](https://github.com/vegaprotocol/vega/pull/4369) - Fix orders still being accepted after market in trading terminated state
- [4395](https://github.com/vegaprotocol/vega/pull/4395) - Fix drone pipeline
- [4398](https://github.com/vegaprotocol/vega/pull/4398) - Fix to set proper status on withdrawal errors
- [4421](https://github.com/vegaprotocol/vega/issues/4421) - Fix to missing pending rewards in LNL checkpoint
- [4419](https://github.com/vegaprotocol/vega/pull/4419) - Fix snapshot cleanup, improve logging when specified block height could not be reloaded.
- [4444](https://github.com/vegaprotocol/vega/pull/4444) - Fix division by zero when all validator scores are 0
- [4467](https://github.com/vegaprotocol/vega/pull/4467) - Fix reward account balance not being saved/loaded to/from checkpoint
- [4474](https://github.com/vegaprotocol/vega/pull/4474) - Wire rewards checkpoint to checkpoint engine and store infrastructure fee accounts in collateral checkpoint

## 0.45.6

_2021-11-16_

### 🐛 Fixes

- [4506](https://github.com/vegaprotocol/vega/pull/4506) - Wire network parameters to time service to flush out pending changes

## 0.45.5

_2021-11-16_

### 🐛 Fixes

- [4403](https://github.com/vegaprotocol/vega/pull/4403) - Fully remove expiry from withdrawals and release version `v0.45.5`

## 0.45.4

_2021-11-05_

### 🐛 Fixes

- [4372](https://github.com/vegaprotocol/vega/pull/4372) - Fix, if all association is nominated, allow association to be unnominated and nominated again in the same epoch

## 0.45.3

_2021-11-04_

### 🐛 Fixes

- [4362](https://github.com/vegaprotocol/vega/pull/4362) - Remove staking of cache at the beginning of the epoch for spam protection

## 0.45.2

_2021-10-27_

### 🛠 Improvements

- [4308](https://github.com/vegaprotocol/vega/pull/4308) - Add Visual Studio Code configuration
- [4319](https://github.com/vegaprotocol/vega/pull/4319) - Add snapshot node topology
- [4321](https://github.com/vegaprotocol/vega/pull/4321) - Release version `v0.45.2` #4321

### 🐛 Fixes

- [4320](https://github.com/vegaprotocol/vega/pull/4320) - Implement retries for notary transactions
- [4312](https://github.com/vegaprotocol/vega/pull/4312) - Implement retries for witness transactions

## 0.45.1

_2021-10-23_

### 🛠 Improvements

- [4246](https://github.com/vegaprotocol/vega/pull/4246) - Add replay protection snapshot
- [4245](https://github.com/vegaprotocol/vega/pull/4245) - Add ABCI snapshot
- [4260](https://github.com/vegaprotocol/vega/pull/4260) - Reconcile delegation more frequently
- [4255](https://github.com/vegaprotocol/vega/pull/4255) - Add staking snapshot
- [4278](https://github.com/vegaprotocol/vega/pull/4278) - Add timestamps to rewards
- [4265](https://github.com/vegaprotocol/vega/pull/4265) - Add witness snapshot
- [4287](https://github.com/vegaprotocol/vega/pull/4287) - Add stake verifier snapshot
- [4292](https://github.com/vegaprotocol/vega/pull/4292) - Update the vega wallet version

### 🐛 Fixes

- [4280](https://github.com/vegaprotocol/vega/pull/4280) - Make event forwarder hashing result more random
- [4270](https://github.com/vegaprotocol/vega/pull/4270) - Prevent overflow with pending delegation
- [4274](https://github.com/vegaprotocol/vega/pull/4274) - Ensure sufficient balances when nominating multiple nodes
- [4286](https://github.com/vegaprotocol/vega/pull/4286) - Checkpoints fixes

## 0.45.0

_2021-10-19_

### 🛠 Improvements

- [4188](https://github.com/vegaprotocol/vega/pull/4188) - Add rewards snapshot
- [4191](https://github.com/vegaprotocol/vega/pull/4191) - Add limit snapshot
- [4192](https://github.com/vegaprotocol/vega/pull/4192) - Ask for passphrase confirmation on init and generate commands when applicable
- [4201](https://github.com/vegaprotocol/vega/pull/4201) - Implement spam snapshot
- [4214](https://github.com/vegaprotocol/vega/pull/4214) - Add golangci-lint to CI
- [4199](https://github.com/vegaprotocol/vega/pull/4199) - Add ERC20 logic signing
- [4211](https://github.com/vegaprotocol/vega/pull/4211) - Implement snapshot for notary
- [4219](https://github.com/vegaprotocol/vega/pull/4219) - Enable linters
- [4218](https://github.com/vegaprotocol/vega/pull/4218) - Run system-tests in separate build
- [4227](https://github.com/vegaprotocol/vega/pull/4227) - Ignore system-tests failures for non PR builds
- [4232](https://github.com/vegaprotocol/vega/pull/4232) - golangci-lint increase timeout
- [4229](https://github.com/vegaprotocol/vega/pull/4229) - Ensure the vega and Ethereum wallet are not nil before accessing
- [4230](https://github.com/vegaprotocol/vega/pull/4230) - Replay protection snapshot
- [4242](https://github.com/vegaprotocol/vega/pull/4242) - Set timeout for system-tests steps
- [4215](https://github.com/vegaprotocol/vega/pull/4215) - Improve handling of expected trades
- [4224](https://github.com/vegaprotocol/vega/pull/4224) - Make evt forward mode deterministic
- [4168](https://github.com/vegaprotocol/vega/pull/4168) - Update code still using uint64
- [4240](https://github.com/vegaprotocol/vega/pull/4240) - Add command to list and describe Vega paths

### 🐛 Fixes

- [4228](https://github.com/vegaprotocol/vega/pull/4228) - Fix readme updates
- [4210](https://github.com/vegaprotocol/vega/pull/4210) - Add min validators network parameter and bug fix for overflow reward

## 0.44.2

_2021-10-11_

### 🐛 Fixes

- [4195](https://github.com/vegaprotocol/vega/pull/4195) - Fix rewards payout with delay

## 0.44.1

_2021-10-08_

### 🐛 Fixes

- [4183](https://github.com/vegaprotocol/vega/pull/4183) - Fix `undelegateNow` to use the passed amount instead of 0
- [4184](https://github.com/vegaprotocol/vega/pull/4184) - Remove 0 balance events from checkpoint of delegations
- [4185](https://github.com/vegaprotocol/vega/pull/4185) - Fix event sent on reward pool creation + fix owner

## 0.44.0

_2021-10-07_

### 🛠 Improvements

- [4159](https://github.com/vegaprotocol/vega/pull/4159) - Clean-up and separate checkpoints and snapshots
- [4172](https://github.com/vegaprotocol/vega/pull/4172) - Added assetActions to banking snapshot
- [4173](https://github.com/vegaprotocol/vega/pull/4173) - Add tools and linting
- [4161](https://github.com/vegaprotocol/vega/pull/4161) - Assets snapshot implemented
- [4142](https://github.com/vegaprotocol/vega/pull/4142) - Add clef wallet
- [4160](https://github.com/vegaprotocol/vega/pull/4160) - Snapshot positions engine
- [4170](https://github.com/vegaprotocol/vega/pull/4170) - Update to latest proto and go mod tidy
- [4157](https://github.com/vegaprotocol/vega/pull/4157) - Adding IDGenerator types
- [4166](https://github.com/vegaprotocol/vega/pull/4166) - Banking snapshot
- [4133](https://github.com/vegaprotocol/vega/pull/4133) - Matching engine snapshots
- [4162](https://github.com/vegaprotocol/vega/pull/4162) - Add fields to validators genesis
- [4154](https://github.com/vegaprotocol/vega/pull/4154) - Port code to use last version of proto (layout change)
- [4141](https://github.com/vegaprotocol/vega/pull/4141) - Collateral snapshots
- [4131](https://github.com/vegaprotocol/vega/pull/4131) - Snapshot epoch engine
- [4143](https://github.com/vegaprotocol/vega/pull/4143) - Add delegation snapshot
- [4114](https://github.com/vegaprotocol/vega/pull/4114) - Document default file location
- [4130](https://github.com/vegaprotocol/vega/pull/4130) - Update proto dependencies to latest
- [4134](https://github.com/vegaprotocol/vega/pull/4134) - Checkpoints and snapshots are 2 different things
- [4121](https://github.com/vegaprotocol/vega/pull/4121) - Additional test scenarios for delegation & rewards
- [4111](https://github.com/vegaprotocol/vega/pull/4111) - Simplify nodewallet integration
- [4110](https://github.com/vegaprotocol/vega/pull/4110) - Auto delegation
- [4123](https://github.com/vegaprotocol/vega/pull/4123) - Add auto delegation to checkpoint
- [4120](https://github.com/vegaprotocol/vega/pull/4120) - Snapshot preparation
- [4060](https://github.com/vegaprotocol/vega/pull/4060) - Edge case scenarios delegation

### 🐛 Fixes

- [4156](https://github.com/vegaprotocol/vega/pull/4156) - Fix filename for checkpoints
- [4158](https://github.com/vegaprotocol/vega/pull/4158) - Remove delay in reward/delegation calculation
- [4150](https://github.com/vegaprotocol/vega/pull/4150) - De-duplicate stake linkings
- [4137](https://github.com/vegaprotocol/vega/pull/4137) - Add missing key to all network parameters key map
- [4132](https://github.com/vegaprotocol/vega/pull/4132) - Send delegation events
- [4128](https://github.com/vegaprotocol/vega/pull/4128) - Simplify checkpointing for network parameters and start fixing collateral checkpoint
- [4124](https://github.com/vegaprotocol/vega/pull/4124) - Fixed non-deterministic checkpoint and added auto delegation to checkpoint
- [4118](https://github.com/vegaprotocol/vega/pull/4118) - Fixed epoch issue

## 0.43.0

_2021-09-22_

### 🛠 Improvements

- [4051](https://github.com/vegaprotocol/vega/pull/4051) - New type to handle signed versions of the uint256 values we already support
- [4090](https://github.com/vegaprotocol/vega/pull/4090) - Update the proto repository dependencies
- [4023](https://github.com/vegaprotocol/vega/pull/4023) - Implement the spam protection engine
- [4063](https://github.com/vegaprotocol/vega/pull/4063) - Migrate to XDG structure
- [4075](https://github.com/vegaprotocol/vega/pull/4075) - Prefix checkpoint files with time and interval for automated tests
- [4050](https://github.com/vegaprotocol/vega/pull/4050) - Extend delegation feature test scenarios
- [4056](https://github.com/vegaprotocol/vega/pull/4056) - Improve message for genesis error with topology
- [4017](https://github.com/vegaprotocol/vega/pull/4017) - Migrate wallet to XGD file structure
- [4024](https://github.com/vegaprotocol/vega/pull/4024) - Extend delegation rewards feature test scenarios
- [4035](https://github.com/vegaprotocol/vega/pull/4035) - Implement multisig control signatures
- [4083](https://github.com/vegaprotocol/vega/pull/4083) - Remove expiry support for withdrawals
- [4068](https://github.com/vegaprotocol/vega/pull/4068) - Allow proposal votes to happen during the validation period
- [4088](https://github.com/vegaprotocol/vega/pull/4088) - Implements the simple JSON oracle source
- [4105](https://github.com/vegaprotocol/vega/pull/4105) - Add more hashes to the app state hash
- [4107](https://github.com/vegaprotocol/vega/pull/4107) - Remove the trading proxy service
- [4101](https://github.com/vegaprotocol/vega/pull/4101) - Remove dependency to the Ethereum client from the Ethereum wallet

### 🐛 Fixes

- [4053](https://github.com/vegaprotocol/vega/pull/4053) - Fix readme explanation for log levels
- [4054](https://github.com/vegaprotocol/vega/pull/4054) - Capture errors with Ethereum iterator and continue
- [4040](https://github.com/vegaprotocol/vega/pull/4040) - Fix bug where the withdrawal signature uses uint64
- [4042](https://github.com/vegaprotocol/vega/pull/4042) - Extended delegation rewards feature test scenario edits
- [4034](https://github.com/vegaprotocol/vega/pull/4034) - Update integration tests now TxErr events are not sent in the execution package
- [4106](https://github.com/vegaprotocol/vega/pull/4106) - Fix a panic when reloading checkpoints
- [4115](https://github.com/vegaprotocol/vega/pull/4115) - Use block height in checkpoint file names

## 0.42.0

_2021-09-10_

### 🛠 Improvements

- [3862](https://github.com/vegaprotocol/vega/pull/3862) - Collateral snapshot: Add checkpoints where needed, update processor (ABCI app) to write checkpoint data to file.
- [3926](https://github.com/vegaprotocol/vega/pull/3926) - Add epoch to delegation balance events and changes to the delegation / reward engines
- [3963](https://github.com/vegaprotocol/vega/pull/3963) - Load tendermint logger configuration
- [3958](https://github.com/vegaprotocol/vega/pull/3958) - Update istake ABI and run abigen
- [3933](https://github.com/vegaprotocol/vega/pull/3933) - Remove redundant API from Validator node
- [3971](https://github.com/vegaprotocol/vega/pull/3971) - Reinstate wallet subcommand tests
- [3961](https://github.com/vegaprotocol/vega/pull/3961) - Implemented feature test for delegation
- [3977](https://github.com/vegaprotocol/vega/pull/3977) - Add undelegate, delegate and register snapshot errors
- [3976](https://github.com/vegaprotocol/vega/pull/3976) - Add network parameter for competition level
- [3975](https://github.com/vegaprotocol/vega/pull/3975) - Add parties stake api
- [3978](https://github.com/vegaprotocol/vega/pull/3978) - Update dependencies
- [3980](https://github.com/vegaprotocol/vega/pull/3980) - Update protobuf dependencies
- [3910](https://github.com/vegaprotocol/vega/pull/3910) - Change all price, amounts, balances from uint64 to string
- [3969](https://github.com/vegaprotocol/vega/pull/3969) - Bump dlv and geth to latest versions
- [3925](https://github.com/vegaprotocol/vega/pull/3925) - Add command to sign a subset of network parameters
- [3981](https://github.com/vegaprotocol/vega/pull/3981) - Remove the `wallet-pubkey` flag on genesis sign command
- [3987](https://github.com/vegaprotocol/vega/pull/3987) - Add genesis verify command to verify signature against local genesis file
- [3984](https://github.com/vegaprotocol/vega/pull/3984) - Update the mainnet addresses in genesis generation command
- [3983](https://github.com/vegaprotocol/vega/pull/3983) - Added action field to epoch events
- [3988](https://github.com/vegaprotocol/vega/pull/3988) - Update the go-ethereum dependency
- [3991](https://github.com/vegaprotocol/vega/pull/3991) - Remove hardcoded address to the Ethereum node
- [3990](https://github.com/vegaprotocol/vega/pull/3990) - Network bootstrapping
- [3992](https://github.com/vegaprotocol/vega/pull/3992) - Check big int conversion from string in ERC20 code
- [3993](https://github.com/vegaprotocol/vega/pull/3993) - Use the vega public key as node id
- [3955](https://github.com/vegaprotocol/vega/pull/3955) - Use staking accounts in governance
- [4004](https://github.com/vegaprotocol/vega/pull/4004) - Broker configuration: change IP to address Address
- [4005](https://github.com/vegaprotocol/vega/pull/4005) - Add a simple subcommand to the vega binary to ease submitting transactions
- [3997](https://github.com/vegaprotocol/vega/pull/3997) - Do not require Ethereum client when starting the nodewallet
- [4009](https://github.com/vegaprotocol/vega/pull/4009) - Add delegation core APIs
- [4014](https://github.com/vegaprotocol/vega/pull/4014) - Implement delegation and epoch for Limited Network Life
- [3914](https://github.com/vegaprotocol/vega/pull/3914) - Implement staking event verification
- [3940](https://github.com/vegaprotocol/vega/pull/3940) - Remove validator signature from configuration and add network parameters
- [3938](https://github.com/vegaprotocol/vega/pull/3938) - Add more logging informations on the witness vote failures
- [3932](https://github.com/vegaprotocol/vega/pull/3932) - Adding asset details to reward events
- [3706](https://github.com/vegaprotocol/vega/pull/3706) - Remove startup markets workaround
- [3905](https://github.com/vegaprotocol/vega/pull/3905) - Add vega genesis new validator sub-command
- [3895](https://github.com/vegaprotocol/vega/pull/3895) - Add command to create a new genesis block with app_state
- [3900](https://github.com/vegaprotocol/vega/pull/3900) - Create reward engine
- [4847](https://github.com/vegaprotocol/vega/pull/3847) - Modified staking account to be backed by governance token account balance
- [3907](https://github.com/vegaprotocol/vega/pull/3907) - Tune system tests
- [3904](https://github.com/vegaprotocol/vega/pull/3904) - Update Jenkins file to run all System Tests
- [3795](https://github.com/vegaprotocol/vega/pull/3795) - Add capability to sent events to a socket stream
- [3832](https://github.com/vegaprotocol/vega/pull/3832) - Update the genesis topology map
- [3891](https://github.com/vegaprotocol/vega/pull/3891) - Verify transaction version 2 signature
- [3813](https://github.com/vegaprotocol/vega/pull/3813) - Implementing epoch time
- [4031](https://github.com/vegaprotocol/vega/pull/4031) - Send error events in processor through wrapper

### 🐛 Fixes

- [3950](https://github.com/vegaprotocol/vega/pull/3950) - `LoadGenesis` returns nil if checkpoint entry is empty
- [3960](https://github.com/vegaprotocol/vega/pull/3960) - Unstaking events are not seen by all validator nodes in DV
- [3973](https://github.com/vegaprotocol/vega/pull/3973) - Set ABCI client so it is possible to submit a transaction
- [3986](https://github.com/vegaprotocol/vega/pull/3986) - Emit Party event when stake link is accepted
- [3979](https://github.com/vegaprotocol/vega/pull/3979) - Add more delegation / reward scenarios and steps and a bug fix in emitted events
- [4007](https://github.com/vegaprotocol/vega/pull/4007) - Changed delegation balance event to use string
- [4006](https://github.com/vegaprotocol/vega/pull/4006) - Sort proposals by timestamp
- [4012](https://github.com/vegaprotocol/vega/pull/4012) - Fix panic with vega watch
- [3937](https://github.com/vegaprotocol/vega/pull/3937) - Include `TX_ERROR` events for type ALL subscribers
- [3930](https://github.com/vegaprotocol/vega/pull/3930) - Added missing function and updated readme with details
- [3918](https://github.com/vegaprotocol/vega/pull/3918) - Fix the build by updating the module version for the vegawallet
- [3901](https://github.com/vegaprotocol/vega/pull/3901) - Emit a `TxErrEvent` if withdraw submission is invalid
- [3874](https://github.com/vegaprotocol/vega/pull/3874) - Fix binary version
- [3884](https://github.com/vegaprotocol/vega/pull/3884) - Always async transaction
- [3877](https://github.com/vegaprotocol/vega/pull/3877) - Use a custom http client for the tendermint client

## 0.41.0

_2021-08-06_

### 🛠 Improvements

- [#3743](https://github.com/vegaprotocol/vega/pull/3743) - Refactor: Rename traders to parties
- [#3758](https://github.com/vegaprotocol/vega/pull/3758) - Refactor: Cleanup naming in the types package
- [#3789](https://github.com/vegaprotocol/vega/pull/3789) - Update ed25519-voi
- [#3589](https://github.com/vegaprotocol/vega/pull/3589) - Update tendermint to a newer version
- [#3591](https://github.com/vegaprotocol/vega/pull/3591) - Implemented market terminated, settled and suspended states via the oracle trigger
- [#3798](https://github.com/vegaprotocol/vega/pull/3798) - Update godog version to 11
- [#3793](https://github.com/vegaprotocol/vega/pull/3793) - Send Commander commands in a goroutine
- [#3805](https://github.com/vegaprotocol/vega/pull/3805) - Checkpoint engine hash and checkpoint creation
- [#3785](https://github.com/vegaprotocol/vega/pull/3785) - Implement delegation commands
- [#3714](https://github.com/vegaprotocol/vega/pull/3714) - Move protobufs into an external repository
- [#3719](https://github.com/vegaprotocol/vega/pull/3719) - Replace vega wallet with call to the vegawallet
- [#3762](https://github.com/vegaprotocol/vega/pull/3762) - Refactor: Cleanup markets in domains types
- [#3822](https://github.com/vegaprotocol/vega/pull/3822) - Testing: vega integration add subfolders for features
- [#3794](https://github.com/vegaprotocol/vega/pull/3794) - Implement rewards transfer
- [#3839](https://github.com/vegaprotocol/vega/pull/3839) - Implement a delegation engine
- [#3842](https://github.com/vegaprotocol/vega/pull/3842) - Imports need reformatting for core code base
- [#3849](https://github.com/vegaprotocol/vega/pull/3849) - Add limits engine + genesis loading
- [#3836](https://github.com/vegaprotocol/vega/pull/3836) - Add a first version of the accounting engine
- [#3859](https://github.com/vegaprotocol/vega/pull/3859) - Enable CGO in CI

### 🐛 Fixes

- [#3751](https://github.com/vegaprotocol/vega/pull/3751) - `Unparam` linting fixes
- [#3776](https://github.com/vegaprotocol/vega/pull/3776) - Ensure expired/settled markets are correctly recorded in app state
- [#3774](https://github.com/vegaprotocol/vega/pull/3774) - Change liquidity fees distribution to general account and not margin account of liquidity provider
- [#3801](https://github.com/vegaprotocol/vega/pull/3801) - Testing: Fixed setup of oracle spec step in integration
- [#3828](https://github.com/vegaprotocol/vega/pull/3828) - 🔥 Check if application context has been cancelled before writing to channel
- [#3838](https://github.com/vegaprotocol/vega/pull/3838) - 🔥 Fix panic on division by 0 with party voting and withdrawing funds

## 0.40.0

_2021-07-12_

### 🛠 Improvements

- [#3718](https://github.com/vegaprotocol/vega/pull/3718) - Run `unparam` over the codebase
- [#3705](https://github.com/vegaprotocol/vega/pull/3705) - Return theoretical target stake when in auction
- [#3703](https://github.com/vegaprotocol/vega/pull/3703) - Remove inefficient metrics calls
- [#3693](https://github.com/vegaprotocol/vega/pull/3693) - Calculation without Decimal in the liquidity target package
- [#3696](https://github.com/vegaprotocol/vega/pull/3696) - Remove some uint <-> Decimal conversion
- [#3689](https://github.com/vegaprotocol/vega/pull/3689) - Do not rely on proto conversion for `GetAsset`
- [#3676](https://github.com/vegaprotocol/vega/pull/3676) - Ad the `tm` subcommand
- [#3569](https://github.com/vegaprotocol/vega/pull/3569) - Migrate from uint64 to uint256 for all balances, amount, prices in the core
- [#3594](https://github.com/vegaprotocol/vega/pull/3594) - Improve probability of trading calculations
- [#3752](https://github.com/vegaprotocol/vega/pull/3752) - Update oracle engine to send events at the end of the block
- [#3745](https://github.com/vegaprotocol/vega/pull/3745) - Add loss socialization for final settlement

### 🐛 Fixes

- [#3722](https://github.com/vegaprotocol/vega/pull/3722) - Added sign to settle return values to allow to determine correctly win/loss
- [#3720](https://github.com/vegaprotocol/vega/pull/3720) - Tidy up max open interest calculations
- [#3704](https://github.com/vegaprotocol/vega/pull/3704) - Fix settlement with network orders
- [#3686](https://github.com/vegaprotocol/vega/pull/3686) -Fixes in the positions engine following migration to uint256
- [#3684](https://github.com/vegaprotocol/vega/pull/3684) - Fix the position engine hash state following migration to uint256
- [#3467](https://github.com/vegaprotocol/vega/pull/3647) - Ensure LP orders are not submitted during auction
- [#3736](https://github.com/vegaprotocol/vega/pull/3736) - Correcting event types and adding panics to catch mistakes

## 0.39.0

_2021-06-30_

### 🛠 Improvements

- [#3642](https://github.com/vegaprotocol/vega/pull/3642) - Refactor integration tests
- [#3637](https://github.com/vegaprotocol/vega/pull/3637) - Rewrite pegged / liquidity order control flow
- [#3635](https://github.com/vegaprotocol/vega/pull/3635) - Unified error system and strict parsing in feature tests
- [#3632](https://github.com/vegaprotocol/vega/pull/3632) - Add documentation on market instantiation in feature tests
- [#3599](https://github.com/vegaprotocol/vega/pull/3599) - Return better errors when replay protection happen

### 🐛 Fixes

- [#3640](https://github.com/vegaprotocol/vega/pull/3640) - Fix send on closed channel using timer (event bus)
- [#3638](https://github.com/vegaprotocol/vega/pull/3638) - Fix decimal instantiation in bond slashing
- [#3621](https://github.com/vegaprotocol/vega/pull/3621) - Remove pegged order from pegged list if order is aggressive and trade
- [#3612](https://github.com/vegaprotocol/vega/pull/3612) - Clean code in the wallet package

## 0.38.0

_2021-06-11_

### 🛠 Improvements

- [#3546](https://github.com/vegaprotocol/vega/pull/3546) - Add Auction Extension trigger field to market data
- [#3538](https://github.com/vegaprotocol/vega/pull/3538) - Testing: Add block time handling & block time variance
- [#3596](https://github.com/vegaprotocol/vega/pull/3596) - Enable replay protection
- [#3497](https://github.com/vegaprotocol/vega/pull/3497) - Implement new transaction format
- [#3461](https://github.com/vegaprotocol/vega/pull/3461) - Implement new commands validation

### 🐛 Fixes

- [#3528](https://github.com/vegaprotocol/vega/pull/3528) - Stop liquidity auctions from extending infinitely
- [#3567](https://github.com/vegaprotocol/vega/pull/3567) - Fix handling of Liquidity Commitments at price bounds
- [#3568](https://github.com/vegaprotocol/vega/pull/3568) - Fix potential nil pointer when fetching proposals
- [#3554](https://github.com/vegaprotocol/vega/pull/3554) - Fix package import for domain types
- [#3549](https://github.com/vegaprotocol/vega/pull/3549) - Remove Oracle prefix from files in the Oracle package
- [#3541](https://github.com/vegaprotocol/vega/pull/3541) - Ensure all votes have weight initialised to 0
- [#3539](https://github.com/vegaprotocol/vega/pull/3541) - Address flaky tests
- [#3540](https://github.com/vegaprotocol/vega/pull/3540) - Rename auction state methods
- [#3533](https://github.com/vegaprotocol/vega/pull/3533) - Refactor auction end logic to its own file
- [#3532](https://github.com/vegaprotocol/vega/pull/3532) - Fix Average Entry valuation during opening auctions
- [#3523](https://github.com/vegaprotocol/vega/pull/3523) - Improve nil pointer checks on proposal submissions
- [#3591](https://github.com/vegaprotocol/vega/pull/3591) - Avoid slice out of access bond in trades store

## 0.37.0

_2021-05-26_

### 🛠 Improvements

- [#3479](https://github.com/vegaprotocol/vega/pull/3479) - Add test coverage for auction interactions
- [#3494](https://github.com/vegaprotocol/vega/pull/3494) - Add `error_details` field to rejected proposals
- [#3491](https://github.com/vegaprotocol/vega/pull/3491) - Market Data no longer returns an error when no market data exists, as this is a valid situation
- [#3461](https://github.com/vegaprotocol/vega/pull/3461) - Optimise transaction format & improve validation
- [#3489](https://github.com/vegaprotocol/vega/pull/3489) - Run `buf breaking` at build time
- [#3487](https://github.com/vegaprotocol/vega/pull/3487) - Refactor `prepare*` command validation
- [#3516](https://github.com/vegaprotocol/vega/pull/3516) - New tests for distressed LP + use margin for bond slashing as fallback
- [#4921](https://github.com/vegaprotocol/vega/issues/4921) - Add comment to document behaviour on margin account in feature test (liquidity-provision-bond-account.feature)

### 🐛 Fixes

- [#3513](https://github.com/vegaprotocol/vega/pull/3513) - Fix reprice of pegged orders on every liquidity update
- [#3457](https://github.com/vegaprotocol/vega/pull/3457) - Fix probability of trading calculation for liquidity orders
- [#3515](https://github.com/vegaprotocol/vega/pull/3515) - Fixes for the resolve close out LP parties flow
- [#3513](https://github.com/vegaprotocol/vega/pull/3513) - Fix redeployment of LP orders
- [#3514](https://github.com/vegaprotocol/vega/pull/3513) - Fix price monitoring bounds

## 0.36.0

_2021-05-13_

### 🛠 Improvements

- [#3408](https://github.com/vegaprotocol/vega/pull/3408) - Add more information on token proportion/weight on proposal votes APIs
- [#3360](https://github.com/vegaprotocol/vega/pull/3360) - :fire: REST: Move deposits endpoint to `/parties/[partyId]/deposits`
- [#3431](https://github.com/vegaprotocol/vega/pull/3431) - Improve caching of values when exiting auctions
- [#3459](https://github.com/vegaprotocol/vega/pull/3459) - Add extra validation for Order, Vote, Withdrawal and LP transactions
- [#3433](https://github.com/vegaprotocol/vega/pull/3433) - Reject non-persistent orders that fall outside price monitoring bounds
- [#3443](https://github.com/vegaprotocol/vega/pull/3443) - Party is no longer required when submitting an order amendment
- [#3446](https://github.com/vegaprotocol/vega/pull/3443) - Party is no longer required when submitting an order cancellation
- [#3449](https://github.com/vegaprotocol/vega/pull/3449) - Party is no longer required when submitting an withdrawal request

### 🐛 Fixes

- [#3451](https://github.com/vegaprotocol/vega/pull/3451) - Remove float usage in liquidity engine
- [#3447](https://github.com/vegaprotocol/vega/pull/3447) - Clean up order submission code
- [#3436](https://github.com/vegaprotocol/vega/pull/3436) - Break up internal proposal definitions
- [#3452](https://github.com/vegaprotocol/vega/pull/3452) - Tidy up LP implementation internally
- [#3458](https://github.com/vegaprotocol/vega/pull/3458) - Fix spelling errors in GraphQL docs
- [#3434](https://github.com/vegaprotocol/vega/pull/3434) - Improve test coverage around Liquidity Provisions on auction close
- [#3411](https://github.com/vegaprotocol/vega/pull/3411) - Fix settlement tests
- [#3418](https://github.com/vegaprotocol/vega/pull/3418) - Rename External Resource Checker to Witness
- [#3419](https://github.com/vegaprotocol/vega/pull/3419) - Fix blank IDs on oracle specs in genesis markets
- [#3412](https://github.com/vegaprotocol/vega/pull/3412) - Refactor internal Vote Submission type to be separate from Vote type
- [#3421](https://github.com/vegaprotocol/vega/pull/3421) - Improve test coverage around order uncrossing
- [#3425](https://github.com/vegaprotocol/vega/pull/3425) - Remove debug steps from feature tests
- [#3430](https://github.com/vegaprotocol/vega/pull/3430) - Remove `LiquidityPoolBalance` from configuration
- [#3468](https://github.com/vegaprotocol/vega/pull/3468) - Increase rate limit that was causing mempools to fill up unnecessarily
- [#3438](https://github.com/vegaprotocol/vega/pull/3438) - Split protobuf definitions
- [#3450](https://github.com/vegaprotocol/vega/pull/3450) - Do not emit amendments from liquidity engine

## 0.35.0

_2021-04-21_

### 🛠 Improvements

- [#3341](https://github.com/vegaprotocol/vega/pull/3341) - Add logging for transactions rejected for having no accounts
- [#3339](https://github.com/vegaprotocol/vega/pull/3339) - Reimplement amending LPs not to be cancel and replace
- [#3371](https://github.com/vegaprotocol/vega/pull/3371) - Optimise calculation of cumulative price levels
- [#3339](https://github.com/vegaprotocol/vega/pull/3339) - Reuse LP orders IDs when they are re-created
- [#3385](https://github.com/vegaprotocol/vega/pull/3385) - Track the time spent in auction via Prometheus metrics
- [#3376](https://github.com/vegaprotocol/vega/pull/3376) - Implement a simple benchmarking framework for the core trading
- [#3371](https://github.com/vegaprotocol/vega/pull/3371) - Optimize indicative price and volume calculation

### 🐛 Fixes

- [#3356](https://github.com/vegaprotocol/vega/pull/3356) - Auctions are extended if exiting auction would leave either side of the book empty
- [#3348](https://github.com/vegaprotocol/vega/pull/3348) - Correctly set time when liquidity engine is created
- [#3321](https://github.com/vegaprotocol/vega/pull/3321) - Fix bond account use on LP submission
- [#3369](https://github.com/vegaprotocol/vega/pull/3369) - Reimplement amending LPs not to be cancel and replace
- [#3358](https://github.com/vegaprotocol/vega/pull/3358) - Improve event bus stability
- [#3363](https://github.com/vegaprotocol/vega/pull/3363) - Fix behaviour when leaving auctions
- [#3321](https://github.com/vegaprotocol/vega/pull/3321) - Do not slash bond accounts on LP submission
- [#3350](https://github.com/vegaprotocol/vega/pull/3350) - Fix equity like share in the market data
- [#3363](https://github.com/vegaprotocol/vega/pull/3363) - Ensure leaving an auction cannot trigger another auction / auction leave
- [#3369](https://github.com/vegaprotocol/vega/pull/3369) - Fix LP order deployments
- [#3366](https://github.com/vegaprotocol/vega/pull/3366) - Set the fee paid in uncrossing auction trades
- [#3364](https://github.com/vegaprotocol/vega/pull/3364) - Improve / fix positions tracking
- [#3358](https://github.com/vegaprotocol/vega/pull/3358) - Fix event bus by deep cloning all messages
- [#3374](https://github.com/vegaprotocol/vega/pull/3374) - Check trades in integration tests

## 0.34.1

_2021-04-08_

### 🐛 Fixes

- [#3324](https://github.com/vegaprotocol/vega/pull/3324) - CI: Fix multi-architecture build

## 0.34.0

_2021-04-07_

### 🛠 Improvements

- [#3302](https://github.com/vegaprotocol/vega/pull/3302) - Add reference to LP in orders created by LP
- [#3183](https://github.com/vegaprotocol/vega/pull/3183) - All orders from LP - including rejected orders - are now sent through the event bus
- [#3248](https://github.com/vegaprotocol/vega/pull/3248) - Store and propagate bond penalty
- [#3266](https://github.com/vegaprotocol/vega/pull/3266) - Add network parameters to control auction duration & extension
- [#3264](https://github.com/vegaprotocol/vega/pull/3264) - Add Liquidity Provision ID to orders created by LP commitments
- [#3126](https://github.com/vegaprotocol/vega/pull/3126) - Add transfer for bond slashing
- [#3281](https://github.com/vegaprotocol/vega/pull/3281) - Update scripts to go 1.16.2
- [#3280](https://github.com/vegaprotocol/vega/pull/3280) - Update to go 1.16.2
- [#3235](https://github.com/vegaprotocol/vega/pull/3235) - Extend unit test coverage for products
- [#3219](https://github.com/vegaprotocol/vega/pull/3219) - Remove `liquidityFee` network parameter
- [#3217](https://github.com/vegaprotocol/vega/pull/3217) - Add an event bus event when a market closes
- [#3214](https://github.com/vegaprotocol/vega/pull/3214) - Add arbitrary data signing wallet endpoint
- [#3316](https://github.com/vegaprotocol/vega/pull/3316) - Add tests for traders closing their own position
- [#3270](https://github.com/vegaprotocol/vega/pull/3270) - _Feature test refactor_: Add Liquidity Provision feature tests
- [#3289](https://github.com/vegaprotocol/vega/pull/3289) - _Feature test refactor_: Remove unused steps
- [#3275](https://github.com/vegaprotocol/vega/pull/3275) - _Feature test refactor_: Refactor order cancellation steps
- [#3230](https://github.com/vegaprotocol/vega/pull/3230) - _Feature test refactor_: Refactor trader amends step
- [#3226](https://github.com/vegaprotocol/vega/pull/3226) - _Feature test refactor_: Refactor features with invalid order specs
- [#3200](https://github.com/vegaprotocol/vega/pull/3200) - _Feature test refactor_: Add step to end opening auction
- [#3201](https://github.com/vegaprotocol/vega/pull/3201) - _Feature test refactor_: Add step to amend order by reference
- [#3204](https://github.com/vegaprotocol/vega/pull/3204) - _Feature test refactor_: Add step to place pegged orders
- [#3207](https://github.com/vegaprotocol/vega/pull/3207) - _Feature test refactor_: Add step to create Liquidity Provision
- [#3212](https://github.com/vegaprotocol/vega/pull/3212) - _Feature test refactor_: Remove unused settlement price step
- [#3203](https://github.com/vegaprotocol/vega/pull/3203) - _Feature test refactor_: Rework Submit Order step
- [#3251](https://github.com/vegaprotocol/vega/pull/3251) - _Feature test refactor_: Split market declaration
- [#3314](https://github.com/vegaprotocol/vega/pull/3314) - _Feature test refactor_: Apply naming convention to assertions
- [#3295](https://github.com/vegaprotocol/vega/pull/3295) - Refactor governance engine tests
- [#3298](https://github.com/vegaprotocol/vega/pull/3298) - Add order book caching
- [#3307](https://github.com/vegaprotocol/vega/pull/3307) - Use `UpdateNetworkParams` to validate network parameter updates
- [#3308](https://github.com/vegaprotocol/vega/pull/3308) - Add probability of trading

### 🐛 Fixes

- [#3249](https://github.com/vegaprotocol/vega/pull/3249) - GraphQL: `LiquidityProvision` is no longer missing from the `EventBus` union
- [#3253](https://github.com/vegaprotocol/vega/pull/3253) - Verify all properties on oracle specs
- [#3224](https://github.com/vegaprotocol/vega/pull/3224) - Check for wash trades when FOK orders uncross
- [#3257](https://github.com/vegaprotocol/vega/pull/3257) - Order Status is now only `Active` when it is submitted to the book
- [#3285](https://github.com/vegaprotocol/vega/pull/3285) - LP provisions are now properly stopped when a market is rejected
- [#3290](https://github.com/vegaprotocol/vega/pull/3290) - Update Market Value Proxy at the end of each block
- [#3267](https://github.com/vegaprotocol/vega/pull/3267) - Ensure Liquidity Auctions are not left if it would result in an empty book
- [#3286](https://github.com/vegaprotocol/vega/pull/3286) - Reduce some log levels
- [#3263](https://github.com/vegaprotocol/vega/pull/3263) - Fix incorrect context object in Liquidity Provisions
- [#3283](https://github.com/vegaprotocol/vega/pull/3283) - Remove debug code
- [#3198](https://github.com/vegaprotocol/vega/pull/3198) - chore: Add spell checking to build pipeline
- [#3303](https://github.com/vegaprotocol/vega/pull/3303) - Reduce market depth updates when nothing changes
- [#3310](https://github.com/vegaprotocol/vega/pull/3310) - Fees are no longer paid to inactive LPs
- [#3305](https://github.com/vegaprotocol/vega/pull/3305) - Fix validation of governance proposal terms
- [#3311](https://github.com/vegaprotocol/vega/pull/3311) - `targetStake` is now an unsigned integer
- [#3313](https://github.com/vegaprotocol/vega/pull/3313) - Fix invalid account wrapping

## 0.33.0

_2021-02-16_

As per the previous release notes, this release brings a lot of fixes, most of which aren't exciting new features but improve either the code quality or the developer experience. This release is pretty hefty, as the last few updates have been patch releases. It represents a lot of heavy testing and bug fixing on Liquidity Commitment orders. Alongside that, the feature test suite (we use [godog](https://github.com/cucumber/godog)) has seen some serious attention so that we can specify more complex scenarios easily.

### 🛠 Improvements

- [#3094](https://github.com/vegaprotocol/vega/pull/3094) - :fire: GraphQL: Use `ID` scalar for IDs, ensure capitalisation is correct (`marketID` -> `marketId`)
- [#3093](https://github.com/vegaprotocol/vega/pull/3093) - :fire: GraphQL: Add LP Commitment field to market proposal
- [#3061](https://github.com/vegaprotocol/vega/pull/3061) - GraphQL: Add market proposal to markets created via governance
- [#3060](https://github.com/vegaprotocol/vega/pull/3060) - Add maximum LP shape size limit network parameter
- [#3089](https://github.com/vegaprotocol/vega/pull/3089) - Add `OracleSpec` to market
- [#3148](https://github.com/vegaprotocol/vega/pull/3148) - Add GraphQL endpoints for oracle spec
- [#3179](https://github.com/vegaprotocol/vega/pull/3179) - Add metrics logging for LPs
- [#3127](https://github.com/vegaprotocol/vega/pull/3127) - Add validation for Oracle Specs on market proposals
- [#3129](https://github.com/vegaprotocol/vega/pull/3129) - Update transfers to use `uint256`
- [#3091](https://github.com/vegaprotocol/vega/pull/3091) - Refactor: Standardise how `InAuction` is detected in the core
- [#3133](https://github.com/vegaprotocol/vega/pull/3133) - Remove `log.error` when TX rate limit is hit
- [#3140](https://github.com/vegaprotocol/vega/pull/3140) - Remove `log.error` when cancel all orders fails
- [#3072](https://github.com/vegaprotocol/vega/pull/3072) - Re-enable disabled static analysis
- [#3068](https://github.com/vegaprotocol/vega/pull/3068) - Add `dlv` to docker container
- [#3067](https://github.com/vegaprotocol/vega/pull/3067) - Add more LP unit tests
- [#3066](https://github.com/vegaprotocol/vega/pull/3066) - Remove `devnet` specific wallet initialisation
- [#3041](https://github.com/vegaprotocol/vega/pull/3041) - Remove obsolete `InitialMarkPrice` network parameter
- [#3035](https://github.com/vegaprotocol/vega/pull/3035) - Documentation fixed for infrastructure fee field
- [#3034](https://github.com/vegaprotocol/vega/pull/3034) - Add `buf` to get tools script
- [#3032](https://github.com/vegaprotocol/vega/pull/3032) - Move documentation generation to [`vegaprotocol/api`](https://github.com/vegaprotocol/api) repository
- [#3030](https://github.com/vegaprotocol/vega/pull/3030) - Add more debug logging in execution engine
- [#3114](https://github.com/vegaprotocol/vega/pull/3114) - _Feature test refactor_: Standardise market definitions
- [#3122](https://github.com/vegaprotocol/vega/pull/3122) - _Feature test refactor_: Remove unused trading modes
- [#3124](https://github.com/vegaprotocol/vega/pull/3124) - _Feature test refactor_: Move submit order step to separate package
- [#3141](https://github.com/vegaprotocol/vega/pull/3141) - _Feature test refactor_: Move oracle data step to separate package
- [#3142](https://github.com/vegaprotocol/vega/pull/3142) - _Feature test refactor_: Move market steps to separate package
- [#3143](https://github.com/vegaprotocol/vega/pull/3143) - _Feature test refactor_: Move confirmed trades step to separate package
- [#3144](https://github.com/vegaprotocol/vega/pull/3144) - _Feature test refactor_: Move cancelled trades step to separate package
- [#3145](https://github.com/vegaprotocol/vega/pull/3145) - _Feature test refactor_: Move traders step to separate package
- [#3146](https://github.com/vegaprotocol/vega/pull/3146) - _Feature test refactor_: Create new step to verify margin accounts for a market
- [#3153](https://github.com/vegaprotocol/vega/pull/3153) - _Feature test refactor_: Create step to verify one account of each type per asset
- [#3152](https://github.com/vegaprotocol/vega/pull/3152) - _Feature test refactor_: Create step to deposit collateral
- [#3151](https://github.com/vegaprotocol/vega/pull/3151) - _Feature test refactor_: Create step to withdraw collateral
- [#3149](https://github.com/vegaprotocol/vega/pull/3149) - _Feature test refactor_: Merge deposit & verification steps
- [#3154](https://github.com/vegaprotocol/vega/pull/3154) - _Feature test refactor_: Create step to verify settlement balance for market
- [#3156](https://github.com/vegaprotocol/vega/pull/3156) - _Feature test refactor_: Rewrite margin levels step
- [#3178](https://github.com/vegaprotocol/vega/pull/3178) - _Feature test refactor_: Unify error handling steps
- [#3157](https://github.com/vegaprotocol/vega/pull/3157) - _Feature test refactor_: Various small fixes
- [#3101](https://github.com/vegaprotocol/vega/pull/3101) - _Feature test refactor_: Remove outdated feature tests
- [#3092](https://github.com/vegaprotocol/vega/pull/3092) - _Feature test refactor_: Add steps to test handling of LPs during auction
- [#3071](https://github.com/vegaprotocol/vega/pull/3071) - _Feature test refactor_: Fix typo

### 🐛 Fixes

- [#3018](https://github.com/vegaprotocol/vega/pull/3018) - Fix crash caused by distressed traders with LPs
- [#3029](https://github.com/vegaprotocol/vega/pull/3029) - API: LP orders were missing their reference data
- [#3031](https://github.com/vegaprotocol/vega/pull/3031) - Parties with cancelled LPs no longer receive fees
- [#3033](https://github.com/vegaprotocol/vega/pull/3033) - Improve handling of genesis block errors
- [#3036](https://github.com/vegaprotocol/vega/pull/3036) - Equity share is now correct when submitting initial order
- [#3048](https://github.com/vegaprotocol/vega/pull/3048) - LP submission now checks margin engine is started
- [#3070](https://github.com/vegaprotocol/vega/pull/3070) - Rewrite amending LPs
- [#3053](https://github.com/vegaprotocol/vega/pull/3053) - Rewrite cancel all order implementation
- [#3050](https://github.com/vegaprotocol/vega/pull/3050) - GraphQL: Order in `LiquidityOrder` is now nullable
- [#3056](https://github.com/vegaprotocol/vega/pull/3056) - Move `vegastream` to a separate repository
- [#3057](https://github.com/vegaprotocol/vega/pull/3057) - Ignore error if Tendermint stats is temporarily unavailable
- [#3058](https://github.com/vegaprotocol/vega/pull/3058) - Fix governance to use total supply rather than total deposited into network
- [#3062](https://github.com/vegaprotocol/vega/pull/3070) - Opening Auction no longer set to null on a market when auction completes
- [#3051](https://github.com/vegaprotocol/vega/pull/3051) - Rewrite LP refresh mechanism
- [#3080](https://github.com/vegaprotocol/vega/pull/3080) - Auctions now leave auction when `maximumDuration` is exceeded
- [#3075](https://github.com/vegaprotocol/vega/pull/3075) - Bond account is now correctly cleared when LPs are cancelled
- [#3074](https://github.com/vegaprotocol/vega/pull/3074) - Switch error reporting mechanism to stream error
- [#3069](https://github.com/vegaprotocol/vega/pull/3069) - Switch more error reporting mechanisms to stream error
- [#3081](https://github.com/vegaprotocol/vega/pull/3081) - Fix fee check for LP orders
- [#3087](https://github.com/vegaprotocol/vega/pull/3087) - GraphQL schema grammar & spelling fixes
- [#3185](https://github.com/vegaprotocol/vega/pull/3185) - LP orders are now accessed deterministically
- [#3131](https://github.com/vegaprotocol/vega/pull/3131) - GRPC api now shuts down gracefully
- [#3110](https://github.com/vegaprotocol/vega/pull/3110) - LP Bond is now returned if a market is rejected
- [#3115](https://github.com/vegaprotocol/vega/pull/3115) - Parties with closed out LPs can now submit new LPs
- [#3123](https://github.com/vegaprotocol/vega/pull/3123) - New market proposals with invalid Oracle definitions no longer crash core
- [#3131](https://github.com/vegaprotocol/vega/pull/3131) - GRPC api now shuts down gracefully
- [#3137](https://github.com/vegaprotocol/vega/pull/3137) - Pegged orders that fail to reprice correctly are now properly removed from the Market Depth API
- [#3168](https://github.com/vegaprotocol/vega/pull/3168) - Fix `intoProto` for `OracleSpecBinding`
- [#3106](https://github.com/vegaprotocol/vega/pull/3106) - Target Stake is now used as the Market Value Proxy during opening auction
- [#3103](https://github.com/vegaprotocol/vega/pull/3103) - Ensure all filled and partially filled orders are remove from the Market Depth API
- [#3095](https://github.com/vegaprotocol/vega/pull/3095) - GraphQL: Fix missing data in proposal subscription
- [#3085](https://github.com/vegaprotocol/vega/pull/3085) - Minor tidy-up of errors reported by `goland`

## 0.32.0

_2021-02-23_

More fixes, primarily related to liquidity provisioning (still disabled in this release) and asset withdrawals, which will soon be enabled in the UI.

Two minor breaking changes in the GraphQL API are included - one fixing a typo, the other changing the content of date fields on the withdrawal object - they're now date formatted.

### 🛠 Improvements

- [#3004](https://github.com/vegaprotocol/vega/pull/3004) - Incorporate `buf.yaml` tidy up submitted by `bufdev` on api-clients repo
- [#3002](https://github.com/vegaprotocol/vega/pull/3002) -🔥GraphQL: Withdrawal fields `expiry`, `createdAt` & `updatedAt` are now `RFC3339Nano` date formatted
- [#3000](https://github.com/vegaprotocol/vega/pull/3002) -🔥GraphQL: Fix typo in `prepareVote` mutation - `propopsalId` is now `proposalId`
- [#2957](https://github.com/vegaprotocol/vega/pull/2957) - REST: Add missing prepare endpoints (`PrepareProposal`, `PrepareVote`, `PrepareLiquiditySubmission`)

### 🐛 Fixes

- [#3011](https://github.com/vegaprotocol/vega/pull/3011) - Liquidity fees are distributed in to margin accounts, not general accounts
- [#2991](https://github.com/vegaprotocol/vega/pull/2991) - Liquidity Provisions are now rejected if there is not enough collateral
- [#2990](https://github.com/vegaprotocol/vega/pull/2990) - Fix a lock caused by GraphQL subscribers unsubscribing from certain endpoints
- [#2996](https://github.com/vegaprotocol/vega/pull/2986) - Liquidity Provisions are now parked when repricing fails
- [#2951](https://github.com/vegaprotocol/vega/pull/2951) - Store reference prices when parking pegs for auction
- [#2982](https://github.com/vegaprotocol/vega/pull/2982) - Fix withdrawal data availability before it is verified
- [#2981](https://github.com/vegaprotocol/vega/pull/2981) - Fix sending multisig bundle for withdrawals before threshold is reached
- [#2964](https://github.com/vegaprotocol/vega/pull/2964) - Extend auctions if uncrossing price is unreasonable
- [#2961](https://github.com/vegaprotocol/vega/pull/2961) - GraphQL: Fix incorrect market in bond account resolver
- [#2958](https://github.com/vegaprotocol/vega/pull/2958) - Create `third_party` folder to avoid excluding vendor protobuf files in build
- [#3009](https://github.com/vegaprotocol/vega/pull/3009) - Remove LP commitments when a trader is closed out
- [#3012](https://github.com/vegaprotocol/vega/pull/3012) - Remove LP commitments when a trader reduces their commitment to 0

## 0.31.0

_2021-02-09_

Many of the fixes below relate to Liquidity Commitments, which are still disabled in testnet, and Data Sourcing, which is also not enabled. Data Sourcing (a.k.a Oracles) is one of the last remaining pieces we need to complete settlement at instrument expiry, and Liquidity Commitment will be enabled when the functionality has been stabilised.

This release does improve protocol documentation, with all missing fields filled in and the explanations for Pegged Orders expanded. Two crashers have been fixed, although the first is already live as hotfix on testnet, and the other is in functionality that is not yet enabled.

This release also makes some major API changes:

- `api.TradingClient` -> `api.v1.TradingServiceClient`
- `api.TradingDataClient` -> `api.v1.TradingDataServiceClient`
- Fields have changed from camel-case to snake-cased (e.g. `someFieldName` is now `some_field_name`)
- All API calls now have request and response messages whose names match the API call name (e.g. `GetSomething` now has a request called `GetSomethingRequest` and a response called `GetSomethingResponse`)
- See [#2879](https://github.com/vegaprotocol/vega/pull/2879) for details

### 🛠 Improvements

- [#2879](https://github.com/vegaprotocol/vega/pull/2879) - 🔥Update all the protobuf files with Buf recommendations
- [#2847](https://github.com/vegaprotocol/vega/pull/2847) - Improve proto documentation, in particular for pegged orders
- [#2905](https://github.com/vegaprotocol/vega/pull/2905) - Update `vega verify` command to verify genesis block files
- [#2851](https://github.com/vegaprotocol/vega/pull/2851) - Enable distribution of liquidity fees to liquidity providers
- [#2871](https://github.com/vegaprotocol/vega/pull/2871) - Add `submitOracleData` command
- [#2887](https://github.com/vegaprotocol/vega/pull/2887) - Add Open Oracle data processing & data normalisation
- [#2915](https://github.com/vegaprotocol/vega/pull/2915) - Add Liquidity Commitments to API responses

### 🐛 Fixes

- [#2913](https://github.com/vegaprotocol/vega/pull/2913) - Fix market lifecycle events not being published through event bus API
- [#2906](https://github.com/vegaprotocol/vega/pull/2906) - Add new process for calculating margins for orders during auction
- [#2887](https://github.com/vegaprotocol/vega/pull/2887) - Liquidity Commitment fix-a-thon
- [#2879](https://github.com/vegaprotocol/vega/pull/2879) - Apply `Buf` lint recommendations
- [#2872](https://github.com/vegaprotocol/vega/pull/2872) - Improve field names in fee distribution package
- [#2867](https://github.com/vegaprotocol/vega/pull/2867) - Fix GraphQL bug: deposits `creditedAt` incorrectly showed `createdAt` time, not credit time
- [#2858](https://github.com/vegaprotocol/vega/pull/2858) - Fix crasher caused by parking pegged orders for auction
- [#2852](https://github.com/vegaprotocol/vega/pull/2852) - Remove unused binaries from CI builds
- [#2850](https://github.com/vegaprotocol/vega/pull/2850) - Fix bug that caused fees to be charged for pegged orders
- [#2893](https://github.com/vegaprotocol/vega/pull/2893) - Remove unused dependency in repricing
- [#2929](https://github.com/vegaprotocol/vega/pull/2929) - Refactor GraphQL resolver for withdrawals
- [#2939](https://github.com/vegaprotocol/vega/pull/2939) - Fix crasher caused by incorrectly loading Fee account for transfers

## 0.30.0

_2021-01-19_

This release enables (or more accurately, re-enables previously disabled) pegged orders, meaning they're finally here :tada:

The Ethereum bridge also received some work - in particular the number of confirmations we wait for on Ethereum is now controlled by a governance parameter. Being a governance parameter, that means that the value can be changed by a governance vote. Slightly related: You can now fetch active governance proposals via REST.

:one: We also switch to [Buf](https://buf.build/) for our protobuf workflow. This was one of the pre-requisites for opening up our api clients build process, and making the protobuf files open source. More on that soon!

:two: This fixes an issue on testnet where votes were not registered when voting on open governance proposals. The required number of Ropsten `VOTE` tokens was being calculated incorrectly on testnet, leading to all votes quietly being ignored. In 0.30.0, voting works as expected again.

### ✨ New

- [#2732](https://github.com/vegaprotocol/vega/pull/2732) Add REST endpoint to fetch all proposals (`/governance/proposals`)
- [#2735](https://github.com/vegaprotocol/vega/pull/2735) Add `FeeSplitter` to correctly split fee portion of an aggressive order
- [#2745](https://github.com/vegaprotocol/vega/pull/2745) Add transfer bus events for withdrawals and deposits
- [#2754](https://github.com/vegaprotocol/vega/pull/2754) Add New Market bus event
- [#2778](https://github.com/vegaprotocol/vega/pull/2778) Switch to [Buf](https://buf.build/) :one:
- [#2785](https://github.com/vegaprotocol/vega/pull/2785) Add configurable required confirmations for bridge transactions
- [#2791](https://github.com/vegaprotocol/vega/pull/2791) Add Supplied State to market data
- [#2793](https://github.com/vegaprotocol/vega/pull/2793) 🔥Rename `marketState` to `marketTradingMode`, add new `marketState` enum (`ACTIVE`, `SUSPENDED` or `PENDING`)
- [#2833](https://github.com/vegaprotocol/vega/pull/2833) Add fees estimate for pegged orders
- [#2838](https://github.com/vegaprotocol/vega/pull/2838) Add bond and fee transfers

### 🛠 Improvements

- [#2835](https://github.com/vegaprotocol/vega/pull/2835) Fix voting for proposals :two:
- [#2830](https://github.com/vegaprotocol/vega/pull/2830) Refactor pegged order repricing
- [#2827](https://github.com/vegaprotocol/vega/pull/2827) Refactor expiring orders lists
- [#2821](https://github.com/vegaprotocol/vega/pull/2821) Handle liquidity commitments on market proposals
- [#2816](https://github.com/vegaprotocol/vega/pull/2816) Add changing liquidity commitment when not enough stake
- [#2805](https://github.com/vegaprotocol/vega/pull/2805) Fix read nodes lagging if they receive votes but not a bridge event
- [#2804](https://github.com/vegaprotocol/vega/pull/2804) Fix various minor bridge confirmation bugs
- [#2800](https://github.com/vegaprotocol/vega/pull/2800) Fix removing pegged orders that are rejected when unparked
- [#2799](https://github.com/vegaprotocol/vega/pull/2799) Fix crasher when proposing an update to network parameters
- [#2797](https://github.com/vegaprotocol/vega/pull/2797) Update target stake to include mark price
- [#2783](https://github.com/vegaprotocol/vega/pull/2783) Fix price monitoring integration tests
- [#2780](https://github.com/vegaprotocol/vega/pull/2780) Add more unit tests for pegged order price amends
- [#2774](https://github.com/vegaprotocol/vega/pull/2774) Fix cancelling all orders
- [#2768](https://github.com/vegaprotocol/vega/pull/2768) Fix GraphQL: Allow `marketId` to be null when it is invalid
- [#2767](https://github.com/vegaprotocol/vega/pull/2767) Fix parked pegged orders to have a price of 0 explicitly
- [#2766](https://github.com/vegaprotocol/vega/pull/2766) Disallow GFN to GTC/GTT amends
- [#2765](https://github.com/vegaprotocol/vega/pull/2765) Fix New Market bus event being sent more than once
- [#2763](https://github.com/vegaprotocol/vega/pull/2763) Add rounding to pegged order mid prices that land on non integer values
- [#2795](https://github.com/vegaprotocol/vega/pull/2795) Fix typos in GraphQL schema documentation
- [#2762](https://github.com/vegaprotocol/vega/pull/2762) Fix more typos in GraphQL schema documentation
- [#2758](https://github.com/vegaprotocol/vega/pull/2758) Fix error handling when amending some pegged order types
- [#2757](https://github.com/vegaprotocol/vega/pull/2757) Remove order from pegged list when it becomes inactive
- [#2756](https://github.com/vegaprotocol/vega/pull/2756) Add more panics to the core
- [#2750](https://github.com/vegaprotocol/vega/pull/2750) Remove expiring orders when amending to non GTT
- [#2671](https://github.com/vegaprotocol/vega/pull/2671) Add extra integration tests for uncrossing at auction end
- [#2746](https://github.com/vegaprotocol/vega/pull/2746) Fix potential divide by 0 in position calculation
- [#2743](https://github.com/vegaprotocol/vega/pull/2743) Add check for pegged orders impacted by order expiry
- [#2737](https://github.com/vegaprotocol/vega/pull/2737) Remove the ability to amend a pegged order's price
- [#2724](https://github.com/vegaprotocol/vega/pull/2724) Add price monitoring tests for order amendment
- [#2723](https://github.com/vegaprotocol/vega/pull/2723) Fix fee monitoring values during auction
- [#2721](https://github.com/vegaprotocol/vega/pull/2721) Fix incorrect reference when amending pegged orders
- [#2717](https://github.com/vegaprotocol/vega/pull/2717) Fix incorrect error codes for IOC and FOK orders during auction
- [#2715](https://github.com/vegaprotocol/vega/pull/2715) Update price monitoring to use reference price when syncing with risk model
- [#2711](https://github.com/vegaprotocol/vega/pull/2711) Refactor governance event subscription

## 0.29.0

_2020-12-07_

Note that you'll see a lot of changes related to **Pegged Orders** and **Liquidity Commitments**. These are still in testing, so these two types cannot currently be used in _Testnet_.

### ✨ New

- [#2534](https://github.com/vegaprotocol/vega/pull/2534) Implements amends for pegged orders
- [#2493](https://github.com/vegaprotocol/vega/pull/2493) Calculate market target stake
- [#2649](https://github.com/vegaprotocol/vega/pull/2649) Add REST governance endpoints
- [#2429](https://github.com/vegaprotocol/vega/pull/2429) Replace inappropriate wording in the codebase
- [#2617](https://github.com/vegaprotocol/vega/pull/2617) Implements proposal to update network parameters
- [#2622](https://github.com/vegaprotocol/vega/pull/2622) Integrate the liquidity engine into the market
- [#2683](https://github.com/vegaprotocol/vega/pull/2683) Use the Ethereum block log index to de-duplicate Ethereum transactions
- [#2674](https://github.com/vegaprotocol/vega/pull/2674) Update ERC20 token and bridges ABIs / codegen
- [#2690](https://github.com/vegaprotocol/vega/pull/2690) Add instruction to debug integration tests with DLV
- [#2680](https://github.com/vegaprotocol/vega/pull/2680) Add price monitoring bounds to the market data API

### 🛠 Improvements

- [#2589](https://github.com/vegaprotocol/vega/pull/2589) Fix cancellation of pegged orders
- [#2659](https://github.com/vegaprotocol/vega/pull/2659) Fix panic in execution engine when GFN order are submit at auction start
- [#2661](https://github.com/vegaprotocol/vega/pull/2661) Handle missing error conversion in GraphQL API
- [#2621](https://github.com/vegaprotocol/vega/pull/2621) Fix pegged order creating duplicated order events
- [#2666](https://github.com/vegaprotocol/vega/pull/2666) Prevent the node to DDOS the Ethereum node when lots of deposits happen
- [#2653](https://github.com/vegaprotocol/vega/pull/2653) Fix indicative price and volume calculation
- [#2649](https://github.com/vegaprotocol/vega/pull/2649) Fix a typo in market price monitoring parameters API
- [#2650](https://github.com/vegaprotocol/vega/pull/2650) Change governance minimum proposer balance to be a minimum amount of token instead of a factor of the total supply
- [#2675](https://github.com/vegaprotocol/vega/pull/2675) Fix an GraphQL enum conversion
- [#2691](https://github.com/vegaprotocol/vega/pull/2691) Fix spelling in a network parameter
- [#2696](https://github.com/vegaprotocol/vega/pull/2696) Fix panic when uncrossing auction
- [#2984](https://github.com/vegaprotocol/vega/pull/2698) Fix price monitoring by feeding it the uncrossing price at end of opening auction
- [#2705](https://github.com/vegaprotocol/vega/pull/2705) Fix a bug related to order being sorted by creating time in the matching engine price levels

## 0.28.0

_2020-11-25_

Vega release logs contain a 🔥 emoji to denote breaking API changes. 🔥🔥 is a new combination denoting something that may significantly change your experience - from this release forward, transactions from keys that have no collateral on the network will _always_ be rejected. As there are no transactions that don't either require collateral themselves, or an action to have been taken that already required collateral, we are now rejecting these as soon as possible.

We've also added support for synchronously submitting transactions. This can make error states easier to catch. Along with this you can now subscribe to error events in the event bus.

Also: Note that you'll see a lot of changes related to **Pegged Orders** and **Liquidity Commitments**. These are still in testing, so these two types cannot currently be used in _Testnet_.

### ✨ New

- [#2634](https://github.com/vegaprotocol/vega/pull/2634) Avoid caching transactions before they are rate/balance limited
- [#2626](https://github.com/vegaprotocol/vega/pull/2626) Add a transaction submit type to GraphQL
- [#2624](https://github.com/vegaprotocol/vega/pull/2624) Add mutexes to assets maps
- [#2593](https://github.com/vegaprotocol/vega/pull/2503) 🔥🔥 Reject transactions
- [#2453](https://github.com/vegaprotocol/vega/pull/2453) 🔥 Remove `baseName` field from markets
- [#2536](https://github.com/vegaprotocol/vega/pull/2536) Add Liquidity Measurement engine
- [#2539](https://github.com/vegaprotocol/vega/pull/2539) Add Liquidity Provisioning Commitment handling to markets
- [#2540](https://github.com/vegaprotocol/vega/pull/2540) Add support for amending pegged orders
- [#2549](https://github.com/vegaprotocol/vega/pull/2549) Add calculation for liquidity order sizes
- [#2553](https://github.com/vegaprotocol/vega/pull/2553) Allow pegged orders to have a price of 0
- [#2555](https://github.com/vegaprotocol/vega/pull/2555) Update Event stream votes to contain proposal ID
- [#2556](https://github.com/vegaprotocol/vega/pull/2556) Update Event stream to contain error events
- [#2560](https://github.com/vegaprotocol/vega/pull/2560) Add Pegged Order details to GraphQL
- [#2607](https://github.com/vegaprotocol/vega/pull/2807) Add support for parking orders during auction

### 🛠 Improvements

- [#2634](https://github.com/vegaprotocol/vega/pull/2634) Avoid caching transactions before they are rate/balance limited
- [#2626](https://github.com/vegaprotocol/vega/pull/2626) Add a transaction submit type to GraphQL
- [#2624](https://github.com/vegaprotocol/vega/pull/2624) Add mutexes to assets maps
- [#2623](https://github.com/vegaprotocol/vega/pull/2623) Fix concurrent map access in assets
- [#2608](https://github.com/vegaprotocol/vega/pull/2608) Add sync/async equivalents for `submitTX`
- [#2618](https://github.com/vegaprotocol/vega/pull/2618) Disable storing API-related data on validator nodes
- [#2615](https://github.com/vegaprotocol/vega/pull/2618) Expand static checks
- [#2613](https://github.com/vegaprotocol/vega/pull/2613) Remove unused internal `cancelOrderById` function
- [#2530](https://github.com/vegaprotocol/vega/pull/2530) Governance asset for the network is now set in the genesis block
- [#2533](https://github.com/vegaprotocol/vega/pull/2533) More efficiently close channels in subscriptions
- [#2554](https://github.com/vegaprotocol/vega/pull/2554) Fix mid-price to 0 when best bid and average are unavailable and pegged order price is 0
- [#2565](https://github.com/vegaprotocol/vega/pull/2565) Cancelled pegged orders now have the correct status
- [#2568](https://github.com/vegaprotocol/vega/pull/2568) Prevent pegged orders from being repriced
- [#2570](https://github.com/vegaprotocol/vega/pull/2570) Expose probability of trading
- [#2576](https://github.com/vegaprotocol/vega/pull/2576) Use static best bid/ask price for pegged order repricing
- [#2581](https://github.com/vegaprotocol/vega/pull/2581) Fix order of messages when cancelling a pegged order
- [#2586](https://github.com/vegaprotocol/vega/pull/2586) Fix blank `txHash` in deposit API types
- [#2591](https://github.com/vegaprotocol/vega/pull/2591) Pegged orders are now cancelled when all orders are cancelled
- [#2609](https://github.com/vegaprotocol/vega/pull/2609) Improve expiry of pegged orders
- [#2610](https://github.com/vegaprotocol/vega/pull/2609) Improve removal of liquidity commitment orders when manual orders satisfy liquidity provisioning commitments

## 0.27.0

_2020-10-30_

This release contains a fix (read: large reduction in memory use) around auction modes with particularly large order books that caused slow block times when handling orders placed during an opening auction. It also contains a lot of internal work related to the liquidity provision mechanics.

### ✨ New

- [#2498](https://github.com/vegaprotocol/vega/pull/2498) Automatically create a bond account for liquidity providers
- [#2596](https://github.com/vegaprotocol/vega/pull/2496) Create liquidity measurement API
- [#2490](https://github.com/vegaprotocol/vega/pull/2490) GraphQL: Add Withdrawal and Deposit events to event bus
- [#2476](https://github.com/vegaprotocol/vega/pull/2476) 🔥`MarketData` now uses RFC339 formatted times, not seconds
-
- [#2473](https://github.com/vegaprotocol/vega/pull/2473) Add network parameters related to target stake calculation
- [#2506](https://github.com/vegaprotocol/vega/pull/2506) Network parameters can now contain JSON configuration

### 🛠 Improvements

- [#2521](https://github.com/vegaprotocol/vega/pull/2521) Optimise memory usage when building cumulative price levels
- [#2520](https://github.com/vegaprotocol/vega/pull/2520) Fix indicative price calculation
- [#2517](https://github.com/vegaprotocol/vega/pull/2517) Improve command line for rate limiting in faucet & wallet
- [#2510](https://github.com/vegaprotocol/vega/pull/2510) Remove reference to external risk model
- [#2509](https://github.com/vegaprotocol/vega/pull/2509) Fix panic when loading an invalid genesis configuration
- [#2502](https://github.com/vegaprotocol/vega/pull/2502) Fix pointer when using amend in place
- [#2487](https://github.com/vegaprotocol/vega/pull/2487) Remove context from struct that didn't need it
- [#2485](https://github.com/vegaprotocol/vega/pull/2485) Refactor event bus event transmission
- [#2481](https://github.com/vegaprotocol/vega/pull/2481) Add `LiquidityProvisionSubmission` transaction
- [#2480](https://github.com/vegaprotocol/vega/pull/2480) Remove unused code
- [#2479](https://github.com/vegaprotocol/vega/pull/2479) Improve validation of external resources
- [#1936](https://github.com/vegaprotocol/vega/pull/1936) Upgrade to Tendermint 0.33.8

## 0.26.1

_2020-10-23_

Fixes a number of issues discovered during the testing of 0.26.0.

### 🛠 Improvements

- [#2463](https://github.com/vegaprotocol/vega/pull/2463) Further reliability fixes for the event bus
- [#2469](https://github.com/vegaprotocol/vega/pull/2469) Fix incorrect error returned when a trader places an order in an asset that they have no account for (was `InvalidPartyID`, now `InsufficientAssetBalance`)
- [#2458](https://github.com/vegaprotocol/vega/pull/2458) REST: Fix a crasher when a market is proposed without specifying auction times

## 0.26.0

_2020-10-20_

The events API added in 0.25.0 had some reliability issues when a large volume of events were being emitted. This release addresses that in two ways:

- The gRPC event stream now takes a parameter that sets a batch size. A client will receive the events when the batch limit is hit.
- GraphQL is now limited to one event type per subscription, and we also removed the ALL event type as an option. This was due to the GraphQL gateway layer taking too long to process the full event stream, leading to sporadic disconnections.

These two fixes combined make both the gRPC and GraphQL streams much more reliable under reasonably heavy load. Let us know if you see any other issues. The release also adds some performance improvements to the way the core processes Tendermint events, some documentation improvements, and some additional debug tools.

### ✨ New

- [#2319](https://github.com/vegaprotocol/vega/pull/2319) Add fee estimate API endpoints to remaining APIs
- [#2321](https://github.com/vegaprotocol/vega/pull/2321) 🔥 Change `estimateFee` to `estimateOrder` in GraphQL
- [#2327](https://github.com/vegaprotocol/vega/pull/2327) 🔥 GraphQL: Event bus API - remove ALL type & limit subscription to one event type
- [#2343](https://github.com/vegaprotocol/vega/pull/2343) 🔥 Add batching support to stream subscribers

### 🛠 Improvements

- [#2229](https://github.com/vegaprotocol/vega/pull/2229) Add Price Monitoring module
- [#2246](https://github.com/vegaprotocol/vega/pull/2246) Add new market depth subscription methods
- [#2298](https://github.com/vegaprotocol/vega/pull/2298) Improve error messages for Good For Auction/Good For Normal rejections
- [#2301](https://github.com/vegaprotocol/vega/pull/2301) Add validation for GFA/GFN orders
- [#2307](https://github.com/vegaprotocol/vega/pull/2307) Implement app state hash
- [#2312](https://github.com/vegaprotocol/vega/pull/2312) Add validation for market proposal risk parameters
- [#2313](https://github.com/vegaprotocol/vega/pull/2313) Add transaction replay protection
- [#2314](https://github.com/vegaprotocol/vega/pull/2314) GraphQL: Improve response when market does not exist
- [#2315](https://github.com/vegaprotocol/vega/pull/2315) GraphQL: Improve response when party does not exist
- [#2316](https://github.com/vegaprotocol/vega/pull/2316) Documentation: Improve documentation for fee estimate endpoint
- [#2318](https://github.com/vegaprotocol/vega/pull/2318) Documentation: Improve documentation for governance data endpoints
- [#2324](https://github.com/vegaprotocol/vega/pull/2324) Cache transactions already seen by `checkTX`
- [#2328](https://github.com/vegaprotocol/vega/pull/2328) Add test covering context cancellation mid data-sending
- [#2331](https://github.com/vegaprotocol/vega/pull/2331) Internal refactor of network parameter storage
- [#2334](https://github.com/vegaprotocol/vega/pull/2334) Rewrite `vegastream` to use the event bus
- [#2333](https://github.com/vegaprotocol/vega/pull/2333) Fix context for events, add block hash and event id
- [#2335](https://github.com/vegaprotocol/vega/pull/2335) Add ABCI event recorder
- [#2341](https://github.com/vegaprotocol/vega/pull/2341) Ensure event slices cannot be empty
- [#2345](https://github.com/vegaprotocol/vega/pull/2345) Handle filled orders in the market depth service before new orders are added
- [#2346](https://github.com/vegaprotocol/vega/pull/2346) CI: Add missing environment variables
- [#2348](https://github.com/vegaprotocol/vega/pull/2348) Use cached transactions in `checkTX`
- [#2349](https://github.com/vegaprotocol/vega/pull/2349) Optimise accounts map accesses
- [#2351](https://github.com/vegaprotocol/vega/pull/2351) Fix sequence ID related to market `OnChainTimeUpdate`
- [#2355](https://github.com/vegaprotocol/vega/pull/2355) Update coding style doc with info on log levels
- [#2358](https://github.com/vegaprotocol/vega/pull/2358) Add documentation and comments for `events.proto`
- [#2359](https://github.com/vegaprotocol/vega/pull/2359) Fix out of bounds index crash
- [#2364](https://github.com/vegaprotocol/vega/pull/2364) Add mutex to protect map access
- [#2366](https://github.com/vegaprotocol/vega/pull/2366) Auctions: Reject IOC/FOK orders
- [#2368](https://github.com/vegaprotocol/vega/pull/2370) Tidy up genesis market instantiation
- [#2369](https://github.com/vegaprotocol/vega/pull/2369) Optimise event bus to reduce CPU usage
- [#2370](https://github.com/vegaprotocol/vega/pull/2370) Event stream: Send batches instead of single events
- [#2376](https://github.com/vegaprotocol/vega/pull/2376) GraphQL: Remove verbose logging
- [#2377](https://github.com/vegaprotocol/vega/pull/2377) Update tendermint stats less frequently for Vega stats API endpoint
- [#2381](https://github.com/vegaprotocol/vega/pull/2381) Event stream: Reduce CPU load, depending on batch size
- [#2382](https://github.com/vegaprotocol/vega/pull/2382) GraphQL: Make event stream batch size mandatory
- [#2401](https://github.com/vegaprotocol/vega/pull/2401) Event stream: Fix CPU spinning after stream close
- [#2404](https://github.com/vegaprotocol/vega/pull/2404) Auctions: Add fix for crash during auction exit
- [#2419](https://github.com/vegaprotocol/vega/pull/2419) Make the price level wash trade check configurable
- [#2432](https://github.com/vegaprotocol/vega/pull/2432) Use `EmitDefaults` on `jsonpb.Marshaler`
- [#2431](https://github.com/vegaprotocol/vega/pull/2431) GraphQL: Add price monitoring
- [#2433](https://github.com/vegaprotocol/vega/pull/2433) Validate amend orders with GFN and GFA
- [#2436](https://github.com/vegaprotocol/vega/pull/2436) Return a permission denied error for a non-allowlisted public key
- [#2437](https://github.com/vegaprotocol/vega/pull/2437) Undo accidental code removal
- [#2438](https://github.com/vegaprotocol/vega/pull/2438) GraphQL: Fix a resolver error when markets are in auction mode
- [#2441](https://github.com/vegaprotocol/vega/pull/2441) GraphQL: Remove unnecessary validations
- [#2442](https://github.com/vegaprotocol/vega/pull/2442) GraphQL: Update library; improve error responses
- [#2447](https://github.com/vegaprotocol/vega/pull/2447) REST: Fix HTTP verb for network parameters query
- [#2443](https://github.com/vegaprotocol/vega/pull/2443) Auctions: Add check for opening auction duration during market creation

## 0.25.1

_2020-10-14_

This release backports two fixes from the forthcoming 0.26.0 release.

### 🛠 Improvements

- [#2354](https://github.com/vegaprotocol/vega/pull/2354) Update `OrderEvent` to copy by value
- [#2379](https://github.com/vegaprotocol/vega/pull/2379) Add missing `/governance/prepare/vote` REST endpoint

## 0.25.0

_2020-09-24_

This release adds the event bus API, allowing for much greater introspection in to the operation of a node. We've also re-enabled the order amends API, as well as a long list of fixes.

### ✨ New

- [#2281](https://github.com/vegaprotocol/vega/pull/2281) Enable opening auctions
- [#2205](https://github.com/vegaprotocol/vega/pull/2205) Add GraphQL event stream API
- [#2219](https://github.com/vegaprotocol/vega/pull/2219) Add deposits API
- [#2222](https://github.com/vegaprotocol/vega/pull/2222) Initial asset list is now loaded from genesis configuration, not external configuration
- [#2238](https://github.com/vegaprotocol/vega/pull/2238) Re-enable order amend API
- [#2249](https://github.com/vegaprotocol/vega/pull/2249) Re-enable TX rate limit by party ID
- [#2240](https://github.com/vegaprotocol/vega/pull/2240) Add time to position responses

### 🛠 Improvements

- [#2211](https://github.com/vegaprotocol/vega/pull/2211) 🔥 GraphQL: Field case change `proposalId` -> `proposalID`
- [#2218](https://github.com/vegaprotocol/vega/pull/2218) 🔥 GraphQL: Withdrawals now return a `Party`, not a party ID
- [#2202](https://github.com/vegaprotocol/vega/pull/2202) Fix time validation for proposals when all times are the same
- [#2206](https://github.com/vegaprotocol/vega/pull/2206) Reduce log noise from statistics endpoint
- [#2207](https://github.com/vegaprotocol/vega/pull/2207) Automatically reload node configuration
- [#2209](https://github.com/vegaprotocol/vega/pull/2209) GraphQL: fix proposal rejection enum
- [#2210](https://github.com/vegaprotocol/vega/pull/2210) Refactor order service to not require blockchain client
- [#2213](https://github.com/vegaprotocol/vega/pull/2213) Improve error clarity for invalid proposals
- [#2216](https://github.com/vegaprotocol/vega/pulls/2216) Ensure all GRPC endpoints use real time, not Vega time
- [#2231](https://github.com/vegaprotocol/vega/pull/2231) Refactor processor to no longer require collateral
- [#2232](https://github.com/vegaprotocol/vega/pull/2232) Clean up logs that dumped raw bytes
- [#2233](https://github.com/vegaprotocol/vega/pull/2233) Remove generate method from execution engine
- [#2234](https://github.com/vegaprotocol/vega/pull/2234) Remove `authEnabled` setting
- [#2236](https://github.com/vegaprotocol/vega/pull/2236) Simply order amendment logging
- [#2237](https://github.com/vegaprotocol/vega/pull/2237) Clarify fees attribution in transfers
- [#2239](https://github.com/vegaprotocol/vega/pull/2239) Ensure margin is released immediately, not on next mark to market
- [#2241](https://github.com/vegaprotocol/vega/pull/2241) Load log level in processor app
- [#2245](https://github.com/vegaprotocol/vega/pull/2245) Fix a concurrent map access in positions API
- [#2247](https://github.com/vegaprotocol/vega/pull/2247) Improve logging on a TX with an invalid signature
- [#2252](https://github.com/vegaprotocol/vega/pull/2252) Fix incorrect order count in Market Depth API
- [#2254](https://github.com/vegaprotocol/vega/pull/2254) Fix concurrent map access in Market Depth API
- [#2269](https://github.com/vegaprotocol/vega/pull/2269) GraphQL: Fix party filtering for event bus API
- [#2266](https://github.com/vegaprotocol/vega/pull/2266) Refactor transaction codec
- [#2275](https://github.com/vegaprotocol/vega/pull/2275) Prevent opening auctions from closing early
- [#2262](https://github.com/vegaprotocol/vega/pull/2262) Clear potential position properly when an order is cancelled for self trading
- [#2286](https://github.com/vegaprotocol/vega/pull/2286) Add sequence ID to event bus events
- [#2288](https://github.com/vegaprotocol/vega/pull/2288) Fix auction events not appearing in GraphQL event bus
- [#2294](https://github.com/vegaprotocol/vega/pull/2294) Fixing incorrect order iteration in auctions
- [#2285](https://github.com/vegaprotocol/vega/pull/2285) Check auction times
- [#2283](https://github.com/vegaprotocol/vega/pull/2283) Better handling of 0 `expiresAt`

## 0.24.0

_2020-09-04_

One new API endpoint allows cancelling multiple orders simultaneously, either all orders by market, a single order in a specific market, or just all open orders.

Other than that it's mainly bugfixes, many of which fix subtly incorrect API output.

### ✨ New

- [#2107](https://github.com/vegaprotocol/vega/pull/2107) Support for cancelling multiple orders at once
- [#2186](https://github.com/vegaprotocol/vega/pull/2186) Add per-party rate-limit of 50 requests over 3 blocks

### 🛠 Improvements

- [#2177](https://github.com/vegaprotocol/vega/pull/2177) GraphQL: Add Governance proposal metadata
- [#2098](https://github.com/vegaprotocol/vega/pull/2098) Fix crashed in event bus
- [#2041](https://github.com/vegaprotocol/vega/pull/2041) Fix a rounding error in the output of Positions API
- [#1934](https://github.com/vegaprotocol/vega/pull/1934) Improve API documentation
- [#2110](https://github.com/vegaprotocol/vega/pull/2110) Send Infrastructure fees to the correct account
- [#2117](https://github.com/vegaprotocol/vega/pull/2117) Prevent creation of withdrawal requests for more than the available balance
- [#2136](https://github.com/vegaprotocol/vega/pull/2136) gRPC: Fetch all accounts for a market did not return all accounts
- [#2151](https://github.com/vegaprotocol/vega/pull/2151) Prevent wasteful event bus subscriptions
- [#2167](https://github.com/vegaprotocol/vega/pull/2167) Ensure events in the event bus maintain their order
- [#2178](https://github.com/vegaprotocol/vega/pull/2178) Fix API returning incorrectly formatted orders when a party has no collateral

## 0.23.1

_2020-08-27_

This release backports a fix from the forthcoming 0.24.0 release that fixes a GraphQL issue with the new `Asset` type. When fetching the Assets from the top level, all the details came through. When fetching them as a nested property, only the ID was filled in. This is now fixed.

### 🛠 Improvements

- [#2140](https://github.com/vegaprotocol/vega/pull/2140) GraphQL fix for fetching assets as nested properties

## 0.23.0

_2020-08-10_

This release contains a lot of groundwork for Fees and Auction mode.

**Fees** are incurred on every trade on Vega. Those fees are divided between up to three recipient types, but traders will only see one collective fee charged. The fees reward liquidity providers, infrastructure providers and market makers.

- The liquidity portion of the fee is paid to market makers for providing liquidity, and is transferred to the market-maker fee pool for the market.
- The infrastructure portion of the fee, which is paid to validators as a reward for running the infrastructure of the network, is transferred to the infrastructure fee pool for that asset. It is then periodically distributed to the validators.
- The maker portion of the fee is transferred to the non-aggressive, or passive party in the trade (the maker, as opposed to the taker).

**Auction mode** is not enabled in this release, but the work is nearly complete for Opening Auctions on new markets.

💥 Please note, **this release disables order amends**. The team uncovered an issue in the Market Depth API output that is caused by order amends, so rather than give incorrect output, we've temporarily disabled the amendment of orders. They will return when the Market Depth API is fixed. For now, _amends will return an error_.

### ✨ New

- 💥 [#2092](https://github.com/vegaprotocol/vega/pull/2092) Disable order amends
- [#2027](https://github.com/vegaprotocol/vega/pull/2027) Add built in asset faucet endpoint
- [#2075](https://github.com/vegaprotocol/vega/pull/2075), [#2086](https://github.com/vegaprotocol/vega/pull/2086), [#2083](https://github.com/vegaprotocol/vega/pull/2083), [#2078](https://github.com/vegaprotocol/vega/pull/2078) Add time & size limits to faucet requests
- [#2068](https://github.com/vegaprotocol/vega/pull/2068) Add REST endpoint to fetch governance proposals by Party
- [#2058](https://github.com/vegaprotocol/vega/pull/2058) Add REST endpoints for fees
- [#2047](https://github.com/vegaprotocol/vega/pull/2047) Add `prepareWithdraw` endpoint

### 🛠 Improvements

- [#2061](https://github.com/vegaprotocol/vega/pull/2061) Fix Network orders being left as active
- [#2034](https://github.com/vegaprotocol/vega/pull/2034) Send `KeepAlive` messages on GraphQL subscriptions
- [#2031](https://github.com/vegaprotocol/vega/pull/2031) Add proto fields required for auctions
- [#2025](https://github.com/vegaprotocol/vega/pull/2025) Add auction mode (currently never triggered)
- [#2013](https://github.com/vegaprotocol/vega/pull/2013) Add Opening Auctions support to market framework
- [#2010](https://github.com/vegaprotocol/vega/pull/2010) Add documentation for Order Errors to proto source files
- [#2003](https://github.com/vegaprotocol/vega/pull/2003) Add fees support
- [#2004](https://github.com/vegaprotocol/vega/pull/2004) Remove @deprecated field from GraphQL input types (as it’s invalid)
- [#2000](https://github.com/vegaprotocol/vega/pull/2000) Fix `rejectionReason` for trades stopped for self trading
- [#1990](https://github.com/vegaprotocol/vega/pull/1990) Remove specified `tickSize` from market
- [#2066](https://github.com/vegaprotocol/vega/pull/2066) Fix validation of proposal timestamps to ensure that datestamps specify events in the correct order
- [#2043](https://github.com/vegaprotocol/vega/pull/2043) Track Event Queue events to avoid processing events from other chains twice

## 0.22.0

### 🐛 Bugfixes

- [#2096](https://github.com/vegaprotocol/vega/pull/2096) Fix concurrent map access in event forward

_2020-07-20_

This release primarily focuses on setting up Vega nodes to deal correctly with events sourced from other chains, working towards bridging assets from Ethereum. This includes responding to asset events from Ethereum, and support for validator nodes notarising asset movements and proposals.

It also contains a lot of bug fixes and improvements, primarily around an internal refactor to using an event bus to communicate between packages. Also included are some corrections for order statuses that were incorrectly being reported or left outdated on the APIs.

### ✨ New

- [#1825](https://github.com/vegaprotocol/vega/pull/1825) Add new Notary package for tracking multisig decisions for governance
- [#1837](https://github.com/vegaprotocol/vega/pull/1837) Add support for two-step governance processes such as asset proposals
- [#1856](https://github.com/vegaprotocol/vega/pull/1856) Implement handling of external chain events from the Event Queue
- [#1927](https://github.com/vegaprotocol/vega/pull/1927) Support ERC20 deposits
- [#1987](https://github.com/vegaprotocol/vega/pull/1987) Add `OpenInterest` field to markets
- [#1949](https://github.com/vegaprotocol/vega/pull/1949) Add `RejectionReason` field to rejected governance proposals

### 🛠 Improvements

- 💥 [#1988](https://github.com/vegaprotocol/vega/pull/1988) REST: Update orders endpoints to use POST, not PUT or DELETE
- 💥 [#1957](https://github.com/vegaprotocol/vega/pull/1957) GraphQL: Some endpoints returned a nullable array of Strings. Now they return an array of nullable strings
- 💥 [#1928](https://github.com/vegaprotocol/vega/pull/1928) GraphQL & GRPC: Remove broken `open` parameter from Orders endpoints. It returned ambiguous results
- 💥 [#1858](https://github.com/vegaprotocol/vega/pull/1858) Fix outdated order details for orders amended by cancel-and-replace
- 💥 [#1849](https://github.com/vegaprotocol/vega/pull/1849) Fix incorrect status on partially filled trades that would have matched with another order by the same user. Was `stopped`, now `rejected`
- 💥 [#1883](https://github.com/vegaprotocol/vega/pull/1883) REST & GraphQL: Market name is now based on the instrument name rather than being set separately
- [#1699](https://github.com/vegaprotocol/vega/pull/1699) Migrate Margin package to event bus
- [#1853](https://github.com/vegaprotocol/vega/pull/1853) Migrate Market package to event bus
- [#1844](https://github.com/vegaprotocol/vega/pull/1844) Migrate Governance package to event
- [#1877](https://github.com/vegaprotocol/vega/pull/1877) Migrate Position package to event
- [#1838](https://github.com/vegaprotocol/vega/pull/1838) GraphQL: Orders now include their `version` and `updatedAt`, which are useful when dealing with amended orders
- [#1841](https://github.com/vegaprotocol/vega/pull/1841) Fix: `expiresAt` on orders was validated at submission time, this has been moved to post-chain validation
- [#1849](https://github.com/vegaprotocol/vega/pull/1849) Improve Order documentation for `Status` and `TimeInForce`
- [#1861](https://github.com/vegaprotocol/vega/pull/1861) Remove single mutex in event bus
- [#1866](https://github.com/vegaprotocol/vega/pull/1866) Add mutexes for event bus access
- [#1889](https://github.com/vegaprotocol/vega/pull/1889) Improve event broker performance
- [#1891](https://github.com/vegaprotocol/vega/pull/1891) Fix context for event subscribers
- [#1889](https://github.com/vegaprotocol/vega/pull/1889) Address event bus performance issues
- [#1892](https://github.com/vegaprotocol/vega/pull/1892) Improve handling for new chain connection proposal
- [#1903](https://github.com/vegaprotocol/vega/pull/1903) Fix regressions in Candles API introduced by event bus
- [#1940](https://github.com/vegaprotocol/vega/pull/1940) Add new asset proposals to GraphQL API
- [#1943](https://github.com/vegaprotocol/vega/pull/1943) Validate list of allowed assets

## 0.21.0

_2020-06-18_

A follow-on from 0.20.1, this release includes a fix for the GraphQL API returning inconsistent values for the `side` field on orders, leading to Vega Console failing to submit orders. As a bonus there is another GraphQL improvement, and two fixes that return more correct values for filled network orders and expired orders.

### 🛠 Improvements

- 💥 [#1820](https://github.com/vegaprotocol/vega/pull/1820) GraphQL: Non existent parties no longer return a GraphQL error
- 💥 [#1784](https://github.com/vegaprotocol/vega/pull/1784) GraphQL: Update schema and fix enum mappings from Proto
- 💥 [#1761](https://github.com/vegaprotocol/vega/pull/1761) Governance: Improve processing of Proposals
- [#1822](https://github.com/vegaprotocol/vega/pull/1822) Remove duplicate updates to `createdAt`
- [#1818](https://github.com/vegaprotocol/vega/pull/1818) Trades: Replace buffer with events
- [#1812](https://github.com/vegaprotocol/vega/pull/1812) Governance: Improve logging
- [#1810](https://github.com/vegaprotocol/vega/pull/1810) Execution: Set order status for fully filled network orders to be `FILLED`
- [#1803](https://github.com/vegaprotocol/vega/pull/1803) Matching: Set `updatedAt` when orders expire
- [#1780](https://github.com/vegaprotocol/vega/pull/1780) APIs: Reject `NETWORK` orders
- [#1792](https://github.com/vegaprotocol/vega/pull/1792) Update Golang to 1.14 and tendermint to 0.33.5

## 0.20.1

_2020-06-18_

This release fixes one small bug that was causing many closed streams, which was a problem for API clients.

## 🛠 Improvements

- [#1813](https://github.com/vegaprotocol/vega/pull/1813) Set `PartyEvent` type to party event

## 0.20.0

_2020-06-15_

This release contains a lot of fixes to APIs, and a minor new addition to the statistics endpoint. Potentially breaking changes are now labelled with 💥. If you have implemented a client that fetches candles, places orders or amends orders, please check below.

### ✨ Features

- [#1730](https://github.com/vegaprotocol/vega/pull/1730) `ChainID` added to statistics endpoint
- 💥 [#1734](https://github.com/vegaprotocol/vega/pull/1734) Start adding `TraceID` to core events

### 🛠 Improvements

- 💥 [#1721](https://github.com/vegaprotocol/vega/pull/1721) Improve API responses for `GetProposalById`
- 💥 [#1724](https://github.com/vegaprotocol/vega/pull/1724) New Order: Type no longer defaults to LIMIT orders
- 💥 [#1728](https://github.com/vegaprotocol/vega/pull/1728) `PrepareAmend` no longer accepts expiry time
- 💥 [#1760](https://github.com/vegaprotocol/vega/pull/1760) Add proto enum zero value "unspecified" to Side
- 💥 [#1764](https://github.com/vegaprotocol/vega/pull/1764) Candles: Interval no longer defaults to 1 minute
- 💥 [#1773](https://github.com/vegaprotocol/vega/pull/1773) Add proto enum zero value "unspecified" to `Order.Status`
- 💥 [#1776](https://github.com/vegaprotocol/vega/pull/1776) Add prefixes to enums, add proto zero value "unspecified" to `Trade.Type`
- 💥 [#1781](https://github.com/vegaprotocol/vega/pull/1781) Add prefix and UNSPECIFIED to `ChainStatus`, `AccountType`, `TransferType`
- [#1714](https://github.com/vegaprotocol/vega/pull/1714) Extend governance error handling
- [#1726](https://github.com/vegaprotocol/vega/pull/1726) Mark Price was not always correctly updated on a partial fill
- [#1734](https://github.com/vegaprotocol/vega/pull/1734) Feature/1577 hash context propagation
- [#1741](https://github.com/vegaprotocol/vega/pull/1741) Fix incorrect timestamps for proposals retrieved by GraphQL
- [#1743](https://github.com/vegaprotocol/vega/pull/1743) Orders amended to be GTT now return GTT in the response
- [#1745](https://github.com/vegaprotocol/vega/pull/1745) Votes blob is now base64 encoded
- [#1747](https://github.com/vegaprotocol/vega/pull/1747) Markets created from proposals now have the same ID as the proposal that created them
- [#1750](https://github.com/vegaprotocol/vega/pull/1750) Added datetime to governance votes
- [#1751](https://github.com/vegaprotocol/vega/pull/1751) Fix a bug in governance vote counting
- [#1752](https://github.com/vegaprotocol/vega/pull/1752) Fix incorrect validation on new orders
- [#1757](https://github.com/vegaprotocol/vega/pull/1757) Fix incorrect party ID validation on new orders
- [#1758](https://github.com/vegaprotocol/vega/pull/1758) Fix issue where markets created via governance were not tradable
- [#1763](https://github.com/vegaprotocol/vega/pull/1763) Expiration settlement date for market changed to 30/10/2020-22:59:59
- [#1777](https://github.com/vegaprotocol/vega/pull/1777) Create `README.md`
- [#1764](https://github.com/vegaprotocol/vega/pull/1764) Add proto enum zero value "unspecified" to Interval
- [#1767](https://github.com/vegaprotocol/vega/pull/1767) Feature/1692 order event
- [#1787](https://github.com/vegaprotocol/vega/pull/1787) Feature/1697 account event
- [#1788](https://github.com/vegaprotocol/vega/pull/1788) Check for unspecified Vote value
- [#1794](https://github.com/vegaprotocol/vega/pull/1794) Feature/1696 party event

## 0.19.0

_2020-05-26_

This release fixes a handful of bugs, primarily around order amends and new market governance proposals.

### ✨ Features

- [#1658](https://github.com/vegaprotocol/vega/pull/1658) Add timestamps to proposal API responses
- [#1656](https://github.com/vegaprotocol/vega/pull/1656) Add margin checks to amends
- [#1679](https://github.com/vegaprotocol/vega/pull/1679) Add topology package to map Validator nodes to Vega keypairs

### 🛠 Improvements

- [#1718](https://github.com/vegaprotocol/vega/pull/1718) Fix a case where a party can cancel another party's orders
- [#1662](https://github.com/vegaprotocol/vega/pull/1662) Start moving to event-based architecture internally
- [#1684](https://github.com/vegaprotocol/vega/pull/1684) Fix order expiry handling when `expiresAt` is amended
- [#1686](https://github.com/vegaprotocol/vega/pull/1686) Fix participation stake to have a maximum of 100%
- [#1607](https://github.com/vegaprotocol/vega/pull/1607) Update `gqlgen` dependency to 0.11.3
- [#1711](https://github.com/vegaprotocol/vega/pull/1711) Remove ID from market proposal input
- [#1712](https://github.com/vegaprotocol/vega/pull/1712) `prepareProposal` no longer returns an ID on market proposals
- [#1707](https://github.com/vegaprotocol/vega/pull/1707) Allow overriding default governance parameters via `ldflags`.
- [#1715](https://github.com/vegaprotocol/vega/pull/1715) Compile testing binary with short-lived governance periods

## 0.18.1

_2020-05-13_

### 🛠 Improvements

- [#1649](https://github.com/vegaprotocol/vega/pull/1649)
  Fix github artefact upload CI configuration

## 0.18.0

_2020-05-12_

From this release forward, compiled binaries for multiple platforms will be attached to the release on GitHub.

### ✨ Features

- [#1636](https://github.com/vegaprotocol/vega/pull/1636)
  Add a default GraphQL query complexity limit of 5. Currently configured to 17 on testnet to support Console.
- [#1656](https://github.com/vegaprotocol/vega/pull/1656)
  Add GraphQL queries for governance proposals
- [#1596](https://github.com/vegaprotocol/vega/pull/1596)
  Add builds for multiple architectures to GitHub releases

### 🛠 Improvements

- [#1630](https://github.com/vegaprotocol/vega/pull/1630)
  Fix amends triggering multiple updates to the same order
- [#1564](https://github.com/vegaprotocol/vega/pull/1564)
  Hex encode keys

## 0.17.0

_2020-04-21_

### ✨ Features

- [#1458](https://github.com/vegaprotocol/vega/issues/1458) Add root GraphQL Orders query.
- [#1457](https://github.com/vegaprotocol/vega/issues/1457) Add GraphQL query to list all known parties.
- [#1455](https://github.com/vegaprotocol/vega/issues/1455) Remove party list from stats endpoint.
- [#1448](https://github.com/vegaprotocol/vega/issues/1448) Add `updatedAt` field to orders.

### 🛠 Improvements

- [#1102](https://github.com/vegaprotocol/vega/issues/1102) Return full Market details in nested GraphQL queries.
- [#1466](https://github.com/vegaprotocol/vega/issues/1466) Flush orders before trades. This fixes a rare scenario where a trade can be available through the API, but not the order that triggered it.
- [#1491](https://github.com/vegaprotocol/vega/issues/1491) Fix `OrdersByMarket` and `OrdersByParty` 'Open' parameter.
- [#1472](https://github.com/vegaprotocol/vega/issues/1472) Fix Orders by the same party matching.

### Upcoming changes

This release contains the initial partial implementation of Governance. This will be finished and documented in 0.18.0.

## 0.16.2

_2020-04-16_

### 🛠 Improvements

- [#1545](https://github.com/vegaprotocol/vega/pull/1545) Improve error handling in `Prepare*Order` requests

## 0.16.1

_2020-04-15_

### 🛠 Improvements

- [!651](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/651) Prevent bad ED25519 key length causing node panic.

## 0.16.0

_2020-03-02_

### ✨ Features

- The new authentication service is in place. The existing authentication service is now deprecated and will be removed in the next release.

### 🛠 Improvements

- [!609](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/609) Show trades resulting from Orders created by the network (for example close outs) in the API.
- [!604](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/604) Add `lastMarketPrice` settlement.
- [!614](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/614) Fix casing of Order parameter `timeInForce`.
- [!615](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/615) Add new order statuses, `Rejected` and `PartiallyFilled`.
- [!622](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/622) GraphQL: Change Buyer and Seller properties on Trades from string to Party.
- [!599](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/599) Pin Market IDs to fixed values.
- [!603](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/603), [!611](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/611) Remove `NotifyTraderAccount` from API documentation.
- [!624](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/624) Add protobuf validators to API requests.
- [!595](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/595), [!621](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/621), [!623](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/623) Fix a flaky integration test.
- [!601](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/601) Improve matching engine coverage.
- [!612](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/612) Improve collateral engine test coverage.
