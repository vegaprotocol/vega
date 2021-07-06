Feature: Fees Calculations

Scenario: Fees Calculations
    
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee | liquidity fee |
      | 0.005     | 0.002              | 0.003         |
    And the price monitoring updated every "1000" seconds named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |
    
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100          | -100         | 0.1                    |

    And the markets:
      | id        | quote name | asset | risk model                | margin calculator         | auction duration | fees          | price monitoring | oracle config          | maturity date        |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 2019-12-31T23:59:59Z |

    #  And the following network parameters are set:
    #   | name                                                | value   |
    #   | market.value.windowLength                           | 1h      |
    #   | market.stake.target.timeWindow                      | 24h     |
    #   | market.stake.target.scalingFactor                   | 1       |
    #   | market.liquidity.targetstake.triggering.ratio       | 1       |
    #   | market.liquidity.providers.fee.distributionTimeStep | 10m     |

    # setup accounts
    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount     |
      | aux1    | ETH   | 100000000  |
      | aux2    | ETH   | 100000000  |
      | trader3 | ETH   | 100000000  |
      | trader4 | ETH   | 100000000  |

    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1    | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1    | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2    | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |

    # And the order book should have the following volumes for market "ETH/DEC21":
    #   | side | price    | volume |
    #   | sell | 1000     | 10     |
    #   | buy  | 1000     | 10     |

   # Then debug transfers
    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | 
      | 1000       | TRADING_MODE_CONTINUOUS |  
    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3 | ETH/DEC21 | buy  | 3      | 1002  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the traders should have the following account balances:
      | trader     | asset | market id | margin | general  |
      | trader3    | ETH   | ETH/DEC21 | 721    | 99999279 |
  
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
      | buyer   | price | size | seller  | maker   | taker   |
      | trader3 | 1002  | 3    | trader4 | trader3 | trader4 |
        
      
# Scenario: Fees Calculations

# Given I have open orders on my account during cont. & auction sessions
# When the order gets matched resulting in one or multple trades 
# Then each fee is correctly calculated in settlement currency of the market as below
#   infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes
#   maker_fee =  fee_factor[maker]  * trade_value_for_fee_purposes
#   liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes
#   total_fee = infrastructure_fee + maker_fee + liquidity_fee
#   trade_value_for_fee_purposes = = size_of_trade * price_of_trade
