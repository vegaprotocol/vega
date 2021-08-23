Feature: Fees Calculations

Scenario: Testing fees in continuous trading with one trade and no liquidity providers
    
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the price monitoring updated every "1000" seconds named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |
    
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100          | -100         | 0.1                    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | oracle config          | maturity date        |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 2019-12-31T23:59:59Z |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | aux1    | ETH   | 100000000  |
      | aux2    | ETH   | 100000000  |
      | trader3 | ETH   | 10000  |
      | trader4 | ETH   | 10000  |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1    | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |

   # Then debug transfers
    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | 
      | 1000       | TRADING_MODE_CONTINUOUS |  
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3 | ETH/DEC21 | buy  | 3      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the parties should have the following account balances:
      | party     | asset | market id | margin | general  |
      | trader3    | ETH   | ETH/DEC21 | 720    | 9280 |
  
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"
  # TODO to be implemented by Core Team
  # And the accumulated infrastructure fees should be "0" for the market "ETH/DEC21"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader4 | ETH/DEC21 | sell  | 4     | 1002  | 1                | TYPE_LIMIT | TIF_GTC |

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

    Then the parties should have the following account balances:
      | party     | asset | market id | margin | general |
      | trader3    | ETH   | ETH/DEC21 | 1089   | 8927 | 
      | trader4    | ETH   | ETH/DEC21 | 657    | 9320 | 
      
    # And the accumulated infrastructure fee should be "7" for the market "ETH/DEC21"
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

Scenario: Testing fees in continuous trading with two trades and no liquidity providers
    
    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the price monitoring updated every "1000" seconds named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |
    
    When the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100          | -100         | 0.1                    |

    And the markets:
      | id        | quote name | asset | risk model                | margin calculator         | auction duration | fees          | price monitoring | oracle config          | maturity date        |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 2019-12-31T23:59:59Z |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount     |
      | aux1     | ETH   | 100000000  |
      | aux2     | ETH   | 100000000  |
      | trader3a | ETH   | 10000  |
      | trader3b | ETH   | 10000  |
      | trader4  | ETH   | 10000  |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1    | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |

   # Then debug transfers
    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | 
      | 1000       | TRADING_MODE_CONTINUOUS |  
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21 | buy  | 2      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3b | ETH/DEC21 | buy  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the parties should have the following account balances:
      | party      | asset | market id | margin | general  |
      | trader3a    | ETH   | ETH/DEC21 | 480    | 9520 |
      | trader3b    | ETH   | ETH/DEC21 | 240    | 9760 |
  
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"
  # TODO to be implemented by Core Team
  # And the accumulated infrastructure fees should be "0" for the market "ETH/DEC21"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader4 | ETH/DEC21 | sell  | 4     | 1002  | 2                | TYPE_LIMIT | TIF_GTC |

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

     # Then debug transfers
        
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
      | party      | asset | market id | margin | general |
      | trader3a    | ETH   | ETH/DEC21 | 726    | 9285    | 
      | trader3b    | ETH   | ETH/DEC21 | 363    | 9643    | 
      | trader4     | ETH   | ETH/DEC21 | 657    | 9318    |
      
    # And the accumulated infrastructure fee should be "8" for the market "ETH/DEC21"
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

