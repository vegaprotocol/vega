Feature: Simple example of successor markets

  Background:
    Given time is updated to "2019-11-30T00:00:00Z"
    And the following assets are registered:
      | id  | decimal places |
      | ETH | 0              |
      | USD | 0              |
    Given the log normal risk model named "lognormal-risk-model-fish":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |
    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 2              |

    # Create some oracles
    ## oracle for parent
    And the oracle spec for settlement data filtering data from "0xCAFECAFE1" named "ethDec19Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |
    And the oracle spec for trading termination filtering data from "0xCAFECAFE1" named "ethDec19Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |
    And the settlement data decimals for the oracle named "ethDec19Oracle" is given in "5" decimal places
    ## oracle for a successor
    And the oracle spec for settlement data filtering data from "0xCAFECAFE" named "ethDec20Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |
    And the oracle spec for trading termination filtering data from "0xCAFECAFE" named "ethDec20Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |
    And the settlement data decimals for the oracle named "ethDec20Oracle" is given in "5" decimal places

    And the following network parameters are set:
      | name                                                | value |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
      | market.liquidity.targetstake.triggering.ratio       | 0.01  |
      | market.stake.target.timeWindow                      | 10s   |
      | market.stake.target.scalingFactor                   | 10    |
      | market.auction.minimumDuration                      | 1     |
      | market.fee.factors.infrastructureFee                | 0.001 |
      | market.fee.factors.makerFee                         | 0.004 |
      | market.value.windowLength                           | 60s   |
      | market.liquidityV2.bondPenaltyParameter             | 0.1   |
      | validators.epoch.length                             | 5s    |
      | market.liquidityV2.stakeToCcyVolume                 | 0.2   |
      | market.liquidity.successorLaunchWindowLength        | 1h    |
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | providers fee calculation time step | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 20                                  | 1                             | 1.0                    |
    And the average block duration is "1"
# All parties have 1,000,000.000,000,000,000,000,000
    # Add as many parties as needed here
    And the parties deposit on asset's general account the following amount:
      | party   | asset | amount      |
      | lpprov1 | USD   | 2000000000  |
      | lpprov2 | USD   | 20000000000 |
      | lpprov3 | USD   | 20000000000 |
      | lpprov4 | USD   | 20000000000 |
      | trader1 | USD   | 2000000     |
      | trader2 | USD   | 2000000     |
      | trader3 | USD   | 2000000     |
      | trader4 | USD   | 2000000     |
      | trader5 | USD   | 22000       |

  @SuccessorMarketActive
  Scenario: 001 Enact a successor market when the parent market is still active; Two proposals that name the same parent can be submitted; 0081-SUCM-005, 0081-SUCM-006, 0081-SUCM-020, 0081-SUCM-021, 0081-SUCM-022
    Given the markets:
      | id        | quote name | asset | risk model                | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | decimal places | position decimal places | parent market id | insurance pool fraction | successor auction | sla params |
      | ETH/DEC19 | ETH        | USD   | lognormal-risk-model-fish | margin-calculator-1       | 1                | default-none | default-none     | ethDec19Oracle         | 0.1                    | 0                         | 0              | 0                       |                  |                         |                   | SLA        |
      | ETH/DEC20 | ETH        | USD   | default-st-risk-model     | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.1                    | 0                         | 0              | 0                       | ETH/DEC19        | 0.6                     | 10                | SLA        |
      | ETH/DEC21 | ETH        | USD   | default-st-risk-model     | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.1                    | 0                         | 0              | 0                       | ETH/DEC19        | 0.6                     | 10                | SLA        |
    And the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov1 | ETH/DEC19 | 9000              | 0.1 | submission |
      | lp1 | lpprov1 | ETH/DEC19 | 9000              | 0.1 | submission |
      | lp2 | lpprov2 | ETH/DEC19 | 1000              | 0.1 | submission |
      | lp2 | lpprov2 | ETH/DEC19 | 1000              | 0.1 | submission |
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type | tif |
      | trader1 | ETH/DEC19 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC19 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
    When the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 150        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 731          | 10000          | 1             |
    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | trader4 | ETH/DEC19 | sell | 290    | 150   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | trader3 | ETH/DEC19 | buy  | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |

    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 150        | TRADING_MODE_CONTINUOUS | 731          | 10000          | 1             |
    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader5 | ETH/DEC19 | buy  | 290    | 150   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader5 | USD   | ETH/DEC19 | 17432  | 0       |

    Then the parties cancel the following orders:
      | party   | reference      |
      | trader3 | buy-provider-1 |
    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | trader3 | ETH/DEC19 | buy  | 290    | 120   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2 |

    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader4 | ETH/DEC19 | sell | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader3 | ETH/DEC19 | buy  | 1      | 140   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the insurance pool balance should be "5077" for the market "ETH/DEC19"
    And the global insurance pool balance should be "0" for the asset "USD"
    And the liquidity provider fee shares for the market "ETH/DEC19" should be:
      | party   | equity like share | average entry valuation |
      | lpprov1 | 0.9               | 9000                    |
      | lpprov2 | 0.1               | 10000                   |

