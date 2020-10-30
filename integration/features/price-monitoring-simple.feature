Feature: Price monitoring test using simple risk model

  Background:
    Given the markets starts on "2020-10-16T00:00:00Z" and expires on "2020-12-31T23:59:59Z"
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/long | tau/short | mu/max move up | r/min move down  | sigma  | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode | makerFee | infrastructureFee | liquidityFee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations |
      | ETH/DEC20 | BTC      | ETH       | ETH   |      1000 | simple     |      0.11 |       0.1 |             10 |              -11 |     -1 |            1.4 |            1.2 |           1.1 |              42 |           0 | continuous   |        0 |                 0 |            0 | 60                 |         60,120 |   0.95,0.99 |         240,360 |

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

  Scenario: Persistent order results in an auction (both triggers breached), no orders placed during auction, auction terminates with a trade from order that originally triggered the auction.
    Given the following traders:
      | name    | amount |
      | trader1 | 10000  |
      | trader2 | 10000  |

    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   100 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   100 |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "100"

        Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   111 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   111 |                0 | TYPE_LIMIT | TIF_GTC |

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "100"

    #T0 + 10min
    Then the time is updated to "2020-10-16T00:10:00Z"

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_MONITORING_AUCTION"

    #T0 + 10min01s
    Then the time is updated to "2020-10-16T00:10:01Z"

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "111"


  Scenario: Non-persistent order results in an auction (both triggers breached), no orders placed during auction, auction terminates.
    Given the following traders:
      | name    | amount |
      | trader1 | 10000  |
      | trader2 | 10000  |

    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   100 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   100 |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "100"

        Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   111 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   111 |                0 | TYPE_LIMIT | TIF_FOK |

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "100"

    #T0 + 10min
    Then the time is updated to "2020-10-16T00:10:00Z"

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_MONITORING_AUCTION"

    #T0 + 10min01s
    Then the time is updated to "2020-10-16T00:10:01Z"

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "100"

  Scenario: Non-persistent order results in an auction (both triggers breached), orders placed during auction result in a trade with indicative price within the price monitoring bounds, hence auction concludes.

    Given the following traders:
      | name    | amount |
      | trader1 | 10000  |
      | trader2 | 10000  |

    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   100 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   100 |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "100"

    Then traders place following orders:
    | trader  | id        | type | volume | price | resulting trades | type       | tif     |
    | trader1 | ETH/DEC20 | sell |      1 |   111 |                0 | TYPE_LIMIT | TIF_GTC |
    | trader2 | ETH/DEC20 | buy  |      1 |   111 |                0 | TYPE_LIMIT | TIF_FOK |

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "100"

    Then traders place following orders:
    | trader  | id        | type | volume  | price | resulting trades | type       | tif     |
    | trader2 | ETH/DEC20 | buy  |      1  |  112  |                0 | TYPE_LIMIT | TIF_GTC |

    #T0 + 10min
    Then the time is updated to "2020-10-16T00:10:00Z"

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "100"

    #T0 + 10min01s
    Then the time is updated to "2020-10-16T00:10:01Z"

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "112"

  Scenario: Persistent order results in an auction (one trigger breached), no orders placed during auction, auction gets extended due to 2nd trigger and eventually terminates with a trade from order that originally triggered the auction.
   
    Given the following traders:
      | name    | amount |
      | trader1 | 10000  |
      | trader2 | 10000  |

    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |    110 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |    110 |                1 | TYPE_LIMIT | TIF_FOK |

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "110"

    #T0 + 10s
    Then the time is updated to "2020-10-16T00:00:10Z"

    Then traders place following orders:
    | trader  | id        | type | volume | price | resulting trades | type       | tif     |
    | trader1 | ETH/DEC20 | sell |      1 |   115 |                0 | TYPE_LIMIT | TIF_GTC |
    | trader2 | ETH/DEC20 | buy  |      1 |   115 |                1 | TYPE_LIMIT | TIF_FOK |

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "115"

    #T0 + 01min10s
    Then the time is updated to "2020-10-16T00:01:10Z"

    Then traders place following orders:
    | trader  | id        | type | volume | price | resulting trades | type       | tif     |
    | trader1 | ETH/DEC20 | sell |      1 |   105 |                0 | TYPE_LIMIT | TIF_GTC |
    | trader2 | ETH/DEC20 | buy  |      1 |   105 |                1 | TYPE_LIMIT | TIF_FOK |

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 = T0 + 02min10s (auction start)
    Then the time is updated to "2020-10-16T00:02:10Z"

    Then traders place following orders:
    | trader  | id        | type | volume | price | resulting trades | type       | tif     |
    | trader1 | ETH/DEC20 | sell |      1 |   120 |                0 | TYPE_LIMIT | TIF_GTC |
    | trader2 | ETH/DEC20 | buy  |      1 |   120 |                0 | TYPE_LIMIT | TIF_GTC |

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 + 04min00s (last second of the initial auction)
    Then the time is updated to "2020-10-16T00:06:10Z"

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_MONITORING_AUCTION"

    #T1 + 04min01s (auction extended due to 2nd trigger)
    Then the time is updated to "2020-10-16T00:06:11Z"

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 + 10min00s (last second of the extended auction)
    Then the time is updated to "2020-10-16T00:12:10Z"

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 + 10min01s (extended auction finished)
    Then the time is updated to "2020-10-16T00:12:11Z"

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "120"

  Scenario: Non-persistent order results in an auction (one trigger breached), no orders placed during auction and auction terminates
   
    Given the following traders:
      | name    | amount |
      | trader1 | 10000  |
      | trader2 | 10000  |

    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type        | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |    110 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |    110 |                1 | TYPE_LIMIT | TIF_FOK |

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "110"

    #T0 + 10s
    Then the time is updated to "2020-10-16T00:00:10Z"

    Then traders place following orders:
    | trader  | id        | type | volume | price | resulting trades | type       | tif     |
    | trader1 | ETH/DEC20 | sell |      1 |   115 |                0 | TYPE_LIMIT | TIF_GTC |
    | trader2 | ETH/DEC20 | buy  |      1 |   115 |                1 | TYPE_LIMIT | TIF_FOK |

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "115"

    #T0 + 01min10s
    Then the time is updated to "2020-10-16T00:01:10Z"

    Then traders place following orders:
    | trader  | id        | type | volume | price | resulting trades | type       | tif     |
    | trader1 | ETH/DEC20 | sell |      1 |   105 |                0 | TYPE_LIMIT | TIF_GTC |
    | trader2 | ETH/DEC20 | buy  |      1 |   105 |                1 | TYPE_LIMIT | TIF_FOK |

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 = T0 + 02min10s (auction start)
    Then the time is updated to "2020-10-16T00:02:10Z"

    Then traders place following orders:
    | trader  | id        | type | volume | price | resulting trades | type       | tif     |
    | trader1 | ETH/DEC20 | sell |      1 |   120 |                0 | TYPE_LIMIT | TIF_GTC |
    | trader2 | ETH/DEC20 | buy  |      1 |   120 |                0 | TYPE_LIMIT | TIF_FOK |

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 + 04min00s (last second of the auction)
    Then the time is updated to "2020-10-16T00:06:10Z"

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_MONITORING_AUCTION"

    #T1 + 04min01s
    Then the time is updated to "2020-10-16T00:06:11Z"

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "105"

    Scenario: Non-persistent order results in an auction (one trigger breached), orders placed during auction result in a trade with indicative price outside the price monitoring bounds, hence auction get extended, no further orders placed, auction concludes.
     
    Given the following traders:
      | name    | amount |
      | trader1 | 10000  |
      | trader2 | 10000  |

    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |    110 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |    110 |                1 | TYPE_LIMIT | TIF_FOK |

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "110"

    #T0 + 10s
    Then the time is updated to "2020-10-16T00:00:10Z"

    Then traders place following orders:
    | trader  | id        | type | volume | price | resulting trades | type       | tif     |
    | trader1 | ETH/DEC20 | sell |      1 |   115 |                0 | TYPE_LIMIT | TIF_GTC |
    | trader2 | ETH/DEC20 | buy  |      1 |   115 |                1 | TYPE_LIMIT | TIF_FOK |

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "115"

    #T0 + 01min10s
    Then the time is updated to "2020-10-16T00:01:10Z"

    Then traders place following orders:
    | trader  | id        | type | volume | price | resulting trades | type       | tif     |
    | trader1 | ETH/DEC20 | sell |      1 |   105 |                0 | TYPE_LIMIT | TIF_GTC |
    | trader2 | ETH/DEC20 | buy  |      1 |   105 |                1 | TYPE_LIMIT | TIF_FOK |

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 = T0 + 02min10s (auction start)
    Then the time is updated to "2020-10-16T00:02:10Z"

    Then traders place following orders:
    | trader  | id        | type | volume | price | resulting trades | type       | tif     |
    | trader1 | ETH/DEC20 | sell |      1 |   120 |                0 | TYPE_LIMIT | TIF_GTC |
    | trader2 | ETH/DEC20 | buy  |      1 |   120 |                0 | TYPE_LIMIT | TIF_FOK |

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 + 04min00s (last second of the auction)
    Then the time is updated to "2020-10-16T00:06:10Z"

    Then traders place following orders:
    | trader  | id        | type | volume | price | resulting trades | type       | tif     |
    | trader1 | ETH/DEC20 | sell |      2 |   133 |                0 | TYPE_LIMIT | TIF_GFA |
    | trader2 | ETH/DEC20 | buy  |      2 |   133 |                0 | TYPE_LIMIT | TIF_GFA | 

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_MONITORING_AUCTION"

    #T1 + 04min01s (auction extended due to 2nd trigger)
    Then the time is updated to "2020-10-16T00:06:11Z"

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 + 10min00s (last second of the extended auction)
    Then the time is updated to "2020-10-16T00:12:10Z"

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 + 10min01s (extended auction finished)
    Then the time is updated to "2020-10-16T00:12:11Z"

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "133"
    
    Scenario: Non-persistent order results in an auction (one trigger breached), orders placed during auction result in trade with indicative price outside the price monitoring bounds, hence auction get extended, additional orders resulting in more trades placed, auction concludes. 

    Given the following traders:
      | name    | amount |
      | trader1 | 10000  |
      | trader2 | 10000  |

    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |    110 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |    110 |                1 | TYPE_LIMIT | TIF_FOK |

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "110"

    #T0 + 10s
    Then the time is updated to "2020-10-16T00:00:10Z"

    Then traders place following orders:
    | trader  | id        | type | volume | price | resulting trades | type       | tif     |
    | trader1 | ETH/DEC20 | sell |      1 |   115 |                0 | TYPE_LIMIT | TIF_GTC |
    | trader2 | ETH/DEC20 | buy  |      1 |   115 |                1 | TYPE_LIMIT | TIF_FOK |

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "115"

    #T0 + 01min10s
    Then the time is updated to "2020-10-16T00:01:10Z"

    Then traders place following orders:
    | trader  | id        | type | volume | price | resulting trades | type       | tif     |
    | trader1 | ETH/DEC20 | sell |      1 |   105 |                0 | TYPE_LIMIT | TIF_GTC |
    | trader2 | ETH/DEC20 | buy  |      1 |   105 |                1 | TYPE_LIMIT | TIF_FOK |

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 = T0 + 02min10s (auction start)
    Then the time is updated to "2020-10-16T00:02:10Z"

    Then traders place following orders:
    | trader  | id        | type | volume | price | resulting trades | type       | tif     |
    | trader1 | ETH/DEC20 | sell |      1 |   120 |                0 | TYPE_LIMIT | TIF_GTC |
    | trader2 | ETH/DEC20 | buy  |      1 |   120 |                0 | TYPE_LIMIT | TIF_FOK |

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 + 04min00s (last second of the auction)
    Then the time is updated to "2020-10-16T00:06:10Z"

    Then traders place following orders:
    | trader  | id        | type | volume | price | resulting trades | type       | tif     |
    | trader1 | ETH/DEC20 | sell |      2 |   133 |                0 | TYPE_LIMIT | TIF_GFA |
    | trader2 | ETH/DEC20 | buy  |      2 |   133 |                0 | TYPE_LIMIT | TIF_GFA | 

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_MONITORING_AUCTION"

    #T1 + 04min01s (auction extended due to 2nd trigger)
    Then the time is updated to "2020-10-16T00:06:11Z"

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 + 10min00s (last second of the extended auction)
    Then the time is updated to "2020-10-16T00:12:10Z"

        Then traders place following orders:
    | trader  | id        | type | volume | price | resulting trades | type       | tif     |
    | trader1 | ETH/DEC20 | sell |     10 |   303 |                0 | TYPE_LIMIT | TIF_GFA |
    | trader2 | ETH/DEC20 | buy  |     10 |   303 |                0 | TYPE_LIMIT | TIF_GFA | 

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 + 10min01s (extended auction finished)
    Then the time is updated to "2020-10-16T00:12:11Z"

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "303"