# Examples

This folder contains examples use-cases which can be built as clients to a local vega system.

## Nullchain

Prerequistes:
- A Vega node in nullchain mode up and running
- Data-node up and running
- The Faucet up and running
- At least 3 users created in a local vega wallet
- The details in `nullchain/config/config.go` updated to reflect your local environment

Build with `make build`, and then run `./cmd/examples/nullchain/nullchain` and cross your fingers. If all is well this will run a scenario where a party proposed a market, it gets voted in, trades ocurr on that market, the market is terminated and settled.