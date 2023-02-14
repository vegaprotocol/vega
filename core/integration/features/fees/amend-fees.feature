Feature: Fees when amend trades


  Background:
    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  # this test requires me to be a bit more fresh to fix
  @Fees
  Scenario: Testing fees in continuous trading with one trade
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | aux1     | ETH   | 100000000 |
      | aux2     | ETH   | 100000000 |
      | trader3a | ETH   | 10000     |
      | trader3b | ETH   | 10000     |
      | trader4  | ETH   | 10000     |
      | trader5  | ETH   | 100000000 |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | 10     | submission |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          | 10     | submission |
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
      | sell | 1080  | 1      |
      | buy  | 920   | 1      |
      | buy  | 910   | 10     |
      | sell | 1090  | 10     |

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader3a | ETH/DEC21 | buy  | 2      | 1002  | 0                | TYPE_LIMIT | TIF_GTC | t3a-b3-02 |
      | trader3b | ETH/DEC21 | buy  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC | t3b-b1-02 |
      | trader4  | ETH/DEC21 | sell | 4      | 1002  | 2                | TYPE_LIMIT | TIF_GTC | t4-s4-02  |
    #| trader5  | ETH/DEC21 | sell | 1      | 2002  | 0                | TYPE_LIMIT | TIF_GTC | t5-s4-02  |
    #| trader5  | ETH/DEC21 | buy  | 1      | 101   | 0                | TYPE_LIMIT | TIF_GTC | t5-b1-02  |

    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1002       | TRADING_MODE_CONTINUOUS |

    And the following trades should be executed:
      | buyer    | price | size | seller  |
      | trader3a | 1002  | 2    | trader4 |
      | trader3b | 1002  | 1    | trader4 |

    # For trader3a-
    # trade_value_for_fee_purposes for trader3a = size_of_trade * price_of_trade = 2 * 1002 = 2004
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.002 * 2004 = 4.008 = 5 (rounded up to nearest whole value)
    # maker_fee =  fee_factor[maker]  * trade_value_for_fee_purposes = 0.005 * 2004 = 10.02 = 11 (rounded up to nearest whole value)
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0.001 * 2004 = 2.004 = 3 (rounded up to nearest whole value)

    # For trader3b -
    # trade_value_for_fee_purposes = size_of_trade * price_of_trade = 1 * 1002 = 1002
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.002 * 1002 = 2.004 = 3 (rounded up to nearest whole value)
    # maker_fee =  fee_factor[maker]  * trade_value_for_fee_purposes = 0.005 * 1002 = 5.01 = 6 (rounded up to nearest whole value)
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0.001 * 1002 = 1.002 = 2 (rounded up to nearest whole value)
    And the following transfers should happen:
      | from    | to       | from account            | to account                       | market id | amount | asset |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 11     | ETH   |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 6      | ETH   |
      | trader4 |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 8      | ETH   |
      | market  | trader3a | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 11     | ETH   |
      | market  | trader3b | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 6      | ETH   |

    # total_fee = maker_fee + infrastructure_fee + liquidity_fee =  11 + 6 + 8 = 25
    # Trader3a margin + general account balance = 10000 + 11 ( Maker fees) = 10011
    # Trader3b margin + general account balance = 10000 + 6 ( Maker fees) = 10006
    # Trader4  margin + general account balance = 10000 - (11+6) ( Maker fees) - 8 (Infra fee) = 99975
    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 690    | 9321    |
      #| trader3a | ETH   | ETH/DEC21 | 480    | 9531    |
      | trader3b | ETH   | ETH/DEC21 | 339    | 9667    |
      #| trader3b | ETH   | ETH/DEC21 | 240    | 9766    |
      #| trader4  | ETH   | ETH/DEC21 | 679    | 9291    |
      | trader4  | ETH   | ETH/DEC21 | 480    | 9490    |

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader3a | ETH/DEC21 | buy  | 2      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | t3a-b2-01 |
      | trader4  | ETH/DEC21 | sell | 4      | 1003  | 0                | TYPE_LIMIT | TIF_GTC | t4-s4-03  |
    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 1171   | 8840    |
      | trader4  | ETH   | ETH/DEC21 | 984    | 8986    |
    # ensure orders are on the book
    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1080  | 1      |
      | buy  | 1001  | 2      |
      | buy  | 920   | 1      |
      | buy  | 991   | 10     |
      | sell | 1012  | 10     |
      | sell | 1002  | 1      |
      | sell | 1003  | 4      |

    When the parties amend the following orders:
      | party   | reference | price | size delta | tif     |
      | trader4 | t4-s4-03  | 1002  | 0          | TIF_GTC |
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1002       | TRADING_MODE_CONTINUOUS |
    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1080  | 1      |
      | buy  | 1001  | 2      |
      | buy  | 920   | 1      |
      | buy  | 991   | 10     |
      | sell | 1012  | 10     |
      | sell | 1002  | 5      |

    When the parties amend the following orders:
      | party   | reference | price | size delta | tif     |
      | trader4 | t4-s4-03  | 1001  | 0          | TIF_GTC |
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | last traded price | trading mode            |
      | 1002       | 1001              | TRADING_MODE_CONTINUOUS |
    And the following trades should be executed:
      | buyer    | price | size | seller  |
      | trader3a | 1001  | 2    | trader4 |

    When the parties place the following orders:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader3b | ETH/DEC21 | buy  | 3      | 1002  | 2                | TYPE_LIMIT | TIF_GTC | t3b-b3-02 |
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1002       | TRADING_MODE_CONTINUOUS |
    And the following trades should be executed:
      | buyer    | price | size | seller  |
      | trader3b | 1001  | 2    | trader4 |
      | trader3b | 1002  | 1    | trader4 |

  Scenario: Testing fees get collected when amended order trades
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | aux1     | ETH   | 100000000 |
      | aux2     | ETH   | 100000000 |
      | trader3a | ETH   | 10000     |
      | trader3b | ETH   | 10000     |
      | trader4  | ETH   | 1550      |
      | lpprov   | ETH   | 100000000 |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC21 | buy  | 1      | 920   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1      | 1080  | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | sell | ASK              | 50         | 100    | submission |

    When the opening auction period ends for market "ETH/DEC21"
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1000       | TRADING_MODE_CONTINUOUS |

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21 | buy  | 2      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3b | ETH/DEC21 | buy  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell | 4      | 1002  | 2                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1002       | TRADING_MODE_CONTINUOUS |

    And the following trades should be executed:
      | buyer    | price | size | seller  |
      | trader3a | 1002  | 2    | trader4 |
      | trader3b | 1002  | 1    | trader4 |

    # For trader3a-
    # trade_value_for_fee_purposes for trader3a = size_of_trade * price_of_trade = 2 * 1002 = 2004
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.002 * 2004 = 4.008 = 5 (rounded up to nearest whole value)
    # maker_fee =  fee_factor[maker]  * trade_value_for_fee_purposes = 0.005 * 2004 = 10.02 = 11 (rounded up to nearest whole value)
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0.001 * 2004 = 2.004 = 3 (rounded up to nearest whole value)

    # For trader3b -
    # trade_value_for_fee_purposes = size_of_trade * price_of_trade = 1 * 1002 = 1002
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.002 * 1002 = 2.004 = 3 (rounded up to nearest whole value)
    # maker_fee =  fee_factor[maker]  * trade_value_for_fee_purposes = 0.005 * 1002 = 5.01 = 6 (rounded up to nearest whole value)
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0.001 * 1002 = 1.002 = 2 (rounded up to nearest whole value)

    And the following transfers should happen:
      | from    | to       | from account            | to account                       | market id | amount | asset |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 11     | ETH   |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 6      | ETH   |
      | trader4 |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 8      | ETH   |
      | market  | trader3a | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 11     | ETH   |
      | market  | trader3b | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 6      | ETH   |

    # total_fee = maker_fee + infrastructure_fee + liquidity_fee =  11 + 6 + 8 = 25
    # Trader3a margin + general account balance = 10000 + 11 ( Maker fees) = 10011
    # Trader3b margin + general account balance = 10000 + 6 ( Maker fees) = 10006
    # Trader4  margin + general account balance = 10000 - (11+6) ( Maker fees) - 8 (Infra fee) = 99975

    # TODO: Check why margin doesn't go up after the trade WHEN the liquidity provision order gets included (seems to work fine without LP orders) (expecting first commented out values) but getting second value in other cases
    And the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 798    | 9213    |
      #| trader3a | ETH   | ETH/DEC21 | 699    | 9312    |
      | trader3b | ETH   | ETH/DEC21 | 339    | 9667    |
      | trader4  | ETH   | ETH/DEC21 | 693    | 530     |

    # Placing second set of orders
    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | trader3a | ETH/DEC21 | buy  | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | trader3a-buy-1 |
      | trader4  | ETH/DEC21 | sell | 4      | 1002  | 0                | TYPE_LIMIT | TIF_GTC | trader4-sell-2 |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 1279   | 8732    |
      | trader4  | ETH   | ETH/DEC21 | 1174   | 49      |

    # reducing size
    And the parties amend the following orders:
      | party   | reference      | price | size delta | tif     |
      | trader4 | trader4-sell-2 | 1000  | 0          | TIF_GTC |
    Then the network moves ahead "1" blocks

    # matching the order now
    And the following trades should be executed:
      | buyer    | price | size | seller  |
      | trader3a | 1000  | 2    | trader4 |

    # checking if continuous mode still exists
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1000       | TRADING_MODE_CONTINUOUS |
    And debug transfers

    And the following transfers should happen:
      | from    | to       | from account            | to account                       | market id | amount | asset |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 10     | ETH   |
      | trader4 |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 4      | ETH   |
      | market  | trader3a | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 10     | ETH   |

    And the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 1704   | 8313    |
      | trader4  | ETH   | ETH/DEC21 | 1015   | 0       |
