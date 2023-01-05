# Vega Visor

A binaries runner for [Core](../core/README.md) and [Data Node](../datanode/README.md) that facilitates automatic protocol upgrades.

***Features:***
- Visor is connected to core nodes and listens for protocol upgrades.
- Visor allows you to run core and data node binaries based on custom configuration.
- When a protocol upgrade is ready to be used, Visor automatically stops currently running binaries and starts new ones with the selected version.
- Visor can be configured to automatically fetch binaries with the correct version during the upgrade.
- Visor is highly configurable and allows you to configure the number of restarts, restarts delays, specific upgrade configuration and much more.

## Architecture

Visor stores all it's required config files and state in a `home` folder. The basic folder structure can be geneate by `visor init` cmd or manually. It is vital that all necessery files and folders are present in the `home` folder, therefore using the `init` command is recommended.

### Home folder structure:
```
HOME_FOLDER_PATH
├── config.toml
├── current
├── genesis
│   └── run-config.toml
└── vX.X.X
    └── run-config.toml
```

- `config.toml` - a [Visor configuration](#visorconfiguration) file.
- `run-config.toml` - a [Run configuration](#runconfiguration) file.
- `current` - a symlink to currently loaded configuration folder used to run binaries. On Visor startup when `current` folder is missing, Visor will link the `genesis` folder as `current` by default. During the upgrade if not specified otherwise Visor will try to link a 
folder named after version of the upgrade - for example `vX.X.X`. This symlink is automatically managed by Visor and it should not be tempered with manually.
- `genesis` - a folder that Visor automatically links to `current` [Run configuration](#runconfiguration) in case `current` folder does not exists.
- `vX.X.X` - any folder with a name of the upgrade version that stores [Run configuration](#runconfiguration) for the upgrade.

### Upgrade flow
1. During the first start up of Visor (when Visor has never been used before) a user provides `run-config.toml` and stores it in `genesis` folder.
2. When Visor starts up, it automatically links the `current` folder to the `genesis` folder and starts the corresponding processes based on the provided `run-config.toml`.
3. Visor connects to the core node and communicates with it via RPC API.
4. When validators agree on executing a protocol upgrade to a certain `version` and the network has reached the proposed `upgrade block height`, the core node will notify Visor about the upgrade.
- When `autoInstall` is enabled by a validator, then Visor automatically fetches the binaries with the correct `version` and prepares the upgrade folder.
- When `autoInstall` is NOT enabled, then a validator has to manually download the right binaries with correspoding `version` and prepare the upgrade folder with `run-config.toml` in it before the `upgrade block height`.
5. When Visor is notified about the upgrade to a specific `version`, it links the upgrade folder assosiated with the upgrade (either by being called as `version` or being specifically mapped manually in [Visor config](#visorconfiguration)). Then it executes the run config from the upgrade folder.

After that, the whole process is repeated from points 3-5 every time another upgrade takes place.

## Configuration

Visor has 2 different types of configuration. The ***Visor configuration*** and ***Run configuration*** where the first one is used to configure Visor itself and the latter is used to specify the protocol upgrade.

### Visor configuration

A configuration for Visor itself. This configuration is automatically reloaded by Visor so the changes in edited file will be automatically
reflected by Visor.

[Docs](visor-config.md)

### Run configuration