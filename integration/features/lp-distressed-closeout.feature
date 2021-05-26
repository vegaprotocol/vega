Feature: Replicate LP getting distressed during continuous trading, and after leaving an auction

  Background:
    Given the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.targetstake.triggering.ratio | 0.1   |
    And the average block duration is "1"
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 10          | -10           | 0.1                    |
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r   | sigma |
      | 0.000001      | 0.1 | 0  | 1.4 | -1    |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee | liquidity fee |
      | 0.004     | 0.001              | 0.3           |
    And the price monitoring updated every "1" seconds named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | oracle config          | maturity date        |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 2021-12-31T23:59:59Z |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 100   |
    And the traders deposit on asset's general account the following amount:
      | trader  | asset | amount     |
      | trader0 | ETH   | 6400       |
      | trader1 | ETH   | 100000000  |
      | trader2 | ETH   | 100000000  |
      | trader3 | ETH   | 100000000  |
      | trader4 | ETH   | 1000000000 |

  Scenario: LP gets distressed during continuous trading

    Given the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
      | lp1 | trader0 | ETH/DEC21 | 5000              | 0.001 | buy        | BID             | 500              | -10          |
      | lp1 | trader0 | ETH/DEC21 | 5000              | 0.001 | sell       | ASK             | 500              | 10           |

    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | trader1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | trader1 | ETH/DEC21 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | trader1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | trader2 | ETH/DEC21 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | trader2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | trader2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

    # this is a bit pointless, we're still in auction, price bounds aren't checked
    # And the price monitoring bounds are []

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 0.1
    And the market data for the market "ETH/DEC21" should be:
     | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
     | 1000       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1000         | 5000           | 10            |
    # check the requried balances
    And the traders should have the following account balances:
      | trader  | asset | market id | margin | general | bond |
      | trader0 | ETH   | ETH/DEC21 | 1320   | 80      | 5000 |

    # Now let's make some trades happen to increase the margin for LP
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3 | ETH/DEC21 | buy  | 3      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 5      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    # target stake 1313 with target trigger on 0.6 -> ~788 triggers liquidity auction
    Then the market data for the market "ETH/DEC21" should be:
     | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
     | 1010       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1313         | 5000           | 13            |
    # LP margin requirement increased, had to dip in to bond account to top up the margin
    And the traders should have the following account balances:
      | trader  | asset | market id | margin | general | bond |
      | trader0 | ETH   | ETH/DEC21 | 1670   | 0       | 4739 |

    # progress time a bit, so the price bounds get updated
    When the network moves ahead "2" blocks
    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3 | ETH/DEC21 | buy  | 10     | 1022  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 15     | 1030  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3 | ETH/DEC21 | buy  | 3      | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 5      | 1030  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/DEC21" should be:
     | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
     | 1010       | TRADING_MODE_CONTINUOUS | 1       | 993       | 1012      | 2323         | 5000           | 23            |
    # getting closer to distressed LP, still in continuous trading
    And the traders should have the following account balances:
      | trader  | asset | market id | margin | general | bond |
      | trader0 | ETH   | ETH/DEC21 | 6060   | 0       | 375  |

    When the network moves ahead "2" blocks
    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3 | ETH/DEC21 | buy  | 3      | 1012  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 5      | 1012  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/DEC21" should be:
     | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
     | 1012       | TRADING_MODE_CONTINUOUS | 1       | 1000      | 1020      | 2834         | 5000           | 28            |
    And the traders should have the following account balances:
      | trader  | asset | market id | margin | general | bond |
      | trader0 | ETH   | ETH/DEC21 | 6072   | 0       | 361  |

    When the network moves ahead "2" blocks
    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3 | ETH/DEC21 | buy  | 10     | 1022  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 15     | 1030  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3 | ETH/DEC21 | buy  | 3      | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 5      | 1030  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 20     | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/DEC21" should be:
     | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
     | 1020       | TRADING_MODE_CONTINUOUS | 1       | 1007      | 1026      | 4182         | 5000           | 41            |
    And the traders should have the following account balances:
      | trader  | asset | market id | margin | general | bond |
      | trader0 | ETH   | ETH/DEC21 | 6384   | 0       | 0    |
    And the traders should have the following margin levels:
      | trader  | market id | maintenance | search | initial | release |
      | trader0 | ETH/DEC21 | 5712        | 6283   | 6854    | 7996    |

    When the network moves ahead "2" blocks
    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3 | ETH/DEC21 | buy  | 10     | 1024  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 15     | 1030  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3 | ETH/DEC21 | buy  | 3      | 1026  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 5      | 1030  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 20     | 1026  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/DEC21" should be:
     | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
     | 1026       | TRADING_MODE_CONTINUOUS | 1       | 1012      | 1031      | 5233         | 5000           | 51            |
    And the traders should have the following account balances:
      | trader  | asset | market id | margin | general | bond |
      | trader0 | ETH   | ETH/DEC21 | 6357   | 0       | 0    |
    And the traders should have the following margin levels:
      | trader  | market id | maintenance | search | initial | release |
      | trader0 | ETH/DEC21 | 5746        | 6320   | 6895    | 8044    |

    When the network moves ahead "2" blocks
    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3 | ETH/DEC21 | buy  | 20     | 1030  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 25     | 1031  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/DEC21" should be:
     | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
     | 1030       | TRADING_MODE_CONTINUOUS | 1       | 1012      | 1031      | 7313         | 5000           | 71            |
    And the traders should have the following account balances:
      | trader  | asset | market id | margin | general | bond |
      | trader0 | ETH   | ETH/DEC21 | 6341   | 0       | 0    |
    And the traders should have the following margin levels:
      | trader  | market id | maintenance | search | initial | release |
      | trader0 | ETH/DEC21 | 5768        | 6344   | 6921    | 8075    |

    When the network moves ahead "2" blocks
    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader4 | ETH/DEC21 | buy  | 60     | 1031  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/DEC21" should be:
     | mark price | trading mode            | auction trigger             | horizon | min bound | max bound | target stake | supplied stake | open interest |
     | 1031       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 1       | 1017      | 1036      | 13507        | 5000           | 131           |
    And the traders should have the following account balances:
      | trader  | asset | market id | margin | general | bond |
      | trader0 | ETH   | ETH/DEC21 | 6356   | 0       | 0    |
    And the traders should have the following margin levels:
      | trader  | market id | maintenance | search | initial | release |
      | trader0 | ETH/DEC21 | 5774        | 6351   | 6928    | 8083    |

  Scenario: LP gets closed out during continuous trading

    Given the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
      | lp1 | trader0 | ETH/DEC21 | 5000              | 0.001 | buy        | BID             | 500              | -10          |
      | lp1 | trader0 | ETH/DEC21 | 5000              | 0.001 | sell       | ASK             | 500              | 10           |

    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | trader1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | trader1 | ETH/DEC21 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | trader1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | trader2 | ETH/DEC21 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | trader2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | trader2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

    # this is a bit pointless, we're still in auction, price bounds aren't checked
    # And the price monitoring bounds are []

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 0.1
    And the market data for the market "ETH/DEC21" should be:
     | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
     | 1000       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1000         | 5000           | 10            |
    # check the requried balances
    And the traders should have the following account balances:
      | trader  | asset | market id | margin | general | bond |
      | trader0 | ETH   | ETH/DEC21 | 1320   | 80      | 5000 |

    # Now let's make some trades happen to increase the margin for LP
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3 | ETH/DEC21 | buy  | 3      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 5      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    # target stake 1313 with target trigger on 0.6 -> ~788 triggers liquidity auction
    Then the market data for the market "ETH/DEC21" should be:
     | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
     | 1010       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1313         | 5000           | 13            |
    # LP margin requirement increased, had to dip in to bond account to top up the margin
    And the traders should have the following account balances:
      | trader  | asset | market id | margin | general | bond |
      | trader0 | ETH   | ETH/DEC21 | 1670   | 0       | 4739 |

    # progress time a bit, so the price bounds get updated
    When the network moves ahead "2" blocks
    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3 | ETH/DEC21 | buy  | 10     | 1022  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 75     | 1050  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3 | ETH/DEC21 | buy  | 3      | 1020  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/DEC21" should be:
     | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
     | 1010       | TRADING_MODE_CONTINUOUS | 1       | 993       | 1012      | 2323         | 5000           | 23            |
    # getting closer to distressed LP, still in continuous trading
    And the traders should have the following account balances:
      | trader  | asset | market id | margin | general | bond |
      | trader0 | ETH   | ETH/DEC21 | 6060   | 0       | 375  |

    # Move price out of bounds
    When the network moves ahead "2" blocks
    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3 | ETH/DEC21 | buy  | 10     | 1060  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/DEC21" should be:
     | mark price | trading mode                    | auction trigger       | target stake | supplied stake | open interest |
     | 1010       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 2323         | 5000           | 23            |
    And the traders should have the following account balances:
      | trader  | asset | market id | margin | general | bond |
      | trader0 | ETH   | ETH/DEC21 | 6072   | 0       | 375  |

    # end price auction
    When the network moves ahead "301" blocks
    Then the market data for the market "ETH/DEC21" should be:
     | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
     | 1055       | TRADING_MODE_CONTINUOUS | 1       | 1045      | 1065      | 3482         | 5000           | 33            |
    And the traders should have the following account balances:
      | trader  | asset | market id | margin | general | bond |
      | trader0 | ETH   | ETH/DEC21 | 6132   | 0       | 0    |
    And the traders should have the following margin levels:
      | trader  | market id | maintenance | search | initial | release |
      | trader0 | ETH/DEC21 | 5803        | 6383   | 6963    | 8124    |

    When the network moves ahead "2" blocks
    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3 | ETH/DEC21 | buy  | 30     | 1060  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 55     | 1062  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 55     | 1061  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3 | ETH/DEC21 | buy  | 60     | 1063  | 0                | TYPE_LIMIT | TIF_GTC |
    # if distressed, we will enter auction
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | auction trigger           | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1060       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY | 1       | 1045      | 1065      | 13038        | 0              | 123           |
    And the traders should have the following account balances:
      | trader  | asset | market id | margin | general | bond |
      | trader0 | ETH   | ETH/DEC21 | 6215   | 0       | 0    |

