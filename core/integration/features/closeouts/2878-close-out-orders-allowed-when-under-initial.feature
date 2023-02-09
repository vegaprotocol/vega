Feature: Trader below initial margin, but above maintenance can submit an order to close their own position

  Background:
    Given time is updated to "2020-10-16T00:00:00Z"
    And the markets:
      | id        | quote name | asset | auction duration | risk model                  | margin calculator         | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC20 | ETH        | ETH   | 1                | default-simple-risk-model-3 | default-margin-calculator | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: Trader under initial margin closes out their own position
    Given the parties deposit on asset's general account the following amount:
      | party     | asset | amount         |
      | party1    | ETH   | 10000000000000 |
      | party2    | ETH   | 10000000000000 |
      | party3    | ETH   | 1220           |
      | party4    | ETH   | 10000000000000 |
      | party5    | ETH   | 10000000000000 |
      | auxiliary | ETH   | 100000000000   |
      | aux2      | ETH   | 100000000000   |
      | auxiliary | ETH   | 100000000000   |
      | lpprov    | ETH   | 10000000000000 |
      | party6    | ETH   | 10000000000000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party     | market id | side | volume | price | resulting trades | type       | tif     |
      | auxiliary | ETH/DEC20 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary | ETH/DEC20 | sell | 1      | 200   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2      | ETH/DEC20 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary | ETH/DEC20 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 90000             | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC20 | 90000             | 0.1 | sell | ASK              | 50         | 100    | submission |
    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "100" for the market "ETH/DEC20"

    # T0 + 1min - this causes the price for comparison of the bounds to be 567
    Then time is updated to "2020-10-16T00:01:00Z"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC20 | sell | 10     | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party6 | ETH/DEC20 | sell | 10     | 200   | 0                | TYPE_LIMIT | TIF_GTC | ref-61    |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party5 | ETH/DEC20 | buy  | 10     | 100   | 1                | TYPE_LIMIT | TIF_FOK | ref-1     |
      | party4 | ETH/DEC20 | buy  | 10     | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | party4 | ETH/DEC20 | sell | 10     | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general       |
      | party4 | ETH   | ETH/DEC20 | 132    | 9999999999868 |
      | party3 | ETH   | ETH/DEC20 | 1220   | 0             |
      | party5 | ETH   | ETH/DEC20 | 1320   | 9999999998580 |
    # Value before uint stuff
    # | party4 | ETH   | ETH/DEC20 | 133    | 9999999999867 |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party3 | ETH/DEC20 | 1100        | 1210   | 1320    | 1540    |

    ## Now party 3, though below initial margin places a buy order to close their position out
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party5 | ETH/DEC20 | sell | 20     | 115   | 0                | TYPE_LIMIT | TIF_GTC | ref-6     |
      | party4 | ETH/DEC20 | buy  | 15     | 115   | 1                | TYPE_LIMIT | TIF_GTC | ref-7     |
      | party3 | ETH/DEC20 | buy  | 10     | 115   | 1                | TYPE_LIMIT | TIF_GTC | ref-8     |
    ## The trades have happened, party 3 bought 5 -> margin requirements go down
    Then the mark price should be "115" for the market "ETH/DEC20"
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party3 | ETH/DEC20 | 83          | 91     | 99      | 116     |
    ## Balances of the party accounts reflect the change, total adds up to 1070 -> party3 lost money
    ## as expected, but was able to close their position
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party3 | ETH   | ETH/DEC20 | 99     | 913     |
