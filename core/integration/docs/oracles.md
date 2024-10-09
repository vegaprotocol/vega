## Using oracles

Sending oracle data for any oracle is done using the property name, and the public key as set up previously in the test. How oracles are set up is covered [in the markets documentation](markets.md#Data-source-configuration).

The public key, in all setup steps is the value specified as _[...] from "0xCAFECAFE1"_. The public key, then, is `0xCAFECAFE1`.

### Sending simple signal

In its simplest form, an oracle can be triggered to send a data signal using the following step:

```cucumber
When the oracles broadcast data signed with "0xCAFECAFE1":
  | name             | value    | eth-block-time |
  | prices.ETH.value | 23456789 | 1725643814     |
```

Where the fields are defined as follows:

```
| name           | string                                    |
| value          | string (compatible with PropertyKey_Type) |
| eth-block-time | timestamp - optional                      |
```

Details on the [`PropertyKey_Type` type](types.md#PropertyKey_Type).

For settlement data, the eth-block-time doesn't matter too much. This value, however, is useful for perpetual markets where the time-weighted average values matter a lot.

It is possible to broadcast the same data using multiple keys (e.g. where 2 markets are configured using the same property key, but with different signers). In the keys can simply be comma-separated in the step:

```cucumber
When the oracles broadcast data signed with "0xCAFECAFE1,0xCAFECAFE2,0xCAFECAFE3":
  | name             | value    | eth-block-time |
  | prices.ETH.value | 23456789 | 1725643814     |
```

### Sending a signal relative to the current block time

For perpetual markets, specifically funding payments, we want to be able to control when (relative to the current block time in the test), certain data-points were received like so:

```cucumber
When the oracles broadcast data with block time signed with "0xCAFECAFE1":
  | name             | value      | time offset |
  | perp.funding.cue | 1511924180 | -100s       |
  | perp.ETH.value   | 975        | -2s         |
  | perp.ETH.value   | 977        | -1s         |
```

Other than that, the step works similarly to the previous one discussed fields are defined as follows:

```
| name           | string                                    |
| value          | string (compatible with PropertyKey_Type) |
| time offset    | duration                                  |
| eth-block-time | timestamp - optional                      |
```
