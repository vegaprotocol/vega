Feature: Replicate unexpected margin issues - no mid price pegs

  Background:
    Given the following assets are registered:
      | id  | decimal places |
      | DAI | 5              |
    And the log normal risk model named "dai-lognormal-risk":
      | risk aversion | tau         | mu | r | sigma |
      | 0.00001       | 0.000114077 | 0  | 0 | 0.41  |
    And the markets:
      | id        | quote name | asset | risk model         | margin calculator         | auction duration | fees         | price monitoring | data source config     | decimal places | linear slippage factor | quadratic slippage factor | sla params      |
      | DAI/DEC22 | DAI        | DAI   | dai-lognormal-risk | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 5              | 1e6                    | 1e6                       | default-futures |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | market.stake.target.scalingFactor       | 10    |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 2     |

  @MidPrice
  Scenario: Mid price works as expected
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | DAI   | 110000000000 |
      | party2 | DAI   | 110000000000 |
      | party3 | DAI   | 110000000000 |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee  | reference | lp type    |
      | lp1 | party1 | DAI/DEC22 | 20000000000       | 0.01 | lp-1      | submission |
      | lp1 | party1 | DAI/DEC22 | 20000000000       | 0.01 | lp-1      | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | party1 | DAI/DEC22 | 5         | 3                    | buy  | MID              | 5          | 10     |
      | party1 | DAI/DEC22 | 5         | 3                    | sell | MID              | 5          | 10     |

    When the parties place the following orders:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     | reference |
      | party2 | DAI/DEC22 | buy  | 1      | 800000000  | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party2 | DAI/DEC22 | buy  | 1      | 3500000000 | 0                | TYPE_LIMIT | TIF_GTC | party2-2  |
      | party3 | DAI/DEC22 | sell | 1      | 3500000000 | 0                | TYPE_LIMIT | TIF_GTC | party3-1  |
      | party3 | DAI/DEC22 | sell | 1      | 8200000000 | 0                | TYPE_LIMIT | TIF_GTC | party3-2  |

    And the opening auction period ends for market "DAI/DEC22"
    Then the following trades should be executed:
      | buyer  | price      | size | seller |
      | party2 | 3500000000 | 1    | party3 |
    And the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 800000000             | 4500000000       | 8200000000              |
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 4500000010 | 5      |
      | buy  | 4499999990 | 5      |
      | buy  | 800000000  | 1      |
