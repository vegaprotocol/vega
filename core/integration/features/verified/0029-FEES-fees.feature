Feature: Fees calculations

  Background:
    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.003              |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 60      | 0.99        | 2                 |
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |

    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 2     |
      | market.fee.factors.buybackFee           | 0.001 |
      | market.fee.factors.treasuryFee          | 0.002 |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC21 | ETH/USD    | USD   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 0.25                   | 0                         | default-futures |

    And the average block duration is "2"
  Scenario: 001: Testing fees get collected when amended order trades (0029-FEES-005)
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount |
      | aux1    | USD   | 100000 |
      | aux2    | USD   | 100000 |
      | aux3    | USD   | 100000 |
      | aux4    | USD   | 100000 |
      | trader1 | USD   | 10000  |
      | trader2 | USD   | 10000  |
      | trader3 | USD   | 490    |
      | trader4 | USD   | 250    |
      | trader5 | USD   | 5000   |
      | trader6 | USD   | 5000   |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.002 | submission |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.002 | submission |
    When the network moves ahead "2" blocks

    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | ETH/DEC21 | buy  | 1      | 820   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux4  | ETH/DEC21 | sell | 1      | 1180  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1000       | TRADING_MODE_CONTINUOUS |

    And the following trades should be executed:
      | buyer | price | size | seller |
      | aux1  | 1000  | 1    | aux2   |
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general |
      | aux1  | USD   | ETH/DEC21 | 540    | 89460   |
    #0029-FEES-036:no fees are collected during opening auction

    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | buy  | 820   | 1      |
      | sell | 1180  | 1      |

    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC21 | buy  | 2      | 1002  | 0                | TYPE_LIMIT | TIF_GTC | t1-b2-01  |
      | trader2 | ETH/DEC21 | sell | 2      | 1004  | 0                | TYPE_LIMIT | TIF_GTC | t2-s4-01  |

    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1000       | TRADING_MODE_CONTINUOUS |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader1 | USD   | ETH/DEC21 | 480    | 9520    |
      | trader2 | USD   | ETH/DEC21 | 240    | 9760    |

    And the parties amend the following orders:
      | party   | reference | price | size delta | tif     |
      | trader2 | t2-s4-01  | 1002  | 0          | TIF_GTC |

    Then the market data for the market "ETH/DEC21" should be:
      | mark price | last traded price | trading mode            |
      | 1000       | 1002              | TRADING_MODE_CONTINUOUS |
    And the following trades should be executed:
      | buyer   | price | size | seller  |
      | trader1 | 1002  | 2    | trader2 |

    # For trader1-
    # trade_value_for_fee_purposes for trader1 = size_of_trade * price_of_trade = 2 * 1002 = 2004
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.003 * 2004 = 6.012 = 7 (rounded up to nearest whole value)
    # maker_fee =  fee_factor[maker]  * trade_value_for_fee_purposes = 0.004 * 2004 = 8.016 = 9 (rounded up to nearest whole value)
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0.002 * 2004 = 4.008 = 5 (rounded up to nearest whole value)
    # buy_back_fee = buy_back_factor * trade_value_for_fee_purposes = 0.001 * 2004 = 2.004 = 3
    # treasury_fee = treasury_fee_factor * trade
    #_value_for_fee_purposes = 0.002 * 2004 = 4.008 = 5

    And the following transfers should happen:
      | from    | to      | from account            | to account                       | market id | amount | asset |
      #0029-FEES-046:Once total fee is collected, `maker_fee = fee_factor[maker]  * trade_value_for_fee_purposes` is transferred to maker at the end of fee distribution time.
      | trader2 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 9      | USD   |
      #0029-FEES-045:Once total fee is collected, `infrastructure_fee = fee_factor[infrastucture]  * trade_value_for_fee_purposes` is transferred to infrastructure fee pool for that asset at the end of fee distribution time.
      | trader2 |         | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 7      | USD   |
      #0029-FEES-048:Once total fee is collected, `liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes` (with appropriate fraction of `high_volume_maker_fee` deducted) is transferred to the treasury fee pool for that asset
      | trader2 |         | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 5      | USD   |
      #0029-FEES-049:Once total fee is collected, `treasury_fee = fee_factor[treasury] * trade_value_for_fee_purposes` (with appropriate fraction of `high_volume_maker_fee` deducted) is transferred to the treasury fee pool for that asset
      | trader2 |         | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_NETWORK_TREASURY    |           | 5      | USD   |
      #0029-FEES-050:Once total fee is collected, `buyback_fee = fee_factor[buyback] * trade_value_for_fee_purposes` (with with appropriate fraction of `high_volume_maker_fee` deducted) is transferred to the buyback fee pool for that asset
      | trader2 |         | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_BUY_BACK_FEES       |           | 3      | USD   |
      | market  | trader1 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 9      | USD   |

    # total_fee = maker_fee + infrastructure_fee + liquidity_fee + buy back + treasury =  9 + 7 + 5 + 8 = 29
    #0029-FEES-038: In a matched trade, if the price taker has enough asset to cover the total fee in their general account, then the total fee should be taken from their general account.
    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader1 | USD   | ETH/DEC21 | 480    | 9529    |
      | trader2 | USD   | ETH/DEC21 | 240    | 9731    |

    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC21 | buy  | 2      | 1002  | 0                | TYPE_LIMIT | TIF_GTC | t1-b2-01  |
      | trader4 | ETH/DEC21 | sell | 2      | 1002  | 1                | TYPE_LIMIT | TIF_GTC | t2-s4-01  |

    #trader4 started with asset of 250, and paid 29 for total trading fee
    #0029-FEES-039:In a matched trade, if the price taker has insufficient asset to cover the total fee in their general account (but has enough in general + margin account), then the remainder will be taken from their margin account.
    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3 | USD   | ETH/DEC21 | 480    | 19      |
      | trader4 | USD   | ETH/DEC21 | 221    | 0       |

    And the following transfers should happen:
      | from    | to      | from account            | to account                       | market id | amount | asset |
      | market  | trader3 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 9      | USD   |
      | trader4 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 9      | USD   |
      | trader4 |         | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_BUY_BACK_FEES       |           | 1      | USD   |
      | trader4 |         | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | ETH/DEC21 | 7      | USD   |
      | trader4 |         | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 5      | USD   |
      | trader4 |         | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_NETWORK_TREASURY    | ETH/DEC21 | 5      | USD   |

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound |
      | 1000       | TRADING_MODE_CONTINUOUS | 60      | 900       | 1100      |

    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader5 | ETH/DEC21 | buy  | 2      | 1101  | 0                | TYPE_LIMIT | TIF_GTC | t1-b2-01  |
      | trader6 | ETH/DEC21 | sell | 2      | 1101  | 0                | TYPE_LIMIT | TIF_GTC | t2-s4-01  |

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | horizon | min bound | max bound |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 60      | 900       | 1100      |

    When the network moves ahead "4" blocks

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound |
      | 1101       | TRADING_MODE_CONTINUOUS | 60      | 1002      | 1200      |

    # trade_value_for_fee_purposes for trader1 = size_of_trade * price_of_trade = 2 * 1101 = 2202
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.003 * 2202 = 6.606 = 7 (rounded up to nearest whole value)
    # maker_fee =  fee_factor[maker]  * trade_value_for_fee_purposes = 0.004 * 2202 = 8.808 = 9 (rounded up to nearest whole value)
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0.002 * 2202 = 4.404 = 5 (rounded up to nearest whole value)
    # buy_back_fee = buy_back_factor * trade_value_for_fee_purposes = 0.001 * 2202 = 2.202 = 3
    # treasury_fee = treasury_fee_factor * trade
    #_value_for_fee_purposes = 0.002 * 2202 = 4.404 = 5

    And the following transfers should happen:
      | from    | to | from account         | to account                       | market id | amount | asset |
      #0029-FEES-037:During normal auction (including market protection), each side in a matched trade should contribute `0.5*(infrastructure_fee + liquidity_fee + treasury_fee + buyback_fee)`
      | trader5 |    | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | ETH/DEC21 | 4      | USD   |
      | trader6 |    | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | ETH/DEC21 | 4      | USD   |
      | trader5 |    | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 3      | USD   |
      | trader6 |    | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 3      | USD   |
      | trader5 |    | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_NETWORK_TREASURY    | ETH/DEC21 | 3      | USD   |
      | trader6 |    | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_NETWORK_TREASURY    | ETH/DEC21 | 3      | USD   |
      | trader5 |    | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_BUY_BACK_FEES       |           | 2      | USD   |
      | trader6 |    | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_BUY_BACK_FEES       |           | 2      | USD   |

