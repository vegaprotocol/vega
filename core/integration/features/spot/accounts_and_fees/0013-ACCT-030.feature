Feature: Spot market

  Scenario: Every party that submits an order on a Spot market will have a holding account
            created for the relevant market asset pair. (0013-ACCT-030)

  Background:

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
      | 360000  | 0.999       | 1                 |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | decimal places | position decimal places | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | 2              | 2                       | default-basic |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 10000  |
      | party2 | BTC   | 100    |
    And the average block duration is "1"

    # No orders have been places so we shouldn't have any holding accounts
    And "party1" should have only the following accounts:
      | type                 | asset   | amount |
      | ACCOUNT_TYPE_GENERAL | ETH     | 10000  |

    And "party2" should have only the following accounts:
      | type                 | asset   | amount |
      | ACCOUNT_TYPE_GENERAL | BTC     | 100    |

    # Place some orders to create the holding accounts
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | party1 | BTC/ETH   | buy  | 100    | 2000  | 0                | TYPE_LIMIT | TIF_GFA | party-order1111 |
      | party2 | BTC/ETH   | sell | 100    | 3000  | 0                | TYPE_LIMIT | TIF_GTC | party-order2    |

    And "party1" should have only the following accounts:
      | type                 | asset   | 
      | ACCOUNT_TYPE_GENERAL | ETH     | 
      | ACCOUNT_TYPE_HOLDING | ETH     | 

    And "party2" should have only the following accounts:
      | type                 | asset   | 
      | ACCOUNT_TYPE_GENERAL | BTC     | 
      | ACCOUNT_TYPE_HOLDING | BTC     | 

