# Faucet

The faucet provides a way to deposit/mint new funds for vega builtin assets (for now, maybe more will be supported later on).
The faucet takes the form of an http server exposing a REST API which triggers new chain events on the Vega.

In order to control exactly who is allowed to broadcast events to the network, the [node configuration](../config/) contains a list of public keys allowed to broadcast chain events. The faucet's keypairs must be on this list before it can start allocating assets.

## Configuration

The configuration of the faucet is done through the vega config.toml file.

The whitelisted public key must be added in the following section.
```toml
[EvtForward]
  Level = "Info"
  RetryRate = "10s"
  BlockchainQueueWhitelist = ["c65af95865b4e970c48860f5c854c5ca8f340416372f9e72a98ff09e365aa0cf"]
```

The faucet also has its own configuration inside the vega config.toml file:
```toml
[Faucet]
  Level = "Info"
  CoolDown = "5h0m0s"
  WalletPath = "/Users/jeremy/.vega/faucet-wallet"
  Port = 1790
  IP = "0.0.0.0"
  [Faucet.Node]
    Port = 3002
    IP = "127.0.0.1"
    Retries = 5
```

This configuration can be generated automatically when running `vega init`. The following command will generate the faucet section in the configuration file, and add the generated public key to the `EvtForward` whitelist section.
```shell
vega init -f --gen-dev-nodewallet --gen-builtinasset-faucet
```

## Run the faucet

The faucet can be started using the core vega command line:
```shell
vega faucet run
```

You can run the help for more details explanation on the available flags:
```
vega faucet run -h
```

# API

## Get new funds

### Request

POST /api/v1/mint

```json
{
	"party": "party_pub_key",
	"amount": "amount_to_be_deposited",
	"asset": "asset_id"
}
```

### Response

```json
{
	"success": true
}
```
Note: the response does not indicate that the deposit succeeded, but that the request was send to the Vega network. It may take a few blocks before your the assets are deposited.
