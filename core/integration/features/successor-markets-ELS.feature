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
        | name                                          | value |
        | network.markPriceUpdateMaximumFrequency       | 0s    |
        | market.liquidity.targetstake.triggering.ratio | 0.01  |
        | market.stake.target.timeWindow                | 10s   |
        | market.stake.target.scalingFactor             | 10    |
        | market.auction.minimumDuration                | 1     |
        | market.fee.factors.infrastructureFee          | 0.001 |
        | market.fee.factors.makerFee                   | 0.004 |
        | market.value.windowLength                     | 60s   |
        | market.liquidity.bondPenaltyParameter         | 0.1   |
        | market.liquidityProvision.shapes.maxSize      | 10    |
        | validators.epoch.length                       | 5s    |
        | market.liquidity.stakeToCcyVolume             | 0.2   |
	    | market.liquidity.successorLaunchWindowLength  | 1h    |
  
    And the average block duration is "1"
    # All parties have 1,000,000.000,000,000,000,000,000 
    # Add as many parties as needed here
    And the parties deposit on asset's general account the following amount:
      | party   | asset | amount      |
      | lpprov1 | USD   | 2000000000  |
      | lpprov2 | USD   | 20000000000 |
      | trader1 | USD   | 2000000     |
      | trader2 | USD   | 2000000     |
      | trader3 | USD   | 2000000     |
      | trader4 | USD   | 2000000     |
      | trader5 | USD   | 22000       |

  @SuccessorMarketSimple
  Scenario: 001 check ESL when an LP has virtual stake on parent market M1, and a new market M2 is proposed with M1 as its parent: 0042-LIQF-033,0042-LIQF-034,0042-LIQF-035
    Given the markets:
      | id        | quote name | asset | risk model                | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | decimal places | position decimal places | parent market id | insurance pool fraction | successor auction | lp price range |
      | ETH/DEC19 | ETH        | USD   | lognormal-risk-model-fish | margin-calculator-1       | 1                | default-none | default-none     | ethDec19Oracle         | 0.1                    | 0                         | 0              | 0                       |                  |                         |                   | 1              |
      | ETH/DEC20 | ETH        | USD   | default-st-risk-model     | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.1                    | 0                         | 0              | 0                       | ETH/DEC19        | 0.6                     | 10                | 1              |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov1 | ETH/DEC19 | 9000              | 0.1 | buy  | BID              | 10        | 100    | submission |
      | lp1 | lpprov1 | ETH/DEC19 | 9000              | 0.1 | sell | ASK              | 10        | 100    | submission |
      | lp2 | lpprov2 | ETH/DEC19 | 1000              | 0.1 | buy  | BID              | 10        | 100    | submission |
      | lp2 | lpprov2 | ETH/DEC19 | 1000              | 0.1 | sell | ASK              | 10        | 100    | submission |
    And the parties place the following orders:
      | party   | market id | side | volume | price  | resulting trades | type      | tif     | 
      | trader1 | ETH/DEC19 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC19 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
    When the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    Then the market data for the market "ETH/DEC19" should be:
        | mark price | trading mode            | auction trigger             | target stake | supplied stake   | open interest |
        | 150        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 731          | 10000            | 1             |
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
    And the network treasury balance should be "0" for the asset "USD"
    And the liquidity provider fee shares for the market "ETH/DEC19" should be:
      | party   | equity like share | average entry valuation |
      | lpprov1 | 0.9               | 9000                    |
      | lpprov2 | 0.1               | 10000                   |

     # make LP commitment while market is still pending
    And the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov1 | ETH/DEC20 | 2000              | 0.1 | buy  | BID              | 10         | 100    | submission |
      | lp1 | lpprov1 | ETH/DEC20 | 2000              | 0.1 | sell | ASK              | 10         | 100    | submission |
      | lp2 | lpprov2 | ETH/DEC20 | 8000              | 0.1 | buy  | BID              | 10         | 100    | submission |
      | lp2 | lpprov2 | ETH/DEC20 | 8000              | 0.1 | sell | ASK              | 10         | 100    | submission |
    
    Then the oracles broadcast data signed with "0xCAFECAFE1":
      | name               | value |
      | trading.terminated | true  |
      | prices.ETH.value   | 975   |

    And the insurance pool balance should be "0" for the market "ETH/DEC19"
    And the insurance pool balance should be "2539" for the market "ETH/DEC20"
    And the network treasury balance should be "2539" for the asset "USD"

    When the successor market "ETH/DEC20" is enacted
   
    Then the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC20 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC20 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
    When the opening auction period ends for market "ETH/DEC20"
    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 150        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 82           | 10000          | 1             |

    # this is from ETH/DEC19 market
    And the liquidity provider fee shares for the market "ETH/DEC20" should be:
      | party   | equity like share | average entry valuation |
      | lpprov1 | 0.2               | 9000                    |
      | lpprov2 | 0.8               | 10000                   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC20"

    And the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type   |
      | lp1 | lpprov1 | ETH/DEC20 | 3000              | 0.1 | buy  | BID              | 10         | 100    | amendment |
      | lp1 | lpprov1 | ETH/DEC20 | 3000              | 0.1 | sell | ASK              | 10         | 100    | amendment |

    And the liquidity provider fee shares for the market "ETH/DEC20" should be:
      | party   | equity like share  | average entry valuation |
      | lpprov1 | 0.2727272727272727 | 9666.6666666666666      |
      | lpprov2 | 0.7272727272727273 | 10000                   |

 Scenario: 002 check ESL when an LP has virtual stake on parent market M1, and a new market M2 is proposed with M1 as its parent: 0042-LIQF-034
    Given the markets:
      | id        | quote name | asset | risk model                | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | decimal places | position decimal places | parent market id | insurance pool fraction | successor auction | lp price range |
      | ETH/DEC19 | ETH        | USD   | lognormal-risk-model-fish | margin-calculator-1       | 1                | default-none | default-none     | ethDec19Oracle         | 0.1                    | 0                         | 0              | 0                       |                  |                         |                   | 1              |
      | ETH/DEC20 | ETH        | USD   | default-st-risk-model     | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.1                    | 0                         | 0              | 0                       | ETH/DEC19        | 0.9                     | 10                | 1              |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov1 | ETH/DEC19 | 9000              | 0.1 | buy  | BID              | 10        | 100    | submission |
      | lp1 | lpprov1 | ETH/DEC19 | 9000              | 0.1 | sell | ASK              | 10        | 100    | submission |
      | lp2 | lpprov2 | ETH/DEC19 | 1000              | 0.1 | buy  | BID              | 10        | 100    | submission |
      | lp2 | lpprov2 | ETH/DEC19 | 1000              | 0.1 | sell | ASK              | 10        | 100    | submission |
    And the parties place the following orders:
      | party   | market id | side | volume | price  | resulting trades | type      | tif     | 
      | trader1 | ETH/DEC19 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC19 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
    When the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    Then the market data for the market "ETH/DEC19" should be:
        | mark price | trading mode            | auction trigger             | target stake | supplied stake   | open interest |
        | 150        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 731          | 10000            | 1             |
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
    And the network treasury balance should be "0" for the asset "USD"
    And the liquidity provider fee shares for the market "ETH/DEC19" should be:
      | party   | equity like share | average entry valuation |
      | lpprov1 | 0.9               | 9000                    |
      | lpprov2 | 0.1               | 10000                   |

     # make LP commitment while market is still pending
    And the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov1 | ETH/DEC20 | 2000              | 0.1 | buy  | BID              | 10         | 100    | submission |
      | lp1 | lpprov1 | ETH/DEC20 | 2000              | 0.1 | sell | ASK              | 10         | 100    | submission |
      | lp2 | lpprov2 | ETH/DEC20 | 8000              | 0.1 | buy  | BID              | 10         | 100    | submission |
      | lp2 | lpprov2 | ETH/DEC20 | 8000              | 0.1 | sell | ASK              | 10         | 100    | submission |
    
    # Then the oracles broadcast data signed with "0xCAFECAFE1":
    #   | name               | value |
    #   | trading.terminated | true  |
    #   | prices.ETH.value   | 975   |

    And the insurance pool balance should be "5077" for the market "ETH/DEC19"
    And the insurance pool balance should be "0" for the market "ETH/DEC20"
    And the network treasury balance should be "0" for the asset "USD"

    When the successor market "ETH/DEC20" is enacted
   
    Then the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC20 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC20 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
    When the opening auction period ends for market "ETH/DEC20"
    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 150        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 82           | 10000          | 1             |

    # this is from ETH/DEC19 market
    And the liquidity provider fee shares for the market "ETH/DEC20" should be:
      | party   | equity like share | average entry valuation |
      | lpprov1 | 0.2               | 9000                    |
      | lpprov2 | 0.8               | 10000                   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC20"

    And the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type   |
      | lp1 | lpprov1 | ETH/DEC20 | 3000              | 0.1 | buy  | BID              | 10         | 100    | amendment |
      | lp1 | lpprov1 | ETH/DEC20 | 3000              | 0.1 | sell | ASK              | 10         | 100    | amendment |

    And the liquidity provider fee shares for the market "ETH/DEC20" should be:
      | party   | equity like share  | average entry valuation |
      | lpprov1 | 0.2727272727272727 | 9666.6666666666666      |
      | lpprov2 | 0.7272727272727273 | 10000                   |

    And the insurance pool balance should be "508" for the market "ETH/DEC19"
    And the insurance pool balance should be "0" for the market "ETH/DEC20"
    And the network treasury balance should be "0" for the asset "USD"

 