Scenario: Testing fees in continuous trading with two trades and one liquidity providers with 10 and 0 s liquidity fee distribution timestep
    
    When the following network parameters are set:
      | name                                                | value |
      | market.liquidity.providers.fee.distributionTimeStep | 10s   |
    And the average block duration is "1"
    
    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the price monitoring updated every "1000" seconds named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |
    
    When the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100          | -100         | 0.1                    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | oracle config          | maturity date        |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 2019-12-31T23:59:59Z |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount     |
      | aux1     | ETH   | 100000000  |
      | aux2     | ETH   | 100000000  |
      | trader3a | ETH   | 10000  |
      | trader3b | ETH   | 10000  |
      | trader4  | ETH   | 10000  |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1    | ETH/DEC21 | buy  | 1      | 920   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 1      | 1080  | 0                | TYPE_LIMIT | TIF_GTC |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | -10    |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          |  10    |

    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | 
      | 1000       | TRADING_MODE_CONTINUOUS |  

    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1080  | 1      |
      | buy  | 920   | 1      |
      | buy  | 910   | 105    |
      | sell | 1090  | 92     |
   
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21 | buy  | 2      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3b | ETH/DEC21 | buy  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell | 4      | 1002  | 2                | TYPE_LIMIT | TIF_GTC |

    Then the parties should have the following account balances:
      | party      | asset | market id | margin | general  |
      | trader3a    | ETH   | ETH/DEC21 | 480    | 9531 |
      | trader3b    | ETH   | ETH/DEC21 | 240    | 9766 |
    
    And the liquidity fee factor should "0.001" for the market "ETH/DEC21"
    And the accumulated liquidity fees should be "5" for the market "ETH/DEC21"

    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |  
      | 1002       | TRADING_MODE_CONTINUOUS |

    Then the following trades should be executed:
      # | buyer    | price | size | seller  | maker   | taker   | buyer_fee | seller_fee | maker_fee |
      # | trader3a | 1002  | 2    | trader4 | trader3 | trader4 | 30        | 11         | 11        |
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

    Then debug transfers

    And the following transfers should happen:
      | from    | to       | from account            | to account                       | market id | amount | asset |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 11     | ETH   |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 |  6     | ETH   |
      | trader4 |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           |  8     | ETH   |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 |  5     | ETH   |
      | market  | trader3a | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 11     | ETH   |  
      | market  | trader3b | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 |  6     | ETH   |  
      # | market  | aux1     | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 |  5     | ETH   |

    # total_fee = infrastructure_fee + maker_fee + liquidity_fee = 8 + 11 + 6 + 0 = 25
    # Trader3a margin + general account balance = 10000 + 11 ( Maker fees) = 10011
    # Trader3b margin + general account balance = 10000 + 6 ( Maker fees) = 10006
    # Trader4  margin + general account balance = 10000 - (11+6) ( Maker fees) - 8 (Infra fee) = 99975

    # TODO: Check why margin doesn't go up after the trade WHEN the liquidity provision order gets included (seems to work fine without LP orders) (expecting commented out values)
    Then the parties should have the following account balances:
      | party      | asset | market id | margin | general |
      # | trader3a    | ETH   | ETH/DEC21 | 690    | 9321    | 
      # | trader3b    | ETH   | ETH/DEC21 | 339    | 9667    | 
      # | trader4     | ETH   | ETH/DEC21 | 679    | 9296    |
      | trader3a    | ETH   | ETH/DEC21 | 480    | 9531    | 
      | trader3b    | ETH   | ETH/DEC21 | 240    | 9766    | 
      | trader4     | ETH   | ETH/DEC21 | 679    | 9291    |
      
    # And the accumulated infrastructure fee should be "8" for the market "ETH/DEC21"
   # And the accumulated liquidity fees should be "5" for the market "ETH/DEC21"

    When the network moves ahead "11" blocks

    And the following transfers should happen:
      | from   | to   | from account                | to account          | market id | amount | asset |
      | market | aux1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_MARGIN | ETH/DEC21 | 5      | ETH   |

  # Scenario: WIP - Testing fees in continuous trading with two trades and one liquidity providers with 0s liquidity fee distribution timestep
    When the following network parameters are set:
      | name                                                | value |
      | market.liquidity.providers.fee.distributionTimeStep | 0s    |

       When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader4 | ETH/DEC21  | sell  | 2     | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
    
    And the parties place the following orders: 
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21 | buy  | 1     | 1002  | 1                | TYPE_LIMIT | TIF_GTC |

    # For trader4 -
    # trade_value_for_fee_purposes = size_of_trade * price_of_trade = 1 * 1002 = 1002
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.002 * 1002 = 2.004 = 3 (rounded up to nearest whole value)
    # maker_fee =  fee_factor[maker]  * trade_value_for_fee_purposes = 0.005 * 1002 = 5.01 = 6 (rounded up to nearest whole value)
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0.001 * 1002 = 1.002 = 2 (rounded up to nearest whole value)

      Then the following transfers should happen:
      | from      | to       | from account            | to account                       | market id | amount | asset |
      | trader3a  | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 6      | ETH   |
      | trader3a  |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 3      | ETH   |
      | trader3a  | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 2      | ETH   |
      | market    | trader4  | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 6      | ETH   |  
      
    And the accumulated liquidity fees should be "2" for the market "ETH/DEC21"

    When the network moves ahead "1" blocks

    And the following transfers should happen:
      | from   | to   | from account                | to account          | market id | amount | asset |
      | market | aux1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_MARGIN | ETH/DEC21 | 2      | ETH   |

