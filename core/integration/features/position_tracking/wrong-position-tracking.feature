Feature: Test position tracking with auctions

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-basic    | default-eth-for-future | 0.01                   | 0                         |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | limits.markets.maxPeggedOrders          | 1500  |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario:
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount     |
      | party0  | ETH   | 1000000000 |
      | party1  | ETH   | 1000000000 |
      | party2  | ETH   | 1000000000 |
      | party3  | ETH   | 1000000000 |
      | partylp | ETH   | 1000000000 |
      | ruser   | ETH   | 75000      |

    # submit our LP
    Then the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | partylp | ETH/DEC19 | 16000000          | 0.3 | buy  | BID              | 2          | 10     | submission |
      | lp1 | partylp | ETH/DEC19 | 16000000          | 0.3 | sell | ASK              | 13         | 10     | amendment  |

    # get out of auction
    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party0 | ETH/DEC19 | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | t0-b-1    |
      | party1 | ETH/DEC19 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-1    |
      | party0 | ETH/DEC19 | buy  | 5      | 95000  | 0                | TYPE_LIMIT | TIF_GTC | t0-b-2    |
      | party1 | ETH/DEC19 | sell | 5      | 107000 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-2    |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 15     | 107500 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-3    |
      | party0 | ETH/DEC19 | buy  | 10     | 107100 | 0                | TYPE_LIMIT | TIF_GTC | t1-b-3    |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference | error               |
      | party3 | ETH/DEC19 | buy  | 10     | 107300 | 0                | TYPE_LIMIT | TIF_GTC | t3-b-1    |                     |
      | party1 | ETH/DEC19 | sell | 10     | 107100 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-4    |                     |
      | ruser  | ETH/DEC19 | buy  | 50     | 107500 | 0                | TYPE_LIMIT | TIF_GTC | lp-b-1    | margin check failed |
      | party3 | ETH/DEC19 | buy  | 70     | 106000 | 0                | TYPE_LIMIT | TIF_GFA | lp-b-2    |                     |

    Then the parties place the following pegged orders:
      | party | market id | side | volume | pegged reference | offset |
      | ruser | ETH/DEC19 | buy  | 35     | BID              | 1000   |
      | ruser | ETH/DEC19 | sell | 35     | ASK              | 3000   |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 80     | 105000 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-5    |
      | party3 | ETH/DEC19 | buy  | 81     | 106000 | 0                | TYPE_LIMIT | TIF_GFA | t3-b-2    |
      | party3 | ETH/DEC19 | buy  | 86     | 107000 | 0                | TYPE_LIMIT | TIF_GTC | t3-b-3    |

    Then the parties place the following pegged orders:
      | party  | market id | side | volume | pegged reference | offset |
      | party0 | ETH/DEC19 | buy  | 100    | BID              | 5000   |
      | party1 | ETH/DEC19 | sell | 95     | ASK              | 1000   |

    And time is updated to "2019-11-30T00:01:00Z"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
