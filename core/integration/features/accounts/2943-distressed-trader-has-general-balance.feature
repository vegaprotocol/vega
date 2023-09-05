Feature: Distressed parties should not have general balance left

  Background:
    Given time is updated to "2020-10-16T00:00:00Z"
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC20 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       | default-futures |
    And the following network parameters are set:
      | name                                               | value |
      | market.auction.minimumDuration                     | 1     |
      | network.markPriceUpdateMaximumFrequency            | 0s    |
      | limits.markets.maxPeggedOrders                     | 4     |
      | market.liquidity.providersFeeCalculationTimeStep | 1s    |

  Scenario: Upper bound breached
    Given the parties deposit on asset's general account the following amount:
      | party     | asset | amount         |
      | party1    | ETH   | 10000000000000 |
      | party2    | ETH   | 10000000000000 |
      | party3    | ETH   | 24000          |
      | party4    | ETH   | 10000000000000 |
      | party5    | ETH   | 10000000000000 |
      | auxiliary | ETH   | 100000000000   |
      | aux2      | ETH   | 100000000000   |
      | lpprov    | ETH   | 10000000000000 |

    # Provide LP so market can leave opening auction
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 10000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference    |
      | lpprov | ETH/DEC20 | 2         | 1                    | buy  | BID              | 50     | 10     | lp1-ice-buy  |
      | lpprov | ETH/DEC20 | 2         | 1                    | sell | ASK              | 50     | 10     | lp1-ice-sell |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party     | market id | side | volume | price | resulting trades | type       | tif     |
      | auxiliary | ETH/DEC20 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary | ETH/DEC20 | sell | 1      | 200   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2      | ETH/DEC20 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary | ETH/DEC20 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders "1" blocks apart:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC20 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC20 | buy  | 1      | 100   | 1                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC20 | sell | 20     | 120   | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC20 | buy  | 20     | 80    | 0                | TYPE_LIMIT | TIF_GTC |

    And the mark price should be "100" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    # T0 + 1min - this causes the price for comparison of the bounds to be 567
    Then time is updated to "2020-10-16T00:01:00Z"

    When the parties place the following orders "1" blocks apart:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/DEC20 | sell | 10     | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | ETH/DEC20 | buy  | 10     | 100   | 1                | TYPE_LIMIT | TIF_FOK |
      | party3 | ETH/DEC20 | buy  | 10     | 110   | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC20 | sell | 40     | 120   | 0                | TYPE_LIMIT | TIF_GTC |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general       |
      | party4 | ETH   | ETH/DEC20 | 360    | 9999999999640 |
      | party5 | ETH   | ETH/DEC20 | 372    | 9999999999528 |
    Then the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp2 | party3 | ETH/DEC20 | 20000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:    
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference    |
      | party3 | ETH/DEC20 | 189       | 1                    | buy  | BID              | 189    | 10     | lp2-ice-buy  |
      | party3 | ETH/DEC20 | 117       | 1                    | sell | ASK              | 117    | 10     | lp2-ice-sell |
    
    Then the liquidity provisions should have the following states:
      | id  | party  | market    | commitment amount | status         |
      | lp2 | party3 | ETH/DEC20 | 20000             | STATUS_PENDING |

    When the network moves ahead "1" blocks
    Then the liquidity provisions should have the following states:
      | id  | party  | market    | commitment amount | status        |
      | lp2 | party3 | ETH/DEC20 | 20000             | STATUS_ACTIVE |
    
    Then the orders should have the following states:
      | party  | market id | side | volume | price | status        |
      | party3 | ETH/DEC20 | buy  | 189    | 100   | STATUS_ACTIVE |
      | party3 | ETH/DEC20 | sell | 117    | 130   | STATUS_ACTIVE |
    ## The sum of the margin + general account == 24000 - 10000 (commitment amount)
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party3 | ETH   | ETH/DEC20 | 2626   | 1374    |
    
    ## Now let's increase the mark price so party3 gets distressed
    When the parties place the following orders "1" blocks apart:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party5 | ETH/DEC20 | buy  | 40     | 165   | 2                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party1 | ETH/DEC20 | sell | 20     | 1850  | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
    Then the mark price should be "120" for the market "ETH/DEC20"
    Then the liquidity provider fee shares for the market "ETH/DEC20" should be:
      | party   | equity like share  | average entry valuation |
      | lpprov  | 0.6428571428571429 | 10000                  |
      | party3  | 0.3571428571428571 | 60000.0000000000000556 |
    Then debug transfers
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party5 | 120   | 20    | party1 |
      | party5 | 120   | 20    | party3 |
   Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party3 | ETH   | ETH/DEC20 | 3152   | 1040    |
    Then the parties cancel the following orders:
       | party  | reference    |
       | party3 | lp2-ice-sell | 
       | party3 | lp2-ice-buy  | 

    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | party3 | ETH/DEC20 | 189       | 1                    | buy  | BID              | 189    | 10     |
      | party3 | ETH/DEC20 | 136       | 1                    | sell | ASK              | 136    | 10     |
    
  Then the network moves ahead "10" blocks

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general       |
      | party3 | ETH   | ETH/DEC20 | 3152   | 1040          |
      | party4 | ETH   | ETH/DEC20 | 160    | 9999999999640 |
    Then debug detailed orderbook volumes for market "ETH/DEC20"
    ## Now let's increase the mark price so party3 gets distressed
    When the parties place the following orders "1" blocks apart:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | ETH/DEC20 | sell | 30     | 165   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party5 | ETH/DEC20 | buy  | 30     | 165   | 3                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the mark price should be "130" for the market "ETH/DEC20"

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party3 | ETH   | ETH/DEC20 | 4899   | 0       |
