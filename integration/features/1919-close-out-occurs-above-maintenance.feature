Feature: Setting up 5 traders so that at once all the orders are places they end up with the following margin account balances: tt_5_0: 23 = searchLevel + 1, tt_5_1: 22=searchLevel, tt_5_2: 21=maintenanceLevel+1=searchLevel-1, tt_5_3=maintenanceLevel, tt_5_4=maintenanceLevel-1


  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 | BTC        | BTC   | simple     | 0.1       | 0.1       | -1             | -1              | -1    | 1.4            | 1.2            | 1.1           | 1                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 100   |

  Scenario: https://drive.google.com/file/d/1bYWbNJvG7E-tcqsK26JMu2uGwaqXqm0L/view
    # setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount    |
      | tt_4    | BTC   | 500000    |
      | tt_5_0  | BTC   | 123       |
      | tt_5_1  | BTC   | 122       |
      | tt_5_2  | BTC   | 121       |
      | tt_5_3  | BTC   | 120       |
      | tt_5_4  | BTC   | 119       |
      | tt_6    | BTC   | 100000000 |
      | tt_10   | BTC   | 10000000  |
      | tt_11   | BTC   | 10000000  |
      | trader1 | BTC   | 100000000 |
      | trader2 | BTC   | 100000000 |
      | tt_aux  | BTC   | 100000000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type        | tif     | 
      | tt_aux  | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT  | TIF_GTC | 
      | tt_aux  | ETH/DEC19 | sell | 1      | 200   | 0                | TYPE_LIMIT  | TIF_GTC | 

    Then traders place following orders with references:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 1      | 200   | 0                | TYPE_LIMIT | TIF_GTC | t1-s-1    |
      | trader2 | ETH/DEC19 | buy  | 1      | 95    | 0                | TYPE_LIMIT | TIF_GTC | t2-b-1    |
      | trader1 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | t1-b-1    |
      | trader2 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | t2-s-1    |

    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "100"

    # place orders and generate trades
    Then traders place following orders with references:
      | trader | market id | side | volume | price | resulting trades | type        | tif     | reference |
      | tt_10  | ETH/DEC19 | buy  | 10     | 100   | 0                | TYPE_LIMIT  | TIF_GTT | tt_10-1   |
      | tt_11  | ETH/DEC19 | sell | 10     | 100   | 1                | TYPE_LIMIT  | TIF_GTT | tt_11-1   |
      | tt_4   | ETH/DEC19 | buy  | 5      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_4-1    |
      | tt_4   | ETH/DEC19 | buy  | 5      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_4-2    |
      | tt_5_0 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_5_0-1  |
      | tt_5_1 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_5_1-1  |
      | tt_5_2 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_5_2-1  |
      | tt_5_3 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_5_3-1  |
      | tt_5_4 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_5_4-1  |
      | tt_6   | ETH/DEC19 | sell | 5      | 150   | 1                | TYPE_LIMIT  | TIF_GTC | tt_6-1    |
      | tt_5_0 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_5_0-2  |
      | tt_5_1 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_5_1-2  |
      | tt_5_2 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_5_2-2  |
      | tt_5_3 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_5_3-2  |
      | tt_5_4 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_5_4-2  |
      | tt_6   | ETH/DEC19 | sell | 5      | 150   | 1                | TYPE_LIMIT  | TIF_GTC | tt_6-2    |
      | tt_10  | ETH/DEC19 | buy  | 25     | 100   | 0                | TYPE_LIMIT  | TIF_GTC | tt_10-2   |
      | tt_11  | ETH/DEC19 | sell | 25     | 0     | 11               | TYPE_MARKET | TIF_FOK | tt_11-2   |


    And the mark price for the market "ETH/DEC19" is "100"

    # checking margins
    And the margins levels for the traders are:
      | trader | market id | maintenance | search | initial | release |
      | tt_5_0 | ETH/DEC19 | 20          | 22     | 24      | 28      |
      | tt_5_1 | ETH/DEC19 | 20          | 22     | 24      | 28      |
      | tt_5_2 | ETH/DEC19 | 20          | 22     | 24      | 28      |
      | tt_5_3 | ETH/DEC19 | 20          | 22     | 24      | 28      |
      | tt_5_4 | ETH/DEC19 | 0           | 0      | 0       | 0       |

    # checking balances
    Then traders have the following account balances:
      | trader | asset | market id | margin | general |
      | tt_5_0 | BTC   | ETH/DEC19 | 23     | 0       |
      | tt_5_1 | BTC   | ETH/DEC19 | 22     | 0       |
      | tt_5_2 | BTC   | ETH/DEC19 | 21     | 0       |
      | tt_5_3 | BTC   | ETH/DEC19 | 20     | 0       |
      | tt_5_4 | BTC   | ETH/DEC19 | 0      | 0       |