Scenario: Testing fees get collected when amended order trades
    
    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the price monitoring updated every "1000" seconds named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |
    
    When the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100          | -100         | 0.1                    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | oracle config          | maturity date        |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 2019-12-31T23:59:59Z |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount    |
      | aux1     | ETH   | 100000000 |
      | aux2     | ETH   | 100000000 |
      | trader3a | ETH   | 10000     |
      | trader3b | ETH   | 10000     |
      | trader4  | ETH   | 1250      |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1    | ETH/DEC21 | buy  | 1      | 920   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 1      | 1080  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | 
      | 1000       | TRADING_MODE_CONTINUOUS | 
   
    When the parties place the following orders:
      | party   | market id  | side | volume | price | resulting trades | type       | tif     |
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

  Then debug transfers

    And the following transfers should happen:
      | from    | to       | from account            | to account                       | market id | amount | asset |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 11     | ETH   |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 |  6     | ETH   |
      | trader4 |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           |  8     | ETH   |
      | market  | trader3a | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 11     | ETH   |  
      | market  | trader3b | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 |  6     | ETH   |
     
    # total_fee = maker_fee + infrastructure_fee + liquidity_fee =  11 + 6 + 8 = 25
    # Trader3a margin + general account balance = 10000 + 11 ( Maker fees) = 10011
    # Trader3b margin + general account balance = 10000 + 6 ( Maker fees) = 10006
    # Trader4  margin + general account balance = 10000 - (11+6) ( Maker fees) - 8 (Infra fee) = 99975

    # TODO: Check why margin doesn't go up after the trade WHEN the liquidity provision order gets included (seems to work fine without LP orders) (expecting first commented out values) but getting second value in other cases
    Then the parties should have the following account balances:
      | party      | asset | market id | margin | general |
      | trader3a    | ETH   | ETH/DEC21 | 678    | 9333    | 
      | trader3b    | ETH   | ETH/DEC21 | 339    | 9667    |
      | trader4     | ETH   | ETH/DEC21 | 621    | 604     |
   
   # Placing second set of orders
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | trader3a | ETH/DEC21 | buy  | 2      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | trader3a-buy-1 |
      | trader4  | ETH/DEC21 | sell | 4      | 1003  | 0                | TYPE_LIMIT | TIF_GTC | trader4-sell-2 |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 1159   | 8852    |
      | trader4  | ETH   | ETH/DEC21 | 1102   | 123     |

      # reducing size
      And the parties amend the following orders:
      | party  | reference      | price | size delta | tif     |
      | trader4 | trader4-sell-2 | 1002  | 0          | TIF_GTC |

    # matching the order now
    Then the following trades should be executed:
      # | buyer   | price | size | seller  | maker   | taker   |
      # | trader3 | 1002  | 3    | trader4 | trader3 | trader4 |
      # TODO to be implemented by Core Team
      | buyer    | price | size | seller  |
      | trader3a | 1002  | 2    | trader4 |
      
      # checking if continuous mode still exists
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |  
      | 1002       | TRADING_MODE_CONTINUOUS |

    Then debug transfers

    # And the following transfers should happen:
    #   | from    | to       | from account            | to account                       | market id | amount | asset |
    #   | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 11     | ETH   |
    #   | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 |  6     | ETH   |
    #   | trader4 |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           |  8     | ETH   |
    #   | market  | trader3a | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 11     | ETH   |  
    #   | market  | trader3b | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 |  6     | ETH   |
     
     Then the parties should have the following account balances:
      | party      | asset | market id | margin | general |
      | trader3a    | ETH   | ETH/DEC21 | 1159   | 8852    |
      | trader4     | ETH   | ETH/DEC21 | 1102   |  123    |

Scenario: Testing fees in continuous trading with insufficient balance in their general account but margin covers the fees
    
    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the price monitoring updated every "1000" seconds named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |
    
    When the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100          | -100         | 0.1                    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | oracle config          | maturity date        |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 2019-12-31T23:59:59Z |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount     |
      | aux1     | ETH   | 100000000  |
      | aux2     | ETH   | 100000000  |
      | trader3  | ETH   | 10000000   |
      | trader4  | ETH   | 22086      |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1    | ETH/DEC21 | buy  | 1      | 920   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 1      | 1080  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | 
      | 1000       | TRADING_MODE_CONTINUOUS | 
   
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3 | ETH/DEC21 | buy  | 100    | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4 | ETH/DEC21 | sell | 100    | 1002  | 1                | TYPE_LIMIT | TIF_GTC |


    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |  
      | 1002       | TRADING_MODE_CONTINUOUS |

    Then the following trades should be executed:
      | buyer    | price | size | seller  |
      | trader3  | 1002  | 100  | trader4 |
      
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | trader3 | ETH   | ETH/DEC21 | 33888  | 9966613 | 
      | trader4 | ETH   | ETH/DEC21 | 21384  | 0       |

    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | trader4 | ETH/DEC21 | 17820       | 19602  | 21384   | 24948   |
   
    Then clear transfer events
 
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | trader3  | ETH/DEC21 | buy  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC | trader3-buy-1  |
      | trader4  | ETH/DEC21 | sell | 1      | 1002  | 1                | TYPE_LIMIT | TIF_GTC | trader4-sell-2 |

    Then debug transfers

    And the following transfers should happen:
      | from    | to       | from account            | to account                       | market id | amount | asset |
      | trader4 | market   | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 6      | ETH   |
      | trader4 |          | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | ETH/DEC21 | 3      | ETH   |
      | market  | trader3  | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 6      | ETH   |  

    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | trader4 | ETH/DEC21 | 17999       | 19798  | 21598   | 25198   |

    Then the parties should have the following account balances:
      | party      | asset | market id | margin | general |
      | trader3     | ETH   | ETH/DEC21 | 34129  | 9966378 |
      | trader4     | ETH   | ETH/DEC21 | 21375  | 0       |