# make LP commitment while market is still pending
    And the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov1 | ETH/DEC20 | 2000              | 0.1 | submission |
      | lp1 | lpprov1 | ETH/DEC20 | 2000              | 0.1 | submission |
      | lp2 | lpprov2 | ETH/DEC20 | 8000              | 0.1 | submission |
      | lp2 | lpprov2 | ETH/DEC20 | 8000              | 0.1 | submission |
      | lp3 | lpprov3 | ETH/DEC21 | 8000              | 0.1 | submission |
      | lp3 | lpprov3 | ETH/DEC21 | 8000              | 0.1 | submission |

#check LP bond account after LP commitment submission
    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general     | bond |
# | lpprov1 | USD   | ETH/DEC19 | 127335 | 1999861665  | 9000 |
# | lpprov2 | USD   | ETH/DEC20 | 0      | 19999976851 | 8000 |
      | lpprov3 | USD   | ETH/DEC21 | 0      | 19999992000 | 8000 |

# market ETH/DEC19 is not settled yet, it still active
    And the insurance pool balance should be "5077" for the market "ETH/DEC19"
    And the insurance pool balance should be "0" for the market "ETH/DEC20"
    And the global insurance pool balance should be "0" for the asset "USD"

    When the successor market "ETH/DEC20" is enacted
    When the successor market "ETH/DEC21" is enacted

    Then the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | trader1 | ETH/DEC20 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC |              |
      | trader1 | ETH/DEC20 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |              |
      | trader1 | ETH/DEC21 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC | order1-DEC21 |
      | trader1 | ETH/DEC21 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC | order2-DEC21 |
      | trader1 | ETH/DEC20 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |              |
      | trader2 | ETH/DEC20 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |              |
    When the opening auction period ends for market "ETH/DEC20"
    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 150        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 82           | 10000          | 1             |

    And the last market state should be "STATE_REJECTED" for the market "ETH/DEC21"

    #check assets held to support trader1's orders in market ETH/DEC21 is released
    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader1 | USD   | ETH/DEC20 | 122    | 1998780 |
      | trader1 | USD   | ETH/DEC21 | 0      | 1998780 |

    #check all the orders in market ETH/DEC21 is canceled
    And the orders should have the following status:
      | party   | reference    | status        |
      | trader1 | order1-DEC21 | STATUS_STOPPED |
      | trader1 | order2-DEC21 | STATUS_STOPPED |

    And the insurance pool balance should be "2031" for the market "ETH/DEC19"
    And the insurance pool balance should be "3046" for the market "ETH/DEC20"
    And the insurance pool balance should be "0" for the market "ETH/DEC21"
    And the global insurance pool balance should be "0" for the asset "USD"

    # check LP account is released after the market ETH/DEC21 is rejceted
    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general     |
      | lpprov3 | USD   | ETH/DEC21 | 0      | 20000000000 |

    # this is from ETH/DEC19 market
    And the liquidity provider fee shares for the market "ETH/DEC20" should be:
      | party   | equity like share | average entry valuation |
      # | lpprov1 | 0.2               | 9000                    |
      # | lpprov2 | 0.8               | 10000                   |
      | lpprov1 | 0.9 | 9000  |
      | lpprov2 | 0.1 | 10000 |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC20"

    When the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | lp type   |
      | lp2 | lpprov2 | ETH/DEC19 | 2000              | 0.1 | amendment |
      | lp2 | lpprov2 | ETH/DEC19 | 2000              | 0.1 | amendment |
    Then the liquidity provider fee shares for the market "ETH/DEC19" should be:
      | party   | equity like share  | average entry valuation |
      # | lpprov1 | 0.8181818181818182 | 9000                    |
      # | lpprov2 | 0.1818181818181818 | 10500                   |
      | lpprov1 | 0.9 | 9000  |
      | lpprov2 | 0.1 | 10000 |
    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name               | value |
      | trading.terminated | true  |
      | prices.ETH.value   | 976   |
    Then the market state should be "STATE_SETTLED" for the market "ETH/DEC19"

    And the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | lp type   |
      | lp1 | lpprov1 | ETH/DEC20 | 3000              | 0.1 | amendment |
      | lp1 | lpprov1 | ETH/DEC20 | 3000              | 0.1 | amendment |

    And the liquidity provider fee shares for the market "ETH/DEC20" should be:
      | party   | equity like share  | average entry valuation |
      | lpprov1 | 0.9 | 9000  |
      | lpprov2 | 0.1 | 10000 |

    When the network moves ahead "1" blocks
