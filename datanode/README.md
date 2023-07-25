# Data node

A service exposing read only APIs built on top of [Vega](https://github.com/vegaprotocol/vega) platform.

**Data node** provides the following core features:

- Consume all events from Vega core
- Aggregates received events and stores the aggregated data
- Serves stored data via [APIs](https://docs.vega.xyz/mainnet/api/overview)
- Allows advanced configuration [Configure a node](#configuration)

## Links

- For **new developers**, see [Getting Started](../GETTING_STARTED.md).
- For **updates**, see the [Change log](../CHANGELOG.md) for major updates.
- Please [open an issue](https://github.com/vegaprotocol/vega/issues/new) if anything is missing or unclear in this documentation.

<details>
  <summary><strong>Table of Contents</strong> (click to expand)</summary>

<!-- toc -->

- [Data node](#data-node)
  - [Links](#links)
  - [Installation and configuration](#installation-and-configuration)
  - [Troubleshooting & debugging](#troubleshooting--debugging)

<!-- tocstop -->

</details>

## Installation and configuration

To install see [Getting Started](https://docs.vega.xyz/mainnet/node-operators/setup-datanode).

## Troubleshooting & debugging

The application has structured logging capability, the first port of call for a crash is probably the Vega and Tendermint logs which are available on the console if running locally or by journal plus syslog if running on test networks. Default location for log files:

* `/var/log/vega.log`

Each internal Go package has a logging level that can be set at runtime by configuration. Setting the logging `Level` to `-1` for a package will enable all debugging messages for the package which can be useful when trying to analyse a crash or issue.

## Testing database migrations

To test database migrations you have added, you can use the goose CLI tool to run the migrations against a local database.

You can use docker to spin up a local postgres database to test against using the `postgres-docker.sh` under the project's `scripts` folder.

```bash
./scripts/postgres-docker.sh
```

This will create a local postgres database with the following credentials:

```bash
user: vega
password: vega
dbname: vega
host: localhost
port: 5432
```

To test the migrations, you can use the following command to apply all the migrations:

```bash
goose -dir migrations postgres "user=<your-user-name> password=<your-password> dbname=<your-db-name> host=<your-db-host> port=<your-db-port> sslmode=disable" up
```

To unwind all migrations, you can use the following command:

```bash
goose -dir migrations postgres "user=<your-user-name> password=<your-password> dbname=<your-db-name> host=<your-db-host> port=<your-db-port> sslmode=disable" down-to 0
```

To migrate to a specific migration, you can use the following command:

```bash
goose -dir migrations postgres "user=<your-user-name> password=<your-password> dbname=<your-db-name> host=<your-db-host> port=<your-db-port> sslmode=disable" up-to <migration-number>
```

Then you can rollback to a previous migration using the following command:

```bash
goose -dir migrations postgres "user=<your-user-name> password=<your-password> dbname=<your-db-name> host=<your-db-host> port=<your-db-port> sslmode=disable" down-to <migration-number)
```

The migration number is a 0 based integer of the migration scripts in the `migrations` folder. For example, if you want to rollback to the first migration, you would use the following command:

```bash
goose -dir migrations postgres "user=<your-user-name> password=<your-password> dbname=<your-db-name> host=<your-db-host> port=<your-db-port> sslmode=disable" down-to 1
```

To unwind just one migration to the previous one, you can use the following command:

```bash
goose -dir migrations postgres "user=<your-user-name> password=<your-password> dbname=<your-db-name> host=<your-db-host> port=<your-db-port> sslmode=disable" down
```
