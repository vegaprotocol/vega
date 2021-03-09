Feature: Long close-out test (see ln 449 of system-tests/grpc/trading/tradesTests.py & https://github.com/vegaprotocol/scenario-runner/tree/develop/scenarios/QA/issues/86)

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | mark price | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlement price | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading |
      | ETH/DEC19 | BTC        | BTC   | 100        | simple     | 0.1       | 0.1       | -1             | -1              | -1    | 1.4            | 1.2            | 1.1           | 100              | 0                | 0.00025   | 0.0005             | 0.001         | 0                  |                |             |                 | 0.1              |

  Scenario: https://drive.google.com/file/d/1bYWbNJvG7E-tcqsK26JMu2uGwaqXqm0L/view
    # setup accounts
    Given the following traders:
      | name  | amount   |
      | tt_12 | 10000000 |
      | tt_13 | 10000000 |
      | tt_14 | 10000000 |
      | tt_15 | 100      |
      | tt_16 | 10000000 |
    Then I Expect the traders to have new general account:
      | name  | asset |
      | tt_12 | BTC   |
      | tt_13 | BTC   |
      | tt_14 | BTC   |
      | tt_15 | BTC   |
      | tt_16 | BTC   |

    # place orders and generate trades
    Then traders place following orders with references:
      | trader | id        | type | volume | price | resulting trades | type       | tif     | reference |
      | tt_12  | ETH/DEC19 | buy  | 5      | 20    | 0                | TYPE_LIMIT | TIF_GTT | tt_12-1   |
      | tt_13  | ETH/DEC19 | sell | 5      | 20    | 1                | TYPE_LIMIT | TIF_GTT | tt_13-1   |
      | tt_14  | ETH/DEC19 | sell | 2      | 50    | 0                | TYPE_LIMIT | TIF_GTC | tt_14-1   |
      | tt_14  | ETH/DEC19 | sell | 2      | 50    | 0                | TYPE_LIMIT | TIF_GTC | tt_14-2   |
      | tt_15  | ETH/DEC19 | sell | 2      | 20    | 0                | TYPE_LIMIT | TIF_GTC | tt_15-1   |
      | tt_16  | ETH/DEC19 | buy  | 2      | 20    | 1                | TYPE_LIMIT | TIF_GTC | tt_16-1   |

    Then dump transfers

    Then traders place following orders with references:
      | tt_15 | ETH/DEC19 | sell | 2 | 20 | 0 | TYPE_LIMIT | TIF_GTC | tt_15-2 |
      | tt_16 | ETH/DEC19 | buy  | 2 | 20 | 1 | TYPE_LIMIT | TIF_GTC | tt_16-2 |

    And the mark price for the market "ETH/DEC19" is "20"

    # checking margins
    Then I expect the trader to have a margin:
      | trader | asset | id        | margin | general |
      | tt_15  | BTC   | ETH/DEC19 | 0      | 0       |

    # then we make sure the insurance pool collected the funds
    #    Note the insurance pool is 96 as tt_15 balance first covers the fees on position resolution order
    #    and only what's left (100+2-6=96) goes into the insurance pool.
    And the insurance pool balance is "96" for the market "ETH/DEC19"

    #check positions
    #   Note that the realisedPNL for tt_15 is -102 as additional 2 was made
    #   on top of initial deposit by earning maker fee on passive orders.
    Then position API produce the following:
      | trader | volume | unrealisedPNL | realisedPNL |
      | tt_12  | 5      | 0             | 0           |
      | tt_13  | -5     | 0             | 0           |
      | tt_14  | -4     | 120           | 0           |
      | tt_15  | 0      | 0             | -102        |
      | tt_16  | 4      | 0             | 0           |

