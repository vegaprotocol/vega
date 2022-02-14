# Examples

This folder contains examples use-cases which can be built into clients to a local vega system.

## Nullchain

This is a go-package with a high-level of abstraction that gives the ability to manipulate a Vega node running with the `null-blockchain`. The idea is that with this package trading-scenarios can be written from a point of view that doesn't require a lot of technical knowledge of how vega works. For example, the below will connect to vega, create 3 parties, and then move the blockchain forward:

```

func main() {
  w := nullchain.NewWallet(config.WalletFolder, config.Passphrase)
  conn, _ := nullchain.NewConnection()
  parties, err := w.MakeParties(3)

  nullchain.MoveByDuration(config.BlockDuration)

}
```

Prerequistes:
- A Vega node in nullchain mode up and running
- Data-node up and running
- The Faucet up and running
- At least 3 users created in a local vega wallet
- The details in `nullchain/config/config.go` updated to reflect your local environment

The following bash should get you some way there:
```
git clone git@github.com:vegaprotocol/vega.git
git clone git@github.com:vegaprotocol/vegawallet.git
git clone git@github.com:vegaprotocol/data-node.git

# cd into vega data-node vegawallet directories and run
go install ./...

# initialise vega
vega init -f --home=vegahome
vega nodewallet generate --chain vega --home=vegahome
vega nodewallet generate --chain ethereum --home=vegahome

# initialise the faucet
vega faucet init -f --home=/vegahome --update-in-place

# initialise TM just so we can auto-generate a genesis file to fill in
vega tm init --home=vegahome
vega genesis generate --home=vegahome
vega genesis update --tm-home=/tenderminthome --home=/vegahome

# initialise the data-node
data-node init -f --home=vegahome

# initialise a vegawallet and make some parties
vegawallet init -f --home=vegahome
vegawallet key generate --wallet=A --home=vegahome
vegawallet key generate --wallet=B --home=vegahome
vegawallet key generate --wallet=C --home=vegahome
```

Next you need to fiddle with the vega config file to switch the blockchain on by changing the `BlockChain` section in `vegahome/config/node/config.toml` to look like this:
```
[Blockchain]
  Level = "Info"
  LogTimeDebug = true
  LogOrderSubmitDebug = true
  LogOrderAmendDebug = false
  LogOrderCancelDebug = false
  ChainProvider = "nullchain"
  [Blockchain.Tendermint]
    Level = "Info"
    LogTimeDebug = true
    ClientAddr = "tcp://0.0.0.0:26657"
    ClientEndpoint = "/websocket"
    ServerPort = 26658
    ServerAddr = "localhost"
    ABCIRecordDir = ""
    ABCIReplayFile = ""
  [Blockchain.Null]
    Level = "Debug"
    BlockDuration = "1s"
    TransactionsPerBlock = 3
    GenesisFile = "vegahome/config/genesis.json"
    IP = "0.0.0.0"
    Port = 3101
```

Now update the genesis file in `vegahome/config/genesis.json` to include the following assets in the appstate:

```
"assets": {
      "VOTE": {
        "name": "VOTE",
        "symbol": "VOTE",
        "total_supply": "0",
        "decimals": 5,
        "min_lp_stake": "1",
        "source": {
          "builtin_asset": {
            "max_faucet_amount_mint": "10000000000"
          }
        }
      },
      "XYZ": {
        "name": "XYZ",
        "symbol": "XYZ",
        "total_supply": "0",
        "decimals": 5,
        "min_lp_stake": "1",
        "source": {
          "builtin_asset": {
            "max_faucet_amount_mint": "10000000000"
          }
        }
      }
    },
```

Now spin up all the services by running the following each in their own terminal:

```
vega node run --home=vegahome
data-node run --home=vegahome
vega faucet run --home=vegahome

```

Once all is running, the example app can be run be doing the following in the `vega` directory
```
go run ./cmd/examples/nullchain/nullchain
```
