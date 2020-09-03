genesis
=======

The genesis package handle loading and dispatching the genesis configuration to the different engine and service.
Upon the genesis block of the underlying blockchain, the application state is loaded, and sent through callback to the engine which registered interest.

This package also provide a command line to generate a default genesis state for a vega application.
This can be used using the following command line:
```bash
vega genesis
```
This command will dump the default state in the standard output, it should then be used in the configuration file of the underlying blockchain.
In the case of tendermint, is should be set to the `"app_state"` field of the genesis.json file.

The command can also update the genesis file directly using the following option:
```bash
vega genesis --in-place=/PATH/TO/.tendermint/config/genesis.json
```