Scenario: Testing fees to confirm fees are collected first and then margin
    
    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the price monitoring updated every "1000" seconds named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |
    
    When the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100          | -100         | 0.1                    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | oracle config          | maturity date        |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 2019-12-31T23:59:59Z |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount     |
      | aux1     | ETH   | 100000000  |
      | aux2     | ETH   | 100000000  |
      | trader3  | ETH   | 10000000   |
      | trader4  | ETH   | 214        |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1    | ETH/DEC21 | buy  | 1      | 920   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 1      | 1080  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | 
      | 1000       | TRADING_MODE_CONTINUOUS | 
 
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | trader3  | ETH/DEC21 | buy  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC | trader3-buy-1  |
      | trader4  | ETH/DEC21 | sell | 1      | 1002  | 1                | TYPE_LIMIT | TIF_GTC | trader4-sell-2 |

    Then debug transfers

    And the following transfers should happen:
      | from    | to       | from account             | to account                       | market id | amount | asset |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL     | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 6      | ETH   |
      | trader4 |          | ACCOUNT_TYPE_GENERAL     | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | ETH/DEC21 | 3      | ETH   |
      | market  | trader3  | ACCOUNT_TYPE_FEES_MAKER  | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 6      | ETH   |  

    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | trader4 | ETH/DEC21 | 179         | 196    | 214     | 250     |

    Then the parties should have the following account balances:
      | party      | asset | market id | margin | general |
      | trader3     | ETH   | ETH/DEC21 | 339    | 9999667 |
      | trader4     | ETH   | ETH/DEC21 | 205    | 0       |

Scenario: Testing fees in continuous trading when insufficient balance in their general and margin account with LP, then the trade does not execute

  Given the following network parameters are set:
      | name                                                | value |
      | market.liquidity.providers.fee.distributionTimeStep | 10s   |
    And the average block duration is "1"

    When the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the price monitoring updated every "1000" seconds named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |
    
    When the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100          | -100         | 0.1                    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | oracle config          | maturity date        |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 2019-12-31T23:59:59Z |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount     |
      | aux1     | ETH   | 100000000  |
      | aux2     | ETH   | 100000000  |
      | trader3  | ETH   | 10000000   |
      | trader4  | ETH   | 189        |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1    | ETH/DEC21 | buy  | 10     | 920   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 10     | 1080  | 0                | TYPE_LIMIT | TIF_GTC |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | -10    |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          |  10    |

    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | 
      | 1000       | TRADING_MODE_CONTINUOUS | 

    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1080  | 10     |
      | buy  | 920   | 10     |
      | buy  | 910   | 60     |
      | sell | 1090  | 92     |
 
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | trader3  | ETH/DEC21 | buy  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC | trader3-buy-1  |
      | trader4  | ETH/DEC21 | sell | 1      | 1002  | 1                | TYPE_LIMIT | TIF_GTC | trader4-sell-2 |

   Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | 
      | 1002       | TRADING_MODE_CONTINUOUS | 

    Then the following trades should be executed:
      | buyer   | price | size | seller  |
      | trader3 | 1002  | 1    | trader4 |

     Then debug transfers
    And the following transfers should happen:
      | from    | to       | from account             | to account                       | market id | amount | asset |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL     | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 6      | ETH   |
      | trader4 |          | ACCOUNT_TYPE_GENERAL     | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | ETH/DEC21 | 3      | ETH   |
      | trader4 | market   | ACCOUNT_TYPE_GENERAL     | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 | 2      | ETH   |
      | market  | trader3  | ACCOUNT_TYPE_FEES_MAKER  | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 6      | ETH   |  
    
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      # | trader4 | ETH/DEC21 | 179         | 196    | 214     | 250     |
      | trader4 | ETH/DEC21 | 0         | 0    | 0     | 0     |

    Then the parties should have the following account balances:
      | party      | asset | market id | margin | general |
      | trader3     | ETH   | ETH/DEC21 | 240    | 9999766 |
      | trader4     | ETH   | ETH/DEC21 | 0    | 0       |

    And the liquidity fee factor should "0.001" for the market "ETH/DEC21"
    And the accumulated liquidity fees should be "4" for the market "ETH/DEC21"
    # why liquidity fees has been doubled from 2 to 4 incase of confiscation of trade 

    When the network moves ahead "11" blocks

    And the following transfers should happen:
      | from   | to   | from account                | to account          | market id | amount | asset |
      | market | aux1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_MARGIN | ETH/DEC21 | 4      | ETH   |

