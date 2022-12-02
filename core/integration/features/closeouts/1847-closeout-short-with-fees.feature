Feature: Short close-out test (see ln 449 of system-tests/grpc/trading/tradesTests.py & https://github.com/vegaprotocol/scenario-runner/tree/develop/scenarios/QA/issues/86)

  Background:

    Given the fees configuration named "my-fees-config":
      | maker fee | infrastructure fee |
      | 0.00025   | 0.0005             |
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees           | price monitoring | data source config          |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-4 | default-margin-calculator | 1                | my-fees-config | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: https://drive.google.com/file/d/1bYWbNJvG7E-tcqsK26JMu2uGwaqXqm0L/view
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount    |
      | tt_12     | BTC   | 10000000  |
      | tt_13     | BTC   | 10000000  |
      | tt_14     | BTC   | 10000000  |
      | tt_15     | BTC   | 100       |
      | tt_16     | BTC   | 10000000  |
      | tt_aux    | BTC   | 100000000 |
      | t2_aux    | BTC   | 100000000 |
      | party-lp | BTC   | 100000000 |
    And the parties submit the following liquidity provision:
      | id  | party    | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party-lp | ETH/DEC19 | 30000000          | 0.001 | buy  | BID              | 50         | 10     | submission |
      | lp1 | party-lp | ETH/DEC19 | 30000000          | 0.001 | sell | ASK              | 50         | 10     | amendment  |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | tt_aux | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1   |
      | tt_aux | ETH/DEC19 | sell | 1      | 200   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1   |
      | t2_aux | ETH/DEC19 | buy  | 1      | 20    | 0                | TYPE_LIMIT | TIF_GTC | aux-b-2   |
      | tt_aux | ETH/DEC19 | sell | 1      | 20    | 0                | TYPE_LIMIT | TIF_GTC | aux-s-2   |
    Then the opening auction period ends for market "ETH/DEC19"

    # place orders and generate trades
    When the parties place the following orders "1" blocks apart:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | expires in |
      | tt_12  | ETH/DEC19 | buy  | 5      | 20    | 0                | TYPE_LIMIT | TIF_GTT | tt_12-1   | 3600       |
      | tt_13  | ETH/DEC19 | sell | 5      | 20    | 1                | TYPE_LIMIT | TIF_GTT | tt_13-1   | 3600       |
      | tt_14  | ETH/DEC19 | sell | 2      | 50    | 0                | TYPE_LIMIT | TIF_GTC | tt_14-1   |            |
      | tt_14  | ETH/DEC19 | sell | 2      | 50    | 0                | TYPE_LIMIT | TIF_GTC | tt_14-2   |            |
      | tt_15  | ETH/DEC19 | sell | 2      | 20    | 0                | TYPE_LIMIT | TIF_GTC | tt_15-1   |            |
      | tt_16  | ETH/DEC19 | buy  | 2      | 20    | 1                | TYPE_LIMIT | TIF_GTC | tt_16-1   |            |

    When the parties place the following orders "1" blocks apart:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | tt_15 | ETH/DEC19 | sell | 2      | 20    | 0                | TYPE_LIMIT | TIF_GTC | tt_15-2   |
      | tt_16 | ETH/DEC19 | buy  | 2      | 20    | 1                | TYPE_LIMIT | TIF_GTC | tt_16-2   |

    And the mark price should be "20" for the market "ETH/DEC19"

    # checking margins
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general |
      | tt_15 | BTC   | ETH/DEC19 | 0      | 0       |

    And the insurance pool balance should be "0" for the market "ETH/DEC19"

    #check positions
    #   Note that the realised pnl for tt_15 is -102 as additional 2 was made
    #   on top of initial deposit by earning maker fee on passive orders.
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | tt_12  | 5      | 0              | 0            |
      | tt_13  | -5     | 0              | 0            |
      | tt_14  | -4     | 120            | -22          |
      | tt_15  | 0      | 0              | -102         |
      | tt_16  | 4      | 0              | 0            |
