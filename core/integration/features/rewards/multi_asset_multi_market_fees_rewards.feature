Feature: Fees rewards with multiple markets and assets

Background:
    Given the following network parameters are set:
      | name                                                |  value   |
      | reward.asset                                        |  VEGA    |
      | validators.epoch.length                             |  10s     |
      | validators.delegation.minAmount                     |  10      |
      | reward.staking.delegation.delegatorShare            |  0.883   |
      | reward.staking.delegation.minimumValidatorStake     |  100     |
      | reward.staking.delegation.maxPayoutPerParticipant   | 100000   |
      | reward.staking.delegation.competitionLevel          |  1.1     |
      | reward.staking.delegation.minValidators             |  5       |
      | reward.staking.delegation.optimalStakeMultiplier    |  5.0     |
      | market.value.windowLength                           | 1h       |
      | market.stake.target.timeWindow                      | 24h      |
      | market.stake.target.scalingFactor                   | 1        |
      | market.liquidity.targetstake.triggering.ratio       | 0        |
      | market.liquidity.providers.fee.distributionTimeStep | 0s       |
      | network.markPriceUpdateMaximumFrequency             | 0s       |

    Given time is updated to "2021-08-26T00:00:00Z"
    Given the average block duration is "2"

    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001             |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |

    Given the fees configuration named "fees-config-2":
      | maker fee | infrastructure fee |
      | 0.02      | 0.002              |

    When the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100          | -100         | 0.1                    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config          |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring | default-eth-for-future |
      | ETH/DEC22 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring | default-eth-for-future |
      | BTC/DEC21 | BTC        | BTC   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-2 | price-monitoring | default-eth-for-future |
      | BTC/DEC22 | BTC        | BTC   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-2 | price-monitoring | default-eth-for-future |

    Given the parties deposit on asset's general account the following amount:
    | party           | asset | amount   |
    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf   | VEGA   | 20000000 |
    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf   | USDT   | 20000000 |
    | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf   | USDC   | 20000000 |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | BTC   | 3000000000 |
      | lp2    | BTC   | 3000000000 |
      | lp3    | BTC   | 3000000000 |
      | party1 | BTC   | 300000000  |
      | party2 | BTC   | 300000000  |
      | lp1    | ETH   | 6000000000 |
      | lp2    | ETH   | 6000000000 |
      | lp3    | ETH   | 6000000000 |
      | party1 | ETH   | 600000000  |
      | party2 | ETH   | 600000000  |
      | lpprov | ETH   | 9000000000 |
      | lpprov | BTC   | 9000000000 |

    #complete the epoch to advance to a meaningful epoch (can't setup transfer to start at epoch 0)
    Then the network moves ahead "7" blocks

