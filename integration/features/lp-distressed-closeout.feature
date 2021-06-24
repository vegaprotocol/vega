Feature: Replicate LP getting distressed during continuous trading, and after leaving an auction

  Background:
    Given the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.bondPenaltyParameter         | 1     |
      | market.liquidity.targetstake.triggering.ratio | 0.1   |
    And the average block duration is "1"
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 10          | -10           | 0.1                    |
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r   | sigma |
      | 0.000001      | 0.1 | 0  | 1.4 | -1    |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
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
      | trader5 | ETH   | 1000000000 |

  Scenario: LP gets distressed during continuous trading

    Given the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | trader0 | ETH/DEC21 | 5000              | 0.001 | buy  | BID              | 500        | -10    |
      | lp1 | trader0 | ETH/DEC21 | 5000              | 0.001 | sell | ASK              | 500        | 10     |

    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | trader1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | trader1 | ETH/DEC21 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | trader1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | trader2 | ETH/DEC21 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | trader2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |
      | trader2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |

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
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | trader3 | ETH/DEC21 | buy  | 3      | 1010  | 2                | TYPE_LIMIT | TIF_GTC | trader3-buy-1  |
      | trader2 | ETH/DEC21 | sell | 5      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | trader2-sell-4 |
    # target stake 1313 with target trigger on 0.6 -> ~788 triggers liquidity auction
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1010       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1313         | 5000           | 13            |
    # LP margin requirement increased, had to dip in to bond account to top up the margin
    And the traders should have the following account balances:
      | trader  | asset | market id | margin | general | bond |
      | trader0 | ETH   | ETH/DEC21 | 1670   | 0       | 4478 |

    # progress time a bit, so the price bounds get updated
    When the network moves ahead "2" blocks
    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | trader2 | ETH/DEC21 | sell | 15     | 1030  | 0                | TYPE_LIMIT | TIF_GTC | trader2-sell-1 |
      | trader3 | ETH/DEC21 | buy  | 10     | 1022  | 2                | TYPE_LIMIT | TIF_GTC | trader3-buy-1  |
      | trader3 | ETH/DEC21 | buy  | 3      | 1020  | 0                | TYPE_LIMIT | TIF_GTC | trader3-buy-2  |
      | trader2 | ETH/DEC21 | sell | 5      | 1030  | 0                | TYPE_LIMIT | TIF_GTC | trader2-sell-2 |
    Then the traders should have the following account balances:
      | trader  | asset | market id | margin | general | bond |
      | trader0 | ETH   | ETH/DEC21 | 0      | 0       | 0    |
    And the insurance pool balance should be "6390" for the market "ETH/DEC21"

    # Then the traders should have the following account balances:
    #   | trader  | asset | market id | margin | general | bond |
    #   | trader0 | ETH   | ETH/DEC21 | 1789   | 0       | 0    |
    # And the insurance pool balance should be "4646" for the market "ETH/DEC21"
    # Then the market data for the market "ETH/DEC21" should be:
    #   | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
    #   | 1010       | TRADING_MODE_CONTINUOUS | 1       | 993       | 1012      | 2323         | 5000           | 23            |
    # # getting closer to distressed LP, still in continuous trading
    # And the traders should have the following account balances:
    #   | trader  | asset | market id | margin | general | bond |
    #   | trader0 | ETH   | ETH/DEC21 | 1789   | 0       | 0    |

    # When the network moves ahead "2" blocks
    # And the traders place the following orders:
    #   | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference      |
    #   | trader3 | ETH/DEC21 | buy  | 3      | 1012  | 0                | TYPE_LIMIT | TIF_GTC | trader3-buy-3  |
    #   | trader2 | ETH/DEC21 | sell | 5      | 1012  | 2                | TYPE_LIMIT | TIF_GTC | trader2-sell-3 |
    # Then the market data for the market "ETH/DEC21" should be:
    #   | mark price | trading mode                    | horizon | min bound | max bound | target stake | supplied stake | open interest |
    #   | 1012       | TRADING_MODE_MONITORING_AUCTION | 1       | 1000      | 1020      | 2834         | 0              | 28            |
    # And the traders should have the following account balances:
    #   | trader  | asset | market id | margin | general | bond |
    #   | trader0 | ETH   | ETH/DEC21 | 1787   | 0       | 0    |
    # # make sure bond slashing moved money to insurance pool
    # And the insurance pool balance should be "4646" for the market "ETH/DEC21"


  Scenario: LP gets distressed after auction

    Given the traders submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | trader0 | ETH/DEC21 | 5000              | 0.001 | buy  | BID              | 500        | -10    |
      | lp1 | trader0 | ETH/DEC21 | 5000              | 0.001 | sell | ASK              | 500        | 10     |
      | lp2 | trader5 | ETH/DEC21 | 5000              | 0.001 | buy  | BID              | 500        | -10    |
      | lp2 | trader5 | ETH/DEC21 | 5000              | 0.001 | sell | ASK              | 500        | 10     |

    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | trader1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | trader1 | ETH/DEC21 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | trader1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-3  |
      | trader2 | ETH/DEC21 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | trader2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |
      | trader2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |

    # this is a bit pointless, we're still in auction, price bounds aren't checked
    # And the price monitoring bounds are []

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 0.1
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1000         | 10000          | 10            |
    # check the requried balances
    And the traders should have the following account balances:
      | trader  | asset | market id | margin | general | bond |
      | trader0 | ETH   | ETH/DEC21 | 1320   | 80      | 5000 |

    # Now let's make some trades happen to increase the margin for LP
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | trader3 | ETH/DEC21 | buy  | 3      | 1010  | 2                | TYPE_LIMIT | TIF_GTC | trader3-buy-4  |
      | trader2 | ETH/DEC21 | sell | 5      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | trader2-sell-4 |
    # target stake 1313 with target trigger on 0.6 -> ~788 triggers liquidity auction
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1010       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1313         | 10000          | 13            |
    # LP margin requirement increased, had to dip in to bond account to top up the margin
    And the traders should have the following account balances:
      | trader  | asset | market id | margin | general | bond |
      | trader0 | ETH   | ETH/DEC21 | 1670   | 0       | 4478 |

    # progress time a bit, so the price bounds get updated
    When the network moves ahead "2" blocks
    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | trader3 | ETH/DEC21 | buy  | 10     | 1022  | 2                | TYPE_LIMIT | TIF_GTC | trader3-buy-5  |
      | trader2 | ETH/DEC21 | sell | 75     | 1050  | 0                | TYPE_LIMIT | TIF_GTC | trader2-sell-5 |
      | trader3 | ETH/DEC21 | buy  | 3      | 1020  | 0                | TYPE_LIMIT | TIF_GTC | trader2-sell-6 |
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1010       | TRADING_MODE_CONTINUOUS | 1       | 993       | 1012      | 2323         | 5000           | 23            |
    # getting closer to distressed LP, still in continuous trading
    And the traders should have the following account balances:
      | trader  | asset | market id | margin | general | bond |
      | trader0 | ETH   | ETH/DEC21 | 1762   | 0       | 0    |
    And the insurance pool balance should be "4649" for the market "ETH/DEC21"

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
      | trader0 | ETH   | ETH/DEC21 | 1762   | 0       | 0    |

    # end price auction
    When the network moves ahead "301" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1055       | TRADING_MODE_CONTINUOUS | 1       | 1045      | 1065      | 3482         | 5000           | 33            |
    And the traders should have the following account balances:
      | trader  | asset | market id | margin | general | bond |
      | trader0 | ETH   | ETH/DEC21 | 253    | 1419    | 0    |
    And the insurance pool balance should be "4649" for the market "ETH/DEC21"
