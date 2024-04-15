Feature: Test liquidity provider reward distribution; Should also cover liquidity-fee-setting and equity-like-share calc and total stake.
  # to look into and test: If an equity-like share is small and LP rewards are distributed immediately, then how do we round? (does a small share get rounded up or down, do they all add up?)
  #Check what happens with time and distribution period (both in genesis and mid-market)

  Background:

    Given the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 500         | 500           | 0.1                    |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee | liquidity fee method | liquidity fee constant |
      | 0.0004    | 0.001              | METHOD_CONSTANT      | 0.08                   |
    And the fees configuration named "fees-config-2":
      | maker fee | infrastructure fee | liquidity fee method | liquidity fee constant |
      | 0.0004    | 0.001              | METHOD_CONSTANT      | 0.01                   |
    And the fees configuration named "fees-config-3":
      | maker fee | infrastructure fee | liquidity fee method    | liquidity fee constant |
      | 0.0004    | 0.001              | METHOD_WEIGHTED_AVERAGE | 0.01                   |
    And the fees configuration named "fees-config-4":
      | maker fee | infrastructure fee | liquidity fee method | liquidity fee constant |
      | 0.0004    | 0.001              | METHOD_MARGINAL_COST | 0.01                   |


    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 1              |
      | BTC | 1              |
    And the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 1h    |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
      | limits.markets.maxPeggedOrders                      | 8     |
      | validators.epoch.length                             | 5s    |
      | market.liquidity.providersFeeCalculationTimeStep    | 5s    |

    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | lqm-params         | 0.0              | 10s          | 0.75           |
    
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model          | auction duration | fees          | price monitoring | sla params | liquidity monitoring |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | simple-risk-model-1 | 2                | fees-config-4 | price-monitoring | SLA        | lqm-params           |

    Given the average block duration is "1"

  Scenario: 001: The resulting liquidity-fee-factor is always equal to one of the liquidity provider's individually nominated fee factors  (0042-LIQF-063)

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | BTC   | 1000000000 |
      | lp2    | BTC   | 1000000000 |
      | party1 | BTC   | 100000000  |
      | party2 | BTC   | 100000000  |
      | lp1    | ETH   | 1000000000 |
      | lp2    | ETH   | 1000000000 |
      | party1 | ETH   | 100000000  |
      | party2 | BTC   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | BTC/ETH   | 5000              | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp1   | BTC/ETH | 4         | 1                    | buy  | MID              | 4          | 1      |
      | lp1   | BTC/ETH | 4         | 1                    | sell | MID              | 4          | 1      |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp2 | lp2   | BTC/ETH   | 5000              | 0.002 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp2   | BTC/ETH | 4         | 1                    | buy  | MID              | 4          | 1      |
      | lp2   | BTC/ETH | 4         | 1                    | sell | MID              | 4          | 1      |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/ETH   | buy  | 90     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 90     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"

    When the network moves ahead "1" blocks

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 90   | party2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "1000" for the market "BTC/ETH"
    And the supplied stake should be "10000" for the market "BTC/ETH"

    # 10,000 staked and scaling factor is 10
    And the target stake should be "7500" for the market "BTC/ETH"

    And the liquidity provider fee shares for the market "BTC/ETH" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.5               | 5000                    |
      | lp2   | 0.5               | 10000                   |


    And the liquidity fee factor should be "0.002" for the market "BTC/ETH"

    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "BTC/ETH"

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | BTC/ETH   | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | BTC/ETH   | buy  | 20     | 1000  | 3                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party2 | 951   | 4    | lp1    |
      | party2 | 951   | 4    | lp2    |
      | party2 | 1000  | 12   | party1 |

    And the accumulated liquidity fees should be "394" for the market "BTC/ETH"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | BTC/ETH   | 197    | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | BTC/ETH   | 197    | ETH   |
   
  Scenario: 002: Liquidity fee factors are recalculated every time the liquidity demand estimate changes.  (0042-LIQF-064)

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | BTC   | 1000000000 |
      | lp2    | BTC   | 1000000000 |
      | party1 | BTC   | 100000000  |
      | party2 | BTC   | 100000000  |
      | lp1    | ETH   | 1000000000 |
      | lp2    | ETH   | 1000000000 |
      | party1 | ETH   | 100000000  |
      | party2 | BTC   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | BTC/ETH   | 5000              | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp1   | BTC/ETH | 4         | 1                    | buy  | MID              | 4          | 1      |
      | lp1   | BTC/ETH | 4         | 1                    | sell | MID              | 4          | 1      |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/ETH   | buy  | 90     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 90     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"

    When the network moves ahead "1" blocks

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 90   | party2 |

    And the liquidity fee factor should be "0.001" for the market "BTC/ETH"

   
    # now another LP enters with a different fee
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp2 | lp2   | BTC/ETH   | 5000              | 0.002 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp2   | BTC/ETH   | 4         | 1                    | buy  | MID              | 4          | 1      |
      | lp2   | BTC/ETH   | 4         | 1                    | sell | MID              | 4          | 1      |

    # no liqudiity fee change
    When the network moves ahead "1" blocks
    And the liquidity fee factor should be "0.001" for the market "BTC/ETH"

    # until we eneter a new epoch and the LP provisions become active
    When the network moves ahead "1" epochs
    And the liquidity fee factor should be "0.002" for the market "BTC/ETH"


  Scenario: 003: If passage of time causes the liquidity demand estimate to change, the fee factor is correctly recalculated.  (0042-LIQF-065)

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | BTC   | 1000000000 |
      | lp2    | BTC   | 1000000000 |
      | lp3    | BTC   | 1000000000 |
      | party1 | BTC   | 100000000  |
      | party2 | BTC   | 100000000  |
      | lp1    | ETH   | 1000000000 |
      | lp2    | ETH   | 1000000000 |
      | lp3    | ETH   | 1000000000 |
      | party1 | ETH   | 100000000  |
      | party2 | BTC   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | BTC/ETH   | 10000             | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp1   | BTC/ETH | 4         | 1                    | buy  | MID              | 4          | 1      |
      | lp1   | BTC/ETH | 4         | 1                    | sell | MID              | 4          | 1      |

    # now another LP enters with a different fee but small commitment
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp2 | lp2   | BTC/ETH   | 1000              | 0.002 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp2   | BTC/ETH   | 4         | 1                    | buy  | MID              | 4          | 1      |
      | lp2   | BTC/ETH   | 4         | 1                    | sell | MID              | 4          | 1      |

     # now another LP enters with a big fee
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp3 | lp3   | BTC/ETH   | 3000              | 0.003 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp3   | BTC/ETH   | 4         | 1                    | buy  | MID              | 4          | 1      |
      | lp3   | BTC/ETH   | 4         | 1                    | sell | MID              | 4          | 1      |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/ETH   | buy  | 90     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH   | sell | 90     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"

    When the network moves ahead "1" blocks

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 90   | party2 |

    Then the liquidity fee factor should be "0.002" for the market "BTC/ETH"

    # LP3 now cancels
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type      |
      | lp3 | lp3   | BTC/ETH   | 3000              | 0.003 | cancellation |

    When the network moves ahead "1" epochs
    Then the liquidity fee factor should be "0.002" for the market "BTC/ETH"

    # the time window is 10s so from cancellation the target stake (and the calculated fee) should only drop after 10s
    When the network moves ahead "9" blocks
    Then the liquidity fee factor should be "0.002" for the market "BTC/ETH"

    When the network moves ahead "1" blocks
    Then the liquidity fee factor should be "0.001" for the market "BTC/ETH"

  Scenario: 004 Liquidity fee setting to METHOD_CONSTANT(0042-LIQF-070), METHOD_MARGINAL_COST (0042-LIQF-071), and METHOD_WEIGHTED_AVERAGE(0042-LIQF-069)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | BTC   | 1000000000 |
      | lp2    | BTC   | 1000000000 |
      | party1 | BTC   | 100000000  |
      | party2 | BTC   | 100000000  |
      | lp1    | ETH   | 1000000000 |
      | lp2    | ETH   | 1000000000 |
      | party1 | ETH   | 100000000 |
      | party2 | ETH   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | BTC/ETH | 800               | 0.005 | submission |
      | lp2 | lp2   | BTC/ETH | 300               | 0.004 | submission |

    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1   | BTC/ETH | 12        | 1                    | buy  | BID              | 12     | 2      |
      | lp1   | BTC/ETH | 12        | 1                    | buy  | MID              | 12     | 1      |
      | lp1   | BTC/ETH | 12        | 1                    | sell | ASK              | 12     | 2      |
      | lp1   | BTC/ETH | 12        | 1                    | sell | MID              | 12     | 1      |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | BTC/ETH | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | BTC/ETH | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "BTC/ETH"

    # update to method constant value 0.08 (0042-LIQF-070)
    When the spot markets are updated:
      | id        | liquidity monitoring | linear slippage factor | quadratic slippage factor | liquidity fee settings |
      | BTC/ETH   | lqm-params             | 1e-3                   | 0                       | fees-config-1          |

    When the network moves ahead "1" blocks

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 10   | party2 |

    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | min bound | max bound | target stake | supplied stake |
      | 1000       | TRADING_MODE_CONTINUOUS | 500       | 1500      | 825          | 1100           |

    And the liquidity fee factor should be "0.08" for the market "BTC/ETH"


    # update to constant 0.01
    When the spot markets are updated:
      | id        | liquidity monitoring | linear slippage factor | quadratic slippage factor | liquidity fee settings |
      | BTC/ETH | lqm-params             | 1e-3                   | 0                         | fees-config-2          |
    Then the network moves ahead "2" blocks
    And the liquidity fee factor should be "0.01" for the market "BTC/ETH"

    # update to weighted average (0042-LIQF-069)
    When the spot markets are updated:
      | id        | liquidity monitoring | linear slippage factor | quadratic slippage factor | liquidity fee settings |
      | BTC/ETH | lqm-params           | 1e-3                   | 0                         | fees-config-3          |
    Then the network moves ahead "1" blocks
    And the liquidity fee factor should be "0.0047272727272727" for the market "BTC/ETH"

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type   |
      | lp1 | lp1   | BTC/ETH | 800               | 0.006 | amendment |
    Then the network moves ahead "1" blocks
    And the liquidity fee factor should be "0.0054545454545455" for the market "BTC/ETH"

    # update to marginal cost (0042-LIQF-071)
    When the spot markets are updated:
      | id        | liquidity monitoring | linear slippage factor | quadratic slippage factor | liquidity fee settings |
      | BTC/ETH | lqm-params           | 1e-3                   | 0                         | fees-config-4          |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee    | lp type   |
      | lp1 | lp1   | BTC/ETH | 800               | 0.0375 | amendment |
    Then the network moves ahead "1" blocks
    And the liquidity fee factor should be "0.0375" for the market "BTC/ETH"