Feature: Price monitoring test using forward risk model (bounds for the valid price moves around price of 100000 for the two horizons are: [95878,104251], [90497,110401])

  Background:
    Given the markets starts on "2020-10-16T00:00:00Z" and expires on "2020-12-31T23:59:59Z"
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset |   markprice  | risk model |     lamd/long |              tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode | makerFee | infrastructureFee | liquidityFee | p. m. update freq. |      p. m. horizons | p. m. probs  | p. m. durations |
      | ETH/DEC20 | BTC      | ETH       | ETH   |      100000  | forward    |      0.000001 | 0.00011407711613050422 |              0 | 0.016           |   2.0 |            1.4 |            1.2 |           1.1 |              42 |           0 | continuous   |        0 |                 0 |            0 | 60                 |         36000,72000 |   0.95,0.999 |         240,360 |

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

  Scenario: Persistent order results in an auction (both triggers breached), no orders placed during auction, auction terminates with a trade from order that originally triggered the auction.
    Given the following traders:
      | name    |      amount  |
      | trader1 | 10000000000  |
      | trader2 | 10000000000  |

    Then traders place following orders:
      | trader  | id        | type | volume |    price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   95878  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   95878  |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "95878"

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

    Then traders place following orders:
      | trader  | id        | type | volume |    price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   111000 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   111000 |                0 | TYPE_LIMIT | TIF_GTC |

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "100100"

    #T0 + 10min
    Then the time is updated to "2020-10-16T00:10:00Z"

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_MONITORING_AUCTION"

    #T0 + 10min01s
    Then the time is updated to "2020-10-16T00:10:01Z"

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "111000"