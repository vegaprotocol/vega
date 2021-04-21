Feature: Leave a monitoring auction, enter a liquidity auction

  Background:

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-basic    | default-eth-for-future |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |
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

# trigger liquidity monitoring
    When the traders place the following orders:
      | trader           | market id | side | volume | price  | resulting trades | type       | tif     | reference       |
      | trader1          | ETH/DEC19 | sell | 1      | 99844  | 0                | TYPE_LIMIT | TIF_GTC | t1-s-3          |
      | trader0          | ETH/DEC19 | buy  | 1      | 99844  | 0                | TYPE_LIMIT | TIF_FOK | t0-b-3          |

    And time is updated to "2019-11-30T00:00:03Z"
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"

    Then the traders cancel the following orders:
      | trader  | reference |
      | trader1 | t1-s-3    |

    Then the traders place the following orders:
      | trader           | market id | side | volume | price  | resulting trades | type       | tif     | reference       |
      | trader1          | ETH/DEC19 | sell | 1      | 100291 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-4          |
      | trader0          | ETH/DEC19 | buy  | 1      | 100291 | 0                | TYPE_LIMIT | TIF_GTC | t0-b-4          |

    And time is updated to "2019-11-30T00:00:10Z"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the mark price should be "100291" for the market "ETH/DEC19"

    Then the traders place the following orders:
      | trader   | market id | side | volume | price  | resulting trades | type       | tif     | reference       |
      | trader3  | ETH/DEC19 | buy | 106    | 110000  | 0                | TYPE_LIMIT | TIF_GTC  | t3-b-1          |

    Then the traders place the following pegged orders:
      | trader  | market id | side | volume | reference | offset |
      | trader3 | ETH/DEC19 | buy  | 3      | BID       | -900   |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"

    Then the traders place the following orders:
      | trader           | market id | side | volume | price  | resulting trades | type       | tif     | reference       |
      | trader0          | ETH/DEC19 | buy  | 5      | 108500 | 0                | TYPE_LIMIT | TIF_GTC | t0-b-5          |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"

    And time is updated to "2019-11-30T00:00:12Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"

    Then the traders place the following orders:
      | trader           | market id | side | volume | price  | resulting trades | type       | tif     | reference       |
      | trader1          | ETH/DEC19 | sell | 125    | 95000  | 0                | TYPE_LIMIT | TIF_GTC | t1-s-5          |

    And time is updated to "2019-11-30T00:10:00Z"
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"
