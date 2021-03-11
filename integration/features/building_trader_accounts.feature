Feature: Test trader accounts

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name         | quote name | asset |  risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlement price | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19    | ETH        | ETH   |  simple     | 0.11      | 0.1       | 0              | 0               | 0     | 1.4            | 1.2            | 1.1           | 42               | 0                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
      | GBPUSD/DEC19 | USD        | VUSD  |  simple     | 0.11      | 0.1       | 0              | 0               | 0     | 1.4            | 1.2            | 1.1           | 42               | 0                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xBADC0FEE            | prices.USD.value     | TYPE_INTEGER              | prices.USD.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: a trader is added to the system. A general account is created for each asset
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount |
      | trader1 | ETH   | 100    |
      | trader1 | VUSD  | 100    |
    And "trader1" have only one account per asset
    And "trader1" have only one margin account per market

  Scenario: a trader deposit collateral onto Vega. The general account for this asset increase
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount |
      | trader1 | ETH   | 100    |
      | trader1 | VUSD  | 100    |
    And "trader1" have only one account per asset
    Then The "trader1" makes a deposit of "200" into the "VUSD" account
    And "trader1" general account for asset "VUSD" balance is "300"

  Scenario: a trader withdraw collateral onto Vega. The general account for this asset decrease
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount |
      | trader1 | ETH   | 100    |
      | trader1 | VUSD  | 100    |
    And "trader1" have only one account per asset
    Then The "trader1" withdraw "70" from the "VUSD" account
    And "trader1" general account for asset "VUSD" balance is "30"
