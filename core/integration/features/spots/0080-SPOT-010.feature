Feature: Spot Markets

  Scenario: If a "sell" order incurs fees through trading, the required amount of the quote_asset to cover the fees will
            be deducted from the total quote_asset resulting from the sale of the base_asset.(0080-SPOT-010)

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
      | party  | asset | amount  |
      | party1 | ETH   | 5000000 |
      | party1 | BTC   | 5000    |
      | party2 | ETH   | 5000000 |
      | party2 | BTC   | 5000    |
      | party4 | ETH   | 5000000 |
      | party4 | BTC   | 5000    |
      | party5 | ETH   | 5000000 |
      | party5 | BTC   | 5000    |
      
    # place orders to get us out of the opening auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party4 | BTC/ETH   | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | open-sell |
      | party5 | BTC/ETH   | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | open-buy  |

    Then the opening auction period ends for market "BTC/ETH"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"
    And the mark price should be "100" for the market "BTC/ETH"
    When the network moves ahead "1" blocks

    # place some orders with an aggressive sell
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | BTC/ETH   | buy  | 1000   | 100   | 0                | TYPE_LIMIT | TIF_GTC | big-buy   |
      | party1 | BTC/ETH   | sell | 1000   | 100   | 1                | TYPE_LIMIT | TIF_GTC | big-sell  |

    # Check that the total amount moved to our general account = (1000*100)*(1-(0.002+0.005))
    #                                                          = (100,000)*(0.993)
    #                                                          = 99,300
    # So we gave up 1000 BTC in exchange for 100,000 ETH and a fee of 700 ETH
    Then "party1" should have general account balance of "5099300" for asset "ETH"
    Then "party1" should have general account balance of "4000" for asset "BTC"


