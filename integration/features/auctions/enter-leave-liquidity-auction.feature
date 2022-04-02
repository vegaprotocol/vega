Feature: Ensure we can enter and leave liquidity auction

  Background:

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |

  Scenario: 001, LP only provides LP orders
    Given the parties deposit on asset's general account the following amount:
      | party           | asset | amount    |
      | party1          | ETH   | 100000000 |
      | sellSideProvider | ETH   | 100000000 |
      | buySideProvider  | ETH   | 100000000 |
      | auxiliary        | ETH   | 100000000 |
      | aux2             | ETH   | 100000000 |

# submit our LP
    Then the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type |
      | lp1 | party1 | ETH/DEC19 | 3000              | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | party1 | ETH/DEC19 | 3000              | 0.1 | sell | ASK              | 50         | 10     | amendment |

# get out of auction
    When the parties place the following orders:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | auxiliary | ETH/DEC19 | buy  | 20     | 80    | 0                | TYPE_LIMIT | TIF_GTC | oa-b-1    |
      | auxiliary | ETH/DEC19 | sell | 20     | 120   | 0                | TYPE_LIMIT | TIF_GTC | oa-s-1    |
      | aux2      | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | oa-b-2    |
      | auxiliary | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | oa-s-2    |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

# add a few pegged orders now
    Then the parties place the following pegged orders:
      | party | market id | side | volume | pegged reference | offset |
      | aux2   | ETH/DEC19 | sell | 10     | ASK              | 9      |
      | aux2   | ETH/DEC19 | buy  | 5      | BID              | 9     |

# now consume all the volume on the sell side
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 20     | 120   | 1                | TYPE_LIMIT | TIF_GTC | t1-1      |

   # enter price monitoring auction
    Then the market state should be "STATE_SUSPENDED" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"
    

# now we move add back some volume
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux2   | ETH/DEC19 | sell | 20     | 120   | 0                | TYPE_LIMIT | TIF_GTC | t1-1      |

# now update the time to get the market out of auction
    Given time is updated to "2019-12-01T00:00:00Z"
     # leave price monitoring auction
    Then the market state should be "STATE_ACTIVE" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Scenario: 002, LP provides LP orders and also limit orders
    Given the parties deposit on asset's general account the following amount:
      | party           | asset | amount    |
      | party1          | ETH   | 100000000 |
      | sellSideProvider | ETH   | 100000000 |
      | buySideProvider  | ETH   | 100000000 |
      | auxiliary        | ETH   | 100000000 |
      | aux2             | ETH   | 100000000 |

# submit our LP
    Then the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type |
      | lp1 | aux2  | ETH/DEC19 | 3000              | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | aux2  | ETH/DEC19 | 3000              | 0.1 | sell | ASK              | 50         | 10     | amendment |

# get out of auction
    When the parties place the following orders:
      | party     | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux2      | ETH/DEC19 | buy  | 20     | 80    | 0                | TYPE_LIMIT | TIF_GTC | oa-b-1    |
      | aux2      | ETH/DEC19 | sell | 20     | 120   | 0                | TYPE_LIMIT | TIF_GTC | oa-s-1    |
      | aux2      | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | oa-b-2    |
      | auxiliary | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | oa-s-2    |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

# add a few pegged orders now
    Then the parties place the following pegged orders:
      | party | market id | side | volume | pegged reference | offset |
      | aux2   | ETH/DEC19 | sell | 10     | ASK              | 9      |
      | aux2   | ETH/DEC19 | buy  | 5      | BID              | 9     |

# now consume all the volume on the sell side
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 20     | 120   | 1                | TYPE_LIMIT | TIF_GTC | t1-1      |

   # enter price monitoring auction
    Then the market state should be "STATE_SUSPENDED" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"
    

# now we move add back some volume
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux2   | ETH/DEC19 | sell | 20     | 120   | 0                | TYPE_LIMIT | TIF_GTC | t1-1      |

# now update the time to get the market out of auction
    Given time is updated to "2019-12-01T00:00:00Z"
     # leave price monitoring auction
    Then the market state should be "STATE_ACTIVE" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
