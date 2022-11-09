Feature: Replicate LP getting distressed during continuous trading, and after leaving an auction

  Background:
    Given the following network parameters are set:
      | name                                                | value |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.bondPenaltyParameter               | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 0.1   |
      | market.liquidity.providers.fee.distributionTimeStep | 10s   |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
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
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | data source config          |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | party0 | ETH   | 6400       |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |
      | party3 | ETH   | 100000000  |
      | party4 | ETH   | 1000000000 |
      | party5 | ETH   | 1000000000 |

  Scenario: LP gets distressed during continuous trading (0042-LIQF-014)

    Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/DEC21 | 5000              | 0.001 | buy  | BID              | 500        | 10     | submission |
      | lp1 | party0 | ETH/DEC21 | 5000              | 0.001 | sell | ASK              | 500        | 10     | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/DEC21 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party2 | ETH/DEC21 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 0.1
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1000         | 5000           | 10            |
    # check the requried balances
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 1320   | 80      | 5000 |

    # Now let's make some trades happen to increase the margin for LP
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party3 | ETH/DEC21 | buy  | 3      | 1010  | 2                | TYPE_LIMIT | TIF_GTC | party3-buy-1  |
      | party2 | ETH/DEC21 | sell | 5      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell-4 |
    # target stake 1313 with target trigger on 0.6 -> ~788 triggers liquidity auction
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1010       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1313         | 5000           | 13            |
    # LP margin requirement increased, had to dip in to bond account to top up the margin
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 1670   | 0       | 4478 |

    And the accumulated liquidity fees should be "5" for the market "ETH/DEC21"

    # progress time a bit, so the price bounds get updated
    When the network moves ahead "2" blocks
    And the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party3 | ETH/DEC21 | buy  | 10     | 1022  | 2                | TYPE_LIMIT | TIF_GTC | party3-buy-1  |
      | party3 | ETH/DEC21 | buy  | 3      | 1020  | 0                | TYPE_LIMIT | TIF_GTC | party3-buy-2  |
      | party2 | ETH/DEC21 | sell | 5      | 1030  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell-2 |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 1800   | 0       | 0    |
    And the insurance pool balance should be "4630" for the market "ETH/DEC21"

    Then the liquidity provisions should have the following states:
      | id  | party  | market    | commitment amount | status           |
      | lp1 | party0 | ETH/DEC21 | 5000              | STATUS_CANCELLED |

    # existing LP position not liquidated as there isn't enough volume on the book
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party0 | -7     | 0              | 0            |
    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1030  | 5      |
      | buy  | 1020  | 3      |
      | buy  | 990   | 1      |
      | buy  | 900   | 1      |
    And the accumulated liquidity fees should be "17" for the market "ETH/DEC21"

    # Make sure that at no point fees get distributed since the LP has been closed out
    Then the network moves ahead "12" blocks
    And the accumulated liquidity fees should be "17" for the market "ETH/DEC21"

  Scenario: LP gets distressed after auction

    Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/DEC21 | 5000              | 0.001 | buy  | BID              | 500        | 10     | submission |
      | lp1 | party0 | ETH/DEC21 | 5000              | 0.001 | sell | ASK              | 500        | 10     | amendment  |
      | lp2 | party5 | ETH/DEC21 | 5000              | 0.001 | buy  | BID              | 500        | 10     | submission |
      | lp2 | party5 | ETH/DEC21 | 5000              | 0.001 | sell | ASK              | 500        | 10     | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/DEC21 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-3  |
      | party2 | ETH/DEC21 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 0.1
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1000         | 10000          | 10            |
    # check the requried balances
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 1320   | 80      | 5000 |

    # Now let's make some trades happen to increase the margin for LP
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party3 | ETH/DEC21 | buy  | 3      | 1010  | 2                | TYPE_LIMIT | TIF_GTC | party3-buy-4  |
      | party2 | ETH/DEC21 | sell | 5      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell-4 |
    # target stake 1313 with target trigger on 0.6 -> ~788 triggers liquidity auction
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1010       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1313         | 10000          | 13            |
    # LP margin requirement increased, had to dip in to bond account to top up the margin
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 1670   | 0       | 4478 |

    # progress time a bit, so the price bounds get updated
    When the network moves ahead "2" blocks
    And the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party3 | ETH/DEC21 | buy  | 10     | 1022  | 2                | TYPE_LIMIT | TIF_GTC | party3-buy-5  |
      | party2 | ETH/DEC21 | sell | 75     | 1050  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell-5 |
      | party3 | ETH/DEC21 | buy  | 3      | 1020  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell-6 |
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1010       | TRADING_MODE_CONTINUOUS | 1       | 993       | 1012      | 2323         | 5000           | 23            |
    # getting closer to distressed LP, still in continuous trading
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 1380   | 0       | 0    |
    # Insurance pool is higher, because of the tick stuff we perform bond slashing more frequently than before 4630 => 5050
    And the insurance pool balance should be "5050" for the market "ETH/DEC21"

    # Move price out of bounds
    When the network moves ahead "2" blocks
    And the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC21 | buy  | 10     | 1060  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger       | target stake | supplied stake | open interest |
      | 1010       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 3481         | 5000           | 23            |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 1380   | 0       | 0    |

    # end price auction
    When the network moves ahead "301" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1055       | TRADING_MODE_CONTINUOUS | 1       | 1045      | 1065      | 3481         | 5000           | 33            |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 1065   | 0       | 0    |

    And the insurance pool balance should be "5050" for the market "ETH/DEC21"
