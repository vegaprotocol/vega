Feature: Test trader accounts

  Background:
    Given theExecutonEngineHaveTheseMarkets:
      | name         | baseName | quoteName | asset |
      | BTCETH/DEC19 | BTC      | ETH       | ETH   |
      | GBPUSD/DEC19 | GPB      | USD       | VUSD  |

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
