Feature: Testing batch orders in spot markets

  Background:
    Given time is updated to "2024-01-01T00:00:00Z"

    Given the following network parameters are set:
      | name                                                | value |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
      | market.value.windowLength                           | 1h    |
    
    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 2              |
      | BTC | 2              |

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.01      | 0.03               |
    Given the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 3600    | 0.999       | 10                |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | decimal places | position decimal places | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | 2              | 2                       | default-basic |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 10000  |
      | party2 | ETH   | 10000  |
      | party4 | BTC   | 100    |
      | party5 | BTC   | 100    |
    And the average block duration is "1"

    # Place some orders to get out of auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    And the opening auction period ends for market "BTC/ETH"
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "1000" for the market "BTC/ETH"

    And the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 959       | 1042      | 0            | 0              | 0             |

  @OrdCan
  Scenario: Funds released by cancellations or amendments within the batch should be immediately available
            for later instructions (0074-BTCH-019)

    # Try to place 3 orders, the third one should fail due to not having enough ETH to put into the holding account
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error |
      | party1 | BTC/ETH   | buy  | 400    | 1000  | 0                | TYPE_LIMIT | TIF_GTC | order-a   |       |
      | party1 | BTC/ETH   | buy  | 400    | 1000  | 0                | TYPE_LIMIT | TIF_GTC | order-b   |       |
      | party1 | BTC/ETH   | buy  | 400    | 1000  | 0                | TYPE_LIMIT | TIF_GTC | order-c   | party does not have sufficient balance to cover the trade and fees |

    Then "party1" should have holding account balance of "8000" for asset "ETH"
    Then the orders should have the following states:
      | party  | market id | side | price | remaining | volume | reference | status          |
      | party1 | BTC/ETH   | buy  | 1000  | 400       | 400    | order-a   | STATUS_ACTIVE   | 
      | party1 | BTC/ETH   | buy  | 1000  | 400       | 400    | order-b   | STATUS_ACTIVE   | 

    # If I cancel an existing order and place a new one of the same size it should work
    Then the party "party1" starts a batch instruction
      Then the party "party1" adds the following cancels to a batch:
        | market id | reference |
        | BTC/ETH   | order-a   |
      Then the party "party1" adds the following orders to a batch:
        | market id | side | volume | price | type        | tif     | reference |
        | BTC/ETH   | buy  | 400    | 1000  | TYPE_LIMIT  | TIF_GTC | order-d   |
    Then the party "party1" submits their batch instruction

    Then "party1" should have holding account balance of "8000" for asset "ETH"
    Then the orders should have the following states:
      | party  | market id | side | price | remaining | volume | reference | status           |
      | party1 | BTC/ETH   | buy  | 1000  | 400       | 400    | order-a   | STATUS_CANCELLED | 
      | party1 | BTC/ETH   | buy  | 1000  | 400       | 400    | order-b   | STATUS_ACTIVE    | 
      | party1 | BTC/ETH   | buy  | 1000  | 400       | 400    | order-d   | STATUS_ACTIVE    | 

    # If we amend an order to size 1, we should be able to place an order of 400 successfully.
    Then the party "party1" starts a batch instruction
      Then the party "party1" adds the following amends to a batch:
        | market id | reference | price | size delta | tif     | 
        | BTC/ETH   | order-b   | 1000  | -399       | TIF_GTC | 

      Then the party "party1" adds the following orders to a batch:
        | market id | side | volume | price | type        | tif     | reference |
        | BTC/ETH   | buy  | 400    | 1000  | TYPE_LIMIT  | TIF_GTC | order-e   |
    Then the party "party1" submits their batch instruction

    Then "party1" should have holding account balance of "8010" for asset "ETH"
    Then the orders should have the following states:
      | party  | market id | side | price | remaining | volume | reference | status           |
      | party1 | BTC/ETH   | buy  | 1000  | 400       | 400    | order-a   | STATUS_CANCELLED | 
      | party1 | BTC/ETH   | buy  | 1000  | 1         | 1      | order-b   | STATUS_ACTIVE    | 
      | party1 | BTC/ETH   | buy  | 1000  | 400       | 400    | order-d   | STATUS_ACTIVE    | 
      | party1 | BTC/ETH   | buy  | 1000  | 400       | 400    | order-e   | STATUS_ACTIVE    | 




