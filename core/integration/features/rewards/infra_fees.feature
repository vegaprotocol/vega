Feature: Fees calculations

Background:
    Given the following network parameters are set:
      | name                                              |  value |
      | reward.asset                                      |  VEGA  |
      | validators.epoch.length                           |  10s   |
      | validators.delegation.minAmount                   |  10    |
      | reward.staking.delegation.delegatorShare          |  0.883 |
      | reward.staking.delegation.minimumValidatorStake   |  100   |
      | reward.staking.delegation.maxPayoutPerParticipant | 100000 |
      | reward.staking.delegation.competitionLevel        |  1.1   |
      | reward.staking.delegation.minValidators           |  5     |
      | reward.staking.delegation.optimalStakeMultiplier  |  5.0   |
      | network.markPriceUpdateMaximumFrequency           | 0s     |

    Given time is updated to "2021-08-26T00:00:00Z"
    Given the average block duration is "2"

    And the validators:
      | id     | staking account balance |
      | node1  |         1000000         |
      | node2  |         1000000         |
      | node3  |         1000000         |
      | node4  |         1000000         |
      | node5  |         1000000         |
      | node6  |         1000000         |
      | node7  |         1000000         |
      | node8  |         1000000         |
      | node9  |         1000000         |
      | node10 |         1000000         |
      | node11 |         1000000         |
      | node12 |         1000000         |
      | node13 |         1000000         |

    #set up the self delegation of the validators
    Then the parties submit the following delegations:
      | party  | node id  | amount |
      | node1  |  node1   | 10000  | 
      | node2  |  node2   | 10000  |       
      | node3  |  node3   | 10000  | 
      | node4  |  node4   | 10000  | 
      | node5  |  node5   | 10000  | 
      | node6  |  node6   | 10000  | 
      | node7  |  node7   | 10000  | 
      | node8  |  node8   | 10000  | 
      | node9  |  node9   | 10000  | 
      | node10 |  node10  | 10000  | 
      | node11 |  node11  | 10000  | 
      | node12 |  node12  | 10000  | 
      | node13 |  node13  | 10000  | 

    And the parties deposit on staking account the following amount:
      | party  | asset  | amount |
      | party1 | VEGA   | 10000  |  

    Then the parties submit the following delegations:
    | party  | node id  | amount |
    | party1 |  node1   |  100   | 
    | party1 |  node2   |  200   |       
    | party1 |  node3   |  300   |     

    And the global reward account gets the following deposits:
      | asset | amount |
      | VEGA  | 50000  |  

    #complete the first epoch for the self delegation to take effect
    Then the network moves ahead "7" blocks

