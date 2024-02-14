Feature: Spot Markets

  Scenario: If the fee rates change for any reason within an auction, the amount required to cover fees must be recalculated at uncrossing time, 
            and the necessary amount should be transferred to or released from the holding_account.(0080-SPOT-021).

  Background:
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
    And the average block duration is "1"

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.05               |
    
    And the simple risk model named "my-simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.01 | 0.01  | 10          | -10           | 0.2                    |

    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 3600    | 0.95        | 3                 |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model           | auction duration | fees          | price monitoring | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | my-simple-risk-model | 1                | fees-config-1 | price-monitoring | default-basic |
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 20000  |
      | party2 | BTC   | 100    |
      | party3 | ETH   | 1200   |
      
    # place orders and generate trades
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | t1-b-1    |
      | party2 | BTC/ETH   | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |
      | party3 | BTC/ETH   | buy  | 10     | 100   | 0                | TYPE_LIMIT | TIF_GTC | bla-bla   |

    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "100" for the market "BTC/ETH"
    When the network moves ahead "1" blocks

    # Check the account details for party3 while in continuous trading
    Then "party3" should have holding account balance of "1000" for asset "ETH"
    Then "party3" should have general account balance of "200" for asset "ETH"

    # Place some orders outside the price range to trigger a price monitoring auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | 
      | party1 | BTC/ETH   | buy  | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | 
      | party2 | BTC/ETH   | sell | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | 

    When the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"

    # We have moved into monitoring auction so will require that fees amounting to 25 are covered in our holding account
    Then "party3" should have holding account balance of "1025" for asset "ETH"
    Then "party3" should have general account balance of "175" for asset "ETH"

    # Let's change the infrastructure fees so that we no longer have enough funds to cover out order
    Given the following network parameters are set:
      | name                                 | value |
      | market.fee.factors.infrastructureFee | 0.5   |

    And the orders should have the following status:
      | party   | reference    | status        |
      | party3  | bla-bla      | STATUS_ACTIVE |

    # Place some new matching orders to get the mark price back to the right level and bring us out of auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | 
      | party1 | BTC/ETH   | buy  | 5      | 101   | 0                | TYPE_LIMIT | TIF_GTC | 
      | party2 | BTC/ETH   | sell | 5      | 101   | 0                | TYPE_LIMIT | TIF_GTC | 

    When the network moves ahead "3" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    # The market will attempt to move the required amount of fees for the orders to the holding account ready for the uncrossing,
    # but there will not be enough for party3, so their order will be cancelled.

    And the orders should have the following status:
      | party   | reference    | status           |
      | party3  | bla-bla      | STATUS_CANCELLED |

    # We move out of auction and everything for party3 will be returned from the holding account to the general account
    Then "party3" should have holding account balance of "0" for asset "ETH"
    Then "party3" should have general account balance of "1200" for asset "ETH"


