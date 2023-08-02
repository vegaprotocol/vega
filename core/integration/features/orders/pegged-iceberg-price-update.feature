Feature: Price of a pegged iceberg order gets update as the reference price changes

  Background:
    Given the following network parameters are set:
      | name                                                | value |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidityV2.bondPenaltyParameter             | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 0.1   |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
      | limits.markets.maxPeggedOrders                      | 2     |
    And the average block duration is "1"
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 100         | 50            | 0.2                    |
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | providers fee calculation time step | performance hysteresis epochs | sla competition factor |
      | 0.01        | 0.5                          | 10                                  | 1                             | 1.0                    |
      
  Scenario: 
    Given the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | data source config     | position decimal places | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | default-none | default-basic | default-eth-for-future | 2                       | 0.5                    | 0                         | SLA        |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | party0 | ETH   | 5721       |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |
      | party3 | ETH   | 100000000  |
      | party4 | ETH   | 1000000000 |
      | party5 | ETH   | 1000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp1 | party0 | ETH/DEC21 | 5000              | 0.001 | submission |
      | lp1 | party0 | ETH/DEC21 | 5000              | 0.001 | amendment  |

    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | party0 | ETH/DEC21 | 2         | 1                    | buy  | BID              | 506        | 10     |
      | party0 | ETH/DEC21 | 2         | 1                    | sell | ASK              | 506        | 0      |

    Then the parties place the following pegged orders:
      | party  | market id | side | volume | pegged reference | offset |
      | party5 | ETH/DEC21 | buy  | 3      | BID              | 11     |
      | party5 | ETH/DEC21 | sell | 3      | ASK              | 1      |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/DEC21 | buy  | 100000 | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/DEC21 | buy  | 100    | 990   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/DEC21 | buy  | 1000   | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party2 | ETH/DEC21 | sell | 100    | 1010  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/DEC21 | sell | 100000 | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |
      | party2 | ETH/DEC21 | sell | 1000   | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "1000" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | best static bid price | static mid price | best static offer price |
      | 1000       | TRADING_MODE_CONTINUOUS | 1000         | 5000           | 1000          | 990                   | 1000             | 1010                    |

    # TODO: Side-issue: seems the limit for pegged orders is not respected with a mix of regular and iceberg pegs as the limit is set to 2 but we have deployed 4.
    And the pegged orders should have the following states:
      | party  | market id | side | volume | reference | offset | price | status        |
      | party0 | ETH/DEC21 | buy  | 506    | BID       | 10     | 980   | STATUS_ACTIVE |
      | party0 | ETH/DEC21 | sell | 506    | ASK       | 0      | 1010  | STATUS_ACTIVE |
      | party5 | ETH/DEC21 | buy  | 3      | BID       | 11     | 979   | STATUS_ACTIVE |
      | party5 | ETH/DEC21 | sell | 3      | ASK       | 1      | 1011  | STATUS_ACTIVE |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | party3 | ETH/DEC21 | buy  | 300    | 1010  | 2                | TYPE_LIMIT | TIF_GTC | party3-buy-1 |
    
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | best static bid price | static mid price | best static offer price |
      | 1010       | TRADING_MODE_CONTINUOUS | 1313         | 5000           | 1300          | 990                   | 1045             | 1100                    |
    And the pegged orders should have the following states:
      | party  | market id | side | volume | reference | offset | price | status        |
      | party0 | ETH/DEC21 | buy  | 506    | BID       | 10     | 980   | STATUS_ACTIVE |
      | party0 | ETH/DEC21 | sell | 506    | ASK       | 0      | 1100  | STATUS_ACTIVE |
    #   | party0 | ETH/DEC21 | sell | 506    | ASK       | 0      | 1010  | STATUS_ACTIVE |
      | party5 | ETH/DEC21 | buy  | 3      | BID       | 11     | 979   | STATUS_ACTIVE |
      | party5 | ETH/DEC21 | sell | 3      | ASK       | 1      | 1101  | STATUS_ACTIVE |
