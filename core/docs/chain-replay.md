# Chain Replay

This is a guide to replay a chain using the backup of an existing chain (e.g. Testnet)

## How it works

A Tendermint Core and Vega Core node store their configuration and data to disk by default at `$HOME/.tendermint` and `$HOME/.vega`. When you start new instances of those nodes using a copy of these directories as their home, Tendermint re-submits (replays) historical blocks/transactions from the genesis height to Vega Core.

## Prerequisites

- [Google Cloud SDK ][gcloud]
- Vega Core Node
- Vega Wallet
- [Tendermint][tendermint]

## Chain backups

Note you need to first authenticate `gcloud`.

You can find backups for the Vega networks stored in Google Cloud Storage, e.g. For Testnet Node 01

```
$ gsutil ls gs://vega-chaindata-n01-testnet/chain_stores
```

## Steps

- Copy backups locally to `<path>`

- Overwrite Vega node wallet with your own development [node wallet][wallet]. 

```
$ cp -rp ~/.vega/node_wallets_dev <path>/.vega
$ cp ~/.vega/nodewalletstore <path>/.vega
```

- Update Vega node configuration

```
$ sed -i 's/\/home\/vega/<path>' <path>/.vega/config.toml
```

- Start Vega and Tendermint using backups

```
$ vega node --root-path=<path>/.vega --stores-enabled=false
$ tendermint node --home=<path>/.tendermint
```


## Tips

The Vega nodes adheres to the Tendermint ABCI contract, therefore breakpoints in the following methods are useful:

```
blockchain/abci/abci.go#BeginBlock
```

## Alternatives

Instead of a backup, which effectively replays the full chain from genesis, you can also use a snapshot of the chain at a given height to bootstrap the Tendermint node. Which only replays blocks/transactions from the given height. This however requires extra tooling.

## References

- https://github.com/cometbft/cometbft/blob/master/docs/introduction/quick-start.md
- https://docs.tendermint.com/master/spec/abci/apps.html
- https://github.com/tendermint/spec/blob/master/spec/abci/README.md
- https://docs.tendermint.com/master/spec/abci/apps.html#state-sync

[wallet]: https://github.com/vegaprotocol/vega#configuration
[gcloud]: https://cloud.google.com/sdk/docs/install
[tendermint]: https://github.com/cometbft/cometbft/blob/master/docs/introduction/install.md