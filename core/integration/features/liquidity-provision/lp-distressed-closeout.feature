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

  Scenario: 001, LP gets distressed during continuous trading (0042-LIQF-014)
    Given the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | data source config     | lp price range | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0.01           | 1e6                    | 1e6                       |
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
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/DEC21 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party2 | ETH/DEC21 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | best static bid price | static mid price | best static offer price |
      | 1000       | TRADING_MODE_CONTINUOUS | 1000         | 5000           | 10            | 990                   | 1000             | 1010                    |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 720    | 1       | 5000 |

    # Now let's trade with LP to increase their margin
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | party3 | ETH/DEC21 | buy  | 3      | 1010  | 2                | TYPE_LIMIT | TIF_GTC | party3-buy-1 |
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 1010       | TRADING_MODE_CONTINUOUS | 1313         | 5000           | 13            |
    # LP margin requirement increased, had to dip in to bond account to top up the margin
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 956    | 0       | 4548 |
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | best static bid price | static mid price | best static offer price |
      | 1010       | TRADING_MODE_CONTINUOUS | 1313         | 5000           | 13            | 990                   | 1045             | 1100                    |

    # Keep trading with LP volume until LP can't support the margin
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC21 | buy  | 5      | 1055  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 1488   | 0       | 3348 |
    And the insurance pool balance should be "826" for the market "ETH/DEC21"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC21 | buy  | 5      | 1055  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 2253   | 0       | 1862 |
    And the insurance pool balance should be "1569" for the market "ETH/DEC21"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC21 | buy  | 5      | 1055  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 2928   | 0       | 556  |
    And the insurance pool balance should be "2222" for the market "ETH/DEC21"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC21 | buy  | 5      | 1055  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 233    | 0       | 0    |
    And the insurance pool balance should be "5495" for the market "ETH/DEC21"

    Then the liquidity provisions should have the following states:
      | id  | party  | market    | commitment amount | status           |
      | lp1 | party0 | ETH/DEC21 | 5000              | STATUS_CANCELLED |

    # existing LP position not liquidated as there isn't enough volume on the book
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party0 | -22    | -90            | 0            |
    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1100  | 1      |
      | buy  | 990   | 1      |
      | buy  | 900   | 1      |
    And the accumulated liquidity fees should be "29" for the market "ETH/DEC21"

    # Make sure that at no point fees get distributed since the LP has been closed out
    Then the network moves ahead "12" blocks
    And the accumulated liquidity fees should be "29" for the market "ETH/DEC21"

  Scenario: 002, LP gets distressed after auction
    Given the simple risk model named "simple-risk-model-2":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 30          | 30            | 0.2                    |
    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | data source config     | lp price range | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-2 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0.5            | 1e6                    | 1e6                       |
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
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/DEC21 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-3  |
      | party2 | ETH/DEC21 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | supplied stake | horizon | min bound | max bound |
      | 1000       | TRADING_MODE_CONTINUOUS | 1000000        | 1       | 970       | 1030      |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 720    | 1       | 5000 |
    And the network moves ahead "1" blocks

    # Now let's trade with LP to increase their margin
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC21 | buy  | 6      | 1020  | 2                | TYPE_LIMIT | TIF_FOK |
    # LP margin requirement increased, had to dip in to bond account to top up the margin
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 1494   | 0       | 3496 |
    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1120  | 889    |
      | sell | 1100  | 1      |
      | sell | 1065  | 5      |
      | buy  | 1025  | 5      |
      | buy  | 990   | 1      |
      | buy  | 970   | 1026   |
      | buy  | 900   | 1      |

    And the network moves ahead "1" blocks
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | supplied stake | horizon | min bound | max bound |
      | 1020       | TRADING_MODE_CONTINUOUS | 1000000        | 1       | 988       | 1048      |

    # move the upper bound up
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC21 | sell | 1      | 1048  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1068  | 932    |
      | sell | 1048  | 1      |
      | sell | 1039  | 5      |
      | buy  | 999   | 6      |
      | buy  | 990   | 1      |
      | buy  | 970   | 1026   |
      | buy  | 900   | 1      |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC21 | buy  | 6      | 1048  | 2                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | supplied stake | horizon | min bound | max bound |
      | 1048       | TRADING_MODE_CONTINUOUS | 1000000        | 1       | 1010      | 1071      |

    # getting closer to distressed LP, still in continuous trading
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC21 | buy  | 5      | 1065  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 1818   | 0       | 0    |
    And the liquidity provisions should have the following states:
      | id  | party  | market    | commitment amount | status        |
      | lp1 | party0 | ETH/DEC21 | 5000              | STATUS_ACTIVE |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search |
      | party0 | ETH/DEC21 | 2655        | 2920   |

    # advance time to change the reference price in price monitoring engine
    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | supplied stake | horizon | min bound | max bound |
      | 1065       | TRADING_MODE_CONTINUOUS | 1000000        | 1       | 1035      | 1095      |
    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1120  | 889    |
      | sell | 1100  | 1      |
      | sell | 1065  | 5      |
      | buy  | 1025  | 5      |
      | buy  | 990   | 1      |
      | buy  | 970   | 1026   |
      | buy  | 900   | 1      |
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party0 | -15    | -355           | 0            |
    And the accumulated liquidity fees should be "22" for the market "ETH/DEC21"

    # trigger price monitoring auction by violating the upper bound
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC21 | buy  | 6      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger       | supplied stake |
      | 1065       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 1000000        |
    # assure party0 volume unchanged (we want a closeout as a result of mark price move post auction and not due to position change)
    And the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party0 | -15    | -355           | 0            |

    # place additional order so that there's something left on the sell side and after generating trades and the market can return to continuous trading
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC21 | sell | 20     | 1200  | 0                | TYPE_LIMIT | TIF_GTC |

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
      | party0 | 0      | 0              | -2173        |
    And the accumulated liquidity fees should be "24" for the market "ETH/DEC21"

    # assure that closing out one LP doesn't prevent fees from being fully distributed
    When the network moves ahead "100" blocks
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

  Scenario: 003, 2 LPs on the market, LP1 gets distressed and closed-out during continuous trading (0042-LIQF-014)
    Given the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | data source config     | lp price range | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0.01           | 1e6                    | 1e6                       |
    And the parties deposit on asset's general account the following amount:
      | party   | asset | amount     |
      | party0  | ETH   | 5721       |
      | party10 | ETH   | 5721       |
      | party1  | ETH   | 100000000  |
      | party2  | ETH   | 100000000  |
      | party3  | ETH   | 100000000  |
      | party4  | ETH   | 1000000000 |
      | party5  | ETH   | 1000000000 |
      | party6  | ETH   | 1000000000 |

    When the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0  | ETH/DEC21 | 5000              | 0.001 | buy  | BID              | 500        | 10     | submission |
      | lp1 | party0  | ETH/DEC21 | 5000              | 0.001 | sell | ASK              | 500        | 10     | amendment  |
      | lp2 | party10 | ETH/DEC21 | 5000              | 0.001 | buy  | BID              | 500        | 10     | submission |
      | lp2 | party10 | ETH/DEC21 | 5000              | 0.001 | sell | ASK              | 500        | 10     | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/DEC21 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party2 | ETH/DEC21 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |
      | party6 | ETH/DEC21 | sell | 100    | 1200  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-4 |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest | best static bid price | static mid price | best static offer price |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 950       | 1100      | 1000         | 10000          | 10            | 990                   | 1000             | 1010                    |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 720    | 1       | 5000 |

    Then the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | buy  | 900   | 1      |
      | buy  | 990   | 13     |
      | sell | 1010  | 11     |
      | sell | 1100  | 1      |

    # Now let's trade with LP to increase their margin
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | party3 | ETH/DEC21 | buy  | 3      | 1010  | 2                | TYPE_LIMIT | TIF_GTC | party3-buy-1 |
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 1010       | TRADING_MODE_CONTINUOUS | 1313         | 10000          | 13            |
    # LP margin requirement increased, had to dip in to bond account to top up the margin
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 956    | 0       | 4548 |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party0 | ETH/DEC21 | 797         | 876    | 956     | 1115    |

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party0 | -2     | 0              | 0            |
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest | best static bid price | static mid price | best static offer price |
      | 1010       | TRADING_MODE_CONTINUOUS | 1       | 950       | 1100      | 1313         | 10000          | 13            | 990                   | 1045             | 1100                    |

    Then the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | buy  | 900   | 1      |
      | buy  | 990   | 1      |
      | buy  | 1035  | 10     |
      | sell | 1055  | 10     |
      | sell | 1100  | 1      |
      | sell | 1200  | 100    |

    # Keep trading with LP volume until LP can't support the margin
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC21 | buy  | 5      | 1055  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 1429   | 0       | 3466 |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party0 | ETH/DEC21 | 1266        | 1392   | 1519    | 1772    |

    Then the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | buy  | 900   | 1      |
      | buy  | 990   | 1      |
      | buy  | 1035  | 10     |
      | sell | 1055  | 10     |
      | sell | 1100  | 1      |
      | sell | 1200  | 100    |

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party0 | -7     | -90            | 0            |
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest | best static bid price | static mid price | best static offer price |
      | 1055       | TRADING_MODE_CONTINUOUS | 1       | 950       | 1100      | 1899         | 10000          | 18            | 990                   | 1045             | 1100                    |

    And the insurance pool balance should be "767" for the market "ETH/DEC21"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC21 | buy  | 5      | 1055  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 2368   | 0       | 1632 |
    And the insurance pool balance should be "1684" for the market "ETH/DEC21"

    Then the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | buy  | 900   | 1      |
      | buy  | 990   | 1      |
      | buy  | 1035  | 10     |
      | sell | 1055  | 10     |
      | sell | 1100  | 1      |
      | sell | 1200  | 100    |
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC21 | buy  | 5      | 1055  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | ETH   | ETH/DEC21 | 0      | 0       | 0    |
    And the insurance pool balance should be "4702" for the market "ETH/DEC21"
    Then the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | buy  | 900   | 1      |
      | buy  | 990   | 1      |
      | buy  | 1035  | 0      |
      | sell | 1055  | 0      |
      | sell | 1100  | 0      |
      | sell | 1200  | 89     |
    #lp1(party0) is closed-out, some of the sell orders had been used for close-out trade
    Then the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl |
      | party0  | 0      | 0              | -2636        |
      | party10 | -5     | 0              | 0            |

    Then the liquidity provisions should have the following states:
      | id  | party  | market    | commitment amount | status           |
      | lp1 | party0 | ETH/DEC21 | 5000              | STATUS_CANCELLED |

    And the accumulated liquidity fees should be "45" for the market "ETH/DEC21"
    # Make sure that at no point fees get distributed since the LP has been closed out
    Then the network moves ahead "12" blocks
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"
    # close-out trade price is not used as mark price, so mark price stays at 1055, supplied stake is from lp2 (lp1 is closed-out)
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest | best static bid price | static mid price | best static offer price |
      | 1055       | TRADING_MODE_CONTINUOUS | 1       | 981       | 1130      | 2954         | 5000           | 28            | 990                   | 1095             | 1200                    |

