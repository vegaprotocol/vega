Feature: Set up a spot market, with an opening auction, then uncross the book. Make sure opening auction can end.
  Background:

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the simple risk model named "my-simple-risk-model":
      | long                   | short                  | max move up | min move down | probability of trading |
      | 0.08628781058136630000 | 0.09370922348428490000 | -1          | -1            | 0.2                    |
    And the fees configuration named "my-fees-config":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 5     |
      | market.auction.maximumDuration | 10s   |
    And the spot markets:
      | id      | name    | base asset | quote asset | risk model           | auction duration | fees          | price monitoring | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | my-simple-risk-model | 5                | fees-config-1 | default-none     | default-basic |

  Scenario: 0043-MKTL-013 Ensure spot markets get cancelled if they fail to leave opening auction
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 100000 |
      | party2 | ETH   | 100000 |
      | party3 | ETH   | 100000 |
      | party2 | BTC   | 5      |

    Then the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | party3 | BTC/ETH   | 3000              | 0.1 | submission |
    # place orders and generate trades
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | BTC/ETH   | buy  | 1      | 95    | 0                | TYPE_LIMIT | TIF_GTC | t2-b-1    |
      | party1 | BTC/ETH   | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | t1-b-1    |
      | party3 | BTC/ETH   | buy  | 1      | 110   | 0                | TYPE_LIMIT | TIF_GFA | t3-b-1    |

    Then "party1" should have holding account balance of "100" for asset "ETH"

    When the network moves ahead "1" blocks
    Then the market data for the market "BTC/ETH" should be:
      | trading mode                 |
      | TRADING_MODE_OPENING_AUCTION |

    When the network moves ahead "9" blocks
    Then the last market state should be "STATE_PENDING" for the market "BTC/ETH"

    When the network moves ahead "1" blocks
    Then the last market state should be "STATE_CANCELLED" for the market "BTC/ETH"

    #orders are cancelled
    And the orders should have the following status:
      | party  | reference | status           |
      | party2 | t2-b-1    | STATUS_CANCELLED |
      | party1 | t1-b-1    | STATUS_CANCELLED |
      | party3 | t3-b-1    | STATUS_CANCELLED |

    #asset is released for party with orders and LP commitment
    Then "party1" should have general account balance of "100000" for asset "ETH"
    Then "party2" should have general account balance of "100000" for asset "ETH"
    Then "party3" should have general account balance of "100000" for asset "ETH"