Scenario: Testing fees when network parameters are changed (in continuous trading with one trade and no liquidity providers) 
 Description : Changing net params does change the fees being collected appropriately even if the market is already running
    
    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |
    
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100          | -100         | 0.1                    |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config          |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount     |
      | aux1    | ETH   | 100000000  |
      | aux2    | ETH   | 100000000  |
      | trader3 | ETH   |   10000    |
      | trader4 | ETH   |   10000    |
      | lpprov  | ETH   | 1000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC21 | 90000             | 0.1 | sell | ASK              | 50         | 100    | submission |

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
      | trader3 | ETH   | ETH/DEC21 | 720    |  9280   |
  
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"
    And the accumulated infrastructure fees should be "0" for the asset "ETH"
  
  #  Changing net params fees factors
   And the following network parameters are set:
      | name                                 | value |
      | market.fee.factors.makerFee          | 0.05  |
      | market.fee.factors.infrastructureFee | 0.5   |

    Then the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader4 | ETH/DEC21 | sell  | 4     | 1002  | 1                | TYPE_LIMIT | TIF_GTC |

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |  
      | 1002       | TRADING_MODE_CONTINUOUS |

    Then the following trades should be executed:
      | buyer   | price | size | seller  |
      | trader3 | 1002  | 3    | trader4 |
      
    # trade_value_for_fee_purposes = size_of_trade * price_of_trade = 3 *1002 = 3006
    # infrastructure_fee = fee_factor[infrastructure] * trade_value_for_fee_purposes = 0.5 * 3006 = 1503
    # maker_fee =  fee_factor[maker]  * trade_value_for_fee_purposes = 0.05 * 3006 = 150.30 = 151 (rounded up to nearest whole value)
    # liquidity_fee = fee_factor[liquidity] * trade_value_for_fee_purposes = 0 * 3006 = 0
    And the following transfers should happen:
      | from    | to      | from account            | to account                       | market id | amount | asset |
      | trader4 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC21 |  151   | ETH   |
      | trader4 |         | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           |  1503  | ETH   |
      | trader4 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC21 |  301   | ETH   |
      | market  | trader3 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC21 |  151   | ETH   |  
    
    # total_fee = infrastructure_fee + maker_fee + liquidity_fee = 1503 + 151 + 0 = 1654
    # Trader3 margin + general account balance = 10000 + 151 ( Maker fees) = 10151
    # Trader4 margin + general account balance = 10000 - 151 ( Maker fees) - 1503 (Infra fee) = 8346
    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader3 | ETH   | ETH/DEC21 | 1089   | 9062    | 
      | trader4 | ETH   | ETH/DEC21 | 715    | 7330    | 
      
    And the accumulated infrastructure fees should be "1503" for the asset "ETH"
    And the accumulated liquidity fees should be "301" for the market "ETH/DEC21"
    Then the network moves ahead "7" blocks 

    #verify validator score 
    Then the validators should have the following val scores for epoch 1:
    | node id | validator score  | normalised score |
    |  node1  |      0.07734     |     0.07734      |    
    |  node2  |      0.07810     |     0.07810      |
    |  node3  |      0.07887     |     0.07887      | 
    |  node4  |      0.07657     |     0.07657      | 

    #staking and delegation rewards - 50k
    #node1 has 10k self delegation + 100 from party1
    #node2 has 10k self delegation + 200 from party2 
    #node3 has 10k self delegation + 300 from party3 
    #all other nodes have 10k self delegation 
    #party1 gets 0.07734 * 50000 * 0.883 * 100/10100 + 0.07810 * 50000 * 0.883 * 200/10200 + 0.07887 * 50000 * 0.883 * 300/10300
    #node1 gets: (1 - 0.883 * 100/10100) * 0.07734 * 50000
    #node2 gets: (1 - 0.883 * 200/10200) * 0.07810 * 50000
    #node3 gets: (1 - 0.883 * 300/10300) * 0.07887 * 50000
    #node4 - node13 gets: 0.07657 * 50000
    #infrastructure fees rewards -> 1503 
    #party1 gets 0.07734 * 1503 * 0.883 * 100/10100 + 0.07810 * 1503 * 0.883 * 200/10200 + 0.07887 * 1503 * 0.883 * 300/10300
    #node1 gets: (1 - 0.883 * 100/10100) * 0.07734 * 1503
    #node2 gets: (1 - 0.883 * 200/10200) * 0.07810 * 1503
    #node3 gets: (1 - 0.883 * 300/10300) * 0.07887 * 1503
    #node4 - node13 gets: 0.07657 * 1503

    And the parties receive the following reward for epoch 1:
    | party  | asset | amount |
    | party1 | ETH   |  6     |
    | node1  | ETH   |  115   |
    | node2  | ETH   |  115   |
    | node3  | ETH   |  115   |
    | node4  | ETH   |  115   |
    | node5  | ETH   |  115   |
    | node6  | ETH   |  115   |
    | node8  | ETH   |  115   |
    | node10 | ETH   |  115   |
    | node11 | ETH   |  115   |
    | node12 | ETH   |  115   |
    | node13 | ETH   |  115   |
    | party1 | VEGA  |  201   | 
    | node1  | VEGA  |  3832  | 
    | node2  | VEGA  |  3837  | 
    | node3  | VEGA  |  3841  | 
    | node4  | VEGA  |  3828  | 
    | node5  | VEGA  |  3828  | 
    | node6  | VEGA  |  3828  | 
    | node8  | VEGA  |  3828  | 
    | node10 | VEGA  |  3828  | 
    | node11 | VEGA  |  3828  | 
    | node12 | VEGA  |  3828  | 
    | node13 | VEGA  |  3828  | 



   
