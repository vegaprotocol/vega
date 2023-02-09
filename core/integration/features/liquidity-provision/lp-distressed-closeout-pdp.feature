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
      | 0.1  | 0.1   | 100         | 50            | 0.2                    |
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r   | sigma |
      | 0.000001      | 0.1 | 0  | 1.4 | -1    |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 5                 |

  Scenario: LP gets distressed during continuous trading (0042-LIQF-014)
    Given the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | data source config     | lp price range | position decimal places | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0.01           | 2                       | 1e6                    | 1e6                       |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | party0 | ETH   | 5721       |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |
      | party3 | ETH   | 100000000  |
      | party4 | ETH   | 1000000000 |
      | party5 | ETH   | 1000000000 |

    When the parties submit the following liquidity provision:
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
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | best static bid price | static mid price | best static offer price |
      | 1000       | TRADING_MODE_CONTINUOUS | 1000         | 5000           | 1000          | 990                   | 1000             | 1010                    |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 607    | 114     | 5000 |

    # Now let's trade with LP to increase their margin
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | party3 | ETH/DEC21 | buy  | 300    | 1010  | 2                | TYPE_LIMIT | TIF_GTC | party3-buy-1 |
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 1010       | TRADING_MODE_CONTINUOUS | 1313         | 5000           | 1300          |
    # LP margin requirement increased, had to dip in to bond account to top up the margin
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 925    | 0       | 4610 |
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | best static bid price | static mid price | best static offer price |
      | 1010       | TRADING_MODE_CONTINUOUS | 1313         | 5000           | 1300          | 990                   | 1045             | 1100                    |

    # Keep trading with LP volume until LP can't support the margin
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC21 | buy  | 474    | 1055  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 1420   | 0       | 3482 |
    And the insurance pool balance should be "759" for the market "ETH/DEC21"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC21 | buy  | 474    | 1055  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 2150   | 0       | 2064 |
    And the insurance pool balance should be "1468" for the market "ETH/DEC21"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC21 | buy  | 474    | 1055  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 2790   | 0       | 826  |
    And the insurance pool balance should be "2087" for the market "ETH/DEC21"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC21 | buy  | 474    | 1055  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 0      | 0       | 0    |
    And the insurance pool balance should be "5724" for the market "ETH/DEC21"

    Then the liquidity provisions should have the following states:
      | id  | party  | market    | commitment amount | status           |
      | lp1 | party0 | ETH/DEC21 | 5000              | STATUS_CANCELLED |

    # existing LP position not liquidated as there isn't enough volume on the book
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party0 | -2096  | -90            | 0            |
    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1100  | 100    |
      | buy  | 990   | 100    |
      | buy  | 900   | 100    |
    And the accumulated liquidity fees should be "29" for the market "ETH/DEC21"

    # Make sure that at no point fees get distributed since the LP has been closed out
    Then the network moves ahead "12" blocks
    And the accumulated liquidity fees should be "29" for the market "ETH/DEC21"

  Scenario: LP gets distressed after auction
    Given the simple risk model named "simple-risk-model-2":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 30          | 30            | 0.2                    |
    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | data source config     | lp price range | position decimal places | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-2 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0.5            | 2                       | 1e6                    | 1e6                       |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | party0 | ETH   | 5721       |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |
      | party3 | ETH   | 100000000  |
      | party4 | ETH   | 1000000000 |
      | party5 | ETH   | 1000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/DEC21 | 5000              | 0.001 | buy  | MID              | 500        | 20     | submission |
      | lp1 | party0 | ETH/DEC21 | 5000              | 0.001 | sell | MID              | 500        | 20     | amendment  |
      | lp2 | party5 | ETH/DEC21 | 995000            | 0.001 | buy  | BID              | 500        | 20     | submission |
      | lp2 | party5 | ETH/DEC21 | 995000            | 0.001 | sell | ASK              | 500        | 20     | amendment  |

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
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | supplied stake | horizon | min bound | max bound |
      | 1000       | TRADING_MODE_CONTINUOUS | 1000000        | 1       | 970       | 1030      |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 613    | 108     | 5000 |
    And the network moves ahead "1" blocks

    # Now let's trade with LP to increase their margin
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC21 | buy  | 591    | 1020  | 2                | TYPE_LIMIT | TIF_FOK |
    # LP margin requirement increased, had to dip in to bond account to top up the margin
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 1448   | 0       | 3588 |
    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1120  | 88840  |
      | sell | 1100  | 100    |
      | sell | 1065  | 470    |
      | buy  | 1025  | 488    |
      | buy  | 990   | 100    |
      | buy  | 970   | 102578 |
      | buy  | 900   | 100    |

    And the network moves ahead "1" blocks
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | supplied stake | horizon | min bound | max bound |
      | 1020       | TRADING_MODE_CONTINUOUS | 1000000        | 1       | 988       | 1048      |

    # move the upper bound up
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC21 | sell | 100    | 1048  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1100  | 100    |
      | sell | 1068  | 93165  |
      | sell | 1048  | 100    |
      | sell | 1039  | 482    |
      | buy  | 999   | 501    |
      | buy  | 990   | 100    |
      | buy  | 970   | 102578 |
      | buy  | 900   | 100    |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC21 | buy  | 582    | 1048  | 2                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | supplied stake | horizon | min bound | max bound |
      | 1048       | TRADING_MODE_CONTINUOUS | 1000000        | 1       | 1010      | 1071      |



    # getting closer to distressed LP, still in continuous trading
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC21 | buy  | 470    | 1065  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 2886   | 0       | 106  |
    And the liquidity provisions should have the following states:
      | id  | party  | market    | commitment amount | status        |
      | lp1 | party0 | ETH/DEC21 | 5000              | STATUS_ACTIVE |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search |
      | party0 | ETH/DEC21 | 2543        | 2797   |

    # advance time to change the reference price in price monitoring engine
    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | supplied stake | horizon | min bound | max bound |
      | 1065       | TRADING_MODE_CONTINUOUS | 1000000        | 1       | 1035      | 1095      |
    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1120  | 88840  |
      | sell | 1100  | 100    |
      | sell | 1065  | 470    |
      | buy  | 1025  | 488    |
      | buy  | 990   | 100    |
      | buy  | 970   | 102578 |
      | buy  | 900   | 100    |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party0 | -1443  | -346           | 0            |
    And the accumulated liquidity fees should be "22" for the market "ETH/DEC21"

    # trigger price monitoring auction by violating the upper bound
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC21 | buy  | 471    | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger       | supplied stake |
      | 1065       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 1000000        |
    # assure party0 volume unchanged (we want a closeout as a result of mark price move post auction and not due to position change)
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party0 | -1443  | -346           | 0            |

    # place additional order so that there's something left on the sell side and after generating trades and the market can return to continuous trading
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC21 | sell | 2000   | 1200  | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "6" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | supplied stake |
      | 1100       | TRADING_MODE_CONTINUOUS | 995000         |
    # the LP gets closed out
    And the liquidity provisions should have the following states:
      | id  | party  | market    | commitment amount | status           |
      | lp1 | party0 | ETH/DEC21 | 5000              | STATUS_CANCELLED |
      | lp2 | party5 | ETH/DEC21 | 995000            | STATUS_ACTIVE    |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 0      | 0       | 0    |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party0 | 0      | 0              | -3338        |
    And the accumulated liquidity fees should be "42" for the market "ETH/DEC21"

    # assure that closing out one LP doesn't prevent fees from being fully distributed
    When the network moves ahead "100" blocks
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"