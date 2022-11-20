Feature: Replicate unexpected margin issues - no mid price pegs

  Background:
    Given the following assets are registered:
      | id  | decimal places |
      | DAI | 5              |
    And the log normal risk model named "dai-lognormal-risk":
      | risk aversion | tau         | mu | r | sigma |
      | 0.00001       | 0.000114077 | 0  | 0 | 0.41  |
    And the markets:
      | id        | quote name | asset | risk model         | margin calculator         | auction duration | fees         | price monitoring | data source config          | decimal places |
      | DAI/DEC22 | DAI        | DAI   | dai-lognormal-risk | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 5              |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | market.stake.target.scalingFactor       | 10    |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  @MidPrice @LPAmend
  Scenario: Changing orders copying the script
    Given the parties deposit on asset's general account the following amount:
      | party           | asset | amount       |
      | party1          | DAI   | 110000000000 |
      | party2          | DAI   | 110000000000 |

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
    And the mark price should be "3500000000" for the market "DAI/DEC22"
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 4510000000 | 3      |
      | sell | 8200000000 | 1      |
      | buy  | 4490000000 | 5      |
      | buy  | 800000000  | 1      |

    ## Now change our orders manually
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     | reference |
      | party2 | DAI/DEC22 | buy  | 1      | 810000000  | 0                | TYPE_LIMIT | TIF_GTC | party1-b  |
    ## LP orders are gone! this is where things go wrong
    ## THIS IS WRONG!!!
    Then the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | buy  | 800000000  | 1      |
      | buy  | 810000000  | 1      |
    And the mark price should be "3500000000" for the market "DAI/DEC22"
    
    ## Let's see if amending the LP in a trivial way changes anything at all
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee  | side | pegged reference | proportion | offset   | reference | lp type   |
      | lp1 | party1 | DAI/DEC22 | 10000000000       | 0.01 | buy  | MID              | 1          | 10000000 | lp-1      | amendment |
      | lp1 | party1 | DAI/DEC22 | 10000000000       | 0.01 | sell | MID              | 1          | 10000000 | lp-1      | amendment |
    Then the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | buy  | 800000000  | 1      |
      | buy  | 810000000  | 1      |
      | sell | 4515000000 | 3      |
      | buy  | 4495000000 | 5      |
    And the mark price should be "3500000000" for the market "DAI/DEC22"

    When the parties cancel the following orders:
      | party  | reference |
      | party2 | party2-1  |
    Then the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 4515000000 | 3      |
      | sell | 8200000000 | 1      |
      | buy  | 4495000000 | 5      |
      | buy  | 810000000  | 1      |
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     | reference |
      | party1 | DAI/DEC22 | sell | 1      | 8190000000 | 0                | TYPE_LIMIT | TIF_GTC | party2-b  |
    Then the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | sell | 8190000000 | 1      |
      | buy  | 810000000  | 1      |
    # The bug is fixed, so we should also see the LP orders
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 4510000000 | 2      |
      | buy  | 4490000000 | 5      |
    ## Let's see if amending the LP in a trivial way changes anything at all
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee  | side | pegged reference | proportion | offset   | reference | lp type   |
      | lp1 | party1 | DAI/DEC22 | 10000000000       | 0.01 | buy  | MID              | 1          | 10000000 | lp-1      | amendment |
      | lp1 | party1 | DAI/DEC22 | 10000000000       | 0.01 | sell | MID              | 1          | 10000000 | lp-1      | amendment |
    Then the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 8200000000 | 1      |
      | buy  | 810000000  | 1      |
      | sell | 4510000000 | 2      |
      | buy  | 4490000000 | 5      |
    And the mark price should be "3500000000" for the market "DAI/DEC22"
    When the parties cancel the following orders:
      | party  | reference |
      | party1 | party3-2  |
    ## Now the volumes are different compared to when we created both orders + deleted both at the same time???
    Then the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 4510000000 | 3      |
      | sell | 8190000000 | 1      |
      | buy  | 4490000000 | 5      |
      | buy  | 810000000  | 1      |
    And the mark price should be "3500000000" for the market "DAI/DEC22"


  @MidPrice @LPAmend
  Scenario: Changing orders copying the script
    Given the parties deposit on asset's general account the following amount:
      | party           | asset | amount       |
      | party1          | DAI   | 110000000000 |
      | party2          | DAI   | 110000000000 |

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
    And the mark price should be "3500000000" for the market "DAI/DEC22"
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 4510000000 | 3      |
      | sell | 8200000000 | 1      |
      | buy  | 4490000000 | 5      |
      | buy  | 800000000  | 1      |

    When the parties amend the following orders:
      | party  | reference | price     | size delta | tif     |
      | party2 | party2-1  | 810000000 | 0          | TIF_GTC |
    Then the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 4515000000 | 3      |
      | sell | 8200000000 | 1      |
      | buy  | 4495000000 | 5      |
      | buy  | 810000000  | 1      |
    And the mark price should be "3500000000" for the market "DAI/DEC22"

    When the parties amend the following orders:
      | party  | reference | price      | size delta | tif     |
      | party1 | party3-2  | 8190000000 | 0          | TIF_GTC |
    Then the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 4510000000 | 3      |
      | sell | 8190000000 | 1      |
      | buy  | 4490000000 | 5      |
      | buy  | 810000000  | 1      |
    And the mark price should be "3500000000" for the market "DAI/DEC22"

