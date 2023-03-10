Feature: Fees rewards with multiple markets and assets

  Background:
    Given the following network parameters are set:
      | name                                                | value  |
      | reward.asset                                        | VEGA   |
      | validators.epoch.length                             | 10s    |
      | validators.delegation.minAmount                     | 10     |
      | reward.staking.delegation.delegatorShare            | 0.883  |
      | reward.staking.delegation.minimumValidatorStake     | 100    |
      | reward.staking.delegation.maxPayoutPerParticipant   | 100000 |
      | reward.staking.delegation.competitionLevel          | 1.1    |
      | reward.staking.delegation.minValidators             | 5      |
      | reward.staking.delegation.optimalStakeMultiplier    | 5.0    |
      | market.value.windowLength                           | 1h     |
      | market.stake.target.timeWindow                      | 24h    |
      | market.stake.target.scalingFactor                   | 1      |
      | market.liquidity.targetstake.triggering.ratio       | 0      |
      | market.liquidity.providers.fee.distributionTimeStep | 0s     |
      | network.markPriceUpdateMaximumFrequency             | 0s     |

    Given time is updated to "2021-08-26T00:00:00Z"
    Given the average block duration is "2"

    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |

    Given the fees configuration named "fees-config-2":
      | maker fee | infrastructure fee |
      | 0.02      | 0.002              |

    When the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         |
      | ETH/DEC22 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         |
      | BTC/DEC21 | BTC        | BTC   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-2 | price-monitoring | default-eth-for-future | 1e0                    | 0                         |
      | BTC/DEC22 | BTC        | BTC   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-2 | price-monitoring | default-eth-for-future | 1e0                    | 0                         |

    Given the parties deposit on asset's general account the following amount:
      | party                                                            | asset | amount   |
      | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | VEGA  | 20000000 |
      | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | USDT  | 20000000 |
      | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | USDC  | 20000000 |

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
      | id | from                                                             | from_account_type    | to                                                               | to_account_type                         | asset | amount | start_epoch | end_epoch | factor | metric                              | metric_asset | markets |
      | 1  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES | VEGA  | 10000  | 1           |           | 1      | DISPATCH_METRIC_MAKER_FEES_RECEIVED | ETH          |         |
      | 2  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES     | USDT  | 20000  | 1           |           | 1      | DISPATCH_METRIC_MAKER_FEES_PAID     | ETH          |         |
      | 3  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES    | USDC  | 5000   | 1           |           | 1      | DISPATCH_METRIC_LP_FEES_RECEIVED    | ETH          |         |
      | 7  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES | VEGA  | 1000   | 1           |           | 1      | DISPATCH_METRIC_MAKER_FEES_RECEIVED | BTC          |         |
      | 8  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES     | USDT  | 2000   | 1           |           | 1      | DISPATCH_METRIC_MAKER_FEES_PAID     | BTC          |         |
      | 9  | a3c024b4e23230c89884a54a813b1ecb4cb0f827a38641c66eeca466da6b2ddf | ACCOUNT_TYPE_GENERAL | 0000000000000000000000000000000000000000000000000000000000000000 | ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES    | USDC  | 500    | 1           |           | 1      | DISPATCH_METRIC_LP_FEES_RECEIVED    | BTC          |         |

    # all LPs use the same shape within a given market so that liquidity score doesn't impact this test
    When the parties submit the following liquidity provision:
      | id       | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      # ETH/DEC21
      | lp1-A    | lp1    | ETH/DEC21 | 4000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1-A    | lp1    | ETH/DEC21 | 4000              | 0.001 | buy  | MID              | 2          | 1      |            |
      | lp1-A    | lp1    | ETH/DEC21 | 4000              | 0.001 | sell | ASK              | 1          | 2      |            |
      | lp1-A    | lp1    | ETH/DEC21 | 4000              | 0.001 | sell | MID              | 2          | 1      |            |
      | lp2-A    | lp2    | ETH/DEC21 | 1000              | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp2-A    | lp2    | ETH/DEC21 | 1000              | 0.002 | buy  | MID              | 2          | 1      |            |
      | lp2-A    | lp2    | ETH/DEC21 | 1000              | 0.002 | sell | ASK              | 1          | 2      |            |
      | lp2-A    | lp2    | ETH/DEC21 | 1000              | 0.002 | sell | MID              | 2          | 1      |            |
      | lpprov-A | lpprov | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lpprov-A | lpprov | ETH/DEC21 | 10000             | 0.001 | buy  | MID              | 2          | 1      |            |
      | lpprov-A | lpprov | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          | 2      |            |
      | lpprov-A | lpprov | ETH/DEC21 | 10000             | 0.001 | sell | MID              | 2          | 1      |            |
      # ETH/DEC22
      | lp1-B    | lp1    | ETH/DEC22 | 8000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1-B    | lp1    | ETH/DEC22 | 8000              | 0.001 | buy  | MID              | 2          | 1      |            |
      | lp1-B    | lp1    | ETH/DEC22 | 8000              | 0.001 | sell | ASK              | 1          | 2      |            |
      | lp1-B    | lp1    | ETH/DEC22 | 8000              | 0.001 | sell | MID              | 2          | 1      |            |
      | lp2-B    | lp2    | ETH/DEC22 | 2000              | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp2-B    | lp2    | ETH/DEC22 | 2000              | 0.002 | buy  | MID              | 2          | 1      |            |
      | lp2-B    | lp2    | ETH/DEC22 | 2000              | 0.002 | sell | ASK              | 1          | 2      |            |
      | lp2-B    | lp2    | ETH/DEC22 | 2000              | 0.002 | sell | MID              | 2          | 1      |            |
      # BTC/DEC21
      | lp1-C    | lp1    | BTC/DEC21 | 2000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1-C    | lp1    | BTC/DEC21 | 2000              | 0.001 | buy  | MID              | 2          | 1      |            |
      | lp1-C    | lp1    | BTC/DEC21 | 2000              | 0.001 | sell | ASK              | 1          | 2      |            |
      | lp1-C    | lp1    | BTC/DEC21 | 2000              | 0.001 | sell | MID              | 2          | 1      |            |
      | lp2-C    | lp2    | BTC/DEC21 | 500               | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp2-C    | lp2    | BTC/DEC21 | 500               | 0.002 | buy  | MID              | 2          | 1      |            |
      | lp2-C    | lp2    | BTC/DEC21 | 500               | 0.002 | sell | ASK              | 1          | 2      |            |
      | lp2-C    | lp2    | BTC/DEC21 | 500               | 0.002 | sell | MID              | 2          | 1      |            |
      | lpprov-C | lpprov | BTC/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lpprov-C | lpprov | BTC/DEC21 | 10000             | 0.001 | buy  | MID              | 2          | 1      |            |
      | lpprov-C | lpprov | BTC/DEC21 | 10000             | 0.001 | sell | ASK              | 1          | 2      |            |
      | lpprov-C | lpprov | BTC/DEC21 | 10000             | 0.001 | sell | MID              | 2          | 1      |            |
      # BTC/DEC22
      | lp1-D    | lp1    | BTC/DEC22 | 4000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1-D    | lp1    | BTC/DEC22 | 4000              | 0.001 | buy  | MID              | 2          | 1      |            |
      | lp1-D    | lp1    | BTC/DEC22 | 4000              | 0.001 | sell | ASK              | 1          | 2      |            |
      | lp1-D    | lp1    | BTC/DEC22 | 4000              | 0.001 | sell | MID              | 2          | 1      |            |
      | lp2-D    | lp2    | BTC/DEC22 | 1000              | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp2-D    | lp2    | BTC/DEC22 | 1000              | 0.002 | buy  | MID              | 2          | 1      |            |
      | lp2-D    | lp2    | BTC/DEC22 | 1000              | 0.002 | sell | ASK              | 1          | 2      |            |
      | lp2-D    | lp2    | BTC/DEC22 | 1000              | 0.002 | sell | MID              | 2          | 1      |            |
      | lpprov-D | lpprov | BTC/DEC22 | 10000             | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lpprov-D | lpprov | BTC/DEC22 | 10000             | 0.001 | buy  | MID              | 2          | 1      |            |
      | lpprov-D | lpprov | BTC/DEC22 | 10000             | 0.001 | sell | ASK              | 1          | 2      |            |
      | lpprov-D | lpprov | BTC/DEC22 | 10000             | 0.001 | sell | MID              | 2          | 1      |            |

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
      | buyer  | price | size | seller |
      | party1 | 1000  | 60   | party2 |

    When the opening auction period ends for market "ETH/DEC22"
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1050  | 30   | party2 |

    When the opening auction period ends for market "BTC/DEC21"
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 850   | 30   | party2 |

    When the opening auction period ends for market "BTC/DEC22"
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1030  | 25   | party2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC22"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/DEC21"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/DEC22"

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | party1 | ETH/DEC21 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell1 |

    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1002  | 7      |
      | sell | 1000  | 20     |
      | sell | 951   | 12     |
      | buy  | 949   | 12     |
      | buy  | 900   | 1      |
      | buy  | 898   | 7      |

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | party2 | ETH/DEC21 | buy  | 20     | 1000  | 4                | TYPE_LIMIT | TIF_GTC | party2-buy1  |
      | party1 | ETH/DEC22 | sell | 30     | 1050  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell2 |
      | party2 | ETH/DEC22 | buy  | 30     | 1050  | 3                | TYPE_LIMIT | TIF_GTC | party2-buy2  |
      | party2 | BTC/DEC21 | sell | 5      | 850   | 3                | TYPE_LIMIT | TIF_GTC | party2-sell1 |
      | party1 | BTC/DEC21 | buy  | 10     | 850   | 0                | TYPE_LIMIT | TIF_GTC | party1-buy1  |
      | party2 | BTC/DEC22 | buy  | 5      | 1030  | 0                | TYPE_LIMIT | TIF_GTC | party2-buy3  |
      | party1 | BTC/DEC22 | sell | 20     | 1030  | 4                | TYPE_LIMIT | TIF_GTC | party1-sell3 |

    And the following trades should be executed:
      | buyer  | seller | price | size |
      # ETH/DEC21
      | party2 | lp1    | 951   | 3    |
      | party2 | lp2    | 951   | 1    |
      | party2 | lpprov | 951   | 8    |
      | party2 | party1 | 1000  | 8    |
      # ETH/DEC22
      | party2 | lp1    | 1001  | 6    |
      | party2 | lp2    | 1001  | 2    |
      | party2 | party1 | 1050  | 22   |
      # BTC/DEC21
      | lp1    | party2 | 949   | 2    |
      | lp2    | party2 | 949   | 1    |
      | lpprov | party2 | 949   | 2    |
      # BTC/DEC22
      | lp1    | party1 | 1089  | 3    |
      | lp2    | party1 | 1089  | 1    |
      | lpprov | party1 | 1089  | 7    |
      | party2 | party1 | 1030  | 5    |

    Then "party1" should have general account balance of "599910476" for asset "ETH"
    Then "party2" should have general account balance of "599947494" for asset "ETH"
    Then "lp1" should have general account balance of "5999984128" for asset "ETH"
    Then "lp2" should have general account balance of "5999995630" for asset "ETH"

    Then "party1" should have general account balance of "299955604" for asset "BTC"
    Then "party2" should have general account balance of "299949367" for asset "BTC"
    Then "lp1" should have general account balance of "2999990841" for asset "BTC"
    Then "lp2" should have general account balance of "2999997067" for asset "BTC"

    #complete the epoch for rewards to take place
    Then the network moves ahead "7" blocks

    Then "party1" should have general account balance of "6067" for asset "VEGA"
    Then "party2" should have general account balance of "234" for asset "VEGA"
    Then "lp1" should have general account balance of "2031" for asset "VEGA"
    Then "lp2" should have general account balance of "723" for asset "VEGA"

    Then "party1" should have general account balance of "1567" for asset "USDT"
    Then "party2" should have general account balance of "20431" for asset "USDT"

    Then "lp1" should have general account balance of "3061" for asset "USDC"
    Then "lp2" should have general account balance of "760" for asset "USDC"
