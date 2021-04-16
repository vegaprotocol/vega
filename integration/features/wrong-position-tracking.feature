Feature: Test position tracking with auctions

  Background:

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-basic    | default-eth-for-future |
    And the following network parameters are set:
      | market.auction.minimumDuration |
      | 1                              |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario:
    Given the traders deposit on asset's general account the following amount:
      | trader           | asset | amount    |
      | trader0          | ETH   | 1000000000 |
      | trader1          | ETH   | 1000000000 |
      | trader2          | ETH   | 1000000000 |
      | trader3          | ETH   | 1000000000 |
      | traderlp         | ETH   | 1000000000 |
      | ruser            | ETH   | 75000     |

# submit our LP
    Then the traders submit the following liquidity provision:
      | id  | party    | market id | commitment amount | fee | order side | order reference | order proportion | order offset |
      | lp1 | traderlp | ETH/DEC19 | 16000000          | 0.3 | buy        | BID             | 2                | -10          |
      | lp1 | traderlp | ETH/DEC19 | 16000000          | 0.3 | sell       | ASK             | 13               | 10           |

# get out of auction
    When the traders place the following orders:
      | trader           | market id | side | volume | price  | resulting trades | type       | tif     | reference       |
      | trader0          | ETH/DEC19 | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | t0-b-1          |
      | trader1          | ETH/DEC19 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-1          |
      | trader0          | ETH/DEC19 | buy  | 5      | 95000  | 0                | TYPE_LIMIT | TIF_GTC | t0-b-2          |
      | trader1          | ETH/DEC19 | sell | 5      | 107000 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-2          |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the traders place the following orders:
      | trader           | market id | side | volume | price  | resulting trades | type       | tif     | reference       |
      | trader1          | ETH/DEC19 | sell | 15     | 107500 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-3          |
      | trader0          | ETH/DEC19 | buy  | 10     | 107100 | 0                | TYPE_LIMIT | TIF_GTC | t1-b-3          |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"

    When the traders place the following orders:
      | trader           | market id | side | volume | price  | resulting trades | type       | tif     | reference       |
      | trader3          | ETH/DEC19 | buy  | 10     | 107300 | 0                | TYPE_LIMIT | TIF_GTC | t3-b-1          |
      | trader1          | ETH/DEC19 | sell | 10     | 107100 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-4          |
      | ruser            | ETH/DEC19 | buy  | 50     | 107500 | 0                | TYPE_LIMIT | TIF_GTC | lp-b-1          |
      | trader3          | ETH/DEC19 | buy  | 70     | 106000 | 0                | TYPE_LIMIT | TIF_GFA | lp-b-1          |

    Then the traders place the following pegged orders:
      | trader   | market id | side | volume | reference | offset |
      | ruser    | ETH/DEC19 | buy  | 35     | BID       | -1000  |
      | ruser    | ETH/DEC19 | sell | 35     | ASK       | 3000   |

    When the traders place the following orders:
      | trader           | market id | side | volume | price  | resulting trades | type       | tif     | reference       |
      | trader1          | ETH/DEC19 | sell | 80     | 105000 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-5          |
      | trader3          | ETH/DEC19 | buy  | 81     | 106000 | 0                | TYPE_LIMIT | TIF_GFA | t3-b-2          |
      | trader3          | ETH/DEC19 | buy  | 86     | 107000 | 0                | TYPE_LIMIT | TIF_GTC | t3-b-3          |

    Then the traders place the following pegged orders:
      | trader   | market id | side | volume | reference | offset |
      | trader0  | ETH/DEC19 | buy  | 100    | BID       | -5000  |
      | trader1  | ETH/DEC19 | sell | 95     | ASK       | 1000  |

    And time is updated to "2019-11-30T00:01:00Z"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
