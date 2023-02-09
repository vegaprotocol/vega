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

  @LPRelease
  Scenario: Mid price works as expected
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | DAI   | 110000000000 |
      | party2 | DAI   | 110000000000 |
      | party3 | DAI   | 110000000000 |
      | party4 | DAI   | 110000000000 |

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
    And the mark price should be "3500000000" for the market "DAI/DEC22"
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 4500000010 | 5      |
      | buy  | 4499999990 | 5      |
      | buy  | 800000000  | 1      |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee  | side | pegged reference | proportion | offset | reference | lp type    |
      | lp2 | party4 | DAI/DEC22 | 10000000000       | 0.01 | buy  | BID              | 1          | 12     | lp-2      | submission |
      | lp2 | party4 | DAI/DEC22 | 10000000000       | 0.01 | sell | ASK              | 1          | 12     | lp-2      | submission |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin     | general     | bond        |
      | party4 | DAI   | DAI/DEC22 | 1060913900 | 98939086100 | 10000000000 |
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000012 | 2      |
      | sell | 8200000000 | 1      |
      | sell | 4500000010 | 5      |
      | buy  | 4499999990 | 5      |
      | buy  | 800000000  | 1      |
      | buy  | 799999988  | 13     |

    # LP cancel -> orders are gone from the book + margin balance is released
    When party "party4" cancels their liquidity provision for market "DAI/DEC22"
    Then the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 4500000010 | 5      |
      | buy  | 4499999990 | 5      |
      | buy  | 800000000  | 1      |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general      | bond |
      | party4 | DAI   | DAI/DEC22 | 0      | 110000000000 | 0    |

  @LPAmendVersion
  Scenario: Amend an LP before cancel, check the version events
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | DAI   | 110000000000 |
      | party2 | DAI   | 110000000000 |
      | party3 | DAI   | 110000000000 |
      | party4 | DAI   | 110000000000 |

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
    And the mark price should be "3500000000" for the market "DAI/DEC22"
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 4500000010 | 5      |
      | buy  | 4499999990 | 5      |
      | buy  | 800000000  | 1      |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee  | side | pegged reference | proportion | offset | reference | lp type    |
      | lp2 | party4 | DAI/DEC22 | 10000000000       | 0.01 | buy  | BID              | 1          | 12     | lp-2      | submission |
      | lp2 | party4 | DAI/DEC22 | 10000000000       | 0.01 | sell | ASK              | 1          | 12     | lp-2      | submission |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin     | general     | bond        |
      | party4 | DAI   | DAI/DEC22 | 1060913900 | 98939086100 | 10000000000 |
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000012 | 2      |
      | sell | 8200000000 | 1      |
      | sell | 4500000010 | 5      |
      | buy  | 4499999990 | 5      |
      | buy  | 800000000  | 1      |
      | buy  | 799999988  | 13     |

    # Amending the LP should result in LP versions being different
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee  | side | pegged reference | proportion | offset | reference | lp type   |
      | lp2 | party4 | DAI/DEC22 | 10000000010       | 0.01 | buy  | BID              | 1          | 12     | lp-2      | amendment |
      | lp2 | party4 | DAI/DEC22 | 10000000010       | 0.01 | sell | ASK              | 1          | 12     | lp-2      | amendment |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin     | general     | bond        |
      | party4 | DAI   | DAI/DEC22 | 1060913900 | 98939086090 | 10000000010 |
    And the following LP events should be emitted:
      | party  | id  | version | commitment amount | final |
      | party4 | lp2 | 1       | 10000000000       | false |
      | party4 | lp2 | 2       | 10000000010       | true  |

    # LP cancel -> orders are gone from the book + margin balance is released
    When party "party4" cancels their liquidity provision for market "DAI/DEC22"
    Then the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 4500000010 | 5      |
      | buy  | 4499999990 | 5      |
      | buy  | 800000000  | 1      |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general      | bond |
      | party4 | DAI   | DAI/DEC22 | 0      | 110000000000 | 0    |
