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
      | id        | quote name | asset | risk model                | margin calculator         | auction duration | fees          | price monitoring | oracle config          | maturity date        |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 2019-12-31T23:59:59Z |

    # setup accounts
    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount     |
      | aux1    | ETH   | 100000000  |
      | aux2    | ETH   | 100000000  |
      | trader3 | ETH   | 10000  |
      | trader4 | ETH   | 10000  |

    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1    | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |

   # Then debug transfers
    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | 
      | 1000       | TRADING_MODE_CONTINUOUS |  
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3 | ETH/DEC21 | buy  | 3      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the traders should have the following account balances:
      | trader     | asset | market id | margin | general  |
      | trader3    | ETH   | ETH/DEC21 | 720    | 9280 |
  
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"
  # TODO to be implemented by Core Team
  # And the accumulated infrastructure fees should be "0" for the market "ETH/DEC21"

    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
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

    Then the traders should have the following account balances:
      | trader     | asset | market id | margin | general |
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
    Given the traders deposit on asset's general account the following amount:
      | trader   | asset | amount     |
      | aux1     | ETH   | 100000000  |
      | aux2     | ETH   | 100000000  |
      | trader3a | ETH   | 10000  |
      | trader3b | ETH   | 10000  |
      | trader4  | ETH   | 10000  |

    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1    | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |

   # Then debug transfers
    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | 
      | 1000       | TRADING_MODE_CONTINUOUS |  
    When the traders place the following orders:
      | trader   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21 | buy  | 2      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3b | ETH/DEC21 | buy  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the traders should have the following account balances:
      | trader      | asset | market id | margin | general  |
      | trader3a    | ETH   | ETH/DEC21 | 480    | 9520 |
      | trader3b    | ETH   | ETH/DEC21 | 240    | 9760 |
  
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"
  # TODO to be implemented by Core Team
  # And the accumulated infrastructure fees should be "0" for the market "ETH/DEC21"

    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
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
      | market  | trader3b | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 |  6     | ETH   |  

    # total_fee = infrastructure_fee + maker_fee + liquidity_fee = 8 + 11 + 6 + 0 = 25 ??
    # Trader3a margin + general account balance = 10000 + 11 ( Maker fees) = 10011
    # Trader3b margin + general account balance = 10000 + 6 ( Maker fees) = 10006
    # Trader4  margin + general account balance = 10000 - (11+6) ( Maker fees) - 8 (Infra fee) = 99975

    Then the traders should have the following account balances:
      | trader      | asset | market id | margin | general |
      | trader3a    | ETH   | ETH/DEC21 | 726    | 9285    | 
      | trader3b    | ETH   | ETH/DEC21 | 363    | 9643    | 
      | trader4     | ETH   | ETH/DEC21 | 657    | 9318    |
      
    # And the accumulated infrastructure fee should be "8" for the market "ETH/DEC21"
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

