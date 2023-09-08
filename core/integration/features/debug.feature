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
      | market.liquidity.bondPenaltyParameter               | 0.2   |
      | validators.epoch.length                             | 5s    |
      | market.liquidity.stakeToCcyVolume                   | 1     |
      | market.liquidity.successorLaunchWindowLength        | 1h    |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0.19  |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax   | 1     |
      | validators.epoch.length                             | 2s    |
      | market.liquidity.providersFeeCalculationTimeStep    | 2s    |
    And the liquidity sla params named "SLA":
      | price range        | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0                | 0.5                          | 1                             | 1.0                    |
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

  @SLADebug
  Scenario: Reproduce opening auction without LP
    Given the markets:
      | id        | quote name | asset | risk model                | margin calculator   | auction duration | fees         | price monitoring | data source config | linear slippage factor | quadratic slippage factor | decimal places | position decimal places | parent market id | insurance pool fraction | successor auction | sla params |
      | ETH/DEC19 | ETH        | USD   | lognormal-risk-model-fish | margin-calculator-1 | 1                | default-none | default-none     | ethDec19Oracle     | 0.1                    | 0                         | 0              | 0                       |                  |                         |                   | SLA        |
    And the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "2" blocks
    Then the market data for the market "ETH/DEC19" should be:
      | trading mode                 | auction trigger         | target stake | supplied stake | open interest |
      | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING | 4878         | 0              | 0             |

    When the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov1 | ETH/DEC19 | 9000              | 0.1 | submission |
      | lp1 | lpprov1 | ETH/DEC19 | 9000              | 0.1 | submission |
    And the opening auction period ends for market "ETH/DEC19"
    And the network moves ahead "20" blocks
    #And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode                 | auction trigger             | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_UNSPECIFIED | 4878         | 9000           | 1             |

  @SLADebug
  Scenario: Reproduce opening auction without LP with upfront LP submission
    Given the markets:
      | id        | quote name | asset | risk model                | margin calculator   | auction duration | fees         | price monitoring | data source config | linear slippage factor | quadratic slippage factor | decimal places | position decimal places | parent market id | insurance pool fraction | successor auction | sla params |
      | ETH/DEC19 | ETH        | USD   | lognormal-risk-model-fish | margin-calculator-1 | 1                | default-none | default-none     | ethDec19Oracle     | 0.1                    | 0                         | 0              | 0                       |                  |                         |                   | SLA        |
    When the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov1 | ETH/DEC19 | 9000              | 0.1 | submission |
      | lp1 | lpprov1 | ETH/DEC19 | 9000              | 0.1 | submission |
    And the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    And the opening auction period ends for market "ETH/DEC19"
    And the network moves ahead "20" blocks
    #And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 4878         | 9000           | 1             |