# Then the insurance pool balance should be "0" for the market "ETH/DEC19"
# And the insurance pool balance should be "4062" for the market "ETH/DEC20"
# And the global insurance pool balance should be "1016" for the asset "USD"

#   @SuccessorMarketSimple
#   Scenario: 002 Successor market enacted with parent market still active, ELS is copied over and both states can change independently. 0042-LIQF-031, 0042-LIQF-048, 0042-LIQF-033
#     Given the markets:
#       | id        | quote name | asset | risk model                | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | decimal places | position decimal places | parent market id | insurance pool fraction | successor auction | sla params |
#       | ETH/DEC19 | ETH        | USD   | lognormal-risk-model-fish | margin-calculator-1       | 1                | default-none | default-none     | ethDec19Oracle         | 0.1                    | 0                         | 0              | 0                       |                  |                         |                   | SLA        |
#       | ETH/DEC20 | ETH        | USD   | default-st-risk-model     | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.1                    | 0                         | 0              | 0                       | ETH/DEC19        | 0.6                     | 10                | SLA        |
#     And the parties submit the following liquidity provision:
#       | id  | party   | market id | commitment amount | fee | lp type    |
#       | lp1 | lpprov1 | ETH/DEC19 | 9000              | 0.1 | submission |
#       | lp1 | lpprov1 | ETH/DEC19 | 9000              | 0.1 | submission |
#       | lp2 | lpprov2 | ETH/DEC19 | 1000              | 0.1 | submission |
#       | lp2 | lpprov2 | ETH/DEC19 | 1000              | 0.1 | submission |
#     And the parties place the following orders:
#       | party | market id | side | volume | price | resulting trades | type | tif |
#       | trader1 | ETH/DEC19 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
#       | trader1 | ETH/DEC19 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
#       | trader1 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
#       | trader2 | ETH/DEC19 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
#     When the opening auction period ends for market "ETH/DEC19"
#     And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
#     Then the market data for the market "ETH/DEC19" should be:
#       | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
#       | 150        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 731          | 10000          | 1             |
#     When the parties place the following orders with ticks:
#       | party   | market id | side | volume | price | resulting trades | type       | tif     | reference       |
#       | trader4 | ETH/DEC19 | sell | 290    | 150   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
#       | trader3 | ETH/DEC19 | buy  | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |

#     And the market data for the market "ETH/DEC19" should be:
#       | mark price | trading mode            | target stake | supplied stake | open interest |
#       | 150        | TRADING_MODE_CONTINUOUS | 731          | 10000          | 1             |
#     When the parties place the following orders with ticks:
#       | party   | market id | side | volume | price | resulting trades | type       | tif     | reference |
#       | trader5 | ETH/DEC19 | buy  | 290    | 150   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |

#     Then the parties should have the following account balances:
#       | party   | asset | market id | margin | general |
#       | trader5 | USD   | ETH/DEC19 | 17432  | 0       |

#     Then the parties cancel the following orders:
#       | party   | reference      |
#       | trader3 | buy-provider-1 |
#     When the parties place the following orders with ticks:
#       | party   | market id | side | volume | price | resulting trades | type       | tif     | reference      |
#       | trader3 | ETH/DEC19 | buy  | 290    | 120   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2 |

#     When the parties place the following orders with ticks:
#       | party   | market id | side | volume | price | resulting trades | type       | tif     | reference |
#       | trader4 | ETH/DEC19 | sell | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
#       | trader3 | ETH/DEC19 | buy  | 1      | 140   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

