Feature: Test funding margin for Perps market

  @Perpetual
  Scenario: (0019-MCAL-026 - cross margin, 0019-MCAL-053 - isolated margin) check funding margin for Perps market when clumps are 0 and 0.9, 0070-MKTD-017
    Given the following assets are registered:
      | id  | decimal places |
      | USD | 3              |
    And the perpetual oracles from "0xCAFECAFE1":
      | name        | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals |
      | perp-oracle | USD   | perp.ETH.value      | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | 0.5                   | 0.05          | 0                 | 0.9               | ETH        | 18                  |
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config | linear slippage factor | quadratic slippage factor | decimal places | position decimal places | market type | sla params      | 
      | ETH/DEC19 | ETH        | USD   | default-simple-risk-model-4 | default-margin-calculator | 1                | default-none | default-none     | perp-oracle        | 0.25                   | 0                         | 0              | 0                       | perp        | default-futures |
    And time is updated to "2024-01-01T00:00:00Z"
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | party1 | USD   | 1000000000 |
      | party2 | USD   | 1000000000 |
      | party3 | USD   | 1000000000 |
      | party4 | USD   | 1000000000 |
      | aux    | USD   | 1000000000 |
      | aux2   | USD   | 1000000000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 1      | 50      | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2   | ETH/DEC19 | sell | 1      | 5000    | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC19 | sell | 1      | 1590    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 1      | 1590    | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     |
      | party4 | ETH/DEC19 | buy  | 1      | 1590    | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC19 | sell | 1      | 1590    | 1                | TYPE_LIMIT | TIF_GTC |
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor |
      | party3 | ETH/DEC19 | isolated margin | 0.5           |
      | party4 | ETH/DEC19 | isolated margin | 0.75          |
    Then the mark price should be "1590" for the market "ETH/DEC19"
    And the parties should have the following margin levels:
      | party  | market id | maintenance | margin mode     |
      | party1 | ETH/DEC19 | 556500      | cross margin    |
      | party2 | ETH/DEC19 | 556500      | cross margin    |
      | party3 | ETH/DEC19 | 556500      | isolated margin |
      | party4 | ETH/DEC19 | 556500      | isolated margin |
      
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value                  | time offset |
      | perp.ETH.value   | 1600000000000000000000 | 0s          |
    And time is updated to "2024-01-01T17:31:03Z"
    # We need a trade a different price to trigger MTM
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 1      | 1591  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2   | ETH/DEC19 | sell | 1      | 1591  | 1                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "5" blocks
    # Now trade at desired price and forward time again to trigger another MTM
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 1      | 1590  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2   | ETH/DEC19 | sell | 1      | 1590  | 1                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "5" blocks
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | margin mode     |
      | party1 | ETH/DEC19 | 556500      | cross margin    |
      | party2 | ETH/DEC19 | 556580      | cross margin    |
      | party3 | ETH/DEC19 | 556500      | isolated margin |
      | party4 | ETH/DEC19 | 556580      | isolated margin |

    When the system unix time is "1704130273"
    And the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value                  | time offset |
      | perp.funding.cue | 1704130273             | 0s          |
    Then the following funding period events should be emitted:
      | start               | end                  | internal twap | external twap | funding payment | funding rate |
      | 1704067201000000000 | 1704130273000000000  | 1590000       | 1600000       | 160             | 0.0001       |
    And the following transfers should happen:
      | from   | to     | from account            | to account              | market id | amount | asset | type                                  |
      | party2 | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 160    | USD   | TRANSFER_TYPE_PERPETUALS_FUNDING_LOSS |
      | market | party1 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 160    | USD   | TRANSFER_TYPE_PERPETUALS_FUNDING_WIN  |
      | party4 | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 160    | USD   | TRANSFER_TYPE_PERPETUALS_FUNDING_LOSS |
      | market | party3 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 160    | USD   | TRANSFER_TYPE_PERPETUALS_FUNDING_WIN  |