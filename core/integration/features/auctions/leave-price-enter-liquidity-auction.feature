Feature: Leave a monitoring auction, enter a liquidity auction

  Background:

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-basic    | default-eth-for-future | 0.01                   | 0                         |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |
      | limits.markets.maxPeggedOrders | 1500  |

  Scenario:
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount     |
      | party0  | ETH   | 1000000000 |
      | party1  | ETH   | 1000000000 |
      | party2  | ETH   | 1000000000 |
      | party3  | ETH   | 1000000000 |
      | partylp | ETH   | 1000000000 |

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

    # trigger liquidity monitoring
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 1      | 99844 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-3    |
      | party0 | ETH/DEC19 | buy  | 1      | 99844 | 0                | TYPE_LIMIT | TIF_GTC | t0-b-3    |

    And time is updated to "2019-11-30T00:00:03Z"
    # enter price monitoring auction
    Then the market state should be "STATE_SUSPENDED" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"

    Then the parties cancel the following orders:
      | party  | reference |
      | party1 | t1-s-3    |

    Then the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 1      | 100291 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-4    |
      | party0 | ETH/DEC19 | buy  | 1      | 100291 | 0                | TYPE_LIMIT | TIF_GTC | t0-b-4    |

    When time is updated to "2019-11-30T00:00:20Z"
    # leave auction
    Then the market state should be "STATE_ACTIVE" for the market "ETH/DEC19"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the mark price should be "100291" for the market "ETH/DEC19"

    Then the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | buy  | 106    | 110000 | 0                | TYPE_LIMIT | TIF_GTC | t3-b-1    |

    Then the parties place the following pegged orders:
      | party  | market id | side | volume | pegged reference | offset |
      | party3 | ETH/DEC19 | buy  | 3      | BID              | 900    |

    Then the market state should be "STATE_SUSPENDED" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"

    Then the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party0 | ETH/DEC19 | buy  | 5      | 108500 | 0                | TYPE_LIMIT | TIF_GTC | t0-b-5    |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"
    And the market state should be "STATE_SUSPENDED" for the market "ETH/DEC19"

    And time is updated to "2019-11-30T00:00:22Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"
    And the market state should be "STATE_SUSPENDED" for the market "ETH/DEC19"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 125    | 95000 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-5    |

    And time is updated to "2019-11-30T00:10:00Z"
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"
    And the market state should be "STATE_SUSPENDED" for the market "ETH/DEC19"
