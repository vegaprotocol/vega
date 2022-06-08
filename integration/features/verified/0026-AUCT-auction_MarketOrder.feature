Feature: Test interactions between different auction types with Market Orders

  Background:
    Given the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.targetstake.triggering.ratio | 0     |
      | network.floatingPointUpdates.delay            | 5s    |
      | market.auction.minimumDuration                | 10    |
    And the average block duration is "1"
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 10          | 10            | 0.1                    |
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.001         | 0.00011407711613050422 | 0  | 0.016 | 2.0   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring updated every "1" seconds named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the price monitoring updated every "1" seconds named "price-monitoring-2":
      | horizon | probability | auction extension |
      | 2       | 0.999995    | 200               |
      | 1       | 0.999999    | 300               |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | oracle config          |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1     | default-margin-calculator | 10               | fees-config-1 | price-monitoring-1 | default-eth-for-future |
      | ETH/DEC22 | ETH        | ETH   | log-normal-risk-model-1 | default-margin-calculator | 10               | fees-config-1 | price-monitoring-2 | default-eth-for-future |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | party0 | ETH   | 1000000000 |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |
      | party3 | ETH   | 100000000  |

  Scenario: Once market is in continuous trading mode: post a non-persistent order that should trigger liquidity auction (not enough target stake), appropriate event is sent and market remains in TRADING_MODE_CONTINUOUS (0026-AUCT-001, 0026-AUCT-005)
    Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0.8   |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | buy  | MID              | 2          | 1      | submission |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell | ASK              | 1          | 2      | submission |
      | lp1 | party0 | ETH/DEC21 | 1000              | 0.001 | sell | MID              | 2          | 1      | submission |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the opening auction period ends for market "ETH/DEC21"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1000         | 1000           | 10            |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error                           |
      | party1 | ETH/DEC21 | buy  | 20     | 1010  | 0                | TYPE_LIMIT | TIF_GFN | ref-ref   | OrderError: Invalid Persistence |

   And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | buy  | 900   | 1      |
      | buy  | 990   | 2      |
      | buy  | 989   | 0      |
      | buy  | 999   | 2      |
      | sell | 1002  | 0      |
      | sell | 1010  | 2      |
      | sell | 1100  | 1      |
      | sell | 1011  | 0      |
      | sell | 1100  | 1      |

    
    #try different trader

    # Then the parties place the following orders:
    #   | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error                           |
    #   | party3 | ETH/DEC21 | buy  | 20     | 1010  | 0                | TYPE_LIMIT | TIF_GFN | ref-ref   | OrderError: Invalid Persistence |

    # And the market data for the market "ETH/DEC21" should be:
    #   | trading mode            | auction trigger             | target stake | supplied stake | open interest |
    #   | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 1000         | 1000           | 10            |
  
    # And the market data for the market "ETH/DEC21" should be:
    #   | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
    #   | 1000       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1000         | 1000           | 10            |

    # try different price 
      Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | 
      | party1 | ETH/DEC21 | buy  | 20     | 1005  | 1                | TYPE_LIMIT | TIF_GFN | ref-ref   | 

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001       | TRADING_MODE_CONTINUOUS | 1       | 990       | 1010      | 1201         | 1000           | 12            |

