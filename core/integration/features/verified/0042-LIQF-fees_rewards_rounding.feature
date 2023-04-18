Feature: All liquidity providers with `average fraction of liquidity provided by committed LP > 0` in the market receive a greater than zero amount of liquidity fee. The only exception is if a non-zero amount is rounded to zero due to integer representation.

  Scenario: 001: 0042-LIQF-015
  Background:

    Given the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 1h    |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 0     |
      | market.liquidity.providers.fee.distributionTimeStep | 10m   |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
    And the following assets are registered:
      | id  | decimal places |
      | ETH | 5              |
      | USD | 5              |
    And the average block duration is "2"

    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 100000  | 0.99        | 3                 |

    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | decimal places | position decimal places | linear slippage factor | quadratic slippage factor |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0              | 0                       | 1e6                    | 1e6                       |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount          |
      | lp1    | USD   | 100000000000000 |
      | lp2    | USD   | 100000000000000 |
      | lp3    | USD   | 100000000000000 |
      | party1 | USD   | 10000000000000  |
      | party2 | USD   | 10000000000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 800000000000      | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | lp1   | ETH/MAR22 | 800000000000      | 0.001 | buy  | MID              | 2          | 1      | submission |
      | lp1 | lp1   | ETH/MAR22 | 800000000000      | 0.001 | sell | ASK              | 1          | 2      | submission |
      | lp1 | lp1   | ETH/MAR22 | 800000000000      | 0.001 | sell | MID              | 2          | 1      | submission |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 100000            | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/MAR22 | 100000            | 0.002 | buy  | MID              | 2          | 1      | submission |
      | lp2 | lp2   | ETH/MAR22 | 100000            | 0.002 | sell | ASK              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/MAR22 | 100000            | 0.002 | sell | MID              | 2          | 1      | submission |

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/MAR22"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 60   | party2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"
    And the mark price should be "1000" for the market "ETH/MAR22"
    And the open interest should be "60" for the market "ETH/MAR22"
    And the target stake should be "21341400000" for the market "ETH/MAR22"
    And the supplied stake should be "800000100000" for the market "ETH/MAR22"

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.9999998750000156 | 800000000000            |
      | lp2   | 0.0000001249999844 | 800000100000            |

    And the liquidity fee factor should be "0.001" for the market "ETH/MAR22"

    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/MAR22 | buy  | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC | party2-buy  |
    And the liquidity fee factor should be "0.001" for the market "ETH/MAR22"

    And the accumulated liquidity fees should be "1902000" for the market "ETH/MAR22"

    # check lp fee distribution
    Then time is updated to "2019-11-30T00:10:05Z"

    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount  | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 1901999 | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 0       | USD   |
