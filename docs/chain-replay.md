# Chain Replay

This is a guide to replay a chain using the backup of an existing chain (e.g. Testnet)

## How it works

Tendermint persist all its data to directory


## Prerequisites

- [Google Cloud SDK ][gcloud]
- Vega Core Node
- Vega Wallet
- [Tendermint][tendermint]

## Chain backups

You can find backups for the Vega networks stored in Google Cloud Storage, e.g. For Testnet Node 01

```
$ gsutil ls gs://vega-chaindata-n01-testnet/chain_stores
```

## Steps

- Copy backups locally  to <path>

- Overwrite Vega node config with your own wallet

On macOS

```
$ cp  ~/.vega/nodewalletstore <path>/.vega
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

Instead of a backup, you can also use a snapshot of the chain at a given height and restore to bootstrap the Tendermint node. This however requires extra tooling.

## References

- https://github.com/tendermint/tendermint/blob/master/docs/introduction/quick-start.md
- https://docs.tendermint.com/master/spec/abci/apps.html
- https://github.com/tendermint/spec/blob/master/spec/abci/README.md


[gcloud]: https://cloud.google.com/sdk/docs/install
[tendermint]: https://github.com/tendermint/tendermint/blob/master/docs/introduction/install.md