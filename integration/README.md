#Integration Tests

This is the home of the system integrations tests. They can be run from the root of vega with:

```shell
make integrationtest
```

or  

```shell
go test ./...
``` 

## Running just the integration tests

The integration tests have been hooked up to run as regular unit tests, so you can run just the integration tests with a simple command:

```shell
go test ./integration/...
```

When running these tests, you'll probably want to get a more verbose output (showing which steps of the tests passed and failed), which can be done by adding 2 flags:

```
go test -v ./integration/... -godog.format=pretty
```

The `-v` flag tells `go test` to run with verbose output (sending logging to stdout). The `-godog.format=pretty` flag (which must be put at the end) instructs godog to print out the scenario's and, in case an assertion fails, show which particular step of a given scenario didn't work.

## Running specific scenario's

To run only certain tests (feature files), you can simply add the paths to a given feature file to the command:

```shell
go test -v ./integration/... -godog.format=pretty $(pwd)/integration/features/my-feature.feature
```

## Race detection and cache

For performance reasons, `go test` will check whether or not the source of a package has changed, and reuse compiled objects or even test results in case it determines nothing has changed. Because the integration tests are tucked away in their own package, and likely won't have changed, changes to _other_ packages might not be compiled, and tests could possibly pass without changes being applied. To ensure no cached results are used, the `-count` flag can be used:

```shell
go test -v -count=1 ./integration/... -godog.format=pretty
```

Should there be tests that are intermittently failing, this could indicate a data race somewhere in the code. To use the race detector to check for this, you can add the `-race` flag to the command. The full commands then would be:

```shell
# Run all integration tests, verbose mode, ensure recompiled binaries, enable race detection, and use godog pretty formatting
go test -v -count=1 -race ./integration/... -godog.format=pretty

# Same as above, but only run a specific feature file:
go test -v -count=1 -race ./integration/... -godog.format=pretty $(pwd)/integration/feature/my-feature.feature
```

Race detection is a complex thing to do, so it will make running tests significantly slower. The pipeline runs the tests with race detection, so this shouldn't be required to do locally.

## Reproducing/replicating system tests

The system tests run on a higher level. They submit a new market proposal, get said market accepted through governance, and then start trading. They use a `LogNormal` risk model, and specific fee parameters. David kindly provided the long/short risk factors for a simple risk model that result in the same margin requirements and same fees being applied to trades. To create an integration test that replicates the system test results (transfers, balances, fees, etc...), simply start your feature file with the following:

```
Feature: A feature that reproduces some system test

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | mark price | risk model | lamd/long              | tau/short              | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlement price | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading |
      | ETH/DEC20 | ETH      | ETH       | ETH   | 100       | simple     | 0.08628781058136630000 | 0.09370922348428490000 | -1 | -1 | -1    | 1.4            | 1.2            | 1.1           | 100             | 1           |    0.004 |             0.001 |          0.3 |                 0  |                |             |                 | 0.1             |
```
