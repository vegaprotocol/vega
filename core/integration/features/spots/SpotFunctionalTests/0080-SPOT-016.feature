Feature: Spot Markets

  Scenario: When entering an auction, for any open "buy" orders, the network must transfer additional funds from the parties'
            general_account to their respective holding_account to cover any potential fees resulting from the order trading
            in the auction.(0080-SPOT-016).

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
      | party3 | ETH   | 10000   |
      
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

    # Place an order outside the price range to trigger a price monitoring auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | 
      | party1 | BTC/ETH   | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | 
      | party2 | BTC/ETH   | sell | 2      | 191   | 0                | TYPE_LIMIT | TIF_GTC | 

    # Check the account details for party3 while in continuous trading
    Then "party3" should have holding account balance of "1000" for asset "ETH"
    Then "party3" should have general account balance of "9000" for asset "ETH"

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error |
      | party1 | BTC/ETH   | buy  | 1      | 191   | 0                | TYPE_LIMIT | TIF_GTC | buy2   |  |

    When the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "BTC/ETH"

    # We are now in a monitoring auction, check that the accounts have been reflected to show the fee amount being budgeted for
    Then "party3" should have holding account balance of "1001" for asset "ETH"
    Then "party3" should have general account balance of "8999" for asset "ETH"



