Feature: Long close-out test (see ln 293 of system-tests/grpc/trading/tradesTests.py & https://github.com/vegaprotocol/scenario-runner/tree/develop/scenarios/QA/issues/86)

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/long | tau/short | mu | r  | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode | makerFee | infrastructureFee | liquidityFee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | Prob of trading |
      | ETH/DEC19 | ETH      | BTC       | BTC   | 100       | simple     | 0.1       | 0.1       | -1 | -1 | -1    | 1.4            | 1.2            | 1.1           | 100             | 0           | continuous   | 0        | 0                 | 0            |                 0  |                |             |                 | 0.1             |

  Scenario: https://drive.google.com/file/d/1bYWbNJvG7E-tcqsK26JMu2uGwaqXqm0L/view
    # setup accounts
    Given the following traders:
      | name  | amount    |
      | tt_4  | 500000    |
      | tt_5  | 100       |
      | tt_6  | 100000000 |
      | tt_10 | 10000000  |
      | tt_11 | 10000000  |
    Then I Expect the traders to have new general account:
      | name  | asset |
      | tt_4  | BTC   |
      | tt_5  | BTC   |
      | tt_6  | BTC   |
      | tt_10 | BTC   |
      | tt_11 | BTC   |

    # place orders and generate trades
    Then traders place following orders with references:
      | trader | id        | type | volume | price | resulting trades | type        | tif     | reference |
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
    Then I expect the trader to have a margin:
      | trader | asset | id        | margin | general |
      | tt_5   | BTC   | ETH/DEC19 | 0      | 0       |

    # then we make sure the insurance pool collected the funds
    And the insurance pool balance is "0" for the market "ETH/DEC19"

    #Then dump orders

    #check positions
    Then position API produce the following:
      | trader | volume | unrealisedPNL | realisedPNL |
      | tt_4   | 4      | -200          | 0           |
      | tt_5   | 0      | 0             | -100        |
      | tt_6   | -4     | 200           | -30         |
      | tt_10  | 30     | 0             | 0           |
      | tt_11  | -30    | 200           | -70         |
