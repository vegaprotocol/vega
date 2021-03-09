Feature: Test trader accounts

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name         | quote name | asset | mark price | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlement price | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading |
      | ETH/DEC19    | ETH        | ETH   | 1000       | simple     | 0.11      | 0.1       | 0              | 0               | 0     | 1.4            | 1.2            | 1.1           | 42               | 0                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              |
      | GBPUSD/DEC19 | USD        | VUSD  | 1000       | simple     | 0.11      | 0.1       | 0              | 0               | 0     | 1.4            | 1.2            | 1.1           | 42               | 0                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              |

  Scenario: a trader is added to the system. A general account is created for each asset
    Given the following traders:
      | name    | amount |
      | trader1 |    100 |
    Then I Expect the traders to have new general account:
      | name    | asset |
      | trader1 | ETH   |
      | trader1 | VUSD  |
    And "trader1" general accounts balance is "100"
    And "trader1" have only one account per asset
    And "trader1" have only on margin account per market

  Scenario: a trader deposit collateral onto Vega. The general account for this asset increase
    Given the following traders:
      | name    | amount |
      | trader1 |    100 |
    Then I Expect the traders to have new general account:
      | name    | asset |
      | trader1 | ETH   |
      | trader1 | VUSD  |
    And "trader1" general accounts balance is "100"
    And "trader1" have only one account per asset
    Then The "trader1" makes a deposit of "200" into the "VUSD" account
    And "trader1" general account for asset "VUSD" balance is "300"

  Scenario: a trader withdraw collateral onto Vega. The general account for this asset decrease
    Given the following traders:
      | name    | amount |
      | trader1 |    100 |
    Then I Expect the traders to have new general account:
      | name    | asset |
      | trader1 | ETH   |
      | trader1 | VUSD  |
    And "trader1" general accounts balance is "100"
    And "trader1" have only one account per asset
    Then The "trader1" withdraw "70" from the "VUSD" account
    And "trader1" general account for asset "VUSD" balance is "30"
