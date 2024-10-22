## Integration test framework for AMM

### Creating, amending and submitting AMMs

To create a new AMM submission:

```
When the parties submit the following AMM:
  | party    | market id | amount            | slippage           | base | lower bound    | upper bound    | lower leverage                   | upper leverage                   | error                                  |
  | party id | market ID | commitment amount | tolerance as float | uint | min price uint | max price uint | margin ratio at lower bound as float | margin ratio at upper bound as float | OPTIONAL: error expected on submission |
```

All fields, except for `error` are required here.

Once an AMM has been created, we may want to amend it, so to amend an existing AMM:

```
Then the parties ammend the following AMM:
  | party               | market id            | amount         | slippage         | base     | lower bound | upper bound | lower leverage | upper leverage | error    |
  | party id (REQUIRED) | market ID (REQUIRED) | optional: uint | float (REQUIRED) | optional | optional    | optional    | optional           | optional           | optional |
```

The only 3 fields that are required are `party`, `market id`, and `slippage`. Any other fields omitted will not be updated.

Lastly, cancelling an existing AMM can be done through:

```
And the parties cancel the following AMM:
  | party    | market id | method           | error                    |
  | party id | market ID | CancelAMM_Method | OPTIONAL: error expected |
```

Details on the [`CancelAMM_Method` type](types.md#Cancel-AMM-Method)

### Checking AMM pools

To see what's going on with an existing AMM, we can check the AMM pool events with the following steps:

```
Then the AMM pool status should be:
  | party    | market id | amount           | status     | reason           | base | lower bound | upper bound | lower leverage | upper leverage |
  | party ID | market ID | commitment amout | AMM_Status | AMM_StatusReason | uint | uint        | uint        | float          | float          |
```

Required fields are `party`, `market id`, `amount`, and `status`. All others are optional.

Details on the [`AMM_Status` type](types.md#AMM-Status)
Details on the [`AMM_StatusReason` type](types.md#AMM-Status-Reason)

Checking the status for a given AMM only checks the most recent AMMPool event that was emitted. If we need to check all statuses a given AMM passed through during a scenario, use the following step:

```
And the following AMM pool events should be emitted:
  | party    | market id | amount           | status     | reason           | base | lower bound | upper bound | lower leverage | upper leverage |
  | party ID | market ID | commitment amout | AMM_Status | AMM_StatusReason | uint | uint        | uint        | float          | float          |
```

The table data is identical to that used in the previous step, with the same optional/required fields. The difference here is that we can check whether the correct events were emitted in a scenario like this:

Details on the [`AMM_Status` type](types.md#AMM-Status)
Details on the [`AMM_StatusReason` type](types.md#AMM-Status-Reason)


```
When 
When the parties submit the following AMM:
  | party  | market id | amount | slippage | base | lower bound | upper bound | lower leverage | upper leverage |
  | party1 | ETH/DEC24 | 10000  | 0.1      | 1000 | 900         | 1100        | 0.2                | 0.15               |
Then the parties amend the following AMM:
  | party  | market id | amount | slippage | base | lower bound | upper bound | upper leverage |
  | party1 | ETH/DEC24 | 20000  | 0.15     | 1010 | 910         | 1110        | 0.2                |
# simple status check, only 1 event can be checked, checking for the initial submission will fail
And the AMM pool status should be:
  | party  | market id | amount | status        | base | lower bound | upper bound | lower leverage | upper leverage |
  | party1 | ETH/DEC24 | 20000  | STATUS_ACTIVE | 1010 | 910         | 1110        | 0.2                | 0.2                |
When the parties cancel the following AMM:
  | party  | market id | method           |
  | party1 | ETH/DEC24 | METHOD_IMMEDIATE |
# check all events emitted so far
Then the following AMM pool events should be emitted:
  | party  | market id | amount | status           | base | lower bound | upper bound | lower leverage | upper leverage | reason                           |
  | party1 | ETH/DEC24 | 10000  | STATUS_ACTIVE    | 1000 | 900         | 1100        | 0.2                | 0.15               |                                  |
  | party1 | ETH/DEC24 | 20000  | STATUS_ACTIVE    | 1010 | 910         | 1110        | 0.2                | 0.2                |                                  |
  | party1 | ETH/DEC24 | 20000  | STATUS_CANCELLED | 1010 | 910         | 1110        | 0.2                | 0.2                | STATUS_REASON_CANCELLED_BY_PARTY |
```

### Checking AMM account balances and transfers

The AMM pool and sub-accounts are assigned derrived ID's, which can't be specified from the integration test scenario. To allow verifying the balances of the accounts, and check whether or not the expected transfers to and from said account happened, it's possible to assign aliases to the derived ID's.

```
Then set the following AMM sub account aliases:
  | party    | market id | alias               |
  | party ID | market ID | account owner alias |
```

This step _must_ be used _after_ the AMM submission has been made (ie after we've created the AMM pool), otherwise it will fail.

Once an alias has been created, we can check the balance of the AMM pool account using the following step:

```
Then parties have the following AMM account balances:
  | account alias     | asset | balance          |
  | alias set earlier | asset | expected balance |
```

The alias set in the first step is mapped to the internally derived ID, and the balance will be checked in the normal way (getting the most recent account balance event, compare the balance to the expected amount).

Checking transfers is done through the existing step, but a new optional field was added:

```
Then the following transfers should happen:
  | from       | from account      | to       | to account      | market id | amount | asset | type                    | is amm         |
  | from owner | from account type | to owner | to account type | market ID | amount | asset | OPTIONAL: transfer type | OPTIONAL: bool |
```

The new field `is amm` should be `true` for transfers involving AMM sub-accounts. An AMM sub-account is defined as being a general account, does not have a market, and the owner is a pre-defined alias (as per above). For example, a transfer from a general account to an AMM pool sub-account would look something like this:

```
When the parties submit the following AMM:
  | party  | market id | amount | slippage | base | lower bound | upper bound | lower leverage | upper leverage |
  | party1 | ETH/DEC24 | 10000  | 0.1      | 1000 | 900         | 1100        | 0.2                | 0.15               |
Then set the the following AMM sub account aliases:
  | party  | market id | alias          |
  | party1 | ETH/DEC24 | party1-amm-acc |
And the following transfers should happen:
  | from   | from account         | to             | to account           | market id | amount | asset | is amm |
  | party1 | ACCOUNT_TYPE_GENERAL | party1-amm-acc | ACCOUNT_TYPE_GENERAL |           | 10000  | ETH   | true   |
```

It's important to note that a vAMM will be receiving fees from particular markets, and when holding a position, will create a margin account on the given market, transfers for this can be checked using the same alias like so:

```
And the following transfers should happen:
  | from     | from account            | to       | to account              | market id | amount | asset | is amm | type                            |
  |          | ACCOUNT_TYPE_FEES_MAKER | vamm1-id | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 1      | USD   | true   | TRANSFER_TYPE_MAKER_FEE_RECEIVE |
  |          | ACCOUNT_TYPE_FEES_MAKER | vamm1-id | ACCOUNT_TYPE_GENERAL    | ETH/MAR22 | 2      | USD   | true   | TRANSFER_TYPE_MAKER_FEE_RECEIVE |
  | vamm1-id | ACCOUNT_TYPE_GENERAL    |          | ACCOUNT_TYPE_SETTLEMENT | ETH/MAR22 | 88     | USD   | true   | TRANSFER_TYPE_MTM_LOSS          |
  | vamm1-id | ACCOUNT_TYPE_GENERAL    | vamm1-id | ACCOUNT_TYPE_MARGIN     | ETH/MAR22 | 274    | USD   | true   | TRANSFER_TYPE_MARGIN_LOW        |
```

For more details on how to check transfer data [see here](transfers.md).

### Checking AMM trades

Because the parties who created the vAMM don't actually trade directly, the derived party ID will appear as the buyer or seller. The account owner alias created above should therefore be used to check the buyer/seller of trades involving the vAMM pools:

```
Then the following trades should be executed:
  | buyer  | price | size | seller      | is amm |
  | party5 | 106   | 1    | vamm1-alias | true   |
  | party5 | 110   | 1    | party2      |        |
  | party5 | 128   | 2    | vamm1-alias | true   |
  | party5 | 140   | 1    | party4      |        |
```

This step has added the `is amm` column (optional) which should be set to `true` to lookup the actual party ID using the vAMM alias created in the step outlined above.

### Checking vAMM position

Like any party, we may want to check the overall position of a vAMM pool. To do this, we can use the existing step to check the profit and loss. Like with the trade-check and transfer checks, a boolean column `is amm` was added to instruct the integration test framewokr to interpret the party ID given as being a vAMM alias.

```
Then the parties should have the following profit and loss:
  | party    | volume | unrealised pnl | realised pnl | is amm |
  | party5   | 5      | 88             | 0            |        |
  | party1   | 1      | 40             | 0            |        |
  | party2   | -2     | -70            | 0            |        |
  | party4   | -1     | 0              | 0            |        |
  | vamm1-id | -3     | -58            | 0            | true   |
```

### Checking AMM account balances

Just like checking trades and positions, checking the `vAMM` account balances can be done using the optional `is amm` column to the table provided to the relevant step.

```
Then the parties should have the following account balances:
  | party    | asset | market    | general | margin | is amm |
  | party1   | USD   | ETH/MAR22 | 100000  | 0      |        |
  | vamm1-id | USD   | ETH/MAR22 | 123     | 456    | true   |
```

### DEBUG STEPS

The debug steps specific to AMMs are simply ways of printing out the AMM pool event data in human-readable form:

```
# simply dump all AMM pool events
debug all AMM pool events
# debug all AMM pool events for a given party
debug AMM pool events for party "([^"]+)"
# debug all AMM pool events for a given market
debug all AMM pool events for market "([^"]+)"
# debug all AMM pool events for a given market and party
debug all AMM pool events for market "([^"]+)" and party "([^"]+)"
```
