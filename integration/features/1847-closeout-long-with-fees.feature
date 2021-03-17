Feature: Long close-out test (see ln 293 of system-tests/grpc/trading/tradesTests.py & https://github.com/vegaprotocol/scenario-runner/tree/develop/scenarios/QA/issues/86)

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlement price | auction duration |  maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 |  BTC        | BTC   |  simple     | 0.1       | 0.1       | -1             | -1              | -1    | 1.4            | 1.2            | 1.1           | 100              | 0                |  0.00025   | 0.0005             | 0.001         | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 100   |

  Scenario: https://drive.google.com/file/d/1bYWbNJvG7E-tcqsK26JMu2uGwaqXqm0L/view
    # setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader | asset | amount    |
      | tt_4   | BTC   | 500000    |
      | tt_5   | BTC   | 100       |
      | tt_6   | BTC   | 100000000 |
      | tt_10  | BTC   | 10000000  |
      | tt_11  | BTC   | 10000000  |

    # place orders and generate trades
    Then traders place following orders with references:
      | trader | market id | side | volume | price | resulting trades | type        | tif     | reference |
      | tt_10  | ETH/DEC19 | buy  | 5      | 100   | 0                | TYPE_LIMIT  | TIF_GTT | tt_10-1   |
      | tt_11  | ETH/DEC19 | sell | 5      | 100   | 1                | TYPE_LIMIT  | TIF_GTT | tt_11-1   |
      | tt_4   | ETH/DEC19 | buy  | 2      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_4-1    |
      | tt_4   | ETH/DEC19 | buy  | 2      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_4-2    |
      | tt_5   | ETH/DEC19 | buy  | 2      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_5-1    |
      | tt_6   | ETH/DEC19 | sell | 2      | 150   | 1                | TYPE_LIMIT  | TIF_GTC | tt_6-1    |
      | tt_5   | ETH/DEC19 | buy  | 2      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_5-2    |
      | tt_6   | ETH/DEC19 | sell | 2      | 150   | 1                | TYPE_LIMIT  | TIF_GTC | tt_6-2    |
      | tt_10  | ETH/DEC19 | buy  | 25     | 100   | 0                | TYPE_LIMIT  | TIF_GTC | tt_10-2   |
      | tt_11  | ETH/DEC19 | sell | 25     | 0     | 3                | TYPE_MARKET | TIF_FOK | tt_11-2   |

    And the mark price for the market "ETH/DEC19" is "100"

    # checking margins
    Then traders have the following account balances:
      | trader | asset | market id | margin | general |
      | tt_5   | BTC   | ETH/DEC19 | 0      | 0       |

    # then we make sure the insurance pool collected the funds
    And the insurance pool balance is "0" for the market "ETH/DEC19"

    #check positions
    #   Note that the realised pnl for tt_15 is -102 as additional 2 was made
    #   on top of initial deposit by earning maker fee on passive orders.
    #   That same income was used to pay up a higher portion of the 200 owed in MTM
    #   settlement by tt_15, hence lower realised pnl loss for tt_11 compared to
    #   the no fees case. The benefit for tt_6 is not visible due to rounding.
    Then traders have the following profit and loss:
      | trader | volume | unrealised pnl | realised pnl |
      | tt_4   | 4      | -200           | 0            |
      | tt_5   | 0      | 0              | -102         |
      | tt_6   | -4     | 200            | -30          |
      | tt_10  | 30     | 0              | 0            |
      | tt_11  | -30    | 200            | -68          |
