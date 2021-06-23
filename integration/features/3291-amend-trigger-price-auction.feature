Feature: Amend order to trigger price monitoring auction

  Background:
    Given time is updated to "2020-10-16T00:00:00Z"
    And the price monitoring updated every "4" seconds named "my-price-monitoring":
      | horizon | probability | auction extension |
      | 5       | 0.95        | 6                 |
      | 10      | 0.99        | 8                 |
    And the markets:
      | id        | quote name | asset | maturity date        | risk model             | margin calculator         | auction duration | fees         | price monitoring    | oracle config          |
      | ETH/DEC20 | ETH        | ETH   | 2020-12-31T23:59:59Z | system-test-risk-model | default-margin-calculator | 1                | default-none | my-price-monitoring | default-eth-for-future |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 6     |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Upper bound breached
    Given the traders deposit on asset's general account the following amount:
      | trader    | asset | amount       |
      | trader1   | ETH   | 100000000000 |
      | trader2   | ETH   | 100000000000 |
      | auxiliary | ETH   | 100000000000 |
      | aux2      | ETH   | 100000000000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader    | market id | side | volume | price    | resulting trades | type       | tif     |
      | auxiliary | ETH/DEC20 | buy  | 1      | 1        | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary | ETH/DEC20 | sell | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary | ETH/DEC20 | sell | 1      | 5670000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2      | ETH/DEC20 | buy  | 1      | 5670000  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 5670000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 10     | 5670010 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    Then the mark price should be "5670000" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders amend the following orders:
      | trader  | reference | price   | size delta | expiresAt | tif     | success |
      | trader2 | ref-2     | 5670005 | 0          | 0         | TIF_GTC | true    |
    Then the mark price should be "5670000" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
