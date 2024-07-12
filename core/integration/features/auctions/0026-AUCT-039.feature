Feature: Set up a market, with an opening auction, then uncross the book. Make sure opening auction can end even if we don't have best bid/ask after uncrossing.


  Background:
    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 5              |
    And the simple risk model named "my-simple-risk-model":
      | long                   | short                  | max move up | min move down | probability of trading |
      | 0.08628781058136630000 | 0.09370922348428490000 | -1          | -1            | 0.2                    |
    And the fees configuration named "my-fees-config":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the markets:
      | id        | quote name | asset | risk model           | margin calculator         | auction duration | fees           | price monitoring | data source config     | decimal places | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC20 | ETH        | ETH   | my-simple-risk-model | default-margin-calculator | 1                | my-fees-config | default-none     | default-eth-for-future | 2              | 0.25                   | 0                         | default-futures |
    And the following network parameters are set:
      | name                                    | value |
      | limits.markets.maxPeggedOrders          | 2     |
    And the average block duration is "1"
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount        |
      | party1 | ETH   | 1000000000000 |
      | party2 | ETH   | 1000000000000 |
      | party3 | ETH   | 1000000000000 |
      | lpprov | ETH   | 1000000000000 |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 937000000         | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC20 | 937000000         | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset | 
      | lpprov | ETH/DEC20 | 2         | 1                    | buy  | BID              | 50     | 100    | 
      | lpprov | ETH/DEC20 | 2         | 1                    | sell | ASK              | 50     | 100    | 

  @NewAuct
  Scenario: 0026-AUCT-039: No best bid after leaving opening auction. Also covers 0016-PFUT-025: normal futures can be submitted without specifying the capped futures fields - all futures tests do this.
    # place orders and generate trades - slippage 100
    When the parties place the following orders:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | buy  | 1      | 1000000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-1    |
      | party2 | ETH/DEC20 | sell | 2      | 1000000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |
    Then the market data for the market "ETH/DEC20" should be:
      | trading mode                 | supplied stake | target stake |
      | TRADING_MODE_OPENING_AUCTION | 937000000      | 937000000    |

    When the network moves ahead "2" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    #And debug detailed orderbook volumes for market "ETH/DEC20"
    And the order book should have the following volumes for market "ETH/DEC20":
      | volume | price   | side |
      | 1      | 1000000 | sell |
      | 2      | 1000100 | sell |

    And the following trades should be executed:
      | buyer  | price   | size | seller |
      | party1 | 1000000 | 1    | party2 |
    And the mark price should be "1000000" for the market "ETH/DEC20"


  @NewAuct
  Scenario: 0026-AUCT-040: No best ask after leaving opening auction.
    # place orders and generate trades - slippage 100
    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | buy  | 2      | 999500 | 0                | TYPE_LIMIT | TIF_GTC | t1-b-1    |
      | party2 | ETH/DEC20 | sell | 1      | 999500 | 0                | TYPE_LIMIT | TIF_GFA | t2-s-1    |
    Then the market data for the market "ETH/DEC20" should be:
      | trading mode                 | supplied stake | target stake |
      | TRADING_MODE_OPENING_AUCTION | 937000000      | 936531500    |

    When the network moves ahead "2" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    #And debug detailed orderbook volumes for market "ETH/DEC20"
    And the order book should have the following volumes for market "ETH/DEC20":
      | volume | price  | side |
      | 1      | 999500 | buy  |

    And the following trades should be executed:
      | buyer  | price  | size | seller |
      | party1 | 999500 | 1    | party2 |
    And the mark price should be "999500" for the market "ETH/DEC20"