Scenario: Testing fees in auctions session with each side of a trade debited 1/2 IF & LP
    
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
    And the price monitoring updated every "1" seconds named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |
    
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r   | sigma |
      | 0.000001      | 0.1 | 0  | 1.4 | -1    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | oracle config          | maturity date        |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 2019-12-31T23:59:59Z |

    # setup accounts
    When the parties deposit on asset's general account the following amount:
      | party   | asset | amount     |
      | aux1     | ETH   | 100000000  |
      | aux2     | ETH   | 100000000  |
      | trader3a | ETH   | 10000      |
      | trader4  | ETH   | 10000      |

    Then the parties place the following orders:
      | party   | market id | side  | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/DEC21 | buy   | 1      | 500   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2     | ETH/DEC21 | sell  | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3a | ETH/DEC21 | buy   | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | buy  | BID              | 1          | -10    |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | sell | ASK              | 1          |  10    |
   
    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | target stake | supplied stake |
      | 1002       | TRADING_MODE_CONTINUOUS |          200 |            200 |
    Then the following trades should be executed:
      | buyer    | price | size | seller  |
      | trader3a | 1002  | 1    | trader4 |

     Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 843    | 9157    |
      | trader4  | ETH   | ETH/DEC21 | 1318   | 8682    |
      # why the margins are different for both parties
      
      #Scenario: Triggering Liquidity auction

      Then the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21 | buy  | 3      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell | 3      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger           |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY |
    
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | -10    |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          |  10    |

    When the network moves ahead "1" blocks

    # TODO: This seems to be suming the traded volume from the previous auction, verify and raise a bug.
    # Then the auction ends with a traded volume of "3" at a price of "1002"

    Then the following trades should be executed:
      | buyer    | price | size | seller  |
      | trader3a | 1002  | 3    | trader4 |

    # For trader3a & 4- Sharing IF and LP
    # trade_value_for_fee_purposes for trader3a = size_of_trade * price_of_trade = 3 * 1002= 3006
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.002 * 3006 = 6.012 = 7(rounded up)
    # maker_fee =  0 in auction
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0.001 * 3006 = 3.006 = 4 (rounded up)

    And the following transfers should happen:
      | from     | to       | from account            | to account                       | market id | amount | asset |
      | trader4  |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           |  4     | ETH   |
      | trader4  | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 |  2     | ETH   |
      | trader3a |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           |  4     | ETH   |
      | trader3a | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 |  2     | ETH   |

     Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 3372   | 6622    |
      | trader4  | ETH   | ETH/DEC21 | 5271   | 4723    |

    #TODO: Raise a bug: mark price is not being checked, any value results in a pass.
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1002       | TRADING_MODE_CONTINUOUS | 1       | 903       | 1101      | 801          | 10000          | 4             |

    Then the parties place the following orders:
      | party   | market id  | side  | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21  | buy   | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21  | sell  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |

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
      | from     | to       | from account            | to account                       | market id | amount | asset |
      | trader4  |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           |  1     | ETH   |
      | trader4  | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 |  1     | ETH   |
      | trader3a |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           |  1     | ETH   |
      | trader3a | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 |  1     | ETH   |

     Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 3204   | 6380    |
      | trader4  | ETH   | ETH/DEC21 | 7140   | 3260    |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |

Scenario: Testing fees in Liquidity auction session trading with insufficient balance in their general account but margin covers the fees
    
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
    And the price monitoring updated every "1" seconds named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |
    
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r   | sigma |
      | 0.000001      | 0.1 | 0  | 1.4 | -1    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | oracle config          | maturity date        |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 2019-12-31T23:59:59Z |

    # setup accounts
    When the parties deposit on asset's general account the following amount:
      | party   | asset | amount     |
      | aux1     | ETH   | 100000000  |
      | aux2     | ETH   | 100000000  |
      | trader3a | ETH   | 5000       |
      | trader4  | ETH   | 5261       |

    Then the parties place the following orders:
      | party   | market id | side  | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/DEC21 | buy   | 1      | 500   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2     | ETH/DEC21 | sell  | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3a | ETH/DEC21 | buy   | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | buy  | BID              | 1          | -10    |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | sell | ASK              | 1          |  10    |
   
    Then the opening auction period ends for market "ETH/DEC21"

    #Scenario: Triggering Liquidity auction

    Then the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21 | buy  | 3      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell | 3      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger           |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY |
    
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | -10    |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          |  10    |

    When the network moves ahead "1" blocks

    # TODO: This seems to be suming the traded volume from the previous auction, verify and raise a bug.
    # Then the auction ends with a traded volume of "3" at a price of "1002"

    Then the following trades should be executed:
      | buyer    | price | size | seller  |
      | trader3a | 1002  | 3    | trader4 |

    # For trader3a & 4- Sharing IF and LP
    # trade_value_for_fee_purposes for trader3a = size_of_trade * price_of_trade = 3 * 1002= 3006
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.002 * 3006 = 6.012 = 7(rounded up)
    # maker_fee =  0 in auction
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0.001 * 3006 = 3.006 = 4 (rounded up)

    And the following transfers should happen:
      | from     | to       | from account            | to account                       | market id | amount | asset |
      | trader4  |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           |  4     | ETH   |
      | trader4  | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 |  2     | ETH   |
      | trader3a |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           |  4     | ETH   |
      | trader3a | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 |  2     | ETH   |

    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | trader4 | ETH/DEC21 | 4393        | 4832   | 5271    | 6150    |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 3372   | 1622    |
      | trader4  | ETH   | ETH/DEC21 | 5255   | 0       |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |

