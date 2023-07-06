Feature: Penalties for Liquidity providers on spot market

  Background:
    Given the spot markets:
      | id      | name    | base asset | quote asset | risk model                | fees          | price monitoring | auction duration |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | default-simple-risk-model | default-none  | default-none     | 1                |
    And the following network parameters are set:
      | name                                        | value |
    #   | market.liquidity.committmentMinTimeFraction | 0.8   | commented until param is in spot-execution branch
      | network.markPriceUpdateMaximumFrequency     | 0s    |
    #   | market.liquidity.providers.fee.calculationTimeStep | 10s | commented until param is in spot-execution branch

    And the average block duration is "1"

    # setup accounts
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 10000  |
      | party2 | BTC   | 50000  |
      | aux1   | ETH   | 10000  |
      | aux2   | BTC   | 50000  |
      | lpprov | ETH   | 500000 |
      | lpprov | BTC   | 500000 |
    
    And the parties submit the following liquidity commitment:
      | id  | party | market id | commitment amount | fee   | lp type    | reference |
      | lp1 | lpprov| BTC/ETH   | 500               | 0.001 | submission | lp1_commitment |
    
    # place orders and generate trades
    And the parties place the following orders:
      | party  | market id | side | volume | price   | resulting trades | type     | tif     | reference      |
      | aux1   | BTC/ETH   | buy  | 1      | 20      | 0                | TYPE_LIMIT | TIF_GTC | aux1-order1    |
      | aux2   | BTC/ETH   | sell | 1      | 20      | 0                | TYPE_LIMIT | TIF_GTC | aux2-order2    |
      | party1 | BTC/ETH   | buy  | 5      | 18      | 0                | TYPE_LIMIT | TIF_GTC | party-order11  |
      | party2 | BTC/ETH   | sell | 5      | 25      | 0                | TYPE_LIMIT | TIF_GTC | party-order12  |

Scenario: If a liquidity provider has fraction_of_time_on_book >= market.liquidity.committmentMinTimeFraction, no penalty will be taken from their general account (0044-LIME-013) 

    Given the parties place the following orders:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | lpprov   | BTC/ETH   | buy  | 100      | 15  | 0                | TYPE_LIMIT | TIF_GTC | lp1-order1  |
      | lpprov   | BTC/ETH   | sell | 100      | 25  | 0                | TYPE_LIMIT | TIF_GTC | lp1-order2  |

    When the opening auction period ends for market "BTC/ETH"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    
    Given the parties should have the following account balances:
      | party  | asset | market id | general | bond |
      | lpprov | ETH   | BTC/ETH   | 498500  | 500  |

    When the network moves ahead "10" blocks 

    Then the parties should have the following account balances:
      | party  | asset | market id | general | bond |
      | lpprov | ETH   | BTC/ETH   | 498500  | 500  |
    
    # penalties are deposited in to the network treasury of the asset
    And the network treasury balance should be "0" for the asset "ETH"

Scenario: If a liquidity provider has fraction_of_time_on_book = 0.3, market.liquidity.committmentMinTimeFraction = 0.6, market.liquidity.sla.nonPerformanceBondPenaltySlope = 0.7, market.liquidity.sla.nonPerformanceBondPenaltyMax = 0.6 at the end of an epoch then they will forfeit 35% of their bond stake, which will be transferred into the network treasury (0044-LIME-014)

    Given the following network parameters are set:
      | name                                        | value |
    #   | market.liquidity.committmentMinTimeFraction | 0.6   | commented until param is in spot-execution branch
      | network.markPriceUpdateMaximumFrequency     | 0s    |
    #   | market.liquidity.providers.fee.calculationTimeStep | 10s | commented until param is in spot-execution branch
    #  | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0.7 |
    #  | market.liquidity.sla.nonPerformanceBondPenaltyMax | 0.6 |
    | validators.epoch.length | 10s |

    And the parties place the following orders:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | lpprov   | BTC/ETH   | buy  | 100      | 15  | 0                | TYPE_LIMIT | TIF_GTC | lp1-order1  |
      | lpprov   | BTC/ETH   | sell | 100      | 25  | 0                | TYPE_LIMIT | TIF_GTC | lp1-order2  |

    When the opening auction period ends for market "BTC/ETH"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    When the network moves ahead "5" blocks 
    Then the parties cancel the following orders:
      | party  | reference  |
      | lpprov | lp1-order1 |
      | lpprov | lp1-order2 |

    When the network moves ahead "5" blocks

    # update the account balances
    Then the parties should have the following account balances:
      | party  | asset | market id | general | bond |
      | lpprov | ETH   | BTC/ETH   | 498500  | 500  |
    
    # penalties are deposited in to the network treasury of the asset
    #  TODO: update the balance
    And the network treasury balance should be "100" for the asset "ETH"
    