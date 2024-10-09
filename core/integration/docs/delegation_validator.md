## Integration test framework for delegation and validators.

### Registering/setting up validators.

To create/register a validator, the following step should be used:

```cucumber
Given the validators:
  | id     | staking account balance | pub_key |
  | node1  | 10000                   | n1pk    |
  | node 2 | 20000                   | n2pk    |
```

Where `id` and `staking account balance` are required, the `pub_key` is optional. The types are as follows:

```
| id                      | string |
| staking account balance | uint   |
| pub_key                 | string |
```

The step _must_ match the pattern `^the validators:$`.

### Verifying the delegation balances for a given epoch.

To validate what the delegation balance is given a party and epoch sequence number, use the following step:

```cucumber
Then the parties should have the following delegation balances for epoch 3:
  | party  | node id | amount |
  | node1  | node1   | 1000   |
  | party1 | node1   | 123    |
  | node2  | node2   | 2000   |
  | party2 | node2   | 100    |
  | party2 | node1   | 100    |
```
All fields in the table are required and are of the following types:

```
| party   | string |
| node id | string |
| amount  | uint   |
```
The step _must_ match the pattern `^the parties should have the following delegation balances for epoch (\d+):$`.

### Verifying the validator scores per epoch.

To check whether or not the validator scores are what we'd expect, use the following step:

```cucumber
Then the validators should have the following val scores for epoch 1:
  | node id | validator score | normalised score |
  | node1   | 0.35            | 0.45             |
  | node2   | 0.65            | 0.55             |
```
All fields are required, and have the following types:

```
| node id          | string                      |
| validator score  | decimal [up to 16 decimals] |
| normalised score | decimal [up to 16 decimals] |
```
The step _must_ match the pattern `^the validators should have the following val scores for epoch (\d+):$`

### Verify the rewards received per epoch.

To validate whether the parties receive the expected rewards for a given epoch, use:

```cucumber
Then the parties receive the following reward for epoch 5:
  | party  | asset | amount |
  | party1 | TOKEN | 12     |
  | party2 | TOKEN | 20     |
  | node1  | TOKEN | 100    |
  | node2  | TOKEN | 200    |
```

All fields are required and of the following types:

```
| party  | string |
| asset  | string |
| amount | uint   |
```
The step _must_ match the pattern `^the parties receive the following reward for epoch (\d+):$`

### Ensure we are in the expected epoch.

To make sure the scenario is indeed in the epoch we expect to be in:

```cucumber
When the current epoch is "2"
```

The step _must_ match the pattern `^the current epoch is "([^"]+)"$`.
**NOTE**: the matched, quoted value should be a `uint`, otherwise the scenario will trigger a panic.