#     And the insurance pool balance should be "5077" for the market "ETH/DEC19"
#     And the global insurance pool balance should be "0" for the asset "USD"
#     And the liquidity provider fee shares for the market "ETH/DEC19" should be:
#       | party   | equity like share | average entry valuation |
#       | lpprov1 | 0.9               | 9000                    |
#       | lpprov2 | 0.1               | 10000                   |

# # make LP commitment while market is still pending
#     And the parties submit the following liquidity provision:
#       | id  | party   | market id | commitment amount | fee | lp type    |
#       | lp1 | lpprov1 | ETH/DEC20 | 4000              | 0.1 | submission |
#       | lp1 | lpprov1 | ETH/DEC20 | 4000              | 0.1 | submission |
#       | lp2 | lpprov2 | ETH/DEC20 | 8000              | 0.1 | submission |
#       | lp2 | lpprov2 | ETH/DEC20 | 8000              | 0.1 | submission |

#     Then the oracles broadcast data signed with "0xCAFECAFE1":
#       | name               | value |
#       | trading.terminated | true  |
#       | prices.ETH.value   | 975   |

#     # pass succession window
#     When the network moves ahead "1" blocks

#     Then the successor market "ETH/DEC20" is enacted

#     And the parties place the following orders:
#       | party   | market id | side | volume | price | resulting trades | type       | tif     |
#       | trader1 | ETH/DEC20 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
#       | trader1 | ETH/DEC20 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
#       | trader1 | ETH/DEC20 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
#       | trader2 | ETH/DEC20 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |

#     # make LP commitment change  while market is still pending
#     And the parties submit the following liquidity provision:
#       | id  | party   | market id | commitment amount | fee | lp type   |
#       | lp1 | lpprov1 | ETH/DEC20 | 2000              | 0.1 | amendment |
#       | lp1 | lpprov1 | ETH/DEC20 | 2000              | 0.1 | amendment |
#     When the opening auction period ends for market "ETH/DEC20"
#     Then the market data for the market "ETH/DEC20" should be:
#       | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
#       | 150        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 82           | 10000          | 1             |

#     And the insurance pool balance should be "0" for the market "ETH/DEC19"
#     And the insurance pool balance should be "4062" for the market "ETH/DEC20"
#     And the global insurance pool balance should be "1016" for the asset "USD"
#     # this is from ETH/DEC19 market
#     And the liquidity provider fee shares for the market "ETH/DEC20" should be:
#       | party   | equity like share | average entry valuation |
#       | lpprov1 | 0.2               | 9000                    |
#       | lpprov2 | 0.8               | 10000                   |

#     And the accumulated liquidity fees should be "0" for the market "ETH/DEC20"

#     And the parties submit the following liquidity provision:
#       | id  | party   | market id | commitment amount | fee | lp type   |
#       | lp1 | lpprov1 | ETH/DEC20 | 3000              | 0.1 | amendment |
#       | lp1 | lpprov1 | ETH/DEC20 | 3000              | 0.1 | amendment |

#     And the liquidity provider fee shares for the market "ETH/DEC20" should be:
#       | party   | equity like share  | average entry valuation |
#       | lpprov1 | 0.2727272727272727 | 9666.6666666666666      |
#       | lpprov2 | 0.7272727272727273 | 10000                   |


#   @SuccessorMarketActive
#   Scenario: 003 Enact a successor market while the parent is still active. Pending successors get rejected
#     Given the markets:
#       | id        | quote name | asset | risk model                | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | decimal places | position decimal places | parent market id | insurance pool fraction | successor auction | sla params |
#       | ETH/DEC19 | ETH        | USD   | lognormal-risk-model-fish | margin-calculator-1       | 1                | default-none | default-none     | ethDec19Oracle         | 0.1                    | 0                         | 0              | 0                       |                  |                         |                   | SLA        |
#       | ETH/DEC20 | ETH        | USD   | default-st-risk-model     | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.1                    | 0                         | 0              | 0                       | ETH/DEC19        | 0.6                     | 10                | SLA        |
#       | ETH/DEC21 | ETH        | USD   | default-st-risk-model     | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.1                    | 0                         | 0              | 0                       | ETH/DEC19        | 0.1                     | 10                | SLA        |
#     And the parties submit the following liquidity provision:
#       | id  | party   | market id | commitment amount | fee | lp type    |
#       | lp1 | lpprov1 | ETH/DEC19 | 9000              | 0.1 | submission |
#       | lp1 | lpprov1 | ETH/DEC19 | 9000              | 0.1 | submission |
#       | lp2 | lpprov2 | ETH/DEC19 | 1000              | 0.1 | submission |
#       | lp2 | lpprov2 | ETH/DEC19 | 1000              | 0.1 | submission |
#     And the parties place the following orders:
#       | party | market id | side | volume | price | resulting trades | type | tif |
#       | trader1 | ETH/DEC19 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
#       | trader1 | ETH/DEC19 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
#       | trader1 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
#       | trader2 | ETH/DEC19 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
#     When the opening auction period ends for market "ETH/DEC19"
#     And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
#     Then the market data for the market "ETH/DEC19" should be:
#       | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
#       | 150        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 731          | 10000          | 1             |
#     When the parties place the following orders with ticks:
#       | party   | market id | side | volume | price | resulting trades | type       | tif     | reference       |
#       | trader4 | ETH/DEC19 | sell | 290    | 150   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
#       | trader3 | ETH/DEC19 | buy  | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |

#     And the market data for the market "ETH/DEC19" should be:
#       | mark price | trading mode            | target stake | supplied stake | open interest |
#       | 150        | TRADING_MODE_CONTINUOUS | 731          | 10000          | 1             |
#     When the parties place the following orders with ticks:
#       | party   | market id | side | volume | price | resulting trades | type       | tif     | reference |
#       | trader5 | ETH/DEC19 | buy  | 290    | 150   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |

#     Then the parties should have the following account balances:
#       | party   | asset | market id | margin | general |
#       | trader5 | USD   | ETH/DEC19 | 17432  | 0       |

#     Then the parties cancel the following orders:
#       | party   | reference      |
#       | trader3 | buy-provider-1 |
#     When the parties place the following orders with ticks:
#       | party   | market id | side | volume | price | resulting trades | type       | tif     | reference      |
#       | trader3 | ETH/DEC19 | buy  | 290    | 120   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2 |

#     When the parties place the following orders with ticks:
#       | party   | market id | side | volume | price | resulting trades | type       | tif     | reference |
#       | trader4 | ETH/DEC19 | sell | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
#       | trader3 | ETH/DEC19 | buy  | 1      | 140   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

#     And the insurance pool balance should be "5077" for the market "ETH/DEC19"
#     And the global insurance pool balance should be "0" for the asset "USD"
#     And the liquidity provider fee shares for the market "ETH/DEC19" should be:
#       | party   | equity like share | average entry valuation |
#       | lpprov1 | 0.9               | 9000                    |
#       | lpprov2 | 0.1               | 10000                   |

# # make LP commitment while market is still pending
#     And the parties submit the following liquidity provision:
#       | id  | party   | market id | commitment amount | fee | lp type    |
#       | lp1 | lpprov1 | ETH/DEC20 | 2000              | 0.1 | submission |
#       | lp1 | lpprov1 | ETH/DEC20 | 2000              | 0.1 | submission |
#       | lp2 | lpprov2 | ETH/DEC20 | 8000              | 0.1 | submission |
#       | lp2 | lpprov2 | ETH/DEC20 | 8000              | 0.1 | submission |

# # market ETH/DEC19 is not settled yet, it still active

#     And the insurance pool balance should be "5077" for the market "ETH/DEC19"
#     And the insurance pool balance should be "0" for the market "ETH/DEC20"
#     And the global insurance pool balance should be "0" for the asset "USD"
#     When the successor market "ETH/DEC20" is enacted

#     Then the parties place the following orders:
#       | party   | market id | side | volume | price | resulting trades | type       | tif     |
#       | trader1 | ETH/DEC20 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
#       | trader1 | ETH/DEC20 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
#       | trader1 | ETH/DEC20 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
#       | trader2 | ETH/DEC20 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
#     When the opening auction period ends for market "ETH/DEC20"
#     Then the last market state should be "STATE_REJECTED" for the market "ETH/DEC21"
#     And the market data for the market "ETH/DEC20" should be:
#       | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
#       | 150        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 82           | 10000          | 1             |
#     And the insurance pool balance should be "2031" for the market "ETH/DEC19"
#     And the insurance pool balance should be "3046" for the market "ETH/DEC20"
#     And the global insurance pool balance should be "0" for the asset "USD"

