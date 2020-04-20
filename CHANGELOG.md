# Changelog

## 0.17.0

*2020-04-21*

### Features
- https://github.com/vegaprotocol/vega/issues/1458 Add root GraphQL Orders query.
- https://github.com/vegaprotocol/vega/issues/1457 AddGraphQL query to list all known parties.
- https://github.com/vegaprotocol/vega/issues/756 Remove party list from stats endpoint.
- https://github.com/vegaprotocol/vega/issues/1448 Add `updatedAt` field to orders.

### Improvements
- https://github.com/vegaprotocol/vega/issues/1102 Return full Market details in nested GraphQL queries.
- https://github.com/vegaprotocol/vega/issues/1466 Flush orders before trades. This fixes a rare scenario where a trade can be available through the API, but not the order that triggered it.
- https://github.com/vegaprotocol/vega/issues/1491 Fix OrdersByMarket and OrdersByParty 'Open' parameter.
- https://github.com/vegaprotocol/vega/issues/1472 Fix Orders by the same party matching.
- https://github.com/vegaprotocol/vega/issues/1462 Fix node crash when incorrect key length is used.

### Upcoming changes
This release contains the unfinalised initial implementation of Governance. This will be finished and documented in 0.18.0.

## 0.16.2

*2020-04-16*

### Improvements

- [#1545](https://github.com/vegaprotocol/vega/pull/1545) Improve error handling in `Prepare*Order` requests

## 0.16.1

*2020-04-15*

### Improvements

- [!651](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/651) Prevent bad ED25519 key length causing node panic.

## 0.16.0

*2020-03-02*

### Features

- The new authentication service is in place. The existing authentication service is now deprecated and will be removed in the next release.

### Improvements

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