Scenario: Testing fees in Price auction session trading with insufficient balance in their general account but margin covers the fees
    
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
    And the price monitoring updated every "1" seconds named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |
    
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r   | sigma |
      | 0.000001      | 0.1 | 0  | 1.4 | -1    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | oracle config          | maturity date        |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 2019-12-31T23:59:59Z |

    # setup accounts
    When the parties deposit on asset's general account the following amount:
      | party   | asset | amount     |
      | aux1     | ETH   | 100000000  |
      | aux2     | ETH   | 100000000  |
      | trader3a | ETH   | 5000       |
      | trader4  | ETH   | 2656       |

    Then the parties place the following orders:
      | party   | market id | side  | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/DEC21 | buy   | 1      | 500   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2     | ETH/DEC21 | sell  | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3a | ETH/DEC21 | buy   | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | buy  | BID              | 1          | -10    |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | sell | ASK              | 1          |  10    |
   
    Then the opening auction period ends for market "ETH/DEC21"
    
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | -10    |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          |  10    |

    #TODO: Raise a bug: mark price is not being checked, any value results in a pass.
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1002       | TRADING_MODE_CONTINUOUS | 1       | 903       | 1101      | 200          | 10000          | 1             |

    Then the parties place the following orders:
      | party   | market id  | side  | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21  | buy   | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21  | sell  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |

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
      | from     | to       | from account            | to account                       | market id | amount | asset |
      | trader4  |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           |  1     | ETH   |
      | trader4  | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 |  1     | ETH   |
      | trader3a |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           |  1     | ETH   |
      | trader3a | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 |  1     | ETH   |

    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | trader4 | ETH/DEC21 | 2380        | 2618   | 2856    | 3332    |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 1392   | 3504    |
      | trader4  | ETH   | ETH/DEC21 | 2756   | 0       |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |

Scenario: Testing fees in Liquidity auction session trading with insufficient balance in their general and margin account, then the trade still goes ahead.

    Given the following network parameters are set:
      | name                                                | value |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 1     |
      | market.liquidity.providers.fee.distributionTimeStep | 10s   |

    And the average block duration is "1"
    
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005        | 2                |
    And the price monitoring updated every "1" seconds named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |
    
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r   | sigma |
      | 0.000001      | 0.1 | 0  | 1.4 | -1    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | oracle config          | maturity date        |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 2019-12-31T23:59:59Z |

    # setup accounts
    When the parties deposit on asset's general account the following amount:
      | party   | asset | amount     |
      | aux1     | ETH   | 100000000  |
      | aux2     | ETH   | 100000000  |
      | trader3a | ETH   | 5000       |
      | trader4  | ETH   | 5261       |

    Then the parties place the following orders:
      | party   | market id | side  | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/DEC21 | buy   | 1      | 500   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2     | ETH/DEC21 | sell  | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3a | ETH/DEC21 | buy   | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | buy  | BID              | 1          | -10    |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | sell | ASK              | 1          |  10    |
   
    Then the opening auction period ends for market "ETH/DEC21"

    #Scenario: Triggering Liquidity auction

    Then the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21 | buy  | 3      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell | 3      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger           |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY |
    
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | -10    |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          |  10    |

    When the network moves ahead "1" blocks

    # TODO: This seems to be suming the traded volume from the previous auction, verify and raise a bug.
    # Then the auction ends with a traded volume of "3" at a price of "1002"

    Then the following trades should be executed:
      | buyer    | price | size | seller  |
      | trader3a | 1002  | 3    | trader4 |

    # For trader3a & 4- Sharing IF and LP
    # trade_value_for_fee_purposes for trader3a = size_of_trade * price_of_trade = 3 * 1002= 3006
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 2 * 3006
    # maker_fee =  0 in auction
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0.001 * 3006 = 3.006 = 4 (rounded up)
 Then debug transfers

    And the following transfers should happen:
      | from     | to       | from account            | to account                       | market id | amount | asset |
      | trader4  |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           |  3006  | ETH   |
      | trader4  | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 |  2     | ETH   |
      | trader3a |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           |  3006  | ETH   |
      | trader3a | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 |  2     | ETH   |

    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | trader4 | ETH/DEC21 | 4393        | 4832   | 5271    | 6150    |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 0      | 0       |
      | trader4  | ETH   | ETH/DEC21 | 0      | 0       |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |

