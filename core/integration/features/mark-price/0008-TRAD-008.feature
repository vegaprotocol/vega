Feature: Mark price move after trades complete in a spot market (0008-TRAD-008)

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

  Scenario: Following all of the matching of trades resulting from a single order or the acceptance of an order onto the order book,
            there may be a change to the Mark Price (0008-TRAD-008).

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

    # Placing an order above the current mark price that does not match with anything will not change the mark price
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | 
      | party1 | BTC/ETH   | buy  | 1      | 101   | 0                | TYPE_LIMIT | TIF_GTC | 
    When the network moves ahead "1" blocks
    And the mark price should be "100" for the market "BTC/ETH"

    # Placing an order that fills with an existing order will move the mark price to the matched price (101)
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     | 
      | party2 | BTC/ETH   | sell | 1      | 101   | 1                | TYPE_MARKET | TIF_IOC | 
    When the network moves ahead "1" blocks
    And the mark price should be "101" for the market "BTC/ETH"

    # An unmatched order does not move the mark price
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | 
      | party1 | BTC/ETH   | buy  | 1      | 102   | 0                | TYPE_LIMIT | TIF_GTC | 
    When the network moves ahead "1" blocks
    And the mark price should be "101" for the market "BTC/ETH"

    # This order matches and moves the mark price (102)
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | 
      | party2 | BTC/ETH   | sell | 1      | 102   | 1                | TYPE_LIMIT | TIF_GTC | 
    When the network moves ahead "1" blocks
    And the mark price should be "102" for the market "BTC/ETH"

    # Placing an order below the current mark price does not move it
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | 
      | party1 | BTC/ETH   | buy  | 1      | 101   | 0                | TYPE_LIMIT | TIF_GTC | 
    When the network moves ahead "1" blocks
    And the mark price should be "102" for the market "BTC/ETH"

    # Matching with the order below the current mark price will move the mark price (101)
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | 
      | party2 | BTC/ETH   | sell | 1      | 101   | 1                | TYPE_LIMIT | TIF_GTC | 
    When the network moves ahead "1" blocks
    And the mark price should be "101" for the market "BTC/ETH"