Scenario: LP gets distressed when leaving auction

  # This ought to be "buy_shape" and "sell_shape" equivalents
  Given the traders submit the following liquidity provision:
    | id  | party   | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
    | lp1 | trader0 | ETH/DEC21 | 2000              | 0.001 | buy        | BID             | 1                | -2           |
    | lp1 | trader0 | ETH/DEC21 | 2000              | 0.001 | buy        | MID             | 2                | -1           |
    | lp1 | trader0 | ETH/DEC21 | 2000              | 0.001 | sell       | ASK             | 1                | 2            |
    | lp1 | trader0 | ETH/DEC21 | 2000              | 0.001 | sell       | MID             | 2                | 1            |

  And the traders place the following orders:
    | trader  | market id | side | volume | price | resulting trades | type       | tif     |
    | trader1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    | trader2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
    | trader2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

  # Again, pointless to check this in auction
  # And the price monitoring bounds are []

  When the opening auction period ends for market "ETH/DEC21"
  # Perhaps the reason for extending could be changed to reflect which check actually failed
  # In this case, though, it's the orderbook status, which applies to all auctions alike
  # So the trigger being AUCTION_TRIGGER_OPENING is as accurate as any
  Then the market data for the market "ETH/DEC21" should be:
    | trading mode                 | auction trigger            |
    | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING    |

  And the traders place the following orders:
    | trader  | market id | side | volume | price | resulting trades | type       | tif     |
    | trader1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |

  When the network moves ahead "1" blocks
  Then the auction ends with a traded volume of "10" at a price of "1000"

  And the market data for the market "ETH/DEC21" should be:
    | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
    | 1000       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1000         | 2000           | 10            |
