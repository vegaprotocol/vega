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
    Given time is updated to "2023-07-20T00:00:00Z"
    Given the average block duration is "1"
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
    And the oracle spec for settlement data filtering data from "0xCAFECAFE20" named "ethDec20Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |
    And the oracle spec for trading termination filtering data from "0xCAFECAFE20" named "ethDec20Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |
    And the settlement data decimals for the oracle named "ethDec20Oracle" is given in "5" decimal places
    And the following network parameters are set:
      | name                                                  | value |
      | network.markPriceUpdateMaximumFrequency               | 0s    |
      | market.liquidity.targetstake.triggering.ratio         | 0.01  |
      | market.stake.target.timeWindow                        | 10s   |
      | market.stake.target.scalingFactor                     | 10    |
      | market.auction.minimumDuration                        | 1     |
      | market.fee.factors.infrastructureFee                  | 0.001 |
      | market.fee.factors.makerFee                           | 0.004 |
      | market.value.windowLength                             | 60s   |
      | market.liquidity.bondPenaltyParameter               | 0.2   |
      | validators.epoch.length                               | 5s    |
      | market.liquidity.stakeToCcyVolume                   | 1     |
      | market.liquidity.successorLaunchWindowLength          | 1h    |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0.19  |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax   | 1     |
      | validators.epoch.length                               | 2s    |
      | market.liquidity.providersFeeCalculationTimeStep    | 2s   |
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |
    And the oracle spec for settlement data filtering data from "0xCAFECAFE1" named "ethDec20Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |
    And the oracle spec for trading termination filtering data from "0xCAFECAFE1" named "ethDec20Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |
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
      | ETH/DEC20 | ETH        | USD   | default-st-risk-model     | default-margin-calculator | 1                | default-none | default-none     | ethDec20Oracle         | 0.1                    | 0                         | 0              | 0                       | ETH/DEC19        | 0.6                     | 10                | SLA        |
      | ETH/DEC21 | ETH        | USD   | default-st-risk-model     | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.1                    | 0                         | 0              | 0                       | ETH/DEC19        | 0.6                     | 10                | SLA        |
    And the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov1 | ETH/DEC19 | 9000              | 0.1 | submission |
      | lp1 | lpprov1 | ETH/DEC19 | 9000              | 0.1 | submission |
      | lp2 | lpprov2 | ETH/DEC19 | 1000              | 0.1 | submission |
      | lp2 | lpprov2 | ETH/DEC19 | 1000              | 0.1 | submission |
    And the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC19 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
    When the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 150        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 731          | 10000          | 1             |
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | trader4 | ETH/DEC19 | sell | 290    | 150   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | trader3 | ETH/DEC19 | buy  | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 150        | TRADING_MODE_CONTINUOUS | 731          | 10000          | 1             |
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader5 | ETH/DEC19 | buy  | 290    | 150   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader5 | USD   | ETH/DEC19 | 17432  | 0       |
    Then the parties cancel the following orders:
      | party   | reference      |
      | trader3 | buy-provider-1 |
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | trader3 | ETH/DEC19 | buy  | 290    | 120   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2 |
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader4 | ETH/DEC19 | sell | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader3 | ETH/DEC19 | buy  | 1      | 140   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
    Then the network moves ahead "2" blocks

    And the insurance pool balance should be "6977" for the market "ETH/DEC19"
    And the global insurance pool balance should be "0" for the asset "USD"

    Then debug transfers

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general     |
      | lpprov1 | USD   | ETH/DEC19 | 0      | 1999991000  |
      | lpprov2 | USD   | ETH/DEC19 | 0      | 19999999000 |
      | trader1 | USD   | ETH/DEC19 | 1088   | 1998902     |
      | trader2 | USD   | ETH/DEC19 | 121    | 1999889     |
      | trader3 | USD   | ETH/DEC19 | 27856  | 1978068     |
      | trader4 | USD   | ETH/DEC19 | 41369  | 1961706     |
      | trader5 | USD   | ETH/DEC19 | 0      | 0           |

    Then the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl |
      | trader1 | 1      | -10            | 0            |
      | trader2 | -1     | 10             | 0            |
      | trader3 | 291    | 5800           | 0            |
      | trader4 | -291   | 2900           | 0            |
      | trader5 | 0      | 0              | -17432       |

    And the liquidity provider fee shares for the market "ETH/DEC19" should be:
      | party   | equity like share  | average entry valuation |
      | lpprov1 | 0.9 | 9000  |
      | lpprov2 | 0.1 | 10000 |

    And then the network moves ahead "2" blocks

    And the insurance pool balance should be "6977" for the market "ETH/DEC19"
    And the global insurance pool balance should be "0" for the asset "USD"

    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name               | value |
      | trading.terminated | true  |

    And time is updated to "2023-07-21T00:00:00Z"
    And the global insurance pool balance should be "0" for the asset "USD"

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general     |
      | lpprov1 | USD   | ETH/DEC19 | 0      | 1999991000  |
      | lpprov2 | USD   | ETH/DEC19 | 0      | 19999999000 |
      | trader1 | USD   | ETH/DEC19 | 1088   | 1998902     |
      | trader2 | USD   | ETH/DEC19 | 121    | 1999889     |
      | trader3 | USD   | ETH/DEC19 | 27856  | 1978068     |
      | trader4 | USD   | ETH/DEC19 | 41369  | 1961706     |
      | trader5 | USD   | ETH/DEC19 | 0      | 0           |

    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC19"
    And the insurance pool balance should be "6977" for the market "ETH/DEC19"
    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name             | value    |
      | prices.ETH.value | 14000000 |

    And the insurance pool balance should be "16359" for the market "ETH/DEC19"

    And then the network moves ahead "10" blocks
    And the insurance pool balance should be "16359" for the market "ETH/DEC19"
    And the global insurance pool balance should be "0" for the asset "USD"

    Then the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl |
      | trader1 | 1      | 0              | -10          |
      | trader2 | -1     | 0              | 10           |
      | trader3 | 291    | 0              | 5800         |
      | trader4 | -291   | 0              | 2900         |
      | trader5 | 0      | 0              | -17432       |

    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general     |
      | lpprov1 | USD | ETH/DEC19 | 0 | 1999996905  |
      | lpprov2 | USD | ETH/DEC19 | 0 | 19999999657 |
      | trader1 | USD   | ETH/DEC19 | 0      | 1999990     |
      | trader2 | USD   | ETH/DEC19 | 0      | 2000010     |
      | trader3 | USD   | ETH/DEC19 | 0      | 2005924     |
      | trader4 | USD   | ETH/DEC19 | 0      | 2003075     |
      | trader5 | USD   | ETH/DEC19 | 0      | 0           |



