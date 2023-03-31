Feature: Replicate LP getting distressed during continuous trading, check if penalty is implemented correctly

  Background:

    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0.7                    | 0                         |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party0 | USD   | 500000    |
      | party1 | USD   | 100000000 |
      | party2 | USD   | 100000000 |
      | party3 | USD   | 100000000 |
      | party4 | USD   | 100000000 |
      | party5 | USD   | 100000000 |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  @Now
  Scenario: 001, LP gets distressed during continuous trading, no DPD setting (0044-LIME-002, 0035-LIQM-004)


    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 0.24             | 24h         | 1              |
    When the markets are updated:
      | id        | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/MAR22 | updated-lqm-params   | 0.7                    | 0                         |
    And the following network parameters are set:
      | name                                  | value |
      | market.liquidity.bondPenaltyParameter | 0.2   |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party0 | USD   | 12500  |
    And the average block duration is "1"

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/MAR22 | 50000             | 0.001 | sell | ASK              | 500        | 20     | submission |
      | lp1 | party0 | ETH/MAR22 | 50000             | 0.001 | buy  | BID              | 500        | 20     | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party4 | ETH/MAR22 | buy  | 100    | 850   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-4  |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/MAR22 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/MAR22 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party2 | ETH/MAR22 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |
      | party2 | ETH/MAR22 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |
      | party5 | ETH/MAR22 | sell | 100    | 1200  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-5 |

    When the opening auction period ends for market "ETH/MAR22"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 0.1
    And the insurance pool balance should be "0" for the market "ETH/MAR22"
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 1000      | 1000      | 35569        | 50000          | 10            |

    #check the volume on the order book
    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1030  | 49     |
      | sell | 1010  | 1      |
      | buy  | 990   | 1      |
      | buy  | 970   | 52     |
      | buy  | 900   | 1      |

    # check the requried balances
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  | bond  |
      | party0 | USD   | ETH/MAR22 | 209146 | 253354   | 50000 |
      | party1 | USD   | ETH/MAR22 | 11425  | 99988575 |       |
      | party2 | USD   | ETH/MAR22 | 51690  | 99948310 |       |
    #check the margin levels
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial |
      | party0 | ETH/MAR22 | 174289      | 209146  |
      | party1 | ETH/MAR22 | 9889        | 11866   |
      | party2 | ETH/MAR22 | 42963       | 51555   |
    #check position (party0 has no position)
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 10     | 0              | 0            |
      | party2 | -10    | 0              | 0            |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | party3 | ETH/MAR22 | buy  | 30     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party3-buy-1 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party2 | ETH/MAR22 | sell | 50     | 1000  | 1                | TYPE_LIMIT | TIF_GTC | party2-sell-4 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 1000      | 1000      | 142276       | 50000          | 40            |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1010 x 13 x 1 x 0.1
    # target stake 1313 with target trigger on 0.6 -> ~788 triggers liquidity auction

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  | bond  |
      | party0 | USD   | ETH/MAR22 | 209146 | 253354   | 50000 |
      | party1 | USD   | ETH/MAR22 | 11425  | 99988575 |       |
      | party2 | USD   | ETH/MAR22 | 264970 | 99734850 |       |
    #check the margin levels
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party0 | ETH/MAR22 | 177846      | 195630 | 213415  | 248984  |

    And the insurance pool balance should be "0" for the market "ETH/MAR22"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party0 | ETH/MAR22 | sell | 70     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party0-sell-3 |
      | party1 | ETH/MAR22 | buy  | 100    | 1000  | 2                | TYPE_LIMIT | TIF_GTC | party1-buy-4  |

    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party0 | ETH/MAR22 | 428592      | 471451 | 514310  | 600028   |
      | party1 | ETH/MAR22 | 89962       | 98958  | 107954  | 125946   |

    And the insurance pool balance should be "12231" for the market "ETH/MAR22"

    #check the requried balances
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  | bond |
      | party0 | USD   | ETH/MAR22 | 500549 | 0        | 0    |
      | party1 | USD   | ETH/MAR22 | 107954 | 99891506 |      |
      | party2 | USD   | ETH/MAR22 | 264970 | 99734930 |      |
      | party3 | USD   | ETH/MAR22 | 28826  | 99971294 |      |

    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party0 | ETH/MAR22 | 428592      | 471451 | 514310  | 600028  |
      | party1 | ETH/MAR22 | 89962       | 98958  | 107954  | 125946  |
      | party2 | ETH/MAR22 | 220809      | 242889 | 264970  | 309132  |

    #documented behaviour why margin account has higher value than margin initial level:
    #When an LP submits a new order, we recalculate the margin requirements as we do for any order. At this point, we don't care if the party is an LP or not. We work out the margin requirements assuming whatever position the party holds stays the same. If the margin requirement increases, we try and top up the margin balance to the initial margin level. If this means dipping in to the bond account, we slash the bond account and apply a penalty.
    #This newly submitted order will change the LP orders/shapes, so we cancel the orders currently on the book, and replace them with new ones. In doing so, we check the margin requirements as we submit the new orders. In this particular case, the party had buy and sell orders (potential long/short) on the book the book like this: sell 106@1020, buy 109@970. The party submitted a sell order of 15@1000. This meant that their worst potential short (and the position we based the margin calculation on) was short 121@(15000+108120)/121 (≃1017.5).
    #The sell LP order was updated to sell 90@1020 (potential short becoming 105@(15000+91800)/105 (≃1017.1). The margin requirement drops, but the release level remains above the margin balance. Because the margin requirement when the sell order for 15@1000 was submitted caused the margin balance to go up to 500000, we didn't release any of the balance later on. The initial margin level dropped down to 448170, but the release level was higher than the margin account balance still.
    #The upshot is that seemingly, we transferred too much to the margin account, because briefly (between placing the new order and repricing the LP orders), that was the balance required.

    # move to the next block to perform liquidity check
    Then the network moves ahead "1" blocks
    # open interest updates to include buy order of size 20
    And the market data for the market "ETH/MAR22" should be:
      | trading mode                    | auction trigger                          | target stake | supplied stake | open interest |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | 462397       | 50000          | 130           |

  @Now
  Scenario: 002, LP gets slashed twice during continuous trading, 0044-LIME-002, No DPD setting

    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 0.1              | 24h         | 1              |
    When the markets are updated:
      | id        | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/MAR22 | updated-lqm-params   | 0.7                    | 0                         |
    And the following network parameters are set:
      | name                                  | value |
      | market.liquidity.bondPenaltyParameter | 0.5   |

    And the average block duration is "1"

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/MAR22 | 50000             | 0.001 | sell | ASK              | 500        | 20     | submission |
      | lp1 | party0 | ETH/MAR22 | 50000             | 0.001 | buy  | BID              | 500        | 20     | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/MAR22 | buy  | 2      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/MAR22 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/MAR22 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party2 | ETH/MAR22 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |
      | party2 | ETH/MAR22 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

    When the opening auction period ends for market "ETH/MAR22"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 0.1
    And the insurance pool balance should be "0" for the market "ETH/MAR22"
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 1000      | 1000      | 35569        | 50000          | 10            |

    #check the volume on the order book
    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1030  | 49     |
      | sell | 1010  | 1      |
      | buy  | 990   | 1      |
      | buy  | 970   | 52     |
      | buy  | 900   | 2      |

    # check the requried balances
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond  |
      | party0 | USD   | ETH/MAR22 | 209146 | 240854  | 50000 |

    #check the margin levels
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party0 | ETH/MAR22 | 174289      | 191717 | 209146  | 244004  |

    #check position (party0 has no position)
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 10     | 0              | 0            |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | party3 | ETH/MAR22 | buy  | 30     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party3-buy-1 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"

    # #check the volume on the order book
    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1010  | 1      |
      | buy  | 1000  | 30     |
      | buy  | 990   | 1      |
      | buy  | 900   | 2      |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party2 | ETH/MAR22 | sell | 50     | 1000  | 1                | TYPE_LIMIT | TIF_GTC | party2-sell-4 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 1000      | 1000      | 142276       | 50000          | 40            |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1010 x 13 x 1 x 0.1
    # target stake 1313 with target trigger on 0.6 -> ~788 triggers liquidity auction

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond  |
      | party0 | USD   | ETH/MAR22 | 209146 | 240854  | 50000 |
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party0 | ETH/MAR22 | 177846      | 195630 | 213415  | 248984  |

    And the insurance pool balance should be "0" for the market "ETH/MAR22"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party0 | ETH/MAR22 | sell | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party0-sell-3 |
      | party1 | ETH/MAR22 | buy  | 100    | 1000  | 2                | TYPE_LIMIT | TIF_GTC | party1-buy-4  |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 1000      | 1000      | 426828       | 50000          | 120           |

    And the insurance pool balance should be "56753" for the market "ETH/MAR22"
    #check the volume on the order book

    #check the requried balances
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond  |
      | party0 | USD   | ETH/MAR22 | 443487 | 0       | 0     |

    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party0 | ETH/MAR22 | 429703      | 472673 | 515643  | 601584  |

