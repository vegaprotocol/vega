Feature: Spot Markets

  Scenario: For a "buy" order to be considered valid during an auction, the party must have a sufficient amount of the quote_asset
            to cover the order size, as well as any potential fees that may be incurred due to the order trading
            in the auction (0080-SPOT-018).

  Background:
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
    And the average block duration is "1"
    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the simple risk model named "my-simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.01 | 0.01  | 10          | -10           | 0.2                    |
    And the fees configuration named "my-fees-config":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
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
      | party3 | ETH   | 100    |
      
    # place orders and generate trades
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | t1-b-1    |
      | party2 | BTC/ETH   | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |

    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "100" for the market "BTC/ETH"
    When the network moves ahead "1" blocks

    # Place an order outside the price range to trigger a price monitoring auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | 
      | party1 | BTC/ETH   | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | 
      | party2 | BTC/ETH   | sell | 2      | 191   | 0                | TYPE_LIMIT | TIF_GTC | 

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | 
      | party1 | BTC/ETH   | buy  | 1      | 191   | 0                | TYPE_LIMIT | TIF_GTC | 

    When the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"

    # We have moved into monitoring auction so let's try to submit some orders to test out
    # the transfer of funds from the general to the holding account

    # Trying to place an order at price 100 will require a holding account amount of 100+(0.01*100)=101 which will fail
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error                                                              |
      | party3 | BTC/ETH   | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | buy1      | party does not have sufficient balance to cover the trade and fees |

    # Trying to place an order at price 99 will require a holding account amount of 99+(0.01*99)=100 which will be fine
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error |
      | party3 | BTC/ETH   | buy  | 1      | 99    | 0                | TYPE_LIMIT | TIF_GTC | buy1      |       |





