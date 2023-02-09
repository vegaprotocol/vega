Feature: Replicate unexpected margin issues - no mid price pegs

  Background:
    Given the following assets are registered:
      | id  | decimal places |
      | DAI | 5              |
    And the log normal risk model named "dai-lognormal-risk":
      | risk aversion | tau         | mu | r | sigma |
      | 0.00001       | 0.000114077 | 0  | 0 | 0.41  |
    And the markets:
      | id        | quote name | asset | risk model         | margin calculator         | auction duration | fees         | price monitoring | data source config     | decimal places | linear slippage factor | quadratic slippage factor |
      | DAI/DEC22 | DAI        | DAI   | dai-lognormal-risk | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 5              | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | market.stake.target.scalingFactor       | 10    |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  @MidPrice @LPAmend
  Scenario: Changing orders copying the script
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | DAI   | 110000000000 |
      | party2 | DAI   | 110000000000 |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee  | side | pegged reference | proportion | offset   | reference | lp type    |
      | lp1 | party1 | DAI/DEC22 | 10000000000       | 0.01 | buy  | MID              | 1          | 10000000 | lp-1      | submission |
      | lp1 | party1 | DAI/DEC22 | 10000000000       | 0.01 | sell | MID              | 1          | 10000000 | lp-1      | submission |

    When the parties place the following orders:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     | reference |
      | party2 | DAI/DEC22 | buy  | 1      | 800000000  | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party2 | DAI/DEC22 | buy  | 1      | 3500000000 | 0                | TYPE_LIMIT | TIF_GTC | party2-2  |
      | party1 | DAI/DEC22 | sell | 1      | 3500000000 | 0                | TYPE_LIMIT | TIF_GTC | party1-1  |
      | party1 | DAI/DEC22 | sell | 1      | 8200000000 | 0                | TYPE_LIMIT | TIF_GTC | party1-2  |

    And the opening auction period ends for market "DAI/DEC22"
    Then the following trades should be executed:
      | buyer  | price      | size | seller |
      | party2 | 3500000000 | 1    | party1 |
    And the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 800000000             | 4500000000       | 8200000000              |
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 4510000000 | 1      |
      | buy  | 4490000000 | 3      |
      | buy  | 800000000  | 1      |

    ## Now raise best bid price
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price     | resulting trades | type       | tif     | reference |
      | party2 | DAI/DEC22 | buy  | 1      | 810000000 | 0                | TYPE_LIMIT | TIF_GTC | party1-b  |
    Then the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 810000000             | 4505000000       | 8200000000              |
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 4515000000 | 1      |
      | sell | 4510000000 | 0      |
      | buy  | 4495000000 | 3      |
      | buy  | 4490000000 | 0      |
      | buy  | 810000000  | 1      |
      | buy  | 800000000  | 1      |

    # Expecting no change as LP amend had no changes compared to the submission and the market composition hasn't change too
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee  | side | pegged reference | proportion | offset   | reference | lp type   |
      | lp1 | party1 | DAI/DEC22 | 10000000000       | 0.01 | buy  | MID              | 1          | 10000000 | lp-1      | amendment |
      | lp1 | party1 | DAI/DEC22 | 10000000000       | 0.01 | sell | MID              | 1          | 10000000 | lp-1      | amendment |
    Then the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 810000000             | 4505000000       | 8200000000              |
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 4515000000 | 1      |
      | sell | 4510000000 | 0      |
      | buy  | 4495000000 | 3      |
      | buy  | 4490000000 | 0      |
      | buy  | 810000000  | 1      |
      | buy  | 800000000  | 1      |

    When the parties cancel the following orders:
      | party  | reference |
      | party2 | party2-1  |
    Then the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 810000000             | 4505000000       | 8200000000              |
    # expecting no change to LP orders as pegs haven't moved to do cancellation (order with reference part2-1 wasn't a best bid at that stage)
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 4515000000 | 1      |
      | sell | 4510000000 | 0      |
      | buy  | 4495000000 | 3      |
      | buy  | 4490000000 | 0      |
      | buy  | 810000000  | 1      |
      | buy  | 800000000  | 0      |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     | reference |
      | party2 | DAI/DEC22 | sell | 1      | 8190000000 | 0                | TYPE_LIMIT | TIF_GTC | party2-b  |
    Then the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 810000000             | 4500000000       | 8190000000              |
    # LP orders change as the mid price changed
    Then debug detailed orderbook volumes for market "DAI/DEC22"
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 4510000000 | 1      |
      | sell | 4515000000 | 0      |
      | buy  | 4495000000 | 0      |
      | buy  | 4490000000 | 3      |
      | buy  | 810000000  | 1      |

    # Null amend should not change anything
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee  | side | pegged reference | proportion | offset   | reference | lp type   |
      | lp1 | party1 | DAI/DEC22 | 10000000000       | 0.01 | buy  | MID              | 1          | 10000000 | lp-1      | amendment |
      | lp1 | party1 | DAI/DEC22 | 10000000000       | 0.01 | sell | MID              | 1          | 10000000 | lp-1      | amendment |
    Then the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 810000000             | 4500000000       | 8190000000              |
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 8190000000 | 1      |
      | sell | 4510000000 | 1      |
      | sell | 4515000000 | 0      |
      | buy  | 4495000000 | 0      |
      | buy  | 4490000000 | 3      |
      | buy  | 810000000  | 1      |

    When the parties cancel the following orders:
      | party  | reference |
      | party1 | party1-2  |
    Then the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 810000000             | 4500000000       | 8190000000              |
    # Pegged order prices unchanged as MID hasn't changed, but volume goes up on sell side as the LP has cancelled their limit order
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 0      |
      | sell | 8190000000 | 1      |
      | sell | 4510000000 | 3      |
      | sell | 4515000000 | 0      |
      | buy  | 4495000000 | 0      |
      | buy  | 4490000000 | 3      |
      | buy  | 810000000  | 1      |

  @MidPrice @LPAmend
  Scenario: Changing orders copying the script
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | DAI   | 110000000000 |
      | party2 | DAI   | 110000000000 |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee  | side | pegged reference | proportion | offset   | reference | lp type    |
      | lp1 | party1 | DAI/DEC22 | 10000000000       | 0.01 | buy  | MID              | 1          | 10000000 | lp-1      | submission |
      | lp1 | party1 | DAI/DEC22 | 10000000000       | 0.01 | sell | MID              | 1          | 10000000 | lp-1      | submission |

    When the parties place the following orders:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     | reference |
      | party2 | DAI/DEC22 | buy  | 1      | 800000000  | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party2 | DAI/DEC22 | buy  | 1      | 3500000000 | 0                | TYPE_LIMIT | TIF_GTC | party2-2  |
      | party1 | DAI/DEC22 | sell | 1      | 3500000000 | 0                | TYPE_LIMIT | TIF_GTC | party1-1  |
      | party1 | DAI/DEC22 | sell | 1      | 8200000000 | 0                | TYPE_LIMIT | TIF_GTC | party1-2  |

    And the opening auction period ends for market "DAI/DEC22"
    Then the following trades should be executed:
      | buyer  | price      | size | seller |
      | party2 | 3500000000 | 1    | party1 |
    And the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 800000000             | 4500000000       | 8200000000              |
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 4510000000 | 1      |
      | buy  | 4490000000 | 3      |
      | buy  | 800000000  | 1      |

    When the parties amend the following orders:
      | party  | reference | price     | size delta | tif     |
      | party2 | party2-1  | 810000000 | 0          | TIF_GTC |
    Then the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 810000000             | 4505000000       | 8200000000              |
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 4515000000 | 1      |
      | sell | 4510000000 | 0      |
      | buy  | 4495000000 | 3      |
      | buy  | 4490000000 | 0      |
      | buy  | 810000000  | 1      |
      | buy  | 800000000  | 0      |

    When the parties amend the following orders:
      | party  | reference | price      | size delta | tif     |
      | party1 | party1-2  | 8190000000 | 0          | TIF_GTC |
    Then the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 810000000             | 4500000000       | 8190000000              |
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 0      |
      | sell | 8190000000 | 1      |
      | sell | 4515000000 | 0      |
      | sell | 4510000000 | 1      |
      | buy  | 4495000000 | 0      |
      | buy  | 4490000000 | 3      |
      | buy  | 810000000  | 1      |
      | buy  | 800000000  | 0      |