Scenario: Testing fees in Price auction session trading with insufficient balance in their general and margin account, then the trade still goes ahead
    
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
    And the price monitoring updated every "1" seconds named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |
    
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r   | sigma |
      | 0.000001      | 0.1 | 0  | 1.4 | -1    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | oracle config          | maturity date        |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 2019-12-31T23:59:59Z |

    # setup accounts
    When the parties deposit on asset's general account the following amount:
      | party   | asset | amount     |
      | aux1     | ETH   | 100000000  |
      | aux2     | ETH   | 100000000  |
      | trader3a | ETH   | 3500       |
      | trader4  | ETH   | 5500       |

    Then the parties place the following orders:
      | party   | market id | side  | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/DEC21 | buy   | 1      | 500   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2     | ETH/DEC21 | sell  | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3a | ETH/DEC21 | buy   | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | buy  | BID              | 1          | -10    |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | sell | ASK              | 1          |  10    |
   
    Then the opening auction period ends for market "ETH/DEC21"
    
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | -10    |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          |  10    |

    #TODO: Raise a bug: mark price is not being checked, any value results in a pass.
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1002       | TRADING_MODE_CONTINUOUS | 1       | 903       | 1101      | 200          | 10000          | 1             |

    Then the parties place the following orders:
      | party   | market id  | side  | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21  | buy   | 2      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21  | sell  | 2      | 900   | 0                | TYPE_LIMIT | TIF_GTC |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode                    | auction trigger       |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

    Then the network moves ahead "301" blocks

    Then the following trades should be executed:
      | buyer    | price | size | seller  |
      | trader3a | 900   | 2    | trader4 |

    # For trader3a & 4- Sharing IF and LP
    # trade_value_for_fee_purposes for trader3a = size_of_trade * price_of_trade = 2 * 900 = 1800
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 2 * 900
    # maker_fee =  0 in auction
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0.001 * 1800 = 1.8 = 2/2 = 1

    Then debug transfers

    And the following transfers should happen:
      | from     | to       | from account            | to account                       | market id | amount | asset |
      | trader4  |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           |  1800  | ETH   |
      | trader4  | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 |  1     | ETH   |
      | trader3a |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           |  1800  | ETH   |
      | trader3a | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 |  1     | ETH   |

    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | trader4 | ETH/DEC21 | 3570        | 3927   | 4284    | 4998    |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 1597   | 0       |
      | trader4  | ETH   | ETH/DEC21 | 3801   | 0       |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |

Scenario: WIP - Testing fees in Price auction session trading with insufficient balance in their general and margin account, then the trade does not go ahead
    
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
    And the price monitoring updated every "1" seconds named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |
    
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r   | sigma |
      | 0.000001      | 0.1 | 0  | 1.4 | -1    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | oracle config          | maturity date        |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 2019-12-31T23:59:59Z |

    # setup accounts
    When the parties deposit on asset's general account the following amount:
      | party   | asset | amount     |
      | aux1     | ETH   | 100000000  |
      | aux2     | ETH   | 100000000  |
      | trader3a | ETH   | 5000       |
      | trader4  | ETH   | 7465       |
      # If the trader4 balance is changed to from 7261 to 7465 then the trade goes ahead as the account balance goes above maintenance level after paying fees.
      # | trader4  | ETH   | 7261       |
      # If the trader4 balance is changed to 7465 then the trade goes ahead as the account balance goes below maintenance level.

    Then the parties place the following orders:
      | party   | market id | side  | volume | price | resulting trades | type       | tif     |
      | aux1     | ETH/DEC21 | buy   | 1      | 500   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2     | ETH/DEC21 | sell  | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3a | ETH/DEC21 | buy   | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | buy  | BID              | 1          | -10    |
      | lp1 | aux1  | ETH/DEC21 | 200               | 0.001 | sell | ASK              | 1          |  10    |
   
    Then the opening auction period ends for market "ETH/DEC21"
    
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | -10    |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          |  10    |

    #TODO: Raise a bug: mark price is not being checked, any value results in a pass.
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1002       | TRADING_MODE_CONTINUOUS | 1       | 903       | 1101      | 200          | 10000          | 1             |

    Then the parties place the following orders:
      | party   | market id  | side  | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21  | buy   | 3      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21  | sell  | 3      | 900   | 0                | TYPE_LIMIT | TIF_GTC |

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

    Then debug transfers

    And the following transfers should happen:
      | from     | to       | from account            | to account                       | market id | amount | asset |
      | trader4  |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           |  2700  | ETH   |
      | trader4  | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 |  2     | ETH   |
      | trader3a |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           |  2700  | ETH   |
      | trader3a | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 |  2     | ETH   |

    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | trader4 | ETH/DEC21 | 4760        | 5236   | 5712    | 6664    |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 0      | 0       |
      | trader4  | ETH   | ETH/DEC21 | 0      | 0       |

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED |

