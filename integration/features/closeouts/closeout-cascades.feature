Feature: Long close-out test (see ln 293 of system-tests/grpc/trading/tradesTests.py & https://github.com/vegaprotocol/scenario-runner/tree/develop/scenarios/QA/issues/86)

  Background:
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-4 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |

  Scenario: https://drive.google.com/file/d/1bYWbNJvG7E-tcqsK26JMu2uGwaqXqm0L/view
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party | asset | amount    |
      | trader1   | BTC   | 59000  |
      | trader2   | BTC   | 54500  |
      | trader3   | BTC   | 45100 |
      | trader4_LP  | BTC   | 10000000 |
      | trader5   | BTC   | 10000 |
      | trader6   | BTC   | 10000 |

#place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
#set up position for trader1, trader2 and trader3
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | trader4_LP     | ETH/DEC19 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | trader4_LP     | ETH/DEC19 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC19 | buy  | 100   | 90  | 1                | TYPE_LIMIT | TIF_GTC |
      | trader4_LP  | ETH/DEC19 | sell | 100      | 90   | 1                | TYPE_LIMIT | TIF_GTC |
      | trader2  | ETH/DEC19 | buy  | 50      | 90   | 1                | TYPE_LIMIT | TIF_GTC |
      | trader4_LP  | ETH/DEC19 | sell | 50      | 90   | 1                | TYPE_LIMIT | TIF_GTC |
      | trader3 | ETH/DEC19 | buy  | 500    | 90   | 1                | TYPE_LIMIT | TIF_GTC |
      | trader4_LP   | ETH/DEC19 | sell | 500     | 90  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "94" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the mark price should be "94" for the market "ETH/DEC19"

       # place orders and generates positions 
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference | expires in |
      | trader1  | ETH/DEC19 | sell | 3      | 250    | 0                | TYPE_LIMIT | TIF_GTT | tt_12-1   |       |
      | trader1  | ETH/DEC19 | sell | 5      | 240    | 0                | TYPE_LIMIT | TIF_GTT | tt_13-1   |       |
      | trader1  | ETH/DEC19 | sell | 3      | 190    | 0                | TYPE_LIMIT | TIF_GTC | tt_14-1   |            |
      | trader2  | ETH/DEC19 | buy | 1      | 130    | 0                | TYPE_LIMIT | TIF_GTC | tt_14-2   |            |
      | trader2  | ETH/DEC19 | buy | 4      | 110    | 0                | TYPE_LIMIT | TIF_GTC | tt_15-1   |            |
      | trader4_LP  | ETH/DEC19 | buy | 5      | 140    | 0                | TYPE_LIMIT | TIF_GTC | tt_16-1   |            |
      | trader5 | ETH/DEC19 | sell | 5      | 140    | 0                | TYPE_LIMIT | TIF_GTC | tt_16-1   |            |

    # checking margins
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general |
      | trader3   | BTC   | ETH/DEC19 | 0      | 0       |

    # then we make sure the insurance pool collected the funds
    And the insurance pool balance should be "0" for the market "ETH/DEC19"



