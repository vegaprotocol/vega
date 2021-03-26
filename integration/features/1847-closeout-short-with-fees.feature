Feature: Long close-out test (see ln 449 of system-tests/grpc/trading/tradesTests.py & https://github.com/vegaprotocol/scenario-runner/tree/develop/scenarios/QA/issues/86)

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 | BTC        | BTC   | simple     | 0.1       | 0.1       | -1             | -1              | -1    | 1.4            | 1.2            | 1.1           | 1                | 0.00025   | 0.0005             | 0.001         | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And the following network parameters are set:
      | market.auction.minimumDuration |
      | 1                              |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 100   |

  Scenario: https://drive.google.com/file/d/1bYWbNJvG7E-tcqsK26JMu2uGwaqXqm0L/view
    # setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader | asset | amount    |
      | tt_12  | BTC   | 10000000  |
      | tt_13  | BTC   | 10000000  |
      | tt_14  | BTC   | 10000000  |
      | tt_15  | BTC   | 100       |
      | tt_16  | BTC   | 10000000  |
      | tt_aux | BTC   | 100000000 |
      | t2_aux | BTC   | 100000000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type        | tif     | reference |
      | tt_aux | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT  | TIF_GTC | aux-b-1   |
      | tt_aux | ETH/DEC19 | sell | 1      | 200   | 0                | TYPE_LIMIT  | TIF_GTC | aux-s-1   |
      | t2_aux | ETH/DEC19 | buy  | 1      | 20    | 0                | TYPE_LIMIT  | TIF_GTC | aux-b-2   |
      | tt_aux | ETH/DEC19 | sell | 1      | 20    | 0                | TYPE_LIMIT  | TIF_GTC | aux-s-2   |
    Then the opening auction period for market "ETH/DEC19" ends

    # place orders and generate trades
    When traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | tt_12  | ETH/DEC19 | buy  | 5      | 20    | 0                | TYPE_LIMIT | TIF_GTT | tt_12-1   |
      | tt_13  | ETH/DEC19 | sell | 5      | 20    | 1                | TYPE_LIMIT | TIF_GTT | tt_13-1   |
      | tt_14  | ETH/DEC19 | sell | 2      | 50    | 0                | TYPE_LIMIT | TIF_GTC | tt_14-1   |
      | tt_14  | ETH/DEC19 | sell | 2      | 50    | 0                | TYPE_LIMIT | TIF_GTC | tt_14-2   |
      | tt_15  | ETH/DEC19 | sell | 2      | 20    | 0                | TYPE_LIMIT | TIF_GTC | tt_15-1   |
      | tt_16  | ETH/DEC19 | buy  | 2      | 20    | 1                | TYPE_LIMIT | TIF_GTC | tt_16-1   |

    When traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | tt_15  | ETH/DEC19 | sell | 2      | 20    | 0                | TYPE_LIMIT | TIF_GTC | tt_15-2   |
      | tt_16  | ETH/DEC19 | buy  | 2      | 20    | 1                | TYPE_LIMIT | TIF_GTC | tt_16-2   |

    And the mark price for the market "ETH/DEC19" is "20"

    # checking margins
    Then traders have the following account balances:
      | trader | asset | market id | margin | general |
      | tt_15  | BTC   | ETH/DEC19 | 0      | 0       |

    # then we make sure the insurance pool collected the funds
    #    Note the insurance pool is 96 as tt_15 balance first covers the fees on position resolution order
    #    and only what's left (100+2-6=96) goes into the insurance pool.
    And the insurance pool balance is "96" for the market "ETH/DEC19"

    #check positions
    #   Note that the realised pnl for tt_15 is -102 as additional 2 was made
    #   on top of initial deposit by earning maker fee on passive orders.
    Then traders have the following profit and loss:
      | trader | volume | unrealised pnl | realised pnl |
      | tt_12  | 5      | 0              | 0            |
      | tt_13  | -5     | 0              | 0            |
      | tt_14  | -4     | 120            | 0            |
      | tt_15  | 0      | 0              | -102         |
      | tt_16  | 4      | 0              | 0            |