Scenario: all sort of fees with multiple assets and multiple markets pay rewards on epoch end

    Given the parties submit the following recurring transfers:
    | id  |                             from                                 |  from_account_type    |                                to                                 |   to_account_type                       | asset  |  amount | start_epoch | end_epoch | factor |               metric                | metric_asset | markets   |
    | 1   | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf |  ACCOUNT_TYPE_GENERAL |  0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES | VEGA   |  10000  |      1      |           |   1    | DISPATCH_METRIC_MAKER_FEES_RECEIVED |    ETH       |           |
    | 2   | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf |  ACCOUNT_TYPE_GENERAL |  0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES     | USDT   |  20000  |      1      |           |   1    | DISPATCH_METRIC_MAKER_FEES_PAID     |    ETH       |           |
    | 3   | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf |  ACCOUNT_TYPE_GENERAL |  0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES    | USDC   |  5000   |      1      |           |   1    | DISPATCH_METRIC_LP_FEES_RECEIVED    |    ETH       |           |
    | 7   | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf |  ACCOUNT_TYPE_GENERAL |  0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES | VEGA   |  1000   |      1      |           |   1    | DISPATCH_METRIC_MAKER_FEES_RECEIVED |    BTC       |           |
    | 8   | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf |  ACCOUNT_TYPE_GENERAL |  0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES     | USDT   |  2000   |      1      |           |   1    | DISPATCH_METRIC_MAKER_FEES_PAID     |    BTC       |           |
    | 9   | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf |  ACCOUNT_TYPE_GENERAL |  0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES    | USDC   |  500    |      1      |           |   1    | DISPATCH_METRIC_LP_FEES_RECEIVED    |    BTC       |           |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type |
      | lp1 | lp1    | ETH/DEC21 | 4000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | lp1    | ETH/DEC21 | 4000              | 0.001 | buy  | MID              | 2          | 1      | amendment |
      | lp1 | lp1    | ETH/DEC21 | 4000              | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | lp1    | ETH/DEC21 | 4000              | 0.001 | sell | MID              | 2          | 1      | amendment |
      | lp2 | lp1    | ETH/DEC22 | 8000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp2 | lp1    | ETH/DEC22 | 8000              | 0.001 | buy  | MID              | 2          | 1      | amendment |
      | lp2 | lp1    | ETH/DEC22 | 8000              | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp2 | lp1    | ETH/DEC22 | 8000              | 0.001 | sell | MID              | 2          | 1      | amendment |
      | lp3 | lp1    | BTC/DEC21 | 2000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp3 | lp1    | BTC/DEC21 | 2000              | 0.001 | buy  | MID              | 2          | 1      | amendment |
      | lp3 | lp1    | BTC/DEC21 | 2000              | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp3 | lp1    | BTC/DEC21 | 2000              | 0.001 | sell | MID              | 2          | 1      | amendment |
      | lp4 | lp1    | BTC/DEC22 | 4000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp4 | lp1    | BTC/DEC22 | 4000              | 0.001 | buy  | MID              | 2          | 1      | amendment |
      | lp4 | lp1    | BTC/DEC22 | 4000              | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp4 | lp1    | BTC/DEC22 | 4000              | 0.001 | sell | MID              | 2          | 1      | amendment |
      | lp5 | lp2    | ETH/DEC21 | 1000              | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp5 | lp2    | ETH/DEC21 | 1000              | 0.002 | buy  | MID              | 2          | 1      | amendment |
      | lp5 | lp2    | ETH/DEC21 | 1000              | 0.002 | sell | ASK              | 1          | 2      | amendment |
      | lp5 | lp2    | ETH/DEC21 | 1000              | 0.002 | sell | MID              | 2          | 1      | amendment |
      | lp6 | lp2    | ETH/DEC22 | 2000              | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp6 | lp2    | ETH/DEC22 | 2000              | 0.002 | buy  | MID              | 2          | 1      | amendment |
      | lp6 | lp2    | ETH/DEC22 | 2000              | 0.002 | sell | ASK              | 1          | 2      | amendment |
      | lp6 | lp2    | ETH/DEC22 | 2000              | 0.002 | sell | MID              | 2          | 1      | amendment |
      | lp7 | lp2    | BTC/DEC21 | 500               | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp7 | lp2    | BTC/DEC21 | 500               | 0.002 | buy  | MID              | 2          | 1      | amendment |
      | lp7 | lp2    | BTC/DEC21 | 500               | 0.002 | sell | ASK              | 1          | 2      | amendment |
      | lp7 | lp2    | BTC/DEC21 | 500               | 0.002 | sell | MID              | 2          | 1      | amendment |
      | lp8 | lp2    | BTC/DEC22 | 1000              | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp8 | lp2    | BTC/DEC22 | 1000              | 0.002 | buy  | MID              | 2          | 1      | amendment |
      | lp8 | lp2    | BTC/DEC22 | 1000              | 0.002 | sell | ASK              | 1          | 2      | amendment |
      | lp8 | lp2    | BTC/DEC22 | 1000              | 0.002 | sell | MID              | 2          | 1      | amendment |
      | lp9 | lpprov | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 50         | 1      | submission |
      | lp9 | lpprov | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 50         | 1      | submission |
      | lpA | lpprov | BTC/DEC21 | 10000             | 0.001 | buy  | BID              | 50         | 1      | submission |
      | lpA | lpprov | BTC/DEC21 | 10000             | 0.001 | sell | ASK              | 50         | 1      | submission |
      | lpB | lpprov | BTC/DEC22 | 10000             | 0.001 | buy  | BID              | 50         | 1      | submission |
      | lpB | lpprov | BTC/DEC22 | 10000             | 0.001 | sell | ASK              | 50         | 1      | submission |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC22 | buy  | 2      | 950   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC22 | buy  | 30     | 1050  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC22 | sell | 2      | 1150  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC22 | sell | 30     | 1050  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/DEC21 | buy  | 3      | 800   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/DEC21 | buy  | 30     | 850   | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/DEC21 | sell | 3      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/DEC21 | sell | 30     | 850   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/DEC22 | buy  | 4      | 950   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/DEC22 | buy  | 25     | 1030  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/DEC22 | sell | 4      | 1150  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/DEC22 | sell | 25     | 1030  | 0                | TYPE_LIMIT | TIF_GTC |

    And the market data for the market "ETH/DEC21" should be:
      | target stake | supplied stake |
      | 12000        | 15000          |
    And the market data for the market "ETH/DEC22" should be:
      | target stake | supplied stake |
      | 6300         | 10000          |
    And the market data for the market "BTC/DEC21" should be:
      | target stake | supplied stake |
      | 5100         | 12500          |
    And the market data for the market "BTC/DEC22" should be:
      | target stake | supplied stake |
      | 5150         | 15000          |
    When the opening auction period ends for market "ETH/DEC21"
    Then the following trades should be executed:
      | buyer   | price | size | seller  |
      | party1  | 1000  | 60   |  party2 |

    When the opening auction period ends for market "ETH/DEC22"
    Then the following trades should be executed:
      | buyer   | price | size | seller  |
      | party1  | 1050  | 30   |  party2 |

    When the opening auction period ends for market "BTC/DEC21"
    Then the following trades should be executed:
      | buyer   | price | size | seller  |
      | party1  | 850  | 30   |  party2 |

    When the opening auction period ends for market "BTC/DEC22"
    Then the following trades should be executed:
      | buyer   | price | size | seller  |
      | party1  | 1030  | 25   |  party2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC22"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/DEC21"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/DEC22"

    Then the parties place the following orders with ticks:
    | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
    | party1 | ETH/DEC21 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell1 |
    | party2 | ETH/DEC21 | buy  | 20     | 1000  | 3                | TYPE_LIMIT | TIF_GTC | party2-buy1  |
    | party1 | ETH/DEC22 | sell | 30     | 1050  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell2 |
    | party2 | ETH/DEC22 | buy  | 30     | 1050  | 3                | TYPE_LIMIT | TIF_GTC | party2-buy2  |
    | party2 | BTC/DEC21 | sell | 5      | 850   | 2                | TYPE_LIMIT | TIF_GTC | party2-sell1 |
    | party1 | BTC/DEC21 | buy  | 10     | 850   | 3                | TYPE_LIMIT | TIF_GTC | party1-buy1  |
    | party2 | BTC/DEC22 | buy  | 5      | 1030  | 0                | TYPE_LIMIT | TIF_GTC | party2-buy3  |
    | party1 | BTC/DEC22 | sell | 20     | 1030  | 3                | TYPE_LIMIT | TIF_GTC | party1-sell3 |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party2 | 951   | 6    | lp1    |
      | party2 | 951   | 2    | lp2    |
      | party2 | 1000  | 12   | party1 |
      | party2 | 1050  | 16   | party1 |
      | party1 | 850   | 30   | party2 |
      | party2 | 1030  | 5    | party1 |

    Then "party1" should have general account balance of "599978284" for asset "ETH"
    Then "party2" should have general account balance of "599995115" for asset "ETH"
    Then "lp1" should have general account balance of "5999973192" for asset "ETH"
    Then "lp2" should have general account balance of "5999993014" for asset "ETH"

    Then "party1" should have general account balance of "299986254" for asset "BTC"
    Then "party2" should have general account balance of "299984316" for asset "BTC"
    Then "lp1" should have general account balance of "2999979830" for asset "BTC"
    Then "lp2" should have general account balance of "2999994373" for asset "BTC"

    #complete the epoch for rewards to take place
    Then the network moves ahead "7" blocks

    # calculation of maker fees received reward - given in VEGA
    # ETH - got 10k VEGA
    # BTC - got 1000 VEGA
    # in ETH ETH/DEC21 contributed (80/206) 0.3883495146 of the maker fees received => 3,883.495146 => 3,883
    # in ETH ETH/DEC22 contributed (126/206) 0.6116504854 of the maker fees received => 6,116.504854 => 6,116
    # in BTC BTC/DEC21 contributed (85/188) 0.4521276596 of the maker fees received => 452.1276596 => 452
    # in BTC BTC/DEC21 contributed (103/188) 0.5478723404 of the maker fees received => 547.8723404 => 547

    # ETH/DEC21 maker fees received:
    # party1 - 0.4 * 3,883 = 1,553.2 => 1553
    # lp1 - 0.4 * 3,883 = 1,553.2 => 1553
    # lp2 - 0.2 * 3,883 = 776.6 => 776

    # ETH/DEC22 maker fees received:
    # party1 - 1 * 6,116 = 6,116

    # BTC/DEC21 maker fees received:
    # party2 - 1 * 452 = 452

    # BTC/DEC22 maker fees received:
    # party2 - 1 * 547 = 547

    # total party1 = 1553 + 6,116 = 7660
    # total party2 = 452 + 547 = 999
    # total lp1 = 1553
    # total lp2 = 776

    Then "party1" should have general account balance of "599978284" for asset "ETH"
    Then "party1" should have general account balance of "5657" for asset "VEGA"
    Then "party2" should have general account balance of "276" for asset "VEGA"
    Then "lp1" should have general account balance of "3851" for asset "VEGA"
    Then "lp2" should have general account balance of "1207" for asset "VEGA"

    # calculation of taker fees paid reward - given in USDT
    # ETH - got 20k USDT
    # BTC - got 2000 USDT
    # in ETH ETH/DEC21 contributed (80/206) 0.3883495146 of the taker fees paid => 7,766.990292 => 7766
    # in ETH ETH/DEC22 contributed (126/206) 0.6116504854 of the taker fees paid => 12,233.009708 => 12233
    # in BTC BTC/DEC21 contributed (85/188) 0.4521276596 of the taker fees paid => 904.2553192 => 904
    # in BTC BTC/DEC21 contributed (103/188) 0.5478723404 of the taker fees paid => 1,095.7446808 => 1095

    # ETH/DEC21 taker fees paid:
    # party2 - 1 * 7766 = 7766 => 7766

    # ETH/DEC22 taker fees paid:
    # party2 - 1 * 12233 = 12233

    # BTC/DEC21 taker fees paid:
    # party1 - 1 * 904 = 904

    # BTC/DEC22 taker fees paid:
    # party1 - 1 * 1095 = 1095

    # total party1 = 904 + 1095 = 1999
    # total party2 = 3106 + 12233 = 19999

    Then "party1" should have general account balance of "1648" for asset "USDT"
    Then "party2" should have general account balance of "20349" for asset "USDT"

    # calculation of LP fees received reward - given in USDC
    # ETH - got 5000 USDC
    # BTC - got 500 USDC
    # in ETH ETH/DEC21 contributed (40/40) 0.3883495146 of the LP fees received => 5000
    # in ETH ETH/DEC22 contributed (0/40) 0 of the LP fees received => 0
    # in BTC BTC/DEC21 contributed (0/0) 0 of the LP fees received => 0
    # in BTC BTC/DEC21 contributed (0/0) 0 of the LP fees received => 0

    # ETH/DEC21 LP fees received:
    # lp1 - 0.8 * 5000 = 4000
    # lp2 - 0.2 * 5000 = 1000

    Then "lp1" should have general account balance of "3023" for asset "USDC"
    Then "lp2" should have general account balance of "774" for asset "USDC"
