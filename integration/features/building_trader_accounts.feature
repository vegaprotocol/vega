Feature: Test trader accounts

  Background:
    Given the markets:
      | id           | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19    | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 0                | default-none | default-none     | default-eth-for-future |
      | GBPUSD/DEC19 | USD        | VUSD  | default-simple-risk-model-3 | default-margin-calculator | 0                | default-none | default-none     | default-usd-for-future |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: a trader is added to the system. A general account is created for each asset
    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount |
      | trader1 | ETH   | 100    |
      | trader1 | VUSD  | 100    |
    Then "trader1" should have one account per asset
    And "trader1" should have one margin account per market

  Scenario: a trader deposit collateral onto Vega. The general account for this asset increase
    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount |
      | trader1 | ETH   | 100    |
      | trader1 | VUSD  | 100    |
    Then "trader1" should have one account per asset
    When the traders deposit on asset's general account the following amount:
      | trader  | asset | amount |
      | trader1 | VUSD  | 200    |
    Then "trader1" should have general account balance of "300" for asset "VUSD"

  Scenario: a trader withdraw collateral onto Vega. The general account for this asset decrease
    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount |
      | trader1 | ETH   | 100    |
      | trader1 | VUSD  | 100    |
    Then "trader1" should have one account per asset
    When the traders withdraw the following assets:
      | trader  | asset | amount |
      | trader1 | VUSD  | 70     |
    Then "trader1" should have general account balance of "30" for asset "VUSD"