#documented behavier why margin account has higher value than margin initial level:
#When an LP submits a new order, we recalculate the margin requirements as we do for any order. At this point, we don't care if the party is an LP or not. We work out the margin requirements assuming whatever position the party holds stays the same. If the margin requirement increases, we try and top up the margin balance to the initial margin level. If this means dipping in to the bond account, we slash the bond account and apply a penalty.
#This newly submitted order will change the LP orders/shapes, so we cancel the orders currently on the book, and replace them with new ones. In doing so, we check the margin requirements as we submit the new orders. In this particular case, the party had buy and sell orders (potential long/short) on the book the book like this: sell 106@1020, buy 109@970. The party submitted a sell order of 15@1000. This meant that their worst potential short (and the position we based the margin calculation on) was short 121@(15000+108120)/121 (≃1017.5).
#The sell LP order was updated to sell 90@1020 (potential short becoming 105@(15000+91800)/105 (≃1017.1). The margin requirement drops, but the release level remains above the margin balance. Because the margin requirement when the sell order for 15@1000 was submitted caused the margin balance to go up to 500000, we didn't release any of the balance later on. The initial margin level dropped down to 448170, but the release level was higher than the margin account balance still.
#The upshot is that seemingly, we transferred too much to the margin account, because briefly (between placing the new order and repricing the LP orders), that was the balance required.
