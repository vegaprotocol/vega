Feature: Test party accounts

  Background:
    Given the markets:
      | id           | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19    | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 0                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
      | GBPUSD/DEC19 | USD        | VUSD  | default-simple-risk-model-3 | default-margin-calculator | 0                | default-none | default-none     | default-usd-for-future | 1e6                    | 1e6                       |

  Scenario: a party is added to the system. A general account is created for each asset
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 100    |
      | party1 | VUSD  | 100    |
    Then "party1" should have one account per asset
    And "party1" should have one margin account per market

  Scenario: a party deposit collateral onto Vega. The general account for this asset increase
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 100    |
      | party1 | VUSD  | 100    |
    Then "party1" should have one account per asset
    When the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | VUSD  | 200    |
    Then "party1" should have general account balance of "300" for asset "VUSD"

  Scenario: a party withdraw collateral onto Vega. The general account for this asset decrease
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 100    |
      | party1 | VUSD  | 100    |
    Then "party1" should have one account per asset
    When the parties withdraw the following assets:
      | party  | asset | amount |
      | party1 | VUSD  | 70     |
    Then "party1" should have general account balance of "30" for asset "VUSD"
