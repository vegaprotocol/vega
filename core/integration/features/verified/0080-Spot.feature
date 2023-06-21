Feature: Spot market

  Scenario: Spot Order gets filled partially

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
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.9999999   | 3                 |
    And the spot markets:
      | id      | name    | base asset | quote asset | risk model           | auction duration | fees          | price monitoring   |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | my-simple-risk-model | 1                | fees-config-1 | price-monitoring-1 |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 100    |
      | party2 | BTC   | 5      |

    # place orders and generate trades
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party1 | BTC/ETH   | buy  | 1      | 20    | 0                | TYPE_LIMIT | TIF_GFA | party-order1  |
      | party2 | BTC/ETH   | sell | 1      | 30    | 0                | TYPE_LIMIT | TIF_GTC | party-order2  |
      | party1 | BTC/ETH   | buy  | 2      | 10    | 0                | TYPE_LIMIT | TIF_GTC | party-order11 |
      | party2 | BTC/ETH   | sell | 1      | 90    | 0                | TYPE_LIMIT | TIF_GTC | party-order12 |

    Then "party1" should have holding account balance of "40" for asset "ETH"
    Then "party2" should have holding account balance of "2" for asset "BTC"

    And the parties amend the following orders:
      | party  | reference    | price | size delta | tif     |
      | party2 | party-order2 | 10    | 0          | TIF_GTC |

    And the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"

    Then "party1" should have holding account balance of "25" for asset "ETH"
    Then "party2" should have holding account balance of "1" for asset "BTC"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 15    | 1    | party2 |

#so far party1 has 1 order left, size 2, price 10

    And the parties amend the following orders:
      | party  | reference     | price | size delta | tif     |
      | party2 | party-order12 | 10    | 0          | TIF_GTC |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 15    | 1    | party2 |

#so far party1 has 1 order left, size 1, price 10

    Then "party1" should have holding account balance of "26" for asset "ETH"
    Then "party2" should have holding account balance of "0" for asset "BTC"




