Feature: LP reduction in commitment takes places at the end of an epoch

  # Setup a spot market in opening auction with an LP commitment
  Background: 
    Given the spot markets:
      | id      | name    | base asset | quote asset | risk model                | fees          | price monitoring | auction duration |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | default-simple-risk-model | default-none  | default-none     | 1                |
    And the following network parameters are set:
      | name                                        | value |
    #   | market.liquidity.committmentMinTimeFraction | 0.8   | commented until param is in spot-execution branch
      | network.markPriceUpdateMaximumFrequency     | 0s    |
    #   | market.liquidity.providers.fee.calculationTimeStep | 10s | commented until param is in spot-execution branch
      | validators.epoch.length                     | 5s    |

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

  Scenario: During opening auction, reduced commitment does not take affect until end of epoch (0044-LIME-018)

    Given the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "BTC/ETH"
    
    # Reduce LP commitment
    And the parties submit the following liquidity commitment:
      | id  | party | market id | commitment amount | fee   | lp type    | reference |
      | lp1 | lpprov| BTC/ETH   | 400               | 0.001 | amendment | lp1_commitment |

    # still within same epoch
    When the network moves ahead "3" blocks  

    # TODO: Step definition
    # Still in same epoch so commitment change is pending
    Then the parties should have the following liquidity commitment:
      | id  | party | market id | commitment amount | fee   | lp type    | reference |
      | lp1 | lpprov| BTC/ETH   | 500               | 0.001 | amendment | lp1_commitment |

    Given the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "BTC/ETH"

    # should be in new epoch
    When the network moves ahead "3" blocks   
    
    # TODO: Step definition
    # Rolled to new epoch so commitment change accepted
    Then the parties should have the following liquidity commitment:
      | id  | party | market id | commitment amount | fee   | reference |
      | lp1 | lpprov| BTC/ETH   | 400               | 0.001 | lp1_commitment |

    And the parties should have the following account balances:
      | party  | asset | market id | bond |
      | lpprov | ETH   | BTC/ETH   | 400  |

  Scenario: During continuous trading, reduced commitment does not take affect until end of epoch (0044-LIME-018)
    
    Given the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "BTC/ETH"

    When the parties place the following orders:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference      |
      | aux1   | BTC/ETH   | buy  | 1      | 20      | 0                | TYPE_LIMIT | TIF_GTC | aux1-order1    |
      | aux2   | BTC/ETH   | sell | 1      | 20      | 0                | TYPE_LIMIT | TIF_GTC | aux2-order2    |
      | party1 | BTC/ETH   | buy  | 5      | 18      | 0                | TYPE_LIMIT | TIF_GTC | party-order11  |
      | party2 | BTC/ETH   | sell | 5      | 25      | 0                | TYPE_LIMIT | TIF_GTC | party-order12  |
    
    And the opening auction period ends for market "BTC/ETH"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    Given the parties submit the following liquidity commitment:
      | id  | party | market id | commitment amount | fee   | lp type    | reference |
      | lp1 | lpprov| BTC/ETH   | 500               | 0.001 | submission | lp1_commitment |
    
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    When the parties submit the following liquidity commitment:
      | id  | party | market id | commitment amount | fee   | lp type    | reference |
      | lp1 | lpprov| BTC/ETH   | 400               | 0.001 | amendment | lp1_commitment |

    And the network moves ahead "3" blocks  

    # TODO: Step definition
    # Still in same epoch so commitment change is still pending
    Then the parties should have the following liquidity commitment:
      | id  | party | market id | commitment amount | fee   | reference |
      | lp1 | lpprov| BTC/ETH   | 500               | 0.001 | lp1_commitment |

    Given the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "BTC/ETH"

    When the network moves ahead "3" blocks   
    
    # TODO: Step definition
    # Rolled to new epoch so commitment change accepted
    Then the parties should have the following liquidity commitment:
      | id  | party | market id | commitment amount | fee   | reference |
      | lp1 | lpprov| BTC/ETH   | 400               | 0.001 | lp1_commitment |

    And the parties should have the following account balances:
      | party  | asset | market id | bond |
      | lpprov | ETH   | BTC/ETH   | 400  |

Scenario: During opening auction, LP provides commitment, then in continuous trading, reduced commitment does not take affect until end of epoch (0044-LIME-018)
    
    Given the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "BTC/ETH"

    When the parties place the following orders:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference      |
      | aux1   | BTC/ETH   | buy  | 1      | 20      | 0                | TYPE_LIMIT | TIF_GTC | aux1-order1    |
      | aux2   | BTC/ETH   | sell | 1      | 20      | 0                | TYPE_LIMIT | TIF_GTC | aux2-order2    |
      | party1 | BTC/ETH   | buy  | 5      | 18      | 0                | TYPE_LIMIT | TIF_GTC | party-order11  |
      | party2 | BTC/ETH   | sell | 5      | 25      | 0                | TYPE_LIMIT | TIF_GTC | party-order12  |
    
    And the opening auction period ends for market "BTC/ETH"
    # New epoch
    And the network moves ahead "5" blocks 
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    Given the parties submit the following liquidity commitment:
      | id  | party | market id | commitment amount | fee   | lp type    | reference |
      | lp1 | lpprov| BTC/ETH   | 500               | 0.001 | submission | lp1_commitment |
    
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    When the parties submit the following liquidity commitment:
      | id  | party | market id | commitment amount | fee   | lp type    | reference |
      | lp1 | lpprov| BTC/ETH   | 400               | 0.001 | amendment | lp1_commitment |

    # current epoch
    And the network moves ahead "3" blocks  

    # TODO: Step definition
    # Still in same epoch so commitment change is still pending
    Then the parties should have the following liquidity commitment:
      | id  | party | market id | commitment amount | fee   | reference |
      | lp1 | lpprov| BTC/ETH   | 500               | 0.001 | lp1_commitment |

    Given the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "BTC/ETH"

    When the network moves ahead "3" blocks   
    
    # TODO: Step definition
    # Rolled to new epoch so commitment change accepted
    Then the parties should have the following liquidity commitment:
      | id  | party | market id | commitment amount | fee   | reference |
      | lp1 | lpprov| BTC/ETH   | 400               | 0.001 | lp1_commitment |

    And the parties should have the following account balances:
      | party  | asset | market id | bond |
      | lpprov | ETH   | BTC/ETH   | 400  |
