Feature: Check we can use LIMIT, PEGGED and ICEBERG orders to cover our commitment

# Once liquidity is committed LPs can meet their commitment by placing limit orders,
# pegged limit orders and iceberg orders. For iceberg orders only the visible peak
# counts towards the commitment (0044-LIME-029)

  Background:
    Given the following network parameters are set:
      | name                                                  | value |
      | market.liquidity.bondPenaltyParameter                 | 1     |
      | network.markPriceUpdateMaximumFrequency               | 0s    |
      | limits.markets.maxPeggedOrders                        | 2     |
      | validators.epoch.length                               | 5s    |
      | market.liquidity.earlyExitPenalty                     | 0.25  |
      | market.liquidity.stakeToCcyVolume                     | 1.0   |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope   | 0.19  |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax     | 1     |
    And the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | lqm-params         | 0.1              | 24h         | 1              |  
    And the average block duration is "1"
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 60          | 50            | 0.2                    |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 5                 |
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.01        | 0.5                          | 1                             | 1.0                    |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model           | auction duration | fees          | price monitoring   | sla params    |  sla params |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | simple-risk-model-1  | 1                | fees-config-1 | price-monitoring-1 | default-basic |  SLA        |


    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | party1 | ETH   | 100000000  |
      | party3 | ETH   | 100000000  |
      | party4 | ETH   | 100000000  |
      | party1 | BTC   | 100000000  |
      | party3 | BTC   | 100000000  |
      | party4 | BTC   | 100000000  |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type     |
      | lp1 | party1 | BTC/ETH | 10000             | 0.001 | submission  |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | BTC/ETH | buy  | 1000   | 900   | 0                | TYPE_LIMIT | TIF_GTC | p3b1      |
      | party3 | BTC/ETH | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | p3b2      |
      | party4 | BTC/ETH | sell | 1000   | 1100  | 0                | TYPE_LIMIT | TIF_GTC | p4s1      |
      | party4 | BTC/ETH | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | p4s2      |

  Scenario: 001, LP is covered fully with LIMIT orders
   When the opening auction period ends for market "BTC/ETH"
   And the auction ends with a traded volume of "10" at a price of "1000"
   Then the market data for the market "BTC/ETH" should be:
     | mark price | trading mode            | supplied stake | best static bid price | static mid price | best static offer price |
     | 1000       | TRADING_MODE_CONTINUOUS | 10000          | 900                   | 1000             | 1100                    |
   # Place LIMIT orders to cover our commitment
   And the parties place the following orders:
     | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
     | party1 | BTC/ETH | buy  | 10     | 999   | 0                | TYPE_LIMIT | TIF_GTC | party1-order1 |
     | party1 | BTC/ETH | sell | 10     | 1001  | 0                | TYPE_LIMIT | TIF_GTC | party1-order2 |
   # Move forward an epoch and make sure the accounts do not change as we have the full epoch covered with LIMIT orders
   When the network moves ahead "7" blocks
   Then the liquidity provisions should have the following states:
     | id  | party  | market    | commitment amount | status           |
     | lp1 | party1 | BTC/ETH   | 10000             | STATUS_ACTIVE    |
   And the parties should have the following account balances:
     | party  | asset | market id | general  |
     | party1 | ETH   | BTC/ETH   | 99980010 |

  Scenario: 002, LP is covered fully with PEGGED orders
    When the opening auction period ends for market "BTC/ETH"
    And the auction ends with a traded volume of "10" at a price of "1000"
    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | supplied stake | best static bid price | static mid price | best static offer price |
      | 1000       | TRADING_MODE_CONTINUOUS | 10000          | 900                   | 1000             | 1100                    |

    # Place PEGGED orders to cover our commitment
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     | pegged reference | pegged offset |
      | party1 | BTC/ETH   | buy  | 10     | 0     | 0                | TYPE_LIMIT | TIF_GTC | party1-order1 | MID              | 5             |
      | party1 | BTC/ETH   | sell | 10     | 0     | 0                | TYPE_LIMIT | TIF_GTC | party1-order2 | MID              | 5             |

    # Move forward an epoch and make sure the accounts do not change as we have the full epoch covered with PEGGED orders
    When the network moves ahead "7" blocks
    Then the liquidity provisions should have the following states:
      | id  | party  | market    | commitment amount | status           |
      | lp1 | party1 | BTC/ETH   | 10000             | STATUS_ACTIVE    |
    And the parties should have the following account balances:
      | party  | asset | market id | general  | 
      | party1 | ETH   | BTC/ETH   | 99980050 |

  Scenario: 003, LP is covered fully with ICEBERG orders
    When the opening auction period ends for market "BTC/ETH"
    And the auction ends with a traded volume of "10" at a price of "1000"
    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | supplied stake | best static bid price | static mid price | best static offer price |
      | 1000       | TRADING_MODE_CONTINUOUS | 10000          | 900                   | 1000             | 1100                    |

    # Place ICEBERG orders to cover our commitment
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | only |
      | party1 | BTC/ETH   | buy  | 100    | 999   | 0                | TYPE_LIMIT | TIF_GTC | 90        | 1                    | post |      
      | party1 | BTC/ETH   | sell | 100    | 1001  | 0                | TYPE_LIMIT | TIF_GTC | 90        | 1                    | post |      

    # Move forward an epoch and make sure the accounts do not change as we have the full epoch covered with ICEBERG orders
    When the network moves ahead "7" blocks
    Then the liquidity provisions should have the following states:
      | id  | party  | market    | commitment amount | status           |
      | lp1 | party1 | BTC/ETH   | 10000             | STATUS_ACTIVE    |
    And the parties should have the following account balances:
      | party  | asset | market id | general  |
      | party1 | ETH   | BTC/ETH   | 99890100 |

  Scenario: 004, LP is covered fully with ICEBERG orders even with small peak size
    When the opening auction period ends for market "BTC/ETH"
    And the auction ends with a traded volume of "10" at a price of "1000"
    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | supplied stake | best static bid price | static mid price | best static offer price |
      | 1000       | TRADING_MODE_CONTINUOUS | 10000          | 900                   | 1000             | 1100                    |

    # Place ICEBERG orders which have a too small peak to cover our commitment
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | only |
      | party1 | BTC/ETH | buy  | 100    | 999   | 0                | TYPE_LIMIT | TIF_GTC | 5         | 2                    | post |      
      | party1 | BTC/ETH | sell | 100    | 1001  | 0                | TYPE_LIMIT | TIF_GTC | 5         | 2                    | post |      

    # Move forward an epoch and make sure we get a penalty
    When the network moves ahead "7" blocks
    Then the liquidity provisions should have the following states:
      | id  | party  | market    | commitment amount | status           |
      | lp1 | party1 | BTC/ETH | 10000             | STATUS_ACTIVE    |
    And the parties should have the following account balances:
      | party  | asset | market id | general  |
      | party1 | ETH   | BTC/ETH   | 99890100 |
