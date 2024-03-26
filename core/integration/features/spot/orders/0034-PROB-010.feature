Feature: Spot market quality of committed liquidity

  Background:
    Given time is updated to "2024-01-01T00:00:00Z"

    Given the following network parameters are set:
      | name                                              | value |
      | network.markPriceUpdateMaximumFrequency           | 0s    |
      | market.value.windowLength                         | 1h    |
      | market.liquidity.probabilityOfTrading.tau.scaling | 1     |
      | limits.markets.maxPeggedOrders                    | 4     |      
    
    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 2              |
      | BTC | 2              |

    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model                    | auction duration | fees          | price monitoring   | decimal places | position decimal places | sla params |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | default-log-normal-risk-model | 1                | default-none  | default-none       | 2              | 2                       | SLA        |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 10000  |
      | party5 | BTC   | 100    |
      | lp1    | ETH   | 100000 |
      | lp1    | BTC   | 1000   |
      | lp2    | ETH   | 100000 |
      | lp2    | BTC   | 1000   |

    And the average block duration is "1"

    # Set up 2 users with liquidity commitments 
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | BTC/ETH   | 10000             | 0.001 | submission |
      | lp2 | lp2   | BTC/ETH   | 10000             | 0.001 | submission |

    # Place some orders to get out of auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | p1-buy1   |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | p1-buy2   |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | p5-sell1  |
      | party5 | BTC/ETH   | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | p5-sell2  |

    # Place pegged orders to cover their commitment, one close to top of the book, one further away
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference |
      | lp1   | BTC/ETH   | 200       | 100                  | buy  | BID              | 200    | 1      | lp1-buy   |
      | lp1   | BTC/ETH   | 200       | 100                  | sell | ASK              | 200    | 1      | lp1-sell  |
      | lp2   | BTC/ETH   | 200       | 100                  | buy  | BID              | 200    | 10     | lp2-buy   |
      | lp2   | BTC/ETH   | 200       | 100                  | sell | ASK              | 200    | 10     | lp2-sell  |   

    And the opening auction period ends for market "BTC/ETH"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "1000" for the market "BTC/ETH"
    And the network moves ahead "1" epochs

  Scenario: Change of market.liquidity.probabilityOfTrading.tau.scaling will immediately change the
            scaling parameter, hence will change the probability of trading used for comparing quality
            of committed liquidity (0034-PROB-010)

    And the orders should have the following states:
      | party  | market id | reference | side | volume | remaining | price | status        |
      | party1 | BTC/ETH   | p1-buy1   | buy  | 1      | 1         | 999   | STATUS_ACTIVE |
      | party1 | BTC/ETH   | p1-buy2   | buy  | 1      | 0         | 1000  | STATUS_FILLED |
      | party5 | BTC/ETH   | p5-sell1  | sell | 1      | 0         | 1000  | STATUS_FILLED |
      | party5 | BTC/ETH   | p5-sell2  | sell | 1      | 1         | 1001  | STATUS_ACTIVE |

    Then the pegged orders should have the following states:
      | party  | market id | side | volume | reference | offset | price | status        |
      | lp1    | BTC/ETH   | buy  | 200    | BID       | 1      | 998   | STATUS_ACTIVE |
      | lp1    | BTC/ETH   | sell | 200    | ASK       | 1      | 1002  | STATUS_ACTIVE |
      | lp2    | BTC/ETH   | buy  | 200    | BID       | 10     | 989   | STATUS_ACTIVE |
      | lp2    | BTC/ETH   | sell | 200    | ASK       | 10     | 1011  | STATUS_ACTIVE |

    Given the network moves ahead "1" epochs
    Then the liquidity provider fee shares for the market "BTC/ETH" should be:
      | party | equity like share | average entry valuation | average score |
      | lp1   | 0.5               | 10000                   | 0.6007854709  |
      | lp2   | 0.5               | 20000                   | 0.3992145291  |

    # Now change the PoT value and move forward an epoch to apply the change
    When the following network parameters are set:
      | name                                              | value |
      | market.liquidity.probabilityOfTrading.tau.scaling | 1000  |
    And the network moves ahead "1" epochs

    # The average score should be updated to be different from the numbers above 
    Then the liquidity provider fee shares for the market "BTC/ETH" should be:
      | party | equity like share | average entry valuation | average score |
      | lp1   | 0.5               | 10000                   | 0.561840354  |
      | lp2   | 0.5               | 20000                   | 0.438159646  |      