Scenario: Testing fees in continuous trading with two trades and one liquidity providers with 10 s liquidity fee distribution timestep
    
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
    Given the traders deposit on asset's general account the following amount:
      | trader   | asset | amount     |
      | aux1     | ETH   | 100000000  |
      | aux2     | ETH   | 100000000  |
      | trader3a | ETH   | 10000  |
      | trader3b | ETH   | 10000  |
      | trader4  | ETH   | 10000  |
      | lp5      | ETH   | 100000000  |

    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1    | ETH/DEC21 | buy  | 1      | 920   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 1      | 1080  | 0                | TYPE_LIMIT | TIF_GTC |
      # | aux1    | ETH/DEC21 | buy  | 105    | 910   | 0                | TYPE_LIMIT | TIF_GTC |
      # | aux2    | ETH/DEC21 | sell | 92     | 1090  | 0                | TYPE_LIMIT | TIF_GTC |

    #TODO: Changing party to lp5 changes order book composition, check why.
    Given the traders submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | -10    |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          |  10    |

   # Then debug transfers
    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | 
      | 1000       | TRADING_MODE_CONTINUOUS |  

    # Then debug liquidity provision events
    # Then debug orders

    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | sell | 1080  | 1      |
      | buy  | 920   | 1      |
      | buy  | 910   | 105    |
      | sell | 1090  | 92     |
   
    When the traders place the following orders:
      | trader   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21 | buy  | 2      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3b | ETH/DEC21 | buy  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the traders should have the following account balances:
      | trader      | asset | market id | margin | general  |
      | trader3a    | ETH   | ETH/DEC21 | 480    | 9520 |
      | trader3b    | ETH   | ETH/DEC21 | 240    | 9760 |
    
    And the liquidity fee factor should "0.001" for the market "ETH/DEC21"
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

  # TODO to be implemented by Core Team
  # And the accumulated infrastructure fees should be "0" for the market "ETH/DEC21"

    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader4 | ETH/DEC21 | sell  | 4     | 1002  | 2                | TYPE_LIMIT | TIF_GTC |

    #Then debug trades

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
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0.001 * 2004 = 2.004 = 3 (rounded up to nearest whole value)

    # For trader3b -
    # trade_value_for_fee_purposes = size_of_trade * price_of_trade = 1 * 1002 = 1002
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.002 * 1002 = 2.004 = 3 (rounded up to nearest whole value)
    # maker_fee =  fee_factor[maker]  * trade_value_for_fee_purposes = 0.005 * 1002 = 5.01 = 6 (rounded up to nearest whole value)
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0.001 * 1002 = 1.002 = 2 (rounded up to nearest whole value)

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
    Then the traders should have the following account balances:
      | trader      | asset | market id | margin | general |
      # | trader3a    | ETH   | ETH/DEC21 | 690    | 9321    | 
      # | trader3b    | ETH   | ETH/DEC21 | 339    | 9667    | 
      # | trader4     | ETH   | ETH/DEC21 | 679    | 9296    |
      | trader3a    | ETH   | ETH/DEC21 | 480    | 9531    | 
      | trader3b    | ETH   | ETH/DEC21 | 240    | 9766    | 
      | trader4     | ETH   | ETH/DEC21 | 679    | 9291    |
      
    # And the accumulated infrastructure fee should be "8" for the market "ETH/DEC21"
   # And the accumulated liquidity fees should be "5" for the market "ETH/DEC21"

    When the network moves ahead "11" blocks

    # Then debug transfers

    And the following transfers should happen:
      | from   | to   | from account                | to account          | market id | amount | asset |
      | market | aux1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_MARGIN | ETH/DEC21 | 5      | ETH   |

  # Scenario: WIP - Testing fees in continuous trading with two trades and one liquidity providers with 0s liquidity fee distribution timestep
    When the following network parameters are set:
      | name                                                | value |
      | market.liquidity.providers.fee.distributionTimeStep | 0s    |

       When the traders place the following orders:
      | trader   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader4 | ETH/DEC21  | sell  | 2     | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
    
    And the traders place the following orders: 
      | trader   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21 | buy  | 1     | 1002  | 1               | TYPE_LIMIT | TIF_GTC |
      # check resulting trade = 2 ?

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

Scenario: WIP - Testing fees in continuous trading with two trades and insufficient balance in their general (but margin covers it) account, then the trade fees gets executed in this order - Maker, IP, LP
    
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
    Given the traders deposit on asset's general account the following amount:
      | trader   | asset | amount     |
      | aux1     | ETH   | 100000000  |
      | aux2     | ETH   | 100000000  |
      | trader3a | ETH   | 10000  |
      | trader3b | ETH   | 10000  |
      | trader4  | ETH   | 630  |
      | lp5      | ETH   | 100000000  |

    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1    | ETH/DEC21 | buy  | 1      | 920   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 1      | 1080  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | 
      | 1000       | TRADING_MODE_CONTINUOUS | 
   
    When the traders place the following orders:
      | trader   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21 | buy  | 2      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3b | ETH/DEC21 | buy  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
    
  # TODO to be implemented by Core Team
  # And the accumulated infrastructure fees should be "0" for the market "ETH/DEC21"

    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
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
    Then the traders should have the following account balances:
      | trader      | asset | market id | margin | general |
      # | trader3a    | ETH   | ETH/DEC21 | 690    | 9321    | 
      # | trader3b    | ETH   | ETH/DEC21 | 339    | 9667    | 
      # | trader4     | ETH   | ETH/DEC21 | 679    | 9296    |
      # | trader3a    | ETH   | ETH/DEC21 | 480    | 9520    | 
      # | trader3b    | ETH   | ETH/DEC21 | 240    | 9760    | 
      # | trader4     | ETH   | ETH/DEC21 | 649    | 0    |
      | trader3a    | ETH   | ETH/DEC21 | 678    | 9333    | 
      | trader3b    | ETH   | ETH/DEC21 | 339    | 9667    | 
      | trader4     | ETH   | ETH/DEC21 | 605    | 0       |