#     # this is from ETH/DEC19 market
#     And the liquidity provider fee shares for the market "ETH/DEC20" should be:
#       | party   | equity like share | average entry valuation |
#       | lpprov1 | 0.2               | 9000                    |
#       | lpprov2 | 0.8               | 10000                   |

#     And the accumulated liquidity fees should be "0" for the market "ETH/DEC20"

#     When the parties submit the following liquidity provision:
#       | id  | party   | market id | commitment amount | fee | lp type   |
#       | lp2 | lpprov2 | ETH/DEC19 | 2000              | 0.1 | amendment |
#       | lp2 | lpprov2 | ETH/DEC19 | 2000              | 0.1 | amendment |
#     Then the liquidity provider fee shares for the market "ETH/DEC19" should be:
#       | party   | equity like share  | average entry valuation |
#       | lpprov1 | 0.8181818181818182 | 9000                    |
#       | lpprov2 | 0.1818181818181818 | 10500                   |
#     When the oracles broadcast data signed with "0xCAFECAFE1":
#       | name               | value |
#       | trading.terminated | true  |
#       | prices.ETH.value   | 976   |
#     Then the market state should be "STATE_SETTLED" for the market "ETH/DEC19"

#     And the parties submit the following liquidity provision:
#       | id  | party   | market id | commitment amount | fee | lp type   |
#       | lp1 | lpprov1 | ETH/DEC20 | 3000              | 0.1 | amendment |
#       | lp1 | lpprov1 | ETH/DEC20 | 3000              | 0.1 | amendment |

#     And the liquidity provider fee shares for the market "ETH/DEC20" should be:
#       | party   | equity like share  | average entry valuation |
#       | lpprov1 | 0.2727272727272727 | 9666.6666666666666      |
#       | lpprov2 | 0.7272727272727273 | 10000                   |
#     When the network moves ahead "1" blocks
#     Then the insurance pool balance should be "0" for the market "ETH/DEC19"
#     And the insurance pool balance should be "4062" for the market "ETH/DEC20"
#     And the global insurance pool balance should be "1016" for the asset "USD"


#   @SuccessorMarketExpires2
#   Scenario: 004 Enact a successor market while the parent is still active. Pending successors get rejected
#     Given the following network parameters are set:
#       | name                                         | value |
#       | market.liquidity.successorLaunchWindowLength | 1s    |
#     And the markets:
#       | id        | quote name | asset | risk model                | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | decimal places | position decimal places | parent market id | insurance pool fraction | successor auction | sla params |
#       | ETH/DEC19 | ETH        | USD   | lognormal-risk-model-fish | margin-calculator-1       | 1                | default-none | default-none     | ethDec19Oracle         | 0.1                    | 0                         | 0              | 0                       |                  |                         |                   | SLA        |
#       | ETH/DEC20 | ETH        | USD   | default-st-risk-model     | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.1                    | 0                         | 0              | 0                       | ETH/DEC19        | 0.6                     | 10                | SLA        |
#     And the parties submit the following liquidity provision:
#       | id  | party   | market id | commitment amount | fee | lp type    |
#       | lp1 | lpprov1 | ETH/DEC19 | 9000              | 0.1 | submission |
#       | lp1 | lpprov1 | ETH/DEC19 | 9000              | 0.1 | submission |
#       | lp2 | lpprov2 | ETH/DEC19 | 1000              | 0.1 | submission |
#       | lp2 | lpprov2 | ETH/DEC19 | 1000              | 0.1 | submission |
#     And the parties place the following orders:
#       | party | market id | side | volume | price | resulting trades | type | tif |
#       | trader1 | ETH/DEC19 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
#       | trader1 | ETH/DEC19 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
#       | trader1 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
#       | trader2 | ETH/DEC19 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
#     When the opening auction period ends for market "ETH/DEC19"
#     And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
#     Then the market data for the market "ETH/DEC19" should be:
#       | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
#       | 150        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 731          | 10000          | 1             |
#     When the parties place the following orders with ticks:
#       | party   | market id | side | volume | price | resulting trades | type       | tif     | reference       |
#       | trader4 | ETH/DEC19 | sell | 290    | 150   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
#       | trader3 | ETH/DEC19 | buy  | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |

#     And the market data for the market "ETH/DEC19" should be:
#       | mark price | trading mode            | target stake | supplied stake | open interest |
#       | 150        | TRADING_MODE_CONTINUOUS | 731          | 10000          | 1             |
#     When the parties place the following orders with ticks:
#       | party   | market id | side | volume | price | resulting trades | type       | tif     | reference |
#       | trader5 | ETH/DEC19 | buy  | 290    | 150   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |

