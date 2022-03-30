Feature: Test traded price and mark price 

Scenario: using lognomal risk model, test traded price at the end of the auction; 0026-AUCT-006; 0026-AUCT-008
  Background:

    And the log normal risk model named "lognormal-risk-model-fish":
      | risk aversion | tau  | mu | r     | sigma |
      | 0.001         | 0.01 | 0  | 0.0   | 1.2   |
      #calculated risk factor long: 0.336895684; risk factor short: 0.4878731

    And the price monitoring updated every "1" seconds named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99999999  | 300               |

    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 2              |

    And the markets:
      | id        | quote name | asset | risk model                | margin calculator   | auction duration | fees         | price monitoring  | oracle config          |
      | ETH/DEC19 | ETH        | USD   | lognormal-risk-model-fish | margin-calculator-1 | 1                | default-none | default-none | default-eth-for-future |

    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |

# setup accounts
    Given the initial insurance pool balance is "15000" for the markets:
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount     |
      | sellSideProvider | USD   | 1000000000 |
      | buySideProvider  | USD   | 1000000000 |
      | party1           | USD   | 30000      |
      | party2           | USD   | 50000000   |
      | party3           | USD   | 30000      |
      | aux1             | USD   | 1000000000 |
      | aux2             | USD   | 1000000000 |
     #And the cumulated balance for all accounts should be worth "4050075000"
# setup order book
    When the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 1000   | 150   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
     # | party1           | ETH/DEC19 | sell | 100    | 120   | 0                | TYPE_LIMIT | TIF_GTC | party1-s-1      |
      | aux1             | ETH/DEC19 | sell | 1      | 90   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-2         |
      | aux2             | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-2         |
      | party2           | ETH/DEC19 | buy  | 100    | 80    | 0                | TYPE_LIMIT | TIF_GTC | party2-b-1      |
      | buySideProvider  | ETH/DEC19 | buy  | 1000   | 70    | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |

    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "95" for the market "ETH/DEC19"
    #testing the traded price here, the traded price during auction should be the mid price of the max price range
    # https://github.com/vegaprotocol/specs-internal/blob/master/protocol/0026-AUCT-auctions.md
    # bug report: traded price should be 95 instead of 90
    Then the auction ends with a traded volume of "1" at a price of "95"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | party1           | ETH/DEC19 | sell | 100    | 120   | 0                | TYPE_LIMIT | TIF_GTC | party1-s-1      |

 # party1 margin account: MarginInitialFactor x MaintenanceMarginLevel = 4879*1.5=7318
    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general  |
      | party1  | USD   | ETH/DEC19 | 6952   |  23048   |

 