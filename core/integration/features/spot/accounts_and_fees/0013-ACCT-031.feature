Feature: Spot market

  Scenario: Each party should only have two holding accounts per market:
            one for the the base_asset and one for the quote_asset. (0013-ACCT-031)

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
      | party1 | BTC   | 100    |
      | party2 | ETH   | 10000  |
      | party2 | BTC   | 100    |
    And the average block duration is "1"

    # No orders have been places so we shouldn't have any holding accounts
    And "party1" should have only the following accounts:
      | type                 | asset   | amount |
      | ACCOUNT_TYPE_GENERAL | ETH     | 10000  |
      | ACCOUNT_TYPE_GENERAL | BTC     | 100    |

    And "party2" should have only the following accounts:
      | type                 | asset   | amount |
      | ACCOUNT_TYPE_GENERAL | ETH     | 10000  |
      | ACCOUNT_TYPE_GENERAL | BTC     | 100    |

    # Place some orders to create the holding accounts
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | 
      | party1 | BTC/ETH   | buy  | 100    | 2000  | 0                | TYPE_LIMIT | TIF_GFA | 
      | party1 | BTC/ETH   | sell | 100    | 3000  | 0                | TYPE_LIMIT | TIF_GFA | 
      | party2 | BTC/ETH   | buy  | 100    | 2000  | 0                | TYPE_LIMIT | TIF_GTC | 
      | party2 | BTC/ETH   | sell | 100    | 3000  | 0                | TYPE_LIMIT | TIF_GTC | 

    And "party1" should have only the following accounts:
      | type                 | asset   | 
      | ACCOUNT_TYPE_GENERAL | ETH     | 
      | ACCOUNT_TYPE_HOLDING | ETH     | 
      | ACCOUNT_TYPE_GENERAL | BTC     | 
      | ACCOUNT_TYPE_HOLDING | BTC     | 

    And "party2" should have only the following accounts:
      | type                 | asset   | 
      | ACCOUNT_TYPE_GENERAL | BTC     | 
      | ACCOUNT_TYPE_HOLDING | BTC     | 
      | ACCOUNT_TYPE_GENERAL | ETH     | 
      | ACCOUNT_TYPE_HOLDING | ETH     | 

