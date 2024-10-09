## Validating transfers

To ensure the correct amounts are transferred to and from the correct accounts, we use the following step:

```cucumber
Then the following transfers should happen:
  | type                                        | from    | to      | from account            | to account                       | market id | amount     | asset |
  | TRANSFER_TYPE_PERPETUALS_FUNDING_LOSS       | trader1 | market  | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT          | ETH/DEC19 | 700000000  | ETH   |
  | TRANSFER_TYPE_PERPETUALS_FUNDING_WIN        | market  | trader2 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN              | ETH/DEC19 | 700000000  | ETH   |
  | TRANSFER_TYPE_PERPETUALS_FUNDING_LOSS       | trader3 | market  | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT          | ETH/DEC19 | 1400000000 | ETH   |
  | TRANSFER_TYPE_PERPETUALS_FUNDING_WIN        | market  | trader4 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN              | ETH/DEC19 | 1400000000 | ETH   |
  | TRANSFER_TYPE_INFRASTRUCTURE_FEE_DISTRIBUTE | party3  |         | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 50         | ETH   |
```

With the fields being defined as follows:

```
| field        | required | type                                                 |
| from         | yes      | string (party or market ID, blank for system)        |
| to           | yes      | string(party or market ID, blank for system)         |
| from account | yes      | ACCOUNT_TYPE                                         |
| to account   | yes      | ACCOUNT_TYPE                                         |
| market id    | yes      | string (blank for general/system account)            |
| asset        | yes      | string (asset ID)                                    |
| type         | no       | TRANSFER_TYPE                                        |
| is amm       | no       | boolean (true if either from or to is an AMM subkey) |
```

Details for the [`ACCOUNT_TYPE` type](types.md#Account-type)
Details for the [`TRANSFER_TYPE` type](types.md#Transfer-type)

## Debugging transfers

To diagnose a problem, it can be useful to dump all the transfers that happened up to that particular point in a given test. To do this, simply add the following:

```cucumber
Then debug transfers
```

This will simply print out all transfer events using the following format:

```go
fmt.Sprintf("\t|%38s |%85s |%85s |%19s |\n", v.Type, v.FromAccount, v.ToAccount, v.Amount)
```
