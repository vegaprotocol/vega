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
      | party  | asset | amount    |
      | tt_4   | BTC   | 500000    |
      | tt_5   | BTC   | 100       |
      | tt_6   | BTC   | 100000000 |
      | tt_10  | BTC   | 10000000  |
      | tt_11  | BTC   | 10000000  |
      | tt_aux | BTC   | 100000000 |
      | t2_aux | BTC   | 100000000 |
      | lpprov | BTC   | 100000000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | tt_aux | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1   |
      | tt_aux | ETH/DEC19 | sell | 1      | 200   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1   |
      | t2_aux | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-2   |
      | tt_aux | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-2   |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | buy  | MID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | sell | MID              | 50         | 100    | submission |
    Then the opening auction period ends for market "ETH/DEC19"

    # place orders and generate trades
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type        | tif     | reference | expires in |
      | tt_10 | ETH/DEC19 | buy  | 5      | 100   | 0                | TYPE_LIMIT  | TIF_GTT | tt_10-1   | 3600       |
      | tt_11 | ETH/DEC19 | sell | 5      | 100   | 1                | TYPE_LIMIT  | TIF_GTT | tt_11-1   | 3600       |
      | tt_4  | ETH/DEC19 | buy  | 2      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_4-1    |            |
      | tt_4  | ETH/DEC19 | buy  | 2      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_4-2    |            |
      | tt_5  | ETH/DEC19 | buy  | 2      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_5-1    |            |
      | tt_6  | ETH/DEC19 | sell | 2      | 150   | 1                | TYPE_LIMIT  | TIF_GTC | tt_6-1    |            |
      | tt_5  | ETH/DEC19 | buy  | 2      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_5-2    |            |
      | tt_6  | ETH/DEC19 | sell | 2      | 150   | 1                | TYPE_LIMIT  | TIF_GTC | tt_6-2    |            |
      | tt_10 | ETH/DEC19 | buy  | 25     | 100   | 0                | TYPE_LIMIT  | TIF_GTC | tt_10-2   |            |
      | tt_11 | ETH/DEC19 | sell | 25     | 0     | 3                | TYPE_MARKET | TIF_FOK | tt_11-2   |            |

    And the mark price should be "100" for the market "ETH/DEC19"

    # checking margins
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general |
      | tt_5  | BTC   | ETH/DEC19 | 0      | 0       |

    # then we make sure the insurance pool collected the funds
    And the insurance pool balance should be "0" for the market "ETH/DEC19"

    #check positions
    Then the parties should have the following profit and loss:
      | party | volume | unrealised pnl | realised pnl |
      | tt_4  | 4      | -200           | 0            |
      | tt_5  | 0      | 0              | -100         |
      | tt_6  | -4     | 200            | -27          |
      | tt_10 | 30     | 0              | 0            |
      | tt_11 | -30    | 200            | -65          |
