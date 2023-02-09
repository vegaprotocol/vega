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

  @MidPrice
  Scenario: Mid price works as expected
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | DAI   | 110000000000 |
      | party2 | DAI   | 110000000000 |
      | party3 | DAI   | 110000000000 |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee  | side | pegged reference | proportion | offset | reference | lp type    |
      | lp1 | party1 | DAI/DEC22 | 20000000000       | 0.01 | buy  | MID              | 1          | 10     | lp-1      | submission |
      | lp1 | party1 | DAI/DEC22 | 20000000000       | 0.01 | sell | MID              | 1          | 10     | lp-1      | submission |

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

  @MidPrice
  Scenario: Mid price should work even if LP has limit order on the book (sell side high)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | DAI   | 110000000000 |
      | party2 | DAI   | 110000000000 |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee  | side | pegged reference | proportion | offset | reference | lp type    |
      | lp1 | party1 | DAI/DEC22 | 20000000000       | 0.01 | buy  | MID              | 1          | 10     | lp-1      | submission |
      | lp1 | party1 | DAI/DEC22 | 20000000000       | 0.01 | sell | MID              | 1          | 10     | lp-1      | submission |

    When the parties place the following orders:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     | reference |
      | party2 | DAI/DEC22 | buy  | 1      | 800000000  | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party2 | DAI/DEC22 | buy  | 1      | 3500000000 | 0                | TYPE_LIMIT | TIF_GTC | party2-2  |
      | party1 | DAI/DEC22 | sell | 1      | 3500000000 | 0                | TYPE_LIMIT | TIF_GTC | party3-1  |
      | party1 | DAI/DEC22 | sell | 1      | 8200000000 | 0                | TYPE_LIMIT | TIF_GTC | party3-2  |

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
      | sell | 4500000010 | 3      |
      | buy  | 4499999990 | 5      |
      | buy  | 800000000  | 1      |

  @MidPrice
  Scenario: Mid price should work even if LP has limit order on the book (buy side low)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | DAI   | 110000000000 |
      | party2 | DAI   | 110000000000 |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee  | side | pegged reference | proportion | offset | reference | lp type    |
      | lp1 | party1 | DAI/DEC22 | 20000000000       | 0.01 | buy  | MID              | 1          | 10     | lp-1      | submission |
      | lp1 | party1 | DAI/DEC22 | 20000000000       | 0.01 | sell | MID              | 1          | 10     | lp-1      | submission |

    When the parties place the following orders:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     | reference |
      | party1 | DAI/DEC22 | buy  | 1      | 800000000  | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party2 | DAI/DEC22 | buy  | 1      | 3500000000 | 0                | TYPE_LIMIT | TIF_GTC | party2-2  |
      | party1 | DAI/DEC22 | sell | 1      | 3500000000 | 0                | TYPE_LIMIT | TIF_GTC | party3-1  |
      | party2 | DAI/DEC22 | sell | 1      | 8200000000 | 0                | TYPE_LIMIT | TIF_GTC | party3-2  |

    And the opening auction period ends for market "DAI/DEC22"
    Then the following trades should be executed:
      | buyer  | price      | size | seller |
      | party2 | 3500000000 | 1    | party1 |
    And the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 800000000             | 4500000000       | 8200000000              |
    # LP bid limit order price is so low that it's not covering enough of the obligation to make the automatically deployed volume smaller
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 4500000010 | 5      |
      | buy  | 4499999990 | 5      |
      | buy  | 800000000  | 1      |

  @MidPrice
  Scenario: Mid price should work even if LP has limit order on the book (sell side high) - low commitment
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
      | party1 | DAI/DEC22 | sell | 1      | 3500000000 | 0                | TYPE_LIMIT | TIF_GTC | party3-1  |
      | party1 | DAI/DEC22 | sell | 1      | 8200000000 | 0                | TYPE_LIMIT | TIF_GTC | party3-2  |

    And the opening auction period ends for market "DAI/DEC22"
    Then the following trades should be executed:
      | buyer  | price      | size | seller |
      | party2 | 3500000000 | 1    | party1 |
    And the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 800000000             | 4500000000       | 8200000000              |
    # LP sell limit order covers majority of the commitment on that side
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 4510000000 | 1      |
      | buy  | 4490000000 | 3      |
      | buy  | 800000000  | 1      |

  @MidPrice
  Scenario: Mid price should work even if LP has limit order on the book (buy side low) - half commitment
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
      | party1 | DAI/DEC22 | buy  | 1      | 800000000  | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party2 | DAI/DEC22 | buy  | 1      | 3500000000 | 0                | TYPE_LIMIT | TIF_GTC | party2-2  |
      | party1 | DAI/DEC22 | sell | 1      | 3500000000 | 0                | TYPE_LIMIT | TIF_GTC | party3-1  |
      | party2 | DAI/DEC22 | sell | 1      | 8200000000 | 0                | TYPE_LIMIT | TIF_GTC | party3-2  |

    And the opening auction period ends for market "DAI/DEC22"
    Then the following trades should be executed:
      | buyer  | price      | size | seller |
      | party2 | 3500000000 | 1    | party1 |
    And the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 800000000             | 4500000000       | 8200000000              |
    # LP bid limit order price is so low that it's not covering enough of the obligation to make the automatically deployed volume smaller
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 4510000000 | 3      |
      | buy  | 4490000000 | 3      |
      | buy  | 800000000  | 1      |

  @MidPrice
  Scenario: Mid price should work even if LP has limit order on the book (buy side low) - updating orders (manually, bring prices closer)
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
      | party1 | DAI/DEC22 | buy  | 1      | 800000000  | 0                | TYPE_LIMIT | TIF_GTC | party1-1  |
      | party2 | DAI/DEC22 | buy  | 1      | 3500000000 | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party1 | DAI/DEC22 | sell | 1      | 3500000000 | 0                | TYPE_LIMIT | TIF_GTC | party1-2  |
      | party2 | DAI/DEC22 | sell | 1      | 8200000000 | 0                | TYPE_LIMIT | TIF_GTC | party2-2  |

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
      | sell | 4510000000 | 3      |
      | buy  | 4490000000 | 3      |
      | buy  | 800000000  | 1      |

    ## Now change our orders manually
    When the parties place the following orders:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     | reference |
      | party1 | DAI/DEC22 | buy  | 1      | 810000000  | 0                | TYPE_LIMIT | TIF_GTC | party1-b  |
      | party2 | DAI/DEC22 | sell | 1      | 8190000000 | 0                | TYPE_LIMIT | TIF_GTC | party2-b  |
    And the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 810000000             | 4500000000       | 8190000000              |
    # Volume at 4490000000 goes down as party1 has fullfiled some of it's obligation by the additional limit order submitted above
    Then the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 8190000000 | 1      |
      | sell | 4510000000 | 3      |
      | buy  | 4490000000 | 2      |
      | buy  | 810000000  | 1      |
      | buy  | 800000000  | 1      |

    When the parties cancel the following orders:
      | party  | reference |
      | party1 | party1-1  |
      | party2 | party2-2  |
    And the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 810000000             | 4500000000       | 8190000000              |
    # Volume at 4490000000 goes up as party1 has cancelled limit order fullfiling part of its obligation
    Then the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 0      |
      | sell | 8190000000 | 1      |
      | sell | 4510000000 | 3      |
      | buy  | 4490000000 | 3      |
      | buy  | 810000000  | 1      |
      | buy  | 800000000  | 0      |

  @MidPrice
  Scenario: Mid price should work even if LP has limit order on the book (buy side low) - updating orders (manually move prices apart)
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
      | party1 | DAI/DEC22 | buy  | 1      | 800000000  | 0                | TYPE_LIMIT | TIF_GTC | party1-1  |
      | party2 | DAI/DEC22 | buy  | 1      | 3500000000 | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party1 | DAI/DEC22 | sell | 1      | 3500000000 | 0                | TYPE_LIMIT | TIF_GTC | party1-2  |
      | party2 | DAI/DEC22 | sell | 1      | 8200000000 | 0                | TYPE_LIMIT | TIF_GTC | party2-2  |

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
      | sell | 4510000000 | 3      |
      | buy  | 4490000000 | 3      |
      | buy  | 800000000  | 1      |

    ## Now change our orders manually
    When the parties place the following orders:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     | reference |
      | party1 | DAI/DEC22 | buy  | 1      | 790000000  | 0                | TYPE_LIMIT | TIF_GTC | party1-b  |
      | party2 | DAI/DEC22 | sell | 1      | 8210000000 | 0                | TYPE_LIMIT | TIF_GTC | party2-b  |
    Then the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 800000000             | 4500000000       | 8200000000              |
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8210000000 | 1      |
      | sell | 8200000000 | 1      |
      | sell | 4510000000 | 3      |
      | buy  | 4490000000 | 2      |
      | buy  | 800000000  | 1      |
      | buy  | 790000000  | 1      |

    When the parties cancel the following orders:
      | party  | reference |
      | party1 | party1-1  |
      | party2 | party2-2  |
    Then the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 790000000             | 4500000000       | 8210000000              |
    # Volume at 4490000000 goes up to make up obligation (limit order party1-1 got cancelled)
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8210000000 | 1      |
      | sell | 8200000000 | 0      |
      | sell | 4510000000 | 3      |
      | buy  | 4490000000 | 3      |
      | buy  | 800000000  | 0      |
      | buy  | 790000000  | 1      |

  @MidPrice
  Scenario: Mid price should work even if LP has limit order on the book (LP high) - updating orders (manually, bring prices closer)
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
    # party1 placed a limit sell order that stayed on the book after auction so the automatically deployed volume is smaller.
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 4510000000 | 1      |
      | buy  | 4490000000 | 3      |
      | buy  | 800000000  | 1      |

    ## Now change our orders manually
    When the parties place the following orders:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     | reference |
      | party2 | DAI/DEC22 | buy  | 1      | 810000000  | 0                | TYPE_LIMIT | TIF_GTC | party2-b  |
      | party1 | DAI/DEC22 | sell | 1      | 8190000000 | 0                | TYPE_LIMIT | TIF_GTC | party1-b  |
    And the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 810000000             | 4500000000       | 8190000000              |
    # Volume at 4510000000 goes to 0 as party1 deployed additional limit sell order above which fullfiled the obligation for that side of the book
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 8190000000 | 1      |
      | sell | 4510000000 | 0      |
      | buy  | 4490000000 | 3      |
      | buy  | 810000000  | 1      |
      | buy  | 800000000  | 1      |

    When the parties cancel the following orders:
      | party  | reference |
      | party2 | party2-1  |
      | party1 | party1-2  |
    And the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 810000000             | 4500000000       | 8190000000              |
    # Volume at 4510000000 goes back up as the limit order fulfiling part of the obligation got cancelled by party1
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 0      |
      | sell | 8190000000 | 1      |
      | sell | 4510000000 | 1      |
      | buy  | 4490000000 | 3      |
      | buy  | 810000000  | 1      |
      | buy  | 800000000  | 0      |

  @MidPrice
  Scenario: Mid price should work even if LP has limit order on the book (LP high) - updating orders (manually move prices apart)
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
    # party1 placed a limit sell order that stayed on the book after auction so the automatically deployed volume is smaller.
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 4510000000 | 1      |
      | buy  | 4490000000 | 3      |
      | buy  | 800000000  | 1      |

    # Now change our orders manually
    When the parties place the following orders:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     | reference |
      | party2 | DAI/DEC22 | buy  | 1      | 790000000  | 0                | TYPE_LIMIT | TIF_GTC | party1-b  |
      | party1 | DAI/DEC22 | sell | 1      | 8210000000 | 0                | TYPE_LIMIT | TIF_GTC | party2-b  |
    Then the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 800000000             | 4500000000       | 8200000000              |
    # Volume at 4510000000 goes to 0 as party1 deployed additional limit sell order above which fullfiled the obligation for that side of the book
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8210000000 | 1      |
      | sell | 8200000000 | 1      |
      | sell | 4510000000 | 0      |
      | buy  | 4490000000 | 3      |
      | buy  | 800000000  | 1      |
      | buy  | 790000000  | 1      |

    When the parties cancel the following orders:
      | party  | reference |
      | party2 | party2-1  |
      | party1 | party1-2  |
    Then the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 790000000             | 4500000000       | 8210000000              |
    # Volume at 4510000000 goes back up as the limit order fulfiling part of the obligation got cancelled by party1
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8210000000 | 1      |
      | sell | 8200000000 | 0      |
      | sell | 4510000000 | 1      |
      | buy  | 4490000000 | 3      |
      | buy  | 800000000  | 0      |
      | buy  | 790000000  | 1      |

  @MidPrice
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
    # party1 placed a limit sell order that stayed on the book after auction so the automatically deployed volume is smaller.
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 4510000000 | 1      |
      | buy  | 4490000000 | 3      |
      | buy  | 800000000  | 1      |

    ## Now change our orders manually
    When the parties place the following orders:
      | party  | market id | side | volume | price     | resulting trades | type       | tif     | reference |
      | party2 | DAI/DEC22 | buy  | 1      | 810000000 | 0                | TYPE_LIMIT | TIF_GTC | party1-b  |
    Then the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 810000000             | 4505000000       | 8200000000              |
    # Mid moved so so did the orders pegged to it
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
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 4515000000 | 1      |
      | sell | 4510000000 | 0      |
      | buy  | 4495000000 | 3      |
      | buy  | 4490000000 | 0      |
      | buy  | 810000000  | 1      |
      | buy  | 800000000  | 0      |

    When the parties place the following orders:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     | reference |
      | party1 | DAI/DEC22 | sell | 1      | 8190000000 | 0                | TYPE_LIMIT | TIF_GTC | party2-b  |
    Then the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 810000000             | 4500000000       | 8190000000              |
    # MID moved so do did the LP buy order pegged to it, since party1 deployed a limit order on the sell side and it's obligation is now fully covered by limit orders sell order pegged to MID doesn't get deployed at all
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 8190000000 | 1      |
      | sell | 4515000000 | 0      |
      | sell | 4510000000 | 0      |
      | buy  | 4495000000 | 0      |
      | buy  | 4490000000 | 3      |
      | buy  | 810000000  | 1      |
      | buy  | 800000000  | 0      |

    When the parties cancel the following orders:
      | party  | reference |
      | party1 | party1-2  |
    Then the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 810000000             | 4500000000       | 8190000000              |
    # Limit order cancelled so volume at 4510000000 gets redeployed to cover the remaining part of obligation
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8190000000 | 1      |
      | sell | 4515000000 | 0      |
      | sell | 4510000000 | 1      |
      | buy  | 4495000000 | 0      |
      | buy  | 4490000000 | 3      |
      | buy  | 810000000  | 1      |
      | buy  | 800000000  | 0      |

  @MidPrice
  Scenario: Changing orders copying the script (same as above, but LP low buy order)
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
      | party1 | DAI/DEC22 | buy  | 1      | 800000000  | 0                | TYPE_LIMIT | TIF_GTC | party1-1  |
      | party2 | DAI/DEC22 | buy  | 1      | 3500000000 | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party1 | DAI/DEC22 | sell | 1      | 3500000000 | 0                | TYPE_LIMIT | TIF_GTC | party1-2  |
      | party2 | DAI/DEC22 | sell | 1      | 8200000000 | 0                | TYPE_LIMIT | TIF_GTC | party2-2  |

    And the opening auction period ends for market "DAI/DEC22"
    Then the following trades should be executed:
      | buyer  | price      | size | seller |
      | party2 | 3500000000 | 1    | party1 |
    And the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 800000000             | 4500000000       | 8200000000              |
    # party1 has no limit sell order so automatically deployed volume at 4510000000 goes up (compared to previous scenario)
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 4510000000 | 3      |
      | buy  | 4490000000 | 3      |
      | buy  | 800000000  | 1      |

    ## Now change our orders manually
    When the parties place the following orders:
      | party  | market id | side | volume | price     | resulting trades | type       | tif     | reference |
      | party1 | DAI/DEC22 | buy  | 1      | 810000000 | 0                | TYPE_LIMIT | TIF_GTC | party1-b  |
    Then the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 810000000             | 4505000000       | 8200000000              |
    # MID moves so so did the orders pegged to it, volume for the buy order pegged to mid now lower as limit order by party1 covers part of the obligation
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 4515000000 | 3      |
      | sell | 4510000000 | 0      |
      | buy  | 4495000000 | 2      |
      | buy  | 4490000000 | 0      |
      | buy  | 810000000  | 1      |
      | buy  | 800000000  | 1      |

    When the parties cancel the following orders:
      | party  | reference |
      | party1 | party1-1  |
    Then the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 810000000             | 4505000000       | 8200000000              |
    # No effect on MID, so pegged order don't move, but volume at 4495000000 goes up since party1 cancelled a limit order covering part of its obligation
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 4515000000 | 3      |
      | sell | 4510000000 | 0      |
      | buy  | 4495000000 | 3      |
      | buy  | 4490000000 | 0      |
      | buy  | 810000000  | 1      |

    When the parties place the following orders:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     | reference |
      | party2 | DAI/DEC22 | sell | 1      | 8190000000 | 0                | TYPE_LIMIT | TIF_GTC | party2-b  |
    Then the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 810000000             | 4500000000       | 8190000000              |
    # MID moved so so did orders pegged to it
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 8190000000 | 1      |
      | sell | 4515000000 | 0      |
      | sell | 4510000000 | 3      |
      | buy  | 4495000000 | 0      |
      | buy  | 4490000000 | 3      |
      | buy  | 810000000  | 1      |

    When the parties cancel the following orders:
      | party  | reference |
      | party2 | party2-2  |
    Then the market data for the market "DAI/DEC22" should be:
      | mark price | best static bid price | static mid price | best static offer price |
      | 3500000000 | 810000000             | 4500000000       | 8190000000              |
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 0      |
      | sell | 8190000000 | 1      |
      | sell | 4515000000 | 0      |
      | sell | 4510000000 | 3      |
      | buy  | 4495000000 | 0      |
      | buy  | 4490000000 | 3      |
      | buy  | 810000000  | 1      |
