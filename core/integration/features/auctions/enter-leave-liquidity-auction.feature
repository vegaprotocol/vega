Feature: Ensure we can enter and leave liquidity auction

  Background:
    Given the following network parameters are set:
      | name                              | value |
      | market.auction.minimumDuration    | 1     |
      | market.stake.target.scalingFactor | 1     |
      | limits.markets.maxPeggedOrders    | 1500  |
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.01                   | 0                         |



  Scenario: 001, LP only provides LP orders
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount    |
      | party1           | ETH   | 100000000 |
      | sellSideProvider | ETH   | 100000000 |
      | buySideProvider  | ETH   | 100000000 |
      | auxiliary        | ETH   | 100000000 |
      | aux2             | ETH   | 100000000 |

    # submit our LP
    Then the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party1 | ETH/DEC19 | 3000              | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | party1 | ETH/DEC19 | 3000              | 0.1 | sell | ASK              | 50         | 10     | submission |

    # get out of auction
    When the parties place the following orders:
      | party     | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | auxiliary | ETH/DEC19 | buy  | 20     | 80    | 0                | TYPE_LIMIT | TIF_GTC | oa-b-1    |
      | auxiliary | ETH/DEC19 | sell | 20     | 120   | 0                | TYPE_LIMIT | TIF_GTC | oa-s-1    |
      | aux2      | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | oa-b-2    |
      | auxiliary | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | oa-s-2    |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # add a few pegged orders now
    Then the parties place the following pegged orders:
      | party | market id | side | volume | pegged reference | offset |
      | aux2  | ETH/DEC19 | sell | 10     | ASK              | 9      |
      | aux2  | ETH/DEC19 | buy  | 5      | BID              | 9      |

    # now consume all the volume on the sell side
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 20     | 120   | 1                | TYPE_LIMIT | TIF_GTC | t1-1      |
    And the network moves ahead "1" blocks

    # enter price monitoring auction
    Then the market state should be "STATE_SUSPENDED" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"


    # now we move add back some volume
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux2  | ETH/DEC19 | sell | 20     | 120   | 0                | TYPE_LIMIT | TIF_GTC | t1-1      |

    # now update the time to get the market out of auction
    Given time is updated to "2019-12-01T00:00:00Z"
    # leave price monitoring auction
    Then the market state should be "STATE_ACTIVE" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

  Scenario: 002, LP provides LP orders and also limit orders
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount    |
      | party1           | ETH   | 100000000 |
      | sellSideProvider | ETH   | 100000000 |
      | buySideProvider  | ETH   | 100000000 |
      | auxiliary        | ETH   | 100000000 |
      | aux2             | ETH   | 100000000 |

    # submit our LP
    Then the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | aux2  | ETH/DEC19 | 3000              | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | aux2  | ETH/DEC19 | 3000              | 0.1 | sell | ASK              | 50         | 10     | submission |

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
      | aux2  | ETH/DEC19 | sell | 10     | ASK              | 9      |
      | aux2  | ETH/DEC19 | buy  | 5      | BID              | 9      |

    # now consume all the volume on the sell side
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 20     | 120   | 1                | TYPE_LIMIT | TIF_GTC | t1-1      |
    And the network moves ahead "1" blocks

    # enter price monitoring auction
    Then the market state should be "STATE_SUSPENDED" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"


    # now we move add back some volume
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux2  | ETH/DEC19 | sell | 20     | 120   | 0                | TYPE_LIMIT | TIF_GTC | t1-1      |

    # now update the time to get the market out of auction
    Given time is updated to "2019-12-01T00:00:00Z"
    # leave price monitoring auction
    Then the market state should be "STATE_ACTIVE" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"


  @LPOA
  Scenario: 003, we do not leave opening auction unless target stake is reached
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount    |
      | party1           | ETH   | 100000000 |
      | sellSideProvider | ETH   | 100000000 |
      | buySideProvider  | ETH   | 100000000 |
      | auxiliary        | ETH   | 100000000 |
      | aux2             | ETH   | 100000000 |

    # submit our LP, amount is 1
    Then the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party1 | ETH/DEC19 | 1                 | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | party1 | ETH/DEC19 | 1                 | 0.1 | sell | ASK              | 50         | 10     | submission |

    # get out of auction
    When the parties place the following orders:
      | party     | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | auxiliary | ETH/DEC19 | buy  | 20     | 80    | 0                | TYPE_LIMIT | TIF_GTC | oa-b-1    |
      | auxiliary | ETH/DEC19 | sell | 20     | 120   | 0                | TYPE_LIMIT | TIF_GTC | oa-s-1    |
      | aux2      | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | oa-b-2    |
      | auxiliary | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | oa-s-2    |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC19"
    And the market data for the market "ETH/DEC19" should be:
      | trading mode                 | extension trigger                        | supplied stake | target stake |
      | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | 1              | 11           |

    # Amend LP, set the commitment amount to be enough to leave opening auction
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type   |
      | lp1 | party1 | ETH/DEC19 | 30000             | 0.1 | buy  | BID              | 50         | 10     | amendment |
      | lp1 | party1 | ETH/DEC19 | 30000             | 0.1 | sell | ASK              | 50         | 10     | amendment |
    Then the market data for the market "ETH/DEC19" should be:
      | trading mode                 | supplied stake | target stake |
      | TRADING_MODE_OPENING_AUCTION | 30000          | 11           |

    # after updating the LP, we now can leave opening auction
    When the network moves ahead "3" blocks
    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | supplied stake | target stake |
      | 100        | TRADING_MODE_CONTINUOUS | 30000          | 11           |

    # add a few pegged orders now
    Then the parties place the following pegged orders:
      | party | market id | side | volume | pegged reference | offset |
      | aux2  | ETH/DEC19 | sell | 10     | ASK              | 9      |
      | aux2  | ETH/DEC19 | buy  | 5      | BID              | 9      |

    # now consume all the volume on the sell side
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 20     | 120   | 1                | TYPE_LIMIT | TIF_GTC | t1-1      |
    And the network moves ahead "1" blocks

    # enter price monitoring auction
    Then the market state should be "STATE_SUSPENDED" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | reference | lp type   |
      | lp1 | party1 | ETH/DEC19 | 3000              | 0.1 | buy  | BID              | 50         | 10     | lp1       | amendment |
      | lp1 | party1 | ETH/DEC19 | 3000              | 0.1 | sell | ASK              | 50         | 10     | lp1       | amendment |

    # now we move add back some volume
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux2  | ETH/DEC19 | sell | 20     | 120   | 0                | TYPE_LIMIT | TIF_GTC | t1-1      |

    # now update the time to get the market out of auction
    Given time is updated to "2019-12-01T00:00:00Z"
    # leave price monitoring auction
    Then the market state should be "STATE_ACTIVE" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
