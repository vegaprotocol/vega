Feature:  Amend the price up above the account balance

  Scenario: Amend the price up above the account balance (0080-SPOT-012)

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
    And the spot markets:
      | id      | name    | base asset | quote asset | risk model           | auction duration | fees          | price monitoring | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | my-simple-risk-model | 1                | fees-config-1 | default-none     | default-basic |
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 1000   |
      | party2 | BTC   | 100    |
    # place orders and generate trades
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 1      | 300   | 0                | TYPE_LIMIT | TIF_GTC | t1-b-1    |
      | party2 | BTC/ETH   | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |
    Then "party2" should have holding account balance of "1" for asset "BTC"
    Then "party1" should have holding account balance of "300" for asset "ETH"

    Then the orders should have the following states:
      | party  | market id | side | volume | remaining | price | status        |
      | party1 | BTC/ETH   | buy  | 1      | 1         | 300   | STATUS_ACTIVE |
      | party2 | BTC/ETH   | sell | 1      | 1         | 1100  | STATUS_ACTIVE |

    Given the parties amend the following orders:
      | party  | reference | price | size delta | tif     | error                                                              |
      | party2 | t2-s-1    | 2100  | 0          | TIF_GTC |                                                                    |
      | party1 | t1-b-1    | 1100  | 0          | TIF_GTC | party does not have sufficient balance to cover the trade and fees |

    Given the parties amend the following orders:
      | party  | reference | price | size delta | tif     |
      | party1 | t1-b-1    | 1000  | 0          | TIF_GTC |
      | party2 | t2-s-1    | 1000  | 0          | TIF_GTC |

    Given the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "1000" for the market "BTC/ETH"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 1    | party2 |

    Then "party2" should have holding account balance of "0" for asset "BTC"
    Then "party1" should have holding account balance of "0" for asset "ETH"
