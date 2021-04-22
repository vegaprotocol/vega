Feature: Test liquidity provider reward distribution

# Spec file: ../spec/0042-setting-fees-and-rewarding-lps.md

  Background:
    Given the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 10          | -10           | 0.1                    |
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau                    | mu | r  | sigma |
      | 0.000001      | 0.00011407711613050422 | -1 | -1 | -1    |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee | liquidity fee |
      | 0.0004    | 0.001              | 0.3         |
    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.1           | 1.2            | 1.4            |
    And the price monitoring updated every "1" seconds named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |
    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | oracle config          | maturity date        |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring-1     | default-eth-for-future | 2019-12-31T23:59:59Z |

    And the following network parameters are set:
      | name                                                | value   |
      | market.value.windowLength                           | 1h      |
      | market.stake.target.timeWindow                      | 24h     |
      | market.stake.target.scalingFactor                   | 1       |
      | market.liquidity.targetstake.triggering.ratio       | 0       |
      | market.liquidity.providers.fee.distributionTimeStep | 10m     |


  Scenario: 1 LP joining at start, checking liquidity rewards over 3 periods, 1 period with no trades
    # setup accounts
    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount     |
      | lp1     | ETH   | 1000000000 |
      | trader1 | ETH   | 100000000  |
      | trader2 | ETH   | 100000000  |

    And the traders submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
      | lp1 | lp1   | ETH/DEC21 | 10000             | 0.001 | buy        | BID             | 1                | -2           |
      | lp1 | lp1   | ETH/DEC21 | 10000             | 0.001 | buy        | MID             | 2                | -1           |
      | lp1 | lp1   | ETH/DEC21 | 10000             | 0.001 | sell       | ASK             | 1                | 2            |
      | lp1 | lp1   | ETH/DEC21 | 10000             | 0.001 | sell       | MID             | 2                | 1            |

    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"

    And the following trades should be executed:
      | buyer   | price | size | seller  |
      | trader1 | 1000  | 10   | trader2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the mark price should be "1000" for the market "ETH/DEC21"
    And the open interest for the market "ETH/DEC21" is "10"
    And the target stake for the market "ETH/DEC21" is "1000"
    And the supplied stake for the market "ETH/DEC21" is "10000"

    And the liquidity provider fee shares for the market "ETH/DEC21" should be:
      | party | equity like share | average entry valuation |
      | lp1   |                 1 |                   10000 |

    And the price monitoring bounds for the market "ETH/DEC21" should be:
      | min bound | max bound |
      |       990 |     1010  |


    # And the liquidity fee factor is "0.001"
