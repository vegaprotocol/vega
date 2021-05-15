Feature: Long close-out test (see ln 449 of system-tests/grpc/trading/tradesTests.py & https://github.com/vegaprotocol/scenario-runner/tree/develop/scenarios/QA/issues/86)

  Background:

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-4 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | name                           | value  |
      | market.auction.minimumDuration | 1      |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 100   |

  Scenario: https://drive.google.com/file/d/1bYWbNJvG7E-tcqsK26JMu2uGwaqXqm0L/view
    # setup accounts
    Given the traders deposit on asset's general account the following amount:
      | trader | asset | amount    |
      | tt_12  | BTC   | 10000000  |
      | tt_13  | BTC   | 10000000  |
      | tt_14  | BTC   | 10000000  |
      | tt_15  | BTC   | 100       |
      | tt_16  | BTC   | 10000000  |
      | tt_aux | BTC   | 100000000 |
      | t2_aux | BTC   | 100000000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type        | tif     | reference |
      | tt_aux | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT  | TIF_GTC | aux-b-1   |
      | tt_aux | ETH/DEC19 | sell | 1      | 200   | 0                | TYPE_LIMIT  | TIF_GTC | aux-s-1   |
      | t2_aux | ETH/DEC19 | buy  | 1      | 20    | 0                | TYPE_LIMIT  | TIF_GTC | aux-b-2   |
      | tt_aux | ETH/DEC19 | sell | 1      | 20    | 0                | TYPE_LIMIT  | TIF_GTC | aux-s-2   |
    Then the opening auction period ends for market "ETH/DEC19"

    # place orders and generate trades
    When the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference | expires in |
      | tt_12  | ETH/DEC19 | buy  | 5      | 20    | 0                | TYPE_LIMIT | TIF_GTT | tt_12-1   | 3600       |
      | tt_13  | ETH/DEC19 | sell | 5      | 20    | 1                | TYPE_LIMIT | TIF_GTT | tt_13-1   | 3600       |
      | tt_14  | ETH/DEC19 | sell | 2      | 50    | 0                | TYPE_LIMIT | TIF_GTC | tt_14-1   |            |
      | tt_14  | ETH/DEC19 | sell | 2      | 50    | 0                | TYPE_LIMIT | TIF_GTC | tt_14-2   |            |
      | tt_15  | ETH/DEC19 | sell | 2      | 20    | 0                | TYPE_LIMIT | TIF_GTC | tt_15-1   |            |
      | tt_16  | ETH/DEC19 | buy  | 2      | 20    | 1                | TYPE_LIMIT | TIF_GTC | tt_16-1   |            |
      | tt_15  | ETH/DEC19 | sell | 2      | 20    | 0                | TYPE_LIMIT | TIF_GTC | tt_15-2   |            |
      | tt_16  | ETH/DEC19 | buy  | 2      | 20    | 1                | TYPE_LIMIT | TIF_GTC | tt_16-2   |            |


    And the mark price should be "20" for the market "ETH/DEC19"

    # checking margins
    Then the traders should have the following account balances:
      | trader | asset | market id | margin | general |
      | tt_15  | BTC   | ETH/DEC19 | 0      | 0       |

    # then we make sure the insurance pool collected the funds
    And the insurance pool balance should be "100" for the market "ETH/DEC19"

    #check positions
    Then the traders should have the following profit and loss:
      | trader | volume | unrealised pnl | realised pnl |
      | tt_12  | 5      | 0              | 0            |
      | tt_13  | -5     | 0              | 0            |
      | tt_14  | -4     | 120            | 0            |
      | tt_15  | 0      | 0              | -100         |
      | tt_16  | 4      | 0              | 0            |