#     Then the parties should have the following account balances:
#       | party   | asset | market id | margin | general |
#       | trader5 | USD   | ETH/DEC19 | 17432  | 0       |

#     Then the parties cancel the following orders:
#       | party   | reference      |
#       | trader3 | buy-provider-1 |
#     When the parties place the following orders with ticks:
#       | party   | market id | side | volume | price | resulting trades | type       | tif     | reference      |
#       | trader3 | ETH/DEC19 | buy  | 290    | 120   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2 |

#     When the parties place the following orders with ticks:
#       | party   | market id | side | volume | price | resulting trades | type       | tif     | reference |
#       | trader4 | ETH/DEC19 | sell | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
#       | trader3 | ETH/DEC19 | buy  | 1      | 140   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

#     Then the insurance pool balance should be "5077" for the market "ETH/DEC19"
#     And the global insurance pool balance should be "0" for the asset "USD"
#     And the liquidity provider fee shares for the market "ETH/DEC19" should be:
#       | party   | equity like share | average entry valuation |
#       | lpprov1 | 0.9               | 9000                    |
#       | lpprov2 | 0.1               | 10000                   |

#     And the insurance pool balance should be "5077" for the market "ETH/DEC19"
#     And the insurance pool balance should be "0" for the market "ETH/DEC20"
#     And the global insurance pool balance should be "0" for the asset "USD"

#     When the oracles broadcast data signed with "0xCAFECAFE1":
#       | name               | value |
#       | trading.terminated | true  |
#       | prices.ETH.value   | 976   |
#     Then the market state should be "STATE_SETTLED" for the market "ETH/DEC19"
#     And the successor market "ETH/DEC20" is enacted

#     When the network moves ahead "5" blocks
# # make LP commitment while market is still pending
#     Then the parties submit the following liquidity provision:
#       | id  | party   | market id | commitment amount | fee | lp type    |
#       | lp1 | lpprov1 | ETH/DEC20 | 2000              | 0.1 | submission |
#       | lp1 | lpprov1 | ETH/DEC20 | 2000              | 0.1 | submission |
#       | lp2 | lpprov2 | ETH/DEC20 | 8000              | 0.1 | submission |
#       | lp2 | lpprov2 | ETH/DEC20 | 8000              | 0.1 | submission |

#     And the parties place the following orders:
#       | party   | market id | side | volume | price | resulting trades | type       | tif     |
#       | trader1 | ETH/DEC20 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
#       | trader1 | ETH/DEC20 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
#       | trader1 | ETH/DEC20 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
#       | trader2 | ETH/DEC20 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
#     When the opening auction period ends for market "ETH/DEC20"
#     Then the market data for the market "ETH/DEC20" should be:
#       | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
#       | 150        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 82           | 10000          | 1             |
#     And the insurance pool balance should be "0" for the market "ETH/DEC19"
#     And the insurance pool balance should be "2539" for the market "ETH/DEC20"
#     And the global insurance pool balance should be "2539" for the asset "USD"

#     # this is from ETH/DEC19 market
#     And the liquidity provider fee shares for the market "ETH/DEC20" should be:
#       | party   | equity like share | average entry valuation |
#       | lpprov1 | 0.2               | 2000                    |
#       | lpprov2 | 0.8               | 10000                   |

#     And the accumulated liquidity fees should be "0" for the market "ETH/DEC20"

#     And the parties submit the following liquidity provision:
#       | id  | party   | market id | commitment amount | fee | lp type   |
#       | lp1 | lpprov1 | ETH/DEC20 | 3000              | 0.1 | amendment |
#       | lp1 | lpprov1 | ETH/DEC20 | 3000              | 0.1 | amendment |

#     And the liquidity provider fee shares for the market "ETH/DEC20" should be:
#       | party   | equity like share  | average entry valuation |
#       | lpprov1 | 0.2727272727272727 | 5000                    |
#       | lpprov2 | 0.7272727272727273 | 10000                   |
#     When the network moves ahead "1" blocks
#     Then the insurance pool balance should be "0" for the market "ETH/DEC19"
#     And the insurance pool balance should be "2539" for the market "ETH/DEC20"
#     And the global insurance pool balance should be "2539" for the asset "USD"

