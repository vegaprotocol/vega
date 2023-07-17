Feature: Simple Spot Order between two parties match successfully
  Scenario: Simple Spot Order matches with counter party
  Background:
    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 5              |
    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.005     | 0.002              |
    And the simple risk model named "my-simple-risk-model":
      | long                   | short                  | max move up | min move down | probability of trading |
      | 0.08628781058136630000 | 0.09370922348428490000 | -1          | -1            | 0.2                    |
    And the fees configuration named "my-fees-config":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the spot markets:
      | id      | name    | base asset | quote asset | risk model           | auction duration | fees          | price monitoring | decimal places | position decimal places | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | my-simple-risk-model | 1                | fees-config-1 | default-none     | 3              | 5                       | default-basic |
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | ETH   | 10000000 |
      | party2 | BTC   | 100      |
    # place orders and generate trades
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 100000 | 10000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-1111 |
      | party2 | BTC/ETH   | sell | 100000 | 10000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1111 |

    Then "party2" should have holding account balance of "1" for asset "BTC"
    Then "party1" should have holding account balance of "1000000" for asset "ETH"

    Then the orders should have the following states:
      | party  | market id | side | volume | price | status        |
      | party1 | BTC/ETH   | buy  | 100000 | 10000 | STATUS_ACTIVE |
      | party2 | BTC/ETH   | sell | 100000 | 10000 | STATUS_ACTIVE |

    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "10000" for the market "BTC/ETH"

    Then debug trades

    And the following trades should be executed:
      | buyer  | price | size   | seller |
      | party1 | 10000 | 100000 | party2 |
    Then "party2" should have holding account balance of "0" for asset "BTC"
    Then "party1" should have holding account balance of "0" for asset "ETH"