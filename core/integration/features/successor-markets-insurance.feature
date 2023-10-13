Feature: Successor markets: Global insurance pool account collects all outstanding funds from closed/expired markets in a risk universe (0013-ACCT-032)

  Background:
    Given time is updated to "2019-11-30T00:00:00Z"
    And the following assets are registered:
      | id  | decimal places |
      | ETH | 1              |
      | USD | 1              |

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
    And the settlement data decimals for the oracle named "ethDec19Oracle" is given in "0" decimal places

    ## oracle for a successor 1
    And the oracle spec for settlement data filtering data from "0xCAFECAAA" named "ethDec20Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |
    And the oracle spec for trading termination filtering data from "0xCAFECAAA" named "ethDec20Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |
    And the settlement data decimals for the oracle named "ethDec20Oracle" is given in "0" decimal places

    ## oracle for a successor 2
    And the oracle spec for settlement data filtering data from "0xCAFECABB" named "ethDec21Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |
    And the oracle spec for trading termination filtering data from "0xCAFECABB" named "ethDec21Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |
    And the settlement data decimals for the oracle named "ethDec21Oracle" is given in "0" decimal places

    ## oracle for a successor 3
    And the oracle spec for settlement data filtering data from "0xCAFECACC" named "ethDec22Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |
    And the oracle spec for trading termination filtering data from "0xCAFECACC" named "ethDec22Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |
    And the settlement data decimals for the oracle named "ethDec22Oracle" is given in "0" decimal places

    ## oracle for a successor 4
    And the oracle spec for settlement data filtering data from "0xCAFECADD" named "ethDec23Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |
    And the oracle spec for trading termination filtering data from "0xCAFECADD" named "ethDec23Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |
    And the settlement data decimals for the oracle named "ethDec23Oracle" is given in "0" decimal places

    And the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | lqm-params         | 0.01             | 10s         | 5              |  
    
    And the following network parameters are set:
      | name                                          | value |
      | network.markPriceUpdateMaximumFrequency       | 0s    |
      | market.auction.minimumDuration                | 1     |
      | market.fee.factors.infrastructureFee          | 0.001 |
      | market.fee.factors.makerFee                   | 0.004 |
      | market.value.windowLength                     | 60s   |
      | market.liquidity.bondPenaltyParameter         | 0.1   |
      | validators.epoch.length                       | 5s    |
      | market.liquidity.stakeToCcyVolume             | 0.2   |
      | market.liquidity.successorLaunchWindowLength  | 8s    |
      | limits.markets.maxPeggedOrders                | 2     |
    And the average block duration is "1"


    # All parties have 1,000,000.000,000,000,000,000,000
    # Add as many parties as needed here
    And the parties deposit on asset's general account the following amount:
      | party   | asset | amount                     |
      | lpprov1 | USD   | 10000000000000000000000000 |
      | lpprov2 | USD   | 10000000000000000000000000 |
      | trader1 | USD   | 10000000000000000000000000 |
      | trader2 | USD   | 10000000000000000000000000 |
      | trader3 | USD   | 10000000000000000000000000 |
      | trader4 | USD   | 10000000000000000000000000 |
      | trader5 | USD   | 10000000000000000000000000 |

  @SMGIP01
  Scenario: Test global insurance pool collects successor markets insurance balances
    Given the markets:
      | id        | quote name | asset | liquidity monitoring | risk model                | margin calculator         | auction duration | fees         | price monitoring | data source config | linear slippage factor | quadratic slippage factor | decimal places | position decimal places | parent market id | insurance pool fraction | successor auction | sla params         |
      | ETH/DEC19 | ETH        | USD   | lqm-params           | lognormal-risk-model-fish | margin-calculator-1       | 1                | default-none | default-none     | ethDec19Oracle     | 0.1                    | 0                         | 1              | 1                       |                  |                         |                   | default-futures    |
      | ETH/DEC20 | ETH        | USD   | lqm-params           | default-st-risk-model     | default-margin-calculator | 1                | default-none | default-none     | ethDec20Oracle     | 0.1                    | 0                         | 1              | 1                       | ETH/DEC19        | 0.5                     | 10                | default-futures        |
      | ETH/DEC21 | ETH        | USD   | lqm-params           | default-st-risk-model     | default-margin-calculator | 1                | default-none | default-none     | ethDec21Oracle     | 0.1                    | 0                         | 1              | 1                       | ETH/DEC19        | 0.5                     | 10                | default-futures       |
      | ETH/DEC22 | ETH        | USD   | lqm-params           | default-st-risk-model     | default-margin-calculator | 1                | default-none | default-none     | ethDec22Oracle     | 0.1                    | 0                         | 1              | 1                       | ETH/DEC19        | 0.5                     | 10                | default-futures        |
      | ETH/DEC23 | ETH        | USD   | lqm-params           | default-st-risk-model     | default-margin-calculator | 1                | default-none | default-none     | ethDec23Oracle     | 0.1                    | 0                         | 1              | 1                       | ETH/DEC19        | 0.5                     | 10                | default-futures        |

    Given the initial insurance pool balance is "1000" for all the markets
    And the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov1 | ETH/DEC19 | 9000              | 0.1 | submission |
      | lp2 | lpprov2 | ETH/DEC20 | 1000              | 0.1 | submission |

    And the parties place the following pegged iceberg orders:
      | party   | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov2 | ETH/DEC20 | 2         | 1                    | buy  | BID              | 500    | 10     |
      | lpprov2 | ETH/DEC20 | 2         | 1                    | sell | ASK              | 500    | 10     |

    And the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC19 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | lpprov1 | ETH/DEC19 | buy  | 225    | 40    | 0                | TYPE_LIMIT | TIF_GTC |
      | lpprov1 | ETH/DEC19 | sell | 36     | 250   | 0                | TYPE_LIMIT | TIF_GTC |

    And the network moves ahead "2" blocks
    Then the mark price should be "150" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC21"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC22"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC23"

    ## Now ETH/DEC20 leaves opening auction
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC20 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |

    And the network moves ahead "11" blocks
    Then the mark price should be "150" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    # enact the successor and get the insurance pool franction form the parent
    And the insurance pool balance should be "0" for the market "ETH/DEC21"
    And the insurance pool balance should be "0" for the market "ETH/DEC22"
    And the insurance pool balance should be "0" for the market "ETH/DEC23"

     ## insurance pool for closed market is distributed
    And the insurance pool balance should be "1500" for the market "ETH/DEC19"
    And the insurance pool balance should be "2500" for the market "ETH/DEC20"

    And the global insurance pool balance should be "1000" for the asset "USD"

    # cancel ETH/DEC19
    When the market states are updated through governance:
      | market id | state                              | settlement price |
      | ETH/DEC19 | MARKET_STATE_UPDATE_TYPE_TERMINATE | 150              |

    ## insurance pool for closed market is distributed
    And the insurance pool balance should be "0" for the market "ETH/DEC19"
    And the insurance pool balance should be "3250" for the market "ETH/DEC20"

    And the global insurance pool balance should be "1750" for the asset "USD"

    And the network moves ahead "1" blocks

    # terminate ETH/DEC20
    Then the oracles broadcast data signed with "0xCAFECAAA":
      | name               | value |
      | trading.terminated | true  |  

    And the network moves ahead "1" blocks

    # settle ETH/DEC20
    When the oracles broadcast data signed with "0xCAFECAAA":
      | name             | value    |
      | prices.ETH.value | 14000000 |

    Then the market state should be "STATE_SETTLED" for the market "ETH/DEC20"

    And then the network moves ahead "1" blocks

    And the insurance pool balance should be "3250" for the market "ETH/DEC20"
    And the global insurance pool balance should be "1750" for the asset "USD"

    And the network moves ahead "10" blocks

    And the global insurance pool balance should be "5000" for the asset "USD"

    ## the insurance pools from the settled/cancelled markets are all drained
    Then the insurance pool balance should be "0" for the market "ETH/DEC19"
    And the insurance pool balance should be "0" for the market "ETH/DEC20"
    And the insurance pool balance should be "0" for the market "ETH/DEC21"
    And the insurance pool balance should be "0" for the market "ETH/DEC22"
    And the insurance pool balance should be "0" for the market "ETH/DEC23"
