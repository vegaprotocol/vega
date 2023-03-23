Feature: Fees calculations

  Background:
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: S001, Testing fees in continuous trading with one trade and no liquidity providers (0029-FEES-001)

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
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount    |
      | aux1    | ETH   | 100000000 |
      | aux2    | ETH   | 100000000 |
      | trader3 | ETH   | 10000     |
      | trader4 | ETH   | 10000     |
      | lpprov  | ETH   | 100000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC21 | 90000000          | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | lpprov | ETH/DEC21 | 90000000          | 0.1 | sell | ASK              | 50         | 10     | submission |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1000       | TRADING_MODE_CONTINUOUS |
    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3 | ETH/DEC21 | buy  | 3      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    # margin_maitenance_trader3 = 3*1000*0.2=600
    # margin_initial_trader3 = 600*1.2=720

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3 | ETH   | ETH/DEC21 | 720    | 9280    |

    And the accumulated infrastructure fees should be "0" for the asset "ETH"
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    Then the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader4 | ETH/DEC21 | sell | 4      | 1002  | 1                | TYPE_LIMIT | TIF_GTC |

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1002       | TRADING_MODE_CONTINUOUS |

    Then the following trades should be executed:
      | buyer   | price | size | seller  | aggressor side |
      | trader3 | 1002  | 3    | trader4 | sell           |

    Then the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | buy  | 890   | 0      |
      | buy  | 900   | 1      |
      | sell | 1100  | 1      |
      | sell | 1110  | 0      |

    # trade_value_for_fee_purposes = size_of_trade * price_of_trade = 3 *1002 = 3006
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.002 * 3006 = 6.012 = 7 (rounded up to nearest whole value)
    # maker_fee =  fee_factor[maker]  * trade_value_for_fee_purposes = 0.005 * 3006 = 15.030 = 16 (rounded up to nearest whole value)
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0 * 3006 = 0

    And the following transfers should happen:
      | from    | to      | from account            | to account                       | market id | amount | asset |
      | trader4 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 16     | ETH   |
      | trader4 |         | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 7      | ETH   |
      | trader4 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 0      | ETH   |
      | market  | trader3 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 16     | ETH   |

    # total_fee = infrastructure_fee + maker_fee + liquidity_fee = 7 + 16 + 0 = 23
    # Trader3 margin + general account balance = 10000 + 16 ( Maker fees) = 10016
    # Trader4 margin + general account balance = 10000 - 16 ( Maker fees) - 7 (Infra fee) = 99977
    # margin_maitenance_trader3 = 1002*(3*1e0+0)+3*0.2*1002=3608
    # margin_initial_trader3 = 3608*1.2=4329

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3 | ETH   | ETH/DEC21 | 4329   | 5687    |
      | trader4 | ETH   | ETH/DEC21 | 4088   | 5889    |

    And the accumulated infrastructure fees should be "7" for the asset "ETH"
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

  Scenario: S002, Testing fees in continuous trading with two trades and no liquidity providers (0029-FEES-001, 0029-FEES-003, 0029-FEES-006)

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |

    When the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | aux1     | ETH   | 100000000 |
      | aux2     | ETH   | 100000000 |
      | trader3a | ETH   | 10000     |
      | trader3b | ETH   | 10000     |
      | trader4  | ETH   | 10000     |
      | lpprov   | ETH   | 100000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC21 | 90000000          | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | lpprov | ETH/DEC21 | 90000000          | 0.1 | sell | ASK              | 50         | 10     | submission |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1000       | TRADING_MODE_CONTINUOUS |
    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21 | buy  | 2      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3b | ETH/DEC21 | buy  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 480    | 9520    |
      | trader3b | ETH   | ETH/DEC21 | 240    | 9760    |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"
    And the accumulated infrastructure fees should be "0" for the asset "ETH"

    Then the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader4 | ETH/DEC21 | sell | 4      | 1002  | 2                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1002       | TRADING_MODE_CONTINUOUS |

    Then the following trades should be executed:
      | buyer    | price | size | seller  | aggressor side |
      | trader3a | 1002  | 2    | trader4 | sell           |
      | trader3b | 1002  | 1    | trader4 | sell           |

    # For trader3a-
    # trade_value_for_fee_purposes for trader3a = size_of_trade * price_of_trade = 2 * 1002 = 2004
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.002 * 2004 = 4.008 = 5 (rounded up to nearest whole value)
    # maker_fee =  fee_factor[maker]  * trade_value_for_fee_purposes = 0.005 * 2004 = 10.02 = 11 (rounded up to nearest whole value)
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0 * 3006 = 0

    # For trader3b -
    # trade_value_for_fee_purposes = size_of_trade * price_of_trade = 1 * 1002 = 1002
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.002 * 1002 = 2.004 = 3 (rounded up to nearest whole value)
    # maker_fee =  fee_factor[maker]  * trade_value_for_fee_purposes = 0.005 * 1002 = 5.01 = 6 (rounded up to nearest whole value)
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0 * 3006 = 0

    And the following transfers should happen:
      | from    | to       | from account            | to account                       | market id | amount | asset |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 11     | ETH   |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 6      | ETH   |
      | trader4 |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 8      | ETH   |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 0      | ETH   |
      | market  | trader3a | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 11     | ETH   |
      | market  | trader3b | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 6      | ETH   |

    # total_fee = infrastructure_fee + maker_fee + liquidity_fee = 8 + 11 + 6 + 0 = 25 ??
    # Trader3a margin + general account balance = 10000 + 11 ( Maker fees) = 10011
    # Trader3b margin + general account balance = 10000 + 6 ( Maker fees) = 10006
    # Trader4  margin + general account balance = 10000 - (11+6) ( Maker fees) - 8 (Infra fee) = 99975

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 2886   | 7125    |
      | trader3b | ETH   | ETH/DEC21 | 363    | 9643    |
      | trader4  | ETH   | ETH/DEC21 | 4088   | 5887    |

    And the accumulated infrastructure fees should be "8" for the asset "ETH"
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

  @WhutBug
  Scenario: S003, Testing fees in continuous trading with two trades and one liquidity providers with 10 and 0 s liquidity fee distribution timestep (0029-FEES-006)

    When the following network parameters are set:
      | name                                                | value |
      | market.liquidity.providers.fee.distributionTimeStep | 10s   |
    And the average block duration is "1"

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |

    When the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | aux1     | ETH   | 100000000 |
      | aux2     | ETH   | 100000000 |
      | trader3a | ETH   | 10000     |
      | trader3b | ETH   | 10000     |
      | trader4  | ETH   | 10000     |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC21 | buy  | 1      | 920   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1      | 1080  | 0                | TYPE_LIMIT | TIF_GTC |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | 10     | submission |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          | 10     | amendment  |

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

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21 | buy  | 2      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3b | ETH/DEC21 | buy  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell | 4      | 1002  | 2                | TYPE_LIMIT | TIF_GTC |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 690    | 9321    |
      | trader3b | ETH   | ETH/DEC21 | 339    | 9667    |

    And the liquidity fee factor should be "0.001" for the market "ETH/DEC21"
    And the accumulated liquidity fees should be "5" for the market "ETH/DEC21"

    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1002       | TRADING_MODE_CONTINUOUS |

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

    Then the following trades should be executed:
      | buyer    | price | size | seller  | aggressor side | buyer fee | seller fee | infrastructure fee | maker fee | liquidity fee |
      | trader3a | 1002  | 2    | trader4 | sell           | 0         | 19         | 5                  | 11        | 3             |
      | trader3b | 1002  | 1    | trader4 | sell           | 0         | 11         | 3                  | 6         | 2             |

    And the following transfers should happen:
      | from    | to       | from account            | to account                       | market id | amount | asset |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 11     | ETH   |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 6      | ETH   |
      | trader4 |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 8      | ETH   |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 5      | ETH   |
      | market  | trader3a | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 11     | ETH   |
      | market  | trader3b | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 6      | ETH   |

    # total_fee = infrastructure_fee + maker_fee + liquidity_fee = 8 + 11 + 6 + 0 = 25
    # Trader3a margin + general account balance = 10000 + 11 ( Maker fees) = 10011
    # Trader3b margin + general account balance = 10000 + 6 ( Maker fees) = 10006
    # Trader4  margin + general account balance = 10000 - (11+6) ( Maker fees) - 8 (Infra fee) = 99975

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 690    | 9321    |
      | trader3b | ETH   | ETH/DEC21 | 339    | 9667    |
      | trader4  | ETH   | ETH/DEC21 | 480    | 9490    |
    #| trader3a | ETH   | ETH/DEC21 | 480    | 9531    |
    #| trader3b | ETH   | ETH/DEC21 | 240    | 9766    |
    #| trader4  | ETH   | ETH/DEC21 | 679    | 9291    |

    And the accumulated infrastructure fees should be "8" for the asset "ETH"
    And the accumulated liquidity fees should be "5" for the market "ETH/DEC21"

    When the network moves ahead "11" blocks

    And the following transfers should happen:
      | from   | to   | from account                | to account           | market id | amount | asset |
      | market | aux1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 5      | ETH   |

    # Scenario: S004, Testing fees in continuous trading with two trades and one liquidity providers with 0s liquidity fee distribution timestep (0029-FEES-004)
    When the following network parameters are set:
      | name                                                | value |
      | market.liquidity.providers.fee.distributionTimeStep | 1s    |

    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader4 | ETH/DEC21 | sell | 2      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    And the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21 | buy  | 1      | 1002  | 1                | TYPE_LIMIT | TIF_GTC |

    # For trader4 -
    # trade_value_for_fee_purposes = size_of_trade * price_of_trade = 1 * 1002 = 1002
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.002 * 1002 = 2.004 = 3 (rounded up to nearest whole value)
    # maker_fee =  fee_factor[maker]  * trade_value_for_fee_purposes = 0.005 * 1002 = 5.01 = 6 (rounded up to nearest whole value)
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0.001 * 1002 = 1.002 = 2 (rounded up to nearest whole value)

    Then the following transfers should happen:
      | from     | to      | from account            | to account                       | market id | amount | asset |
      | trader3a | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 6      | ETH   |
      | trader3a |         | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 3      | ETH   |
      | trader3a | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 2      | ETH   |
      | market   | trader4 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 6      | ETH   |

    And the accumulated liquidity fees should be "2" for the market "ETH/DEC21"
    When the network moves ahead "11" blocks

    And the following transfers should happen:
      | from   | to   | from account                | to account           | market id | amount | asset |
      | market | aux1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 2      | ETH   |

  @WhutMargin
  Scenario: S005, Testing fees get collected when amended order trades (0029-FEES-005)

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |

    When the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | aux1     | ETH   | 100000000 |
      | aux2     | ETH   | 100000000 |
      | trader3a | ETH   | 10000     |
      | trader3b | ETH   | 10000     |
      | trader4  | ETH   | 5000      |
      | lpprov   | ETH   | 100000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC21 | 90000000          | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | lpprov | ETH/DEC21 | 90000000          | 0.1 | sell | ASK              | 50         | 10     | submission |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC21 | buy  | 1      | 920   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1      | 1080  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
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
      
    Then the following trades should be executed:
      # | buyer   | price | size | seller  | maker   | taker   |
      # | trader3 | 1002  | 3    | trader4 | trader3 | trader4 |
      # TODO to be implemented by Core Team
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

    # margin_maitenance_trader3a = 1002*(2*1e0+0)+2*0.2*1002=2405
    # margin_initial_trader3a = 2405*1.2=2886
    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 2886   | 7125    |
      | trader3b | ETH   | ETH/DEC21 | 339    | 9667    |
      | trader4  | ETH   | ETH/DEC21 | 4088   | 887     |

    # Placing second set of orders
    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | trader3a | ETH/DEC21 | buy  | 2      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | trader3a-buy-1 |
      | trader4  | ETH/DEC21 | sell | 4      | 1002  | 0                | TYPE_LIMIT | TIF_GTC | trader4-sell-2 |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 3367   | 6644    |
      | trader4  | ETH   | ETH/DEC21 | 4569   | 406     |

    # reducing size
    And the parties amend the following orders:
      | party   | reference      | price | size delta | tif     |
      | trader4 | trader4-sell-2 | 1000  | 0          | TIF_GTC |

    # matching the order now
    Then the following trades should be executed:
      # | buyer   | price | size | seller  | maker   | taker   |
      # | trader3 | 1002  | 3    | trader4 | trader3 | trader4 |
      # TODO to be implemented by Core Team
      | buyer    | price | size | seller  |
      | trader3a | 1000  | 2    | trader4 |

    # checking if continuous mode still exists
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | last traded price | trading mode            |
      | 1002       | 1000              | TRADING_MODE_CONTINUOUS |

    And the following transfers should happen:
      | from    | to       | from account            | to account                       | market id | amount | asset |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 10     | ETH   |
      | trader4 |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 4      | ETH   |
      | market  | trader3a | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 10     | ETH   |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 3367   | 6654    |
      | trader4  | ETH   | ETH/DEC21 | 4569   | 392     |

  @WhutMargin
  Scenario: S006, Testing fees in continuous trading with insufficient balance in their general account but margin covers the fees (0029-FEES-008)

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |

    When the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount    |
      | aux1    | ETH   | 100000000 |
      | aux2    | ETH   | 100000000 |
      | trader3 | ETH   | 10000000  |
      | trader4 | ETH   | 30000     |
      | lpprov  | ETH   | 100000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC21 | 90000000          | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | lpprov | ETH/DEC21 | 90000000          | 0.1 | sell | ASK              | 50         | 10     | submission |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC21 | buy  | 300    | 920   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 300    | 1080  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | target stake |supplied stake |
      | 1000       | TRADING_MODE_CONTINUOUS |   2000       | 0             |

    Then the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | buy  | 910   | 0      |
      | buy  | 920   | 300    |
      | sell | 1080  | 300    |
      | sell | 1090  | 0      |

    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3 | ETH/DEC21 | buy  | 100    | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4 | ETH/DEC21 | sell | 100    | 1002  | 1                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1002       | TRADING_MODE_CONTINUOUS |

    Then the following trades should be executed:
      | buyer   | price | size | seller  |
      | trader3 | 1002  | 100  | trader4 |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3 | ETH   | ETH/DEC21 | 33888  | 9966613 |
      | trader4 | ETH   | ETH/DEC21 | 21384  | 7914    |

    Then the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader4 | ETH/DEC21 | 17820       | 19602  | 21384   | 24948   |

    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | trader3 | ETH/DEC21 | buy  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC | trader3-buy-1  |
      | trader4 | ETH/DEC21 | sell | 1      | 1002  | 1                | TYPE_LIMIT | TIF_GTC | trader4-sell-2 |

    # BUG: the following transfers should happen but did not, raise a bug 
    # And the following transfers should happen:
    #   | from    | to      | from account            | to account                       | market id | amount | asset |
    #   | trader4 | market  | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 6      | ETH   |
    #   | trader4 |         | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | ETH/DEC21 | 3      | ETH   |
    #   | market  | trader3 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 6      | ETH   |

    Then the market data for the market "ETH/DEC21" should be:
      | mark price | last traded price | trading mode            |
      | 1002       | 1002              | TRADING_MODE_CONTINUOUS |
    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader4 | ETH   | ETH/DEC21 | 21505  | 7784    |

      Then the following trades should be executed:
      | buyer    | price | size | seller  |
      | trader3  | 1002  | 1    | trader4 |
    
    # For trader4 -
    # trade_value_for_fee_purposes = size_of_trade * price_of_trade = 1 * 1002 = 1002
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.002 * 1002 = 2.004 = 3 (rounded up to nearest whole value)
    # maker_fee =  fee_factor[maker]  * trade_value_for_fee_purposes = 0.005 * 1002 = 5.01 = 6 (rounded up to nearest whole value)
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0.001 * 1002 = 1.002 = 2 (rounded up to nearest whole value)

    # And the following transfers should happen:
    #   | from    | to      | from account            | to account                       | market id | amount | asset |
    #   | trader4 | market  | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 6      | ETH   |
    #   | trader4 |         | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | ETH/DEC21 | 3      | ETH   |
    #   | market  | trader3 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 6      | ETH   |
      
    Then the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader4 | ETH/DEC21 | 17999       | 19798  | 21598   | 25198   |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3 | ETH   | ETH/DEC21 | 34129  | 9966378 |
      | trader4 | ETH   | ETH/DEC21 | 21505  | 7784    |

  Scenario: S007, Testing fees to confirm fees are collected first and then margin (0029-FEES-002, 0029-FEES-008)

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |

    When the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount    |
      | aux1    | ETH   | 100000000 |
      | aux2    | ETH   | 100000000 |
      | trader3 | ETH   | 10000000  |
      | trader4 | ETH   | 214       |
      | lpprov  | ETH   | 100000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC21 | 90000000          | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | lpprov | ETH/DEC21 | 90000000          | 0.1 | sell | ASK              | 50         | 10     | submission |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC21 | buy  | 1      | 920   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1      | 1080  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1000       | TRADING_MODE_CONTINUOUS |

    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | trader3 | ETH/DEC21 | buy  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC | trader3-buy-1  |
      | trader4 | ETH/DEC21 | sell | 1      | 1002  | 1                | TYPE_LIMIT | TIF_GTC | trader4-sell-2 |

    And the following transfers should happen:
      | from    | to      | from account            | to account                       | market id | amount | asset |
      | trader4 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 6      | ETH   |
      | trader4 |         | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | ETH/DEC21 | 3      | ETH   |
      | market  | trader3 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 6      | ETH   |

    Then the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader4 | ETH/DEC21 | 179         | 196    | 214     | 250     |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3 | ETH   | ETH/DEC21 | 339    | 9999667 |
      | trader4 | ETH   | ETH/DEC21 | 205    | 0       |

  Scenario: S008, Testing fees in continuous trading when insufficient balance in their general and margin account with LP, then the trade does not execute (0029-FEES-007，0029-FEES-008)

    Given the following network parameters are set:
      | name                                                | value |
      | market.liquidity.providers.fee.distributionTimeStep | 10s   |
    And the average block duration is "1"

    When the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |

    When the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount    |
      | aux1    | ETH   | 100000000 |
      | aux2    | ETH   | 100000000 |
      | trader3 | ETH   | 10000000  |
      | trader4 | ETH   | 189       |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC21 | buy  | 10     | 920   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 10     | 1080  | 0                | TYPE_LIMIT | TIF_GTC |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | 10     | submission |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          | 10     | amendment  |

    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1000       | TRADING_MODE_CONTINUOUS |

    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | buy  | 910   | 1      |
      | buy  | 920   | 10     |
      | sell | 1080  | 10     |
      | sell | 1090  | 10     |

    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | trader3 | ETH/DEC21 | buy  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC | trader3-buy-1  |
      | trader4 | ETH/DEC21 | sell | 1      | 1002  | 1                | TYPE_LIMIT | TIF_GTC | trader4-sell-2 |

    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1002       | TRADING_MODE_CONTINUOUS |

    Then the following trades should be executed:
      | buyer   | price | size | seller  |
      | trader3 | 1002  | 1    | trader4 |

    And the following transfers should happen:
      | from    | to      | from account            | to account                       | market id | amount | asset |
      | trader4 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 6      | ETH   |
      | trader4 |         | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | ETH/DEC21 | 3      | ETH   |
      | trader4 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 2      | ETH   |
      | trader4 | market  | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 2      | ETH   |
      | market  | trader3 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 6      | ETH   |

    Then the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader4 | ETH/DEC21 | 0           | 0      | 0       | 0       |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3 | ETH   | ETH/DEC21 | 339    | 9999667 |
      #| trader3 | ETH   | ETH/DEC21 | 240    | 9999766 |
      | trader4 | ETH   | ETH/DEC21 | 0      | 0       |

    And the liquidity fee factor should be "0.001" for the market "ETH/DEC21"
    And the accumulated liquidity fees should be "4" for the market "ETH/DEC21"

    When the network moves ahead "11" blocks

    And the following transfers should happen:
      | from   | to   | from account                | to account           | market id | amount | asset |
      | market | aux1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 4      | ETH   |

  Scenario:S009, Testing fees in auctions session with each side of a trade debited 1/2 IF & LP (0029-FEES-006, 0029-FEES-008)

    Given the following network parameters are set:
      | name                                                | value |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 1     |
      | market.liquidity.providers.fee.distributionTimeStep | 10s   |

    And the average block duration is "1"

    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |

    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r   | sigma |
      | 0.000001      | 0.1 | 0  | 1.4 | -1    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e0                    | 0                         |

    # setup accounts
    When the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | aux1     | ETH   | 100000000 |
      | aux2     | ETH   | 100000000 |
      | trader3a | ETH   | 10000     |
      | trader4  | ETH   | 10000     |

    Then the parties place the following orders:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/DEC21 | buy  | 1      | 500   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2     | ETH/DEC21 | sell | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3a | ETH/DEC21 | buy  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | buy  | BID              | 1          | 10     | submission |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | sell | ASK              | 1          | 10     | amendment  |

    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | target stake | supplied stake |
      | 1002       | TRADING_MODE_CONTINUOUS | 200          | 200            |
    Then the following trades should be executed:
      | buyer    | price | size | seller  |
      | trader3a | 1002  | 1    | trader4 |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 843    | 9157    |
      | trader4  | ETH   | ETH/DEC21 | 1318   | 8682    |

    #Scenario:S010, Triggering Liquidity auction (0029-FEES-006)

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21 | buy  | 3      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell | 3      | 1002  | 1                | TYPE_LIMIT | TIF_GTC |

    Then the following trades should be executed:
      | buyer    | price | size | seller  |
      | trader3a | 1002  | 3    | trader4 |

    # fees during normal trading
    And the following transfers should happen:
      | from    | to     | from account         | to account                       | market id | amount | asset |
      | trader4 |        | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 7      | ETH   |
      | trader4 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 4      | ETH   |

    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger                          |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET |

    # now place orders during auction
    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21 | buy  | 3      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell | 3      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | 10     | amendment |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          | 10     | amendment |

    # leave auction
    When the network moves ahead "2" blocks
    Then the following trades should be executed:
      | buyer    | price | size | seller  |
      | trader3a | 1002  | 3    | trader4 |

    # For trader3a & 4- Sharing IF and LP
    # trade_value_for_fee_purposes for trader3a = size_of_trade * price_of_trade = 3 * 1002= 3006
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.002 * 3006 = 6.012 = 7(rounded up)
    # maker_fee =  0 in auction
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0.001 * 3006 = 3.006 = 4 (rounded up)
    And the following transfers should happen:
      | from     | to     | from account         | to account                       | market id | amount | asset |
      | trader4  |        | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 4      | ETH   |
      | trader4  | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 2      | ETH   |
      | trader3a |        | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 4      | ETH   |
      | trader3a | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 2      | ETH   |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 6493   | 3517    |
      | trader4  | ETH   | ETH/DEC21 | 9259   | 708     |

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1002       | TRADING_MODE_CONTINUOUS | 1       | 903       | 1101      | 1402         | 10000          | 7             |

    Then the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger       |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

    Then the network moves ahead "301" blocks

    Then the following trades should be executed:
      | buyer    | price | size | seller  |
      | trader3a | 900   | 1    | trader4 |

    # For trader3a & 4- Sharing IF and LP
    # trade_value_for_fee_purposes for trader3a = size_of_trade * price_of_trade = 1 * 900 = 900
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.002 * 900 = 1.800 = 2(rounded up)
    # maker_fee =  0 in auction
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0.001 * 900 = 0.900 = 1 (rounded up)

    And the following transfers should happen:
      | from     | to     | from account         | to account                       | market id | amount | asset |
      | trader4  |        | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 1      | ETH   |
      | trader4  | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 1      | ETH   |
      | trader3a |        | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 1      | ETH   |
      | trader3a | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 1      | ETH   |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader4  | ETH   | ETH/DEC21 | 10093  | 586     |
      | trader3a | ETH   | ETH/DEC21 | 5779   | 3515    |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |

  Scenario: S011, Testing fees in Liquidity auction session trading with insufficient balance in their general account but margin covers the fees (0029-FEES-006)

    Given the following network parameters are set:
      | name                                                | value |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 1     |
      | market.liquidity.providers.fee.distributionTimeStep | 10s   |

    And the average block duration is "1"

    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |

    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r   | sigma |
      | 0.000001      | 0.1 | 0  | 1.4 | -1    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e0                    | 0                         |

    # setup accounts
    When the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | aux1     | ETH   | 100000000 |
      | aux2     | ETH   | 100000000 |
      | trader3a | ETH   | 5000      |
      | trader4  | ETH   | 5261      |
      | tradera3 | ETH   | 100000000 |
      | tradera4 | ETH   | 100000000 |

    Then the parties place the following orders:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/DEC21 | buy  | 1      | 500   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2     | ETH/DEC21 | sell | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3a | ETH/DEC21 | buy  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | buy  | BID              | 1          | 10     | submission |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | sell | ASK              | 1          | 10     | amendment  |

    And the opening auction period ends for market "ETH/DEC21"

    # Scenario: S012, Triggering Liquidity auction (0029-FEES-006)

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | tradera3 | ETH/DEC21 | buy  | 3      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | tradera4 | ETH/DEC21 | sell | 3      | 1002  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer    | price | size | seller   |
      | tradera3 | 1002  | 3    | tradera4 |

    When the network moves ahead "1" blocks

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger                          |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET |

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21 | buy  | 3      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell | 3      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | 10     | amendment |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          | 10     | amendment |

    Then the network moves ahead "2" blocks


    # For trader3a & 4- Sharing IF and LP
    # trade_value_for_fee_purposes for trader3a = size_of_trade * price_of_trade = 3 * 1002= 3006
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.002 * 3006 = 6.012 = 7(rounded up)
    # maker_fee =  0 in auction
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0.001 * 3006 = 3.006 = 4 (rounded up)

    And the following transfers should happen:
      | from     | to     | from account         | to account                       | market id | amount | asset |
      | trader4  |        | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 4      | ETH   |
      | trader4  | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 2      | ETH   |
      | trader3a |        | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 4      | ETH   |
      | trader3a | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 2      | ETH   |

    Then the parties should have the following margin levels:
      | party   | market id | maintenance | initial |
      | trader4 | ETH/DEC21 | 4409        | 5290    |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 3410   | 1584    |
      | trader4  | ETH   | ETH/DEC21 | 5255   | 0       |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |

  Scenario: S013, Testing fees in Price auction session trading with insufficient balance in their general account but margin covers the fees (0029-FEES-008)

    Given the following network parameters are set:
      | name                                                | value |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 1     |
      | market.liquidity.providers.fee.distributionTimeStep | 10s   |

    And the average block duration is "1"

    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |

    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r   | sigma |
      | 0.000001      | 0.1 | 0  | 1.4 | -1    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e0                    | 0                         |

    # setup accounts
    When the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | aux1     | ETH   | 100000000 |
      | aux2     | ETH   | 100000000 |
      | trader3a | ETH   | 5000      |
      | trader4  | ETH   | 2656      |

    Then the parties place the following orders:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/DEC21 | buy  | 1      | 500   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2     | ETH/DEC21 | sell | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3a | ETH/DEC21 | buy  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | buy  | BID              | 1          | 10     | submission |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | sell | ASK              | 1          | 10     | amendment  |

    Then the opening auction period ends for market "ETH/DEC21"

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | 10     | amendment |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          | 10     | amendment |

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1002       | TRADING_MODE_CONTINUOUS | 1       | 903       | 1101      | 200          | 10000          | 1             |

    Then the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger       |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

    Then the network moves ahead "301" blocks

    Then the following trades should be executed:
      | buyer    | price | size | seller  |
      | trader3a | 900   | 1    | trader4 |

    # For trader3a & 4- Sharing IF and LP
    # trade_value_for_fee_purposes for trader3a = size_of_trade * price_of_trade = 1 * 900 = 900
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.002 * 900 = 1.800 = 2(rounded up)
    # maker_fee =  0 in auction
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0.001 * 900 = 0.900 = 1 (rounded up)

    And the following transfers should happen:
      | from     | to     | from account         | to account                       | market id | amount | asset |
      | trader4  |        | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 1      | ETH   |
      | trader4  | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 1      | ETH   |
      | trader3a |        | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 1      | ETH   |
      | trader3a | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 1      | ETH   |

    Then the parties should have the following margin levels:
      | party    | market id | maintenance | initial |
      | trader3a | ETH/DEC21 | 1170        | 1404    |
      | trader4  | ETH/DEC21 | 1980        | 2376    |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 1404   | 3492    |
      | trader4  | ETH   | ETH/DEC21 | 2376   | 380     |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |

  Scenario: S014, Testing fees in Liquidity auction session trading with insufficient balance in their general and margin account, then the trade still goes ahead, (0029-FEES-008)

    Given the following network parameters are set:
      | name                                                | value |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 1     |
      | market.liquidity.providers.fee.distributionTimeStep | 10s   |

    And the average block duration is "1"

    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 2                  |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |

    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r   | sigma |
      | 0.000001      | 0.1 | 0  | 1.4 | -1    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e0                    | 0                         |

    # setup accounts
    When the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | aux1     | ETH   | 100000000 |
      | aux2     | ETH   | 100000000 |
      | trader3a | ETH   | 5173      |
      | trader4  | ETH   | 5432      |
      | trader5  | ETH   | 100000000 |
      | trader6  | ETH   | 100000000 |

    Then the parties place the following orders:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/DEC21 | buy  | 1      | 500   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2     | ETH/DEC21 | sell | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3a | ETH/DEC21 | buy  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | buy  | BID              | 1          | 10     | submission |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | sell | ASK              | 1          | 10     | amendment  |

    When the opening auction period ends for market "ETH/DEC21"

    #Scenario: S015, Triggering Liquidity auction (0029-FEES-008)

    Then the parties place the following orders:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | trader5 | ETH/DEC21 | buy  | 3      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader6 | ETH/DEC21 | sell | 3      | 1002  | 1                | TYPE_LIMIT | TIF_GTC |
    And the following trades should be executed:
      | buyer   | price | size | seller   |
      | trader5 | 1002  | 3    | trader6 |

    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger                          |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | aux1  | ETH/DEC21 | 20000             | 0.001 | buy  | BID              | 1          | 10     | amendment |
      | lp1 | aux1  | ETH/DEC21 | 20000             | 0.001 | sell | ASK              | 1          | 10     | amendment |
    And the parties place the following orders:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21 | buy  | 3      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell | 3      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "2" blocks
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |

    And the following trades should be executed:
      | buyer    | price | size | seller  |
      | trader3a | 1002  | 3    | trader4 |

    # For trader3a & 4- Sharing IF and LP
    # trade_value_for_fee_purposes for trader3a = size_of_trade * price_of_trade = 3 * 1002= 3006
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 2 * 3006
    # maker_fee =  0 in auction
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0.001 * 3006 = 3.006 = 4 (rounded up)

    And the following transfers should happen:
      | from     | to     | from account         | to account                       | market id | amount | asset |
      | trader4  |        | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 3006   | ETH   |
      | trader4  | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 2      | ETH   |
      | trader3a |        | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 3006   | ETH   |
      | trader3a | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 2      | ETH   |

    Then the parties should have the following margin levels:
      | party   | market id | maintenance | initial |
      | trader4 | ETH/DEC21 | 4409        | 5290    |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 0      | 0       |
      | trader4  | ETH   | ETH/DEC21 | 0      | 0       |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |

  Scenario:S016,  Testing fees in Price auction session trading with insufficient balance in their general and margin account, then the trade still goes ahead (0029-FEES-008)

    Given the following network parameters are set:
      | name                                                | value |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 1     |
      | market.liquidity.providers.fee.distributionTimeStep | 10s   |

    And the average block duration is "1"

    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 2                  |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |

    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r   | sigma |
      | 0.000001      | 0.1 | 0  | 1.4 | -1    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e0                    | 0                         |

    # setup accounts
    When the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | aux1     | ETH   | 100000000 |
      | aux2     | ETH   | 100000000 |
      | trader3a | ETH   | 3500      |
      | trader4  | ETH   | 5500      |

    Then the parties place the following orders:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/DEC21 | buy  | 1      | 500   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2     | ETH/DEC21 | sell | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3a | ETH/DEC21 | buy  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | buy  | BID              | 1          | 10     | submission |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | sell | ASK              | 1          | 10     | amendment  |

    Then the opening auction period ends for market "ETH/DEC21"

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | 10     | amendment |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          | 10     | amendment |

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1002       | TRADING_MODE_CONTINUOUS | 1       | 903       | 1101      | 200          | 10000          | 1             |

    Then the parties place the following orders:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21 | buy  | 2      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell | 2      | 900   | 0                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger       |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

    Then the network moves ahead "301" blocks

    Then the following trades should be executed:
      | buyer    | price | size | seller   |
      | trader3a | 1002  | 1    | trader4  |
      | trader3a | 900   | 2    | trader4  |
      | aux1     | 500   | 1    | network  |
      | aux1     | 490   | 2    | network  |
      | network  | 493   | 3    | trader3a |

    # For trader3a & 4- Sharing IF and LP
    # trade_value_for_fee_purposes for trader3a = size_of_trade * price_of_trade = 2 * 900 = 1800
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 2 * 900
    # maker_fee =  0 in auction
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0.001 * 1800 = 1.8 = 2/2 = 1

    And the following transfers should happen:
      | from     | to     | from account         | to account                       | market id | amount | asset |
      | trader4  |        | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 1800   | ETH   |
      | trader4  | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 1      | ETH   |
      | trader3a |        | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 1800   | ETH   |
      | trader3a | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 1      | ETH   |

    Then the parties should have the following margin levels:
      | party   | market id | maintenance | search | initial | release |
      | trader4 | ETH/DEC21 | 2970        | 3267   | 3564    | 4158    |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 |    0   | 0       |
      | trader4  | ETH   | ETH/DEC21 | 3564   | 237     |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |

  Scenario:S017, Testing fees in Price auction session trading with insufficient balance in their general and margin account, then the trade does not go ahead (0029-FEES-008)

    Given the following network parameters are set:
      | name                                                | value |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 1     |
      | market.liquidity.providers.fee.distributionTimeStep | 10s   |

    And the average block duration is "1"

    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 2                  |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |

    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r   | sigma |
      | 0.000001      | 0.1 | 0  | 1.4 | -1    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e0                    | 0                         |

    # setup accounts
    When the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | aux1     | ETH   | 100000000 |
      | aux2     | ETH   | 100000000 |
      | trader3a | ETH   | 5000      |
      | trader4  | ETH   | 7261      |
    # If the trader4 balance is changed to from 7261 to 7465 then the trade goes ahead as the account balance goes above maintenance level after paying fees.
    # | trader4  | ETH   | 7465       |

    Then the parties place the following orders:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/DEC21 | buy  | 1      | 500   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2     | ETH/DEC21 | sell | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3a | ETH/DEC21 | buy  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | buy  | BID              | 1          | 10     | submission |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | sell | ASK              | 1          | 10     | amendment  |

    Then the opening auction period ends for market "ETH/DEC21"

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | 10     | amendment |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          | 10     | amendment |

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1002       | TRADING_MODE_CONTINUOUS | 1       | 903       | 1101      | 200          | 10000          | 1             |

    Then the parties place the following orders:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21 | buy  | 3      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell | 3      | 900   | 0                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger       |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

    Then the network moves ahead "301" blocks

    Then the following trades should be executed:
      | buyer    | price | size | seller  |
      | trader3a | 900   | 3    | trader4 |

    # For trader3a & 4- Sharing IF and LP
    # trade_value_for_fee_purposes for trader3a = size_of_trade * price_of_trade = 3 * 900 = 2700
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 2 * 2700
    # maker_fee =  0 in auction
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0.001 * 2700 = 2.7 = 3/2 = 1.5 = 2 (rounded up)

    And the following transfers should happen:
      | from     | to     | from account         | to account                       | market id | amount | asset |
      | trader4  |        | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 2700   | ETH   |
      | trader4  | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 2      | ETH   |
      | trader3a |        | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 2700   | ETH   |
      | trader3a | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 2      | ETH   |

    Then the parties should have the following margin levels:
      | party   | market id | maintenance | initial |
      | trader4 | ETH/DEC21 | 3960        | 4752    |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 0      | 0       |
      | trader4  | ETH   | ETH/DEC21 | 4661   | 0       |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |

  @now
  Scenario: S018, Testing fees in continuous trading during position resolution (0029-FEES-001)

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator                  | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | default-simple-risk-model-2 | default-overkill-margin-calculator | 2                | fees-config-1 | default-none     | default-eth-for-future | 1e0                    | 0                         |

    And the parties deposit on asset's general account the following amount:
      | party    | asset | amount        |
      | aux1     | ETH   | 1000000000000 |
      | aux2     | ETH   | 1000000000000 |
      | trader3a | ETH   | 10000         |
      | trader3b | ETH   | 30000         |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1   |
      | aux2  | ETH/DEC21 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1   |
      | aux1  | ETH/DEC21 | sell | 10     | 180   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-2   |
      | aux2  | ETH/DEC21 | buy  | 10     | 180   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-2   |

    Then the opening auction period ends for market "ETH/DEC21"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the mark price should be "180" for the market "ETH/DEC21"

    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | aux1  | ETH/DEC21 | sell | 150    | 200   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | aux2  | ETH/DEC21 | buy  | 50     | 190   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | aux2  | ETH/DEC21 | buy  | 350    | 180   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2  |

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader3a | ETH/DEC21 | sell | 100    | 180   | 2                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader3b | ETH/DEC21 | sell | 300    | 180   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    Then the following trades should be executed:
      | buyer | price | size | seller   |
      | aux2  | 190   | 50   | trader3a |
      | aux2  | 180   | 50   | trader3a |
      | aux2  | 180   | 300  | trader3b |

    Then the parties should have the following margin levels:
      | party    | market id | maintenance | search | initial | release |
      | trader3a | ETH/DEC21 | 2000        | 6400   | 8000    | 10000   |
      | trader3b | ETH/DEC21 | 54000       | 172800 | 216000  | 270000  |

    Then the parties cancel the following orders:
      | party | reference       |
      | aux1  | sell-provider-1 |

    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | aux1  | ETH/DEC21 | sell | 500    | 350   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-2 |

    And the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC21 | sell | 1      | 300   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2  | ETH/DEC21 | buy  | 1      | 300   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "300" for the market "ETH/DEC21"

    Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl |
      | trader3a | 0      | 0              | -9870        |
      | trader3b | 0      | 0              | -29622       |

    # trade_value_for_fee_purposes for party 3a = size_of_trade * price_of_trade = 50 *190 = 9500 And 50 * 180 = 9000
    # maker_fee for party 3a = fee_factor[maker] * trade_value_for_fee_purposes = 0.005 * 9500 = 47.5 = 48 (rounded up to nearest whole value) And 0.005 * 9000 = 45
    # infrastructure_fee for party 3a = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.002 * 9500 = 19 And 0.002 * 9000 = 18 + 19 = 37
    # trade_value_for_fee_purposes for party 3b = size_of_trade * price_of_trade = 300 *180 = 54000
    # maker_fee for party 3b =  fee_factor[maker]  * trade_value_for_fee_purposes = 0.005 * 54000 = 270
    # infrastructure_fee for party 3b = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.002 * 54000 = 108
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0

    And the following transfers should happen:
      | from     | to     | from account            | to account                       | market id | amount | asset |
      | trader3a | market | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 48     | ETH   |
      | trader3a | market | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 45     | ETH   |
      | trader3a |        | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | ETH/DEC21 | 37     | ETH   |
      | trader3b | market | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 270    | ETH   |
      | trader3b |        | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | ETH/DEC21 | 108    | ETH   |
      | market   | aux2   | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 48     | ETH   |
      | market   | aux2   | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 45     | ETH   |
      | market   | aux2   | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 270    | ETH   |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 0      | 0       |
      | trader3b | ETH   | ETH/DEC21 | 0      | 0       |

    And the insurance pool balance should be "0" for the market "ETH/DEC21"

  Scenario: S019, Testing fees in continuous trading during position resolution with insufficient balance in their general and margin account, partial or full fees does not get paid (0029-FEES-008)

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.003              |

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator                  | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | default-simple-risk-model-2 | default-overkill-margin-calculator | 2                | fees-config-1 | default-none     | default-eth-for-future | 1e0                    | 0                         |

    And the parties deposit on asset's general account the following amount:
      | party    | asset | amount        |
      | aux1     | ETH   | 1000000000000 |
      | aux2     | ETH   | 1000000000000 |
      | trader3a | ETH   | 10000         |
      | trader3b | ETH   | 30000         |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1   |
      | aux2  | ETH/DEC21 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1   |
      | aux1  | ETH/DEC21 | sell | 10     | 180   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-2   |
      | aux2  | ETH/DEC21 | buy  | 10     | 180   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-2   |

    Then the opening auction period ends for market "ETH/DEC21"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the mark price should be "180" for the market "ETH/DEC21"

    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | aux1  | ETH/DEC21 | sell | 150    | 200   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | aux2  | ETH/DEC21 | buy  | 50     | 190   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | aux2  | ETH/DEC21 | buy  | 350    | 180   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2  |

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader3a | ETH/DEC21 | sell | 100    | 180   | 2                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader3b | ETH/DEC21 | sell | 300    | 180   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    Then the following trades should be executed:
      | buyer | price | size | seller   |
      | aux2  | 190   | 50   | trader3a |
      | aux2  | 180   | 50   | trader3a |
      | aux2  | 180   | 300  | trader3b |

    Then the parties should have the following margin levels:
      | party    | market id | maintenance | search | initial | release |
      | trader3a | ETH/DEC21 | 2000        | 6400   | 8000    | 10000   |
      | trader3b | ETH/DEC21 | 54000       | 172800 | 216000  | 270000  |

    Then the parties cancel the following orders:
      | party | reference       |
      | aux1  | sell-provider-1 |

    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | aux1  | ETH/DEC21 | sell | 500    | 350   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-2 |

    And the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC21 | sell | 1      | 300   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2  | ETH/DEC21 | buy  | 1      | 300   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "300" for the market "ETH/DEC21"

    Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl |
      | trader3a | 0      | 0              | -9851        |
      | trader3b | 0      | 0              | -29568       |

    # trade_value_for_fee_purposes for party 3a = size_of_trade * price_of_trade = 50 *190 = 9500 And 50 * 180 = 9000
    # maker_fee for party 3a = fee_factor[maker] * trade_value_for_fee_purposes = 0.005 * 9500 = 47.5 = 48 (rounded up to nearest whole value) And 0.005 * 9000 = 45
    # infrastructure_fee for party 3a = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.002 * 9500 = 19 And 0.002 * 9000 = 18 + 19 = 37
    # trade_value_for_fee_purposes for party 3b = size_of_trade * price_of_trade = 300 *180 = 54000
    # maker_fee for party 3b =  fee_factor[maker]  * trade_value_for_fee_purposes = 0.005 * 54000 = 270
    # infrastructure_fee for party 3b = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.002 * 54000 = 108
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0

    And the following transfers should happen:
      | from     | to     | from account            | to account                       | market id | amount | asset |
      | trader3a | market | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 48     | ETH   |
      | trader3a | market | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 45     | ETH   |
      | trader3a |        | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | ETH/DEC21 | 56     | ETH   |
      | trader3b | market | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 270    | ETH   |
      | trader3b |        | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | ETH/DEC21 | 162    | ETH   |
      | market   | aux2   | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 48     | ETH   |
      | market   | aux2   | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 45     | ETH   |
      | market   | aux2   | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 270    | ETH   |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 0      | 0       |
      | trader3b | ETH   | ETH/DEC21 | 0      | 0       |

    And the insurance pool balance should be "0" for the market "ETH/DEC21"

  Scenario: S020, Testing fees in continuous trading with two pegged trades and one liquidity providers (0029-FEES-002)

    When the following network parameters are set:
      | name                                                | value |
      | market.liquidity.providers.fee.distributionTimeStep | 10s   |
    And the average block duration is "1"

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |

    When the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | aux1     | ETH   | 100000000 |
      | aux2     | ETH   | 100000000 |
      | trader3a | ETH   | 100000    |
      | trader4  | ETH   | 100000    |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC21 | buy  | 1      | 920   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1      | 1080  | 0                | TYPE_LIMIT | TIF_GTC |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | buy  | MID              | 1          | 10     | submission |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | sell | MID              | 1          | 10     | amendment  |

    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1000       | TRADING_MODE_CONTINUOUS |

    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | buy  | 920   | 1      |
      | buy  | 990   | 10     |
      | sell | 1010  | 10     |
      | sell | 1080  | 1      |

    Then the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21 | buy  | 10     | 990   | 0                | TYPE_LIMIT | TIF_GTC |

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1000       | TRADING_MODE_CONTINUOUS |

    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | buy  | 920   | 1      |
      | buy  | 990   | 10     |
      | buy  | 1025  | 9      |
      | sell | 1045  | 10     |
      | sell | 1080  | 1      |

    Then the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader4 | ETH/DEC21 | sell | 30     | 990   | 2                | TYPE_LIMIT | TIF_GTC |

    Then the following trades should be executed:
      | buyer    | price | size | seller  |
      | trader3a | 990   | 10   | trader4 |
      | aux1     | 1025  | 9    | trader4 |

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general  | bond  |
      | aux1  | ETH   | ETH/DEC21 | 5375   | 99984347 | 10000 |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 2916   | 97134   |
      | trader4  | ETH   | ETH/DEC21 | 3915   | 96244   |

    And the liquidity fee factor should be "0.001" for the market "ETH/DEC21"
    And the accumulated liquidity fees should be "20" for the market "ETH/DEC21"

    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 990        | TRADING_MODE_CONTINUOUS |

    # For trader4 -

    # trade_value_for_fee_purposes with trader3a = size_of_trade * price_of_trade = 10 * 990 = 9900
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.002 * 9900 = 19.8 = 20 (rounded up to nearest whole value)
    # maker_fee =  fee_factor[maker]  * trade_value_for_fee_purposes = 0.005 * 9900 = 49.5 = 50 (rounded up to nearest whole value)
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0.001 * 9900 = 9.9 = 10 (rounded up to nearest whole value)

    # trade_value_for_fee_purposes with trader4 = size_of_trade * price_of_trade = 9 * 1025 = 9225
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.002 * 9255 = 18.45 = 19 (rounded up to nearest whole value)
    # maker_fee =  fee_factor[maker]  * trade_value_for_fee_purposes = 0.005 * 9255 = 46.125 = 47 (rounded up to nearest whole value)
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0.001 * 9255 = 9.225 = 10 (rounded up to nearest whole value)

    And the following transfers should happen:
      | from    | to       | from account            | to account                       | market id | amount | asset |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 50     | ETH   |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 20     | ETH   |
      | trader4 |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 39     | ETH   |
      | market  | trader3a | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 50     | ETH   |
      | market  | aux1     | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 47     | ETH   |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 2916   | 97134   |
      | trader4  | ETH   | ETH/DEC21 | 3915   | 96244   |

    # And the accumulated infrastructure fee should be "20" for the market "ETH/DEC21"
    And the accumulated liquidity fees should be "20" for the market "ETH/DEC21"

    When the network moves ahead "11" blocks

    And the following transfers should happen:
      | from   | to   | from account                | to account           | market id | amount | asset |
      | market | aux1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 20     | ETH   |

  Scenario: S021, Testing fees when network parameters are changed (in continuous trading with one trade and no liquidity providers) (0029-FEES-002, 0029-FEES-009)
    Description : Changing net params does change the fees being collected appropriately even if the market is already running

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
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount    |
      | aux1    | ETH   | 100000000 |
      | aux2    | ETH   | 100000000 |
      | trader3 | ETH   | 10000     |
      | trader4 | ETH   | 10000     |
      | lpprov  | ETH   | 100000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC21 | 90000000          | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | lpprov | ETH/DEC21 | 90000000          | 0.1 | sell | ASK              | 50         | 10     | submission |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1000       | TRADING_MODE_CONTINUOUS |
    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3 | ETH/DEC21 | buy  | 3      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3 | ETH   | ETH/DEC21 | 720    | 9280    |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"
    And the accumulated infrastructure fees should be "0" for the asset "ETH"

    #  Changing net params fees factors
    And the following network parameters are set:
      | name                                 | value |
      | market.fee.factors.makerFee          | 0.05  |
      | market.fee.factors.infrastructureFee | 0.02  |

    Then the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader4 | ETH/DEC21 | sell | 4      | 1002  | 1                | TYPE_LIMIT | TIF_GTC |

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1002       | TRADING_MODE_CONTINUOUS |

    Then the following trades should be executed:
      # | buyer   | price | size | seller  | maker   | taker   |
      # | trader3 | 1002  | 3    | trader4 | trader3 | trader4 |
      # TODO to be implemented by Core Team
      | buyer   | price | size | seller  |
      | trader3 | 1002  | 3    | trader4 |

    # trade_value_for_fee_purposes = size_of_trade * price_of_trade = 3 *1002 = 3006
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.02 * 3006 = 60.12 = 61 (rounded up to nearest whole value)
    # maker_fee =  fee_factor[maker]  * trade_value_for_fee_purposes = 0.05 * 3006 = 150.30 = 151 (rounded up to nearest whole value)
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0 * 3006 = 0

    And the following transfers should happen:
      | from    | to      | from account            | to account                       | market id | amount | asset |
      | trader4 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 151    | ETH   |
      | trader4 |         | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 61     | ETH   |
      | trader4 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 0      | ETH   |
      | market  | trader3 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 151    | ETH   |

    # total_fee = infrastructure_fee + maker_fee + liquidity_fee = 61 + 151 + 0 = 212
    # Trader3 margin + general account balance = 10000 + 151 ( Maker fees) = 10151
    # Trader4 margin + general account balance = 10000 - 151 ( Maker fees) - 61 (Infra fee) = 9788

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3 | ETH   | ETH/DEC21 | 4329   | 5822    |
      | trader4 | ETH   | ETH/DEC21 | 4088   | 5700    |

    And the accumulated infrastructure fees should be "61" for the asset "ETH"
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"
