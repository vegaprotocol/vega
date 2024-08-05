Feature: Fees calculations

  Background:
    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.003              |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |
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
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 0.25                   | 0                         | default-futures |

    And the average block duration is "2"
  Scenario: 001: Testing fees get collected when amended order trades (0029-FEES-005)
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount    |
      | aux1    | ETH   | 100000000 |
      | aux2    | ETH   | 100000000 |
      | trader1 | ETH   | 10000     |
      | trader2 | ETH   | 10000     |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.002 | submission |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.002 | submission |
    When the network moves ahead "2" blocks
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | aux1  | ETH/DEC21 | 10        | 1                    | buy  | BID              | 10     | 10     |
      | aux2  | ETH/DEC21 | 10        | 1                    | sell | ASK              | 10     | 10     |
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC21 | buy  | 1      | 920   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1      | 1080  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1000       | TRADING_MODE_CONTINUOUS |

    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | buy  | 910   | 10     |
      | buy  | 920   | 1      |
      | sell | 1080  | 1      |
      | sell | 1090  | 10     |

    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC21 | buy  | 2      | 1002  | 0                | TYPE_LIMIT | TIF_GTC | t1-b2-01  |
      | trader2 | ETH/DEC21 | sell | 4      | 1004  | 0                | TYPE_LIMIT | TIF_GTC | t2-s4-01  |

    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1000       | TRADING_MODE_CONTINUOUS |

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
    # treasury_fee = treasury_fee_factor * trade_value_for_fee_purposes = 0.002 * 2004 = 4.008 = 5

    And the following transfers should happen:
      | from    | to      | from account            | to account                       | market id | amount | asset |
      | trader2 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 9      | ETH   |
      | trader2 |         | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 7      | ETH   |
      | trader2 |         | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_NETWORK_TREASURY    |           | 5      | ETH   |
      | trader2 |         | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_BUY_BACK_FEES       |           | 3      | ETH   |
      | trader2 |         | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 7      | ETH   |
      | market  | trader1 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 9      | ETH   |

    # total_fee = maker_fee + infrastructure_fee + liquidity_fee + buy back + treasury =  9 + 7 + 5 + 8 = 29
    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC21 | 480    | 9529    |
      | trader2 | ETH   | ETH/DEC21 | 480    | 9491    |
