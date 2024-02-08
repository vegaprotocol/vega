Feature: Spot order gets filled partially among three parties

  Scenario: Spot Order gets filled partially among three parties

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
      | party  | asset | amount     |
      | party1 | ETH   | 100000000  |
      | party2 | BTC   | 5          |
      | party3 | ETH   | 1000000000 |

    # place orders and generate trades
    And the parties place the following orders:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference    |
      | party1 | BTC/ETH   | buy  | 1      | 750000  | 0                | TYPE_LIMIT | TIF_GFA | party-order1 |
      | party2 | BTC/ETH   | sell | 5      | 1000000 | 0                | TYPE_LIMIT | TIF_GTC | party-order2 |
      | party3 | BTC/ETH   | buy  | 5      | 700000  | 0                | TYPE_LIMIT | TIF_GTC | party-order3 |

    And the opening auction period ends for market "BTC/ETH"

    Then "party1" should have holding account balance of "750000" for asset "ETH"
    Then "party2" should have holding account balance of "5" for asset "BTC"
    Then "party3" should have holding account balance of "3500000" for asset "ETH"

    And "party1" should have general account balance of "99250000" for asset "ETH"
    And "party2" should have general account balance of "0" for asset "BTC"
    And "party3" should have general account balance of "996500000" for asset "ETH"

    # Force a partial fill to make sure the assets in the holding account are reduced (0080-SPOT-009)
    And the parties amend the following orders:
      | party  | reference    | price  | size delta | tif     |
      | party2 | party-order2 | 750000 | 0          | TIF_GTC |

    Then "party1" should have holding account balance of "750000" for asset "ETH"
    Then "party2" should have holding account balance of "5" for asset "BTC"
    Then "party3" should have holding account balance of "3500000" for asset "ETH"

    And the opening auction period ends for market "BTC/ETH"

    And the following trades should be executed:
      | buyer  | price  | size | seller |
      | party1 | 750000 | 1    | party2 |

    # holding account is reduced when the order is partially filled (0080-SPOT-013)
    Then "party1" should have holding account balance of "0" for asset "ETH"
    Then "party2" should have holding account balance of "4" for asset "BTC"
    Then "party3" should have holding account balance of "3500000" for asset "ETH"

    And "party1" should have general account balance of "99250000" for asset "ETH"
    And "party1" should have general account balance of "1" for asset "BTC"
    And "party2" should have general account balance of "0" for asset "BTC"
    And "party3" should have general account balance of "996500000" for asset "ETH"

    And the parties amend the following orders:
      | party  | reference    | price  | size delta | tif     |
      | party2 | party-order2 | 700000 | 0          | TIF_GTC |

    And the following trades should be executed:
      | buyer  | price  | size | seller |
      | party3 | 700000 | 4    | party2 |

    Then "party1" should have holding account balance of "0" for asset "ETH"
    Then "party2" should have holding account balance of "0" for asset "BTC"
    Then "party3" should have holding account balance of "700000" for asset "ETH"
    Then "party1" should have general account balance of "99250000" for asset "ETH"
    Then "party1" should have general account balance of "1" for asset "BTC"
    Then "party2" should have general account balance of "3530400" for asset "ETH"
    Then "party2" should have general account balance of "0" for asset "BTC"
    Then "party3" should have general account balance of "996514000" for asset "ETH"
    Then "party3" should have general account balance of "4" for asset "BTC"
