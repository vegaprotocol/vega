Feature: Test trader accounts

  Background:

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: a trader place a new order in the system, margin are calculated
    Given the traders deposit on asset's general account the following amount:
      | trader    | asset | amount       |
      | traderGuy | ETH   | 10000        |
      | trader1   | ETH   | 1000000      |
      | trader2   | ETH   | 1000000      |
      | aux       | ETH   | 100000000000 |


     # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 1      | 9     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader1-1 |
      | trader2 | ETH/DEC19 | sell | 1      | 10000 | 0                | TYPE_LIMIT | TIF_GTC | trader2-1 |
      | trader1 | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"
    Then the traders cancel the following orders:
      | trader  | reference |
      | trader1 | trader1-1 |
      | trader2 | trader2-1 |

    When the traders place the following orders:
      | trader    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | traderGuy | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the traders should have the following margin levels:
      | trader    | market id | maintenance | search | initial | release |
      | traderGuy | ETH/DEC19 | 100         | 110    | 120     | 140     |
    Then the traders should have the following account balances:
      | trader    | asset | market id | margin | general |
      | traderGuy | ETH   | ETH/DEC19 | 120    | 9880    |

  Scenario: an order is rejected if a trader have insufficient margin
    Given the traders deposit on asset's general account the following amount:
      | trader    | asset | amount  |
      | traderGuy | ETH   | 1       |
      | trader1   | ETH   | 1000000 |
      | trader2   | ETH   | 1000000 |

    # Trigger an auction to set the mark price
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader1-1 |
      | trader2 | ETH/DEC19 | sell | 1      | 10000 | 0                | TYPE_LIMIT | TIF_GTC | trader2-1 |
      | trader1 | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"
    Then the traders cancel the following orders:
      | trader  | reference |
      | trader1 | trader1-1 |
      | trader2 | trader2-1 |

    When the traders place the following orders:
      | trader    | market id | side | volume | price | type       | tif     | reference | error               |
      | traderGuy | ETH/DEC19 | sell | 1      | 1000  | TYPE_LIMIT | TIF_GTC | trader1-1 | margin check failed |
    Then the following orders should be rejected:
      | trader    | market id | reason                          |
      | traderGuy | ETH/DEC19 | ORDER_ERROR_MARGIN_CHECK_FAILED |
