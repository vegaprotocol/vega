# Faucet

The faucet provides a way to deposit/mint new funds for vega builtin assets (for
now, maybe more will be supported later on). The faucet takes the form of an
http server exposing a REST API which triggers new chain events on the Vega.

In order to control exactly who is allowed to broadcast events to the network,
the [node configuration](../config/) contains a list of public keys allowed to
broadcast chain events. The faucet's keypairs must be on this list before it can
start allocating assets.

## Request rate limiting

To prevent the users to request unlimited amount of funds, the CoolDown field in
the configuration allow operator to specify a minimum amount of time between 2
request for funds.

## Spam prevention

In order to prevent spam from non-validator node, the faucet needs to be
connected to a validator node.

## Initialisation

The faucet exists as its own entity next to the vega node, with its own configuration file. It can be initialised by running the following:

```shell
vega faucet init -f --update-in-place
```

this will create a new config file that can be found at `VEGA_HOME/faucet/config.toml`. Providing the `--update-in-place` flag will cause the command to automatically update the Event Forward section in the main vega config with the public key of the faucet:

```toml
[EvtForward]
  Level = "Info"
  RetryRate = "10s"
  BlockchainQueueAllowlist = ["c65af95865b4e970c48860f5c854c5ca8f340416372f9e72a98ff09e365aa0cf"]
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

Note: the response does not indicate that the deposit succeeded, but that the
request was send to the Vega network. It may take a few blocks before your the
assets are deposited.