Scenario: WIP - Testing fees in continuous trading with two trades and insufficient balance in their general and margin account, then the trade doesn't execute.
    
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
    Given the traders deposit on asset's general account the following amount:
      | trader   | asset | amount     |
      | aux1     | ETH   | 100000000  |
      | aux2     | ETH   | 100000000  |
      | trader3a | ETH   | 10000  |
      | trader3b | ETH   | 10000  |
      # | trader4  | ETH   | 679  |
      | trader4  | ETH   | 400  |
      | lp5      | ETH   | 100000000  |

    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1    | ETH/DEC21 | buy  | 1      | 920   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 1      | 1080  | 0                | TYPE_LIMIT | TIF_GTC |
     
    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | 
      | 1000       | TRADING_MODE_CONTINUOUS |
   
    When the traders place the following orders:
      | trader   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3a | ETH/DEC21 | buy  | 2      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3b | ETH/DEC21 | buy  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader4 | ETH/DEC21 | sell  | 4     | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

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

    # And the following transfers should happen:
    #   | from    | to       | from account            | to account                       | market id | amount | asset |
    #   | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 | 11     | ETH   |
    #   | trader4 | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 |  6     | ETH   |
    #   | trader4 |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           |  8     | ETH   |
    #   | market  | trader3a | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 | 11     | ETH   |  
    #   | market  | trader3b | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 |  6     | ETH   |  

    # total_fee = infrastructure_fee + maker_fee + liquidity_fee = 8 + 11 + 6 + 0 = 25 ??
    # Trader3a margin + general account balance = 10000 + 11 ( Maker fees) = 10011
    # Trader3b margin + general account balance = 10000 + 6 ( Maker fees) = 10006
    # Trader4  margin + general account balance = 10000 - (11+6) ( Maker fees) - 8 (Infra fee) = 99975

    # TODO: Check why margin doesn't go up after the trade WHEN the liquidity provision order gets included (seems to work fine without LP orders) (expecting commented out values)
    Then the traders should have the following account balances:
      | trader      | asset | market id | margin | general |
      # | trader3a    | ETH   | ETH/DEC21 | 690    | 9321    | 
      # | trader3b    | ETH   | ETH/DEC21 | 339    | 9667    | 
      # | trader4     | ETH   | ETH/DEC21 | 679    | 9296    |
      | trader3a    | ETH   | ETH/DEC21 | 678    | 9333    | 
      | trader3b    | ETH   | ETH/DEC21 | 339    | 9667    |
      | trader4     | ETH   | ETH/DEC21 | 375     | 0       |

