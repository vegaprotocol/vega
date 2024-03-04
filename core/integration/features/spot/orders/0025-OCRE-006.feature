Feature: In Spot market, holding will taken before the order is entered into the book

  Background:    
    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 2              |
      | BTC | 2              |

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
      | id      | name    | base asset | quote asset | risk model           | auction duration | fees          | price monitoring | sla params    | tick size |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | my-simple-risk-model | 1                | fees-config-1 | default-none     | default-basic |     1     |

    And the average block duration is "1"

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | ETH   | 1000000  |
      | party2 | ETH   | 1000000  |
      | party3 | BTC   | 100      |
      | party4 | BTC   | 100      |
      | party5 | BTC   | 10000    |
      | party6 | ETH   | 100      |

  Scenario: In Spot market, holding will taken before the order is entered into the book (0025-OCRE-005, 0025-OCRE-006)
    # Place an iceberg order that we want to full match
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | only | reference |
      | party1 | BTC/ETH   | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | 2         | 2                    | post | iceberg1  |


    # Place normal GFA orders to match with the full amount of the iceberg order
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party5 | BTC/ETH   | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GFA | sell1     |

    And the opening auction period ends for market "BTC/ETH"
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    And the parties place the following orders:
      | party  | market id | side | volume  | price | resulting trades | type       | tif     | reference | error |
      | party3 | BTC/ETH   | sell | 1       | 1000  | 0                | TYPE_LIMIT | TIF_GTC | p3-s1     |       |
      | party6 | BTC/ETH   | buy  | 100     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | p6-s1     | party does not have sufficient balance to cover the trade and fees  |

    Then the orders should have the following states:
      | party  | market id   | reference | side | volume | remaining | price | status        |
      | party3 | BTC/ETH     | p3-s1     | sell | 1      | 1         | 1000  | STATUS_ACTIVE |
