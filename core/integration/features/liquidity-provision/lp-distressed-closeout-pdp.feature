Feature: Replicate LP getting distressed during continuous trading, and after leaving an auction

  Background:
    Given the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.bondPenaltyParameter         | 1     |
      | market.liquidity.targetstake.triggering.ratio | 0.1   |
      | network.markPriceUpdateMaximumFrequency       | 0s    |
    And the average block duration is "1"
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 10          | 10            | 0.2                    |
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r   | sigma |
      | 0.000001      | 0.1 | 0  | 1.4 | -1    |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | data source config          | position decimal places |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 2                       |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | party0 | ETH   | 6400       |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |
      | party3 | ETH   | 100000000  |
      | party4 | ETH   | 1000000000 |
      | party5 | ETH   | 1000000000 |

  Scenario: LP gets distressed during continuous trading

    Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/DEC21 | 5000              | 0.001 | buy  | BID              | 500        | 10     | submission |
      | lp1 | party0 | ETH/DEC21 | 5000              | 0.001 | sell | ASK              | 500        | 10     | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/DEC21 | buy  | 100    | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/DEC21 | buy  | 100    | 990   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/DEC21 | buy  | 1000   | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party2 | ETH/DEC21 | sell | 100    | 1010  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/DEC21 | sell | 100    | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |
      | party2 | ETH/DEC21 | sell | 1000   | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "1000" at a price of "1000"
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 0.1
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1000         | 5000           | 1000          |
    # check the requried balances
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 1213   | 187     | 5000 |

    # Now let's make some trades happen to increase the margin for LP
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party3 | ETH/DEC21 | buy  | 300    | 1010  | 2                | TYPE_LIMIT | TIF_GTC | party3-buy-1  |
      | party2 | ETH/DEC21 | sell | 500    | 1010  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell-4 |
    # target stake 1313 with target trigger on 0.6 -> ~788 triggers liquidity auction
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1010       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1313         | 5000           | 1300          |
    # LP margin requirement increased, had to dip in to bond account to top up the margin
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 1562   | 0       | 4694 |

    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1100  | 100    |
      | sell | 1010  | 1491   |
      | buy  | 990   | 1111   |
      | buy  | 900   | 100    |

    # progress time a bit, so the price bounds get updated
    When the network moves ahead "2" blocks
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 1562   | 0       | 4699 |
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1010       | TRADING_MODE_CONTINUOUS | 1       | 993       | 1012      | 1313         | 5000           | 1300          |
    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1100  | 100    |
      | sell | 1010  | 1491   |
      | buy  | 990   | 1111   |
      | buy  | 900   | 100    |
    And the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party3 | ETH/DEC21 | buy  | 1000   | 1022  | 2                | TYPE_LIMIT | TIF_GTC | party3-buy-1  |
      | party3 | ETH/DEC21 | buy  | 300    | 1020  | 0                | TYPE_LIMIT | TIF_GTC | party3-buy-2  |
      | party2 | ETH/DEC21 | sell | 500    | 1030  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell-2 |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 0      | 0       | 0    |
    And the insurance pool balance should be "6435" for the market "ETH/DEC21"

    Then the liquidity provisions should have the following states:
      | id  | party  | market    | commitment amount | status           |
      | lp1 | party0 | ETH/DEC21 | 5000              | STATUS_CANCELLED |

    # existing LP position not liquidated as there isn't enough volume on the book
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party0 | -700   | 0              | 0            |

    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1100  | 100    |
      | sell | 1030  | 500    |
      | buy  | 1020  | 300    |
      | buy  | 990   | 100    |
      | buy  | 900   | 100    |

  Scenario: LP gets distressed after auction

    Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/DEC21 | 5000              | 0.001 | buy  | BID              | 500        | 10     | submission |
      | lp1 | party0 | ETH/DEC21 | 5000              | 0.001 | sell | ASK              | 500        | 10     | amendment  |
      | lp2 | party5 | ETH/DEC21 | 5000              | 0.001 | buy  | BID              | 500        | 10     | submission |
      | lp2 | party5 | ETH/DEC21 | 5000              | 0.001 | sell | ASK              | 500        | 10     | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/DEC21 | buy  | 100    | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/DEC21 | buy  | 100    | 990   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party1 | ETH/DEC21 | buy  | 1000   | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-3  |
      | party2 | ETH/DEC21 | sell | 100    | 1010  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/DEC21 | sell | 100    | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |
      | party2 | ETH/DEC21 | sell | 1000   | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "1000" at a price of "1000"
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 0.1
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1000         | 10000          | 1000          |
    # check the requried balances
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 1213   | 187     | 5000 |

    # Now let's make some trades happen to increase the margin for LP
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party3 | ETH/DEC21 | buy  | 300    | 1010  | 2                | TYPE_LIMIT | TIF_GTC | party3-buy-4  |
      | party2 | ETH/DEC21 | sell | 500    | 1010  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell-4 |
    # target stake 1313 with target trigger on 0.6 -> ~788 triggers liquidity auction
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1010       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1313         | 10000          | 1300          |
    # LP margin requirement increased, had to dip in to bond account to top up the margin
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 1562   | 0       | 4694 |

    # progress time a bit, so the price bounds get updated
    When the network moves ahead "2" blocks
    And the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party3 | ETH/DEC21 | buy  | 1000   | 1022  | 2                | TYPE_LIMIT | TIF_GTC | party3-buy-5  |
      | party2 | ETH/DEC21 | sell | 7500   | 1050  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell-5 |
      | party3 | ETH/DEC21 | buy  | 300    | 1020  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell-6 |
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1010       | TRADING_MODE_CONTINUOUS | 1       | 993       | 1012      | 2323         | 10000          | 2300          |
    # getting closer to distressed LP, still in continuous trading
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 1861   | 0       | 0    |
    And the insurance pool balance should be "4571" for the market "ETH/DEC21"

    # Move price out of bounds
    When the network moves ahead "2" blocks
    And the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC21 | buy  | 1000   | 1060  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger       | target stake | supplied stake | open interest |
      | 1010       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 3481         | 10000          | 2300          |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 1861   | 0       | 6    |

    # end price auction
    When the network moves ahead "301" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1055       | TRADING_MODE_CONTINUOUS | 1       | 1045      | 1065      | 3481         | 5000           | 3300          |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 0      | 0       | 0    |

    And the insurance pool balance should be "6123" for the market "ETH/DEC21"