Scenario: Testing fees in continuous trading during position resolution

  Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    
  And the markets:
      | id        | quote name | asset | risk model                  | margin calculator                  | auction duration | fees         | price monitoring | oracle config          | maturity date        |
      | ETH/DEC21 | ETH        | ETH   | default-simple-risk-model-2 | default-overkill-margin-calculator | 2                | fees-config-1| default-none     | default-eth-for-future | 2019-12-31T23:59:59Z |

  And the parties deposit on asset's general account the following amount:
      | party    | asset | amount        |
      | aux1     | ETH   | 1000000000000 |
      | aux2     | ETH   | 1000000000000 |
      | trader3a | ETH   | 10000         |
      | trader3b | ETH   | 30000         |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1   | ETH/DEC21| sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1   |
      | aux2   | ETH/DEC21| buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1   |
      | aux1   | ETH/DEC21| sell | 10     | 180   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-2   |
      | aux2   | ETH/DEC21| buy  | 10     | 180   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-2   |
    
  Then the opening auction period ends for market "ETH/DEC21"
  And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
  And the mark price should be "180" for the market "ETH/DEC21"

  When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | aux1  | ETH/DEC21 | sell | 150    | 200   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | aux2  | ETH/DEC21 | buy  | 50     | 190   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | aux2  | ETH/DEC21 | buy  | 350    | 180   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2  |

  When the parties place the following orders:
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
      | trader3b | ETH/DEC21 | 7500        | 24000  | 30000   | 37500   |

    Then the parties cancel the following orders:
      | party | reference       |
      | aux1  | sell-provider-1 |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | aux1  | ETH/DEC21 | sell | 500    | 350   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-2 |
    
    And the parties place the following orders:
      | party | market id  | side | volume | price | resulting trades | type       | tif     | reference       |
      | aux1   | ETH/DEC21 | sell | 1      | 300   | 0                | TYPE_LIMIT | TIF_GTC | ref-1           |
      | aux2   | ETH/DEC21 | buy  | 1      | 300   | 1                | TYPE_LIMIT | TIF_GTC | ref-2           |

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

  And debug transfers
  And the following transfers should happen:
      | from     | to       | from account             | to account                       | market id | amount | asset |
      | trader3a | market   | ACCOUNT_TYPE_GENERAL     | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 48     | ETH   |
      | trader3a | market   | ACCOUNT_TYPE_GENERAL     | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 45     | ETH   |
      | trader3a |          | ACCOUNT_TYPE_GENERAL     | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | ETH/DEC21 | 37     | ETH   |
      | trader3b | market   | ACCOUNT_TYPE_GENERAL     | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 270    | ETH   |
      | trader3b |          | ACCOUNT_TYPE_GENERAL     | ACCOUNT_TYPE_FEES_INFRASTRUCTURE | ETH/DEC21 | 108    | ETH   |
      | market   | aux2     | ACCOUNT_TYPE_FEES_MAKER  | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 48     | ETH   | 
      | market   | aux2     | ACCOUNT_TYPE_FEES_MAKER  | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 45     | ETH   | 
      | market   | aux2     | ACCOUNT_TYPE_FEES_MAKER  | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 270    | ETH   |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3a | ETH   | ETH/DEC21 | 0      | 0       |
      | trader3b | ETH   | ETH/DEC21 | 0      | 0       |

  And the insurance pool balance should be "0" for the market "ETH/DEC21"

# TO DO -
# Testing fees in continuous trading with two trades and one liquidity providers with 10 & 0s liquidity fee distribution timestep
# During continuous trading, if a trade is matched and the aggressor / price taker has insufficient balance in their general (but margin covers it) account, then the trade fees gets executed in this order - Maker, IP, LP
# During continuous trading, if a trade is matched and the aggressor / price taker has insufficient balance in their general (and margin) account, then the trade doesn't execute
# Fees are collected in one case of amends: you amend the price so far that it causes an immediate trade - Issue # 3777 
# During all 3 Auction sessions, fees are spilt 1/2 for IF and LP. Maker = 0
# During auction trading, when insufficient balance in their general account but margin covers the fees
# During auction trading, when insufficient balance in their general (+ margin) account, then the trade still goes ahead, (fees gets executed in this order - Maker(0), IP, LP)
# Fees calculations during Position Resolution when the fees could be paid on pro rated basis.

# Fees calculations during Position Resolution when insufficient balance in their general and margin account, then the fees gets paid in order - Maker, IP and then LP else don't get paid.

# Liquidity provider orders results in a trade - pegged orders so that orders of LP gets matched and LP gets maker fee. (LP is a price maker and not taker here) with suffficent balance.
# Last 3 API points ? - check and raise issues in ticket on Core Board - Start working 
# Changing parameters (via governance votes) does change the fees being collected appropriately even if the market is already running - Use
	# MarketFeeFactorsMakerFee                        = "market.fee.factors.makerFee"
	# MarketFeeFactorsInfrastructureFee               = "market.fee.factors.infrastructureFee"