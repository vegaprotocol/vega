# Config

Config is a central place where all sub-system specific configurations come together. Sub-system specific config are defined on each corresponding packages.
The CLI defines all its parameters as structs. Out of this structs we generate a configuration file (in [toml](https://github.com/toml-lang/toml) format) and CLI flags (via [go-flags](github.com/jessevdk/go-flags)).

Ideally all parameters defined in the toml config should be exposed via flags, we might have a case where a parameter exists as a flag but not in the config, like `-c | --config` which defines the path of the config. Thereby, cli flags are a super set of the config parameters.

## How to use it
Structs define fields which maps to parameters:

```go
// Config is the configuration of the execution package
type Config struct {
  ...
  Level encoding.LogLevel             `long:"log-level"`
  InsurancePoolInitialBalance uint64  `long:"insurance-pool-initial-balance" description:"Some description"`
  Matching   matching.Config          `group:"Matching" namespace:"matching"`
  ...
}
```

A Config struct can hold native to types like `uint64`, custom types like `encoding.LogLevel` and other structures like `matching.Config`.
The `long:log-level` tag will be mapped to a `--log-level=` flag, also the `description:` tag will be displayed as documentation for that particular flag.
These are the two main tag that we use, see [Availabl field tags](https://godoc.org/github.com/jessevdk/go-flags#hdr-Available_field_tags) for reference.
When there are nested structs, we use the `group` and `namespace` tag. While the `group` tag will be displayed in the help to group the documentation, the `namespace` tag will affect the final flag name.
In this case the Matching options are going to be prefixed with the `--matching.` See [matching.Config]() for reference
```
$ vega node --help
Usage:
  vega [OPTIONS] node [node-OPTIONS]

Runs a vega node

    Execution:
          --execution.log-level=
          --execution.insurance-pool-initial-balance=         Some description (default: 

    Execution::Matching:
          --execution.matching.log-level=
          --execution.matching.log-price-levels-debug
          --execution.matching.log-removed-orders-debug
```

### Default values
Default values are displayed in the help if a) `description` annotation is set and b) if the value of the parameter is anything different from its [zero value](https://dave.cheney.net/2013/01/19/what-is-the-zero-value-and-why-is-it-useful)

