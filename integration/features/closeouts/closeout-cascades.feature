Feature: closeout-cascases & https://github.com/vegaprotocol/vega/pull/4138/files)

  Background:

    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1             | 1              | 1              | 
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator   | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-4 | margin-calculator-1 | 1                | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |

  Scenario: https://drive.google.com/file/d/1bYWbNJvG7E-tcqsK26JMu2uGwaqXqm0L/view
   # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party        | asset | amount        |
      | trader1      | BTC   | 200           |
      | trader2      | BTC   | 4010          |
      | trader3      | BTC   | 50010         |
      | auxiliary1   | BTC   | 1000000000000 |
      | auxiliary2   | BTC   | 1000000000000 |

  # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party     | market id | side | volume | price  | resulting trades| type       | tif     | reference |
      | auxiliary2| ETH/DEC19 | buy  | 10   | 1      | 0               | TYPE_LIMIT | TIF_GTC | aux-b-1   |
      | auxiliary1| ETH/DEC19 | sell | 10   | 10000   | 0               | TYPE_LIMIT | TIF_GTC | aux-s-1   |
      | auxiliary2| ETH/DEC19 | buy  | 10     | 10     | 0               | TYPE_LIMIT | TIF_GTC | aux-b-1   |
      | auxiliary1| ETH/DEC19 | sell | 10     | 10     | 0               | TYPE_LIMIT | TIF_GTC | aux-s-1   |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

# insurance pool generation - setup orderbook and setup position
    When the parties place the following orders:
      | party     | market id | side | volume| price  | resulting trades | type       | tif     | reference       |
      | trader2   | ETH/DEC19 | buy  | 40    | 100    | 1                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | trader3   | ETH/DEC19 | buy  | 500   | 100    | 1                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | auxiliary1| ETH/DEC19 | sell | 540   | 100    | 1                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | trader1   | ETH/DEC19 | buy  | 40    | 8      | 1                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | trader2   | ETH/DEC19 | buy  | 500   | 9      | 1                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

# check whether trader 1/2/3 are all closed out, trader3 first, and forced to transfer 500 position to trader2 for price 9
# then trader2 is closed out, and forced to transfer 40 position to trader 1 for price 8

    Then the parties should have the following account balances:
      | party     | asset | market id | margin | general |
      | trader1   | BTC   | ETH/DEC19 | 0      | 0       |
      | trader2   | BTC   | ETH/DEC19 | 0      | 0       |
      | trader3   | BTC   | ETH/DEC19 | 0      | 0       |


#check positions
    Then the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl |
      | trader1 | 0      | 0              | -200          |
      | trader2 | 0      | 0              | -4010         |
      | trader3 | 0      | 0              | -50010       |