Scenario: Testing fees in auction trading with two trades and one liquidity providers with 10 s liquidity fee distribution timestep; each side of a trade is debited 1/2 IF & LP
    
    Given the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.targetstake.triggering.ratio | 0     |
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
      | 0.2  | 0.1   | 100          | -100         | 0.1                    |
    
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r   | sigma |
      | 0.000001      | 0.1 | 0  | 1.4 | -1    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | oracle config          | maturity date        |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 2019-12-31T23:59:59Z |

    # setup accounts
    When the traders deposit on asset's general account the following amount:
      | trader   | asset | amount     |
      | aux1     | ETH   | 100000000  |
      | aux2     | ETH   | 100000000  |
      | trader3a | ETH   | 10000  |
      | trader3b | ETH   | 10000  |
      | trader4  | ETH   | 10000  |
      | lp5      | ETH   | 100000000  |

    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/DEC21 | buy  | 1      | 1050  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 1      | 1050  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1    | ETH/DEC21 | buy  | 1      | 920   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 1      | 1080  | 0                | TYPE_LIMIT | TIF_GTC |
      # | aux1    | ETH/DEC21 | buy  | 105    | 910   | 0                | TYPE_LIMIT | TIF_GTC |
      # | aux2    | ETH/DEC21 | sell | 92     | 1090  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3a | ETH/DEC21 | buy   | 2      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4  | ETH/DEC21 | sell  | 4      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    #TODO: Changing party to lp5 changes order book composition, check why.
    Given the traders submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | -10    |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          |  10    |

   
    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | 
       | 1002      | TRADING_MODE_CONTINUOUS |  
     # | 1002      | TRADING_MODE_OPENING_AUCTION |  

    # Then debug liquidity provision events
     Then debug trades
   Then debug transfers

  # For trader3a-
    # trade_value_for_fee_purposes for trader3a = size_of_trade * price_of_trade = 2 * 1002 = 2004
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.002 * 2004 = 4.008 = 5 (rounded up to nearest whole value)
    # maker_fee =  0 in auction
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0.001 * 2004 = 2.004 = 3 (rounded up to nearest whole value)

    And the following transfers should happen:
      | from     | to       | from account            | to account                       | market id | amount | asset |
      # | trader4  |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           |  8     | ETH   |
      # | trader4  | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 |  5     | ETH   |
      | trader3a |          | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           |  8     | ETH   |
      | trader3a | market   | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 |  5     | ETH   |


    # total_fee = infrastructure_fee + maker_fee + liquidity_fee = 8 + 11 + 6 + 0 = 25
    # Trader3a margin + general account balance = 10000 + 11 ( Maker fees) = 10011
  #   # Trader4  margin + general account balance = 10000 - (11+6) ( Maker fees) - 8 (Infra fee) = 99975ß

  #   And the order book should have the following volumes for market "ETH/DEC21":
  #     | side | price | volume |
  #     | sell | 1080  | 1      |
  #     | buy  | 920   | 1      |
  #     | buy  | 910   | 105    |
  #     | sell | 1090  | 92     |
   
  #   When the traders place the following orders:
  #     | trader   | market id | side | volume | price | resulting trades | type       | tif     |
  #     | trader3a | ETH/DEC21 | buy  | 2      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |
  #     | trader3b | ETH/DEC21 | buy  | 1      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

  #   Then the traders should have the following account balances:
  #     | trader      | asset | market id | margin | general  |
  #     | trader3a    | ETH   | ETH/DEC21 | 480    | 9520 |
  #     | trader3b    | ETH   | ETH/DEC21 | 240    | 9760 |
    
  #   And the liquidity fee factor should "0.001" for the market "ETH/DEC21"
  #   And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

  # # TODO to be implemented by Core Team
  # # And the accumulated infrastructure fees should be "0" for the market "ETH/DEC21"

  #   Then the traders place the following orders:
  #     | trader  | market id | side | volume | price | resulting trades | type       | tif     |
  #     | trader4 | ETH/DEC21 | sell  | 4     | 1002  | 2                | TYPE_LIMIT | TIF_GTC |

  #   #Then debug trades

  #   Then the market data for the market "ETH/DEC21" should be:
  #     | mark price | trading mode            |  
  #     | 1002       | TRADING_MODE_CONTINUOUS |

  #   Then the following trades should be executed:
  #     # | buyer   | price | size | seller  | maker   | taker   |
  #     # | trader3 | 1002  | 3    | trader4 | trader3 | trader4 |
  #     # TODO to be implemented by Core Team
  #     | buyer    | price | size | seller  |
  #     | trader3a | 1002  | 2    | trader4 |
  #     | trader3b | 1002  | 1    | trader4 |

  #    # Then debug transfers

  #   # TODO: Check why margin doesn't go up after the trade WHEN the liquidity provision order gets included (seems to work fine without LP orders) (expecting commented out values)
  #   Then the traders should have the following account balances:
  #     | trader      | asset | market id | margin | general |
  #     # | trader3a    | ETH   | ETH/DEC21 | 690    | 9321    | 
  #     # | trader3b    | ETH   | ETH/DEC21 | 339    | 9667    | 
  #     # | trader4     | ETH   | ETH/DEC21 | 679    | 9296    |
  #     | trader3a    | ETH   | ETH/DEC21 | 480    | 9531    | 
  #     | trader3b    | ETH   | ETH/DEC21 | 240    | 9766    | 
  #     | trader4     | ETH   | ETH/DEC21 | 679    | 9291    |
      
  #   # And the accumulated infrastructure fee should be "8" for the market "ETH/DEC21"
  #  # And the accumulated liquidity fees should be "5" for the market "ETH/DEC21"

  #   When the network moves ahead "11" blocks

  #   # Then debug transfers

  #   And the following transfers should happen:
  #     | from   | to   | from account                | to account          | market id | amount | asset |
  #     | market | aux1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_MARGIN | ETH/DEC21 | 5      | ETH   |

# TO DO -
# Testing fees in continuous trading with two trades and one liquidity providers with 0s liquidity fee distribution timestep - Expand the above scenario
# Scenario with insuffcient funds - Both Cont + Auction
# During continuous trading, if a trade is matched and the aggressor / price taker has insufficient balance in their general (but margin covers it) account, then the trade fees gets executed in this order - Maker, IP, LP
# During continuous trading, if a trade is matched and the aggressor / price taker has insufficient balance in their general (and margin) account, then the trade doesn't execute.

# Auction sessions with multiple trades - Normal positive scenario- During auctions, each side of a trade is debited 1/2 (infrastructure_fee + liquidity_fee) from their general (+ margin if needed) account. The infrastructure_fee fee is credited to the staking pool, the liquidity_fee is credited to the market making pool.
# During auction trading, if a trade is matched and the aggressor / price taker has insufficient balance in their general (but margin covers it) account, then the trade fees gets executed in this order - Maker(0), IP, LP
# During auction trading, if a trade is matched and the aggressor / price taker has insufficient balance in their general (+ margin if needed) account, then the trade fees gets executed in this order - Maker(0), IP, LP

# Liquidity provider orders results in a trade - pegged orders so that orders of LP gets matched and LP gets maker fee.
# Negative Fees Scenario 
# Fees are collected in one case of amends: you amend the price so far that it causes an immediate trade.
# Changing parameters (via governance votes) does change the fees being collected appropriately even if the market is already running.
# Last 3 API points ?