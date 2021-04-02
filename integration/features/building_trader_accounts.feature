Feature: Test trader accounts

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the markets:
      | id           | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19    | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 0                | default-none | default-none     | default-eth-for-future |
      | GBPUSD/DEC19 | USD        | VUSD  | default-simple-risk-model-3 | default-margin-calculator | 0                | default-none | default-none     | default-usd-for-future |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: a trader is added to the system. A general account is created for each asset
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount |
      | trader1 | ETH   | 100    |
      | trader1 | VUSD  | 100    |
    And "trader1" has only one account per asset
    And "trader1" has only one margin account per market

  Scenario: a trader deposit collateral onto Vega. The general account for this asset increase
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount |
      | trader1 | ETH   | 100    |
      | trader1 | VUSD  | 100    |
    And "trader1" has only one account per asset
    Then The "trader1" makes a deposit of "200" into the "VUSD" account
    And "trader1" general account for asset "VUSD" balance is "300"

  Scenario: a trader withdraw collateral onto Vega. The general account for this asset decrease
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount |
      | trader1 | ETH   | 100    |
      | trader1 | VUSD  | 100    |
    And "trader1" has only one account per asset
    Then "trader1" withdraws "70" from the "VUSD" account
    And "trader1" general account for asset "VUSD" balance is "30"
