Feature: Setting up 5 traders so that at once all the orders are places they end up with the following margin account balances: tt_5_0: 23 = searchLevel + 1, tt_5_1: 22=searchLevel, tt_5_2: 21=maintenanceLevel+1=searchLevel-1, tt_5_3=maintenanceLevel, tt_5_4=maintenanceLevel-1


  Background:

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-4 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 100   |

  Scenario: https://drive.google.com/file/d/1bYWbNJvG7E-tcqsK26JMu2uGwaqXqm0L/view
    # setup accounts
    Given the traders deposit on asset's general account the following amount:
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
    When the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | tt_aux | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | tt_aux | ETH/DEC19 | sell | 1      | 200   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 1      | 200   | 0                | TYPE_LIMIT | TIF_GTC | t1-s-1    |
      | trader2 | ETH/DEC19 | buy  | 1      | 95    | 0                | TYPE_LIMIT | TIF_GTC | t2-b-1    |
      | trader1 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | t1-b-1    |
      | trader2 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | t2-s-1    |

    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "100" for the market "ETH/DEC19"

    # place orders and generate trades
    When the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type        | tif     | reference | expires in |
      | tt_10  | ETH/DEC19 | buy  | 10     | 100   | 0                | TYPE_LIMIT  | TIF_GTT | tt_10-1   | 3600       |
      | tt_11  | ETH/DEC19 | sell | 10     | 100   | 1                | TYPE_LIMIT  | TIF_GTT | tt_11-1   | 3600       |
      | tt_4   | ETH/DEC19 | buy  | 5      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_4-1    |            |
      | tt_4   | ETH/DEC19 | buy  | 5      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_4-2    |            |
      | tt_5_0 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_5_0-1  |            |
      | tt_5_1 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_5_1-1  |            |
      | tt_5_2 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_5_2-1  |            |
      | tt_5_3 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_5_3-1  |            |
      | tt_5_4 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_5_4-1  |            |
      | tt_6   | ETH/DEC19 | sell | 5      | 150   | 1                | TYPE_LIMIT  | TIF_GTC | tt_6-1    |            |
      | tt_5_0 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_5_0-2  |            |
      | tt_5_1 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_5_1-2  |            |
      | tt_5_2 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_5_2-2  |            |
      | tt_5_3 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_5_3-2  |            |
      | tt_5_4 | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_5_4-2  |            |
      | tt_6   | ETH/DEC19 | sell | 5      | 150   | 1                | TYPE_LIMIT  | TIF_GTC | tt_6-2    |            |
      | tt_10  | ETH/DEC19 | buy  | 25     | 100   | 0                | TYPE_LIMIT  | TIF_GTC | tt_10-2   |            |
      | tt_11  | ETH/DEC19 | sell | 25     | 0     | 11               | TYPE_MARKET | TIF_FOK | tt_11-2   |            |


    And the mark price should be "100" for the market "ETH/DEC19"

    # checking margins
    And the traders should have the following margin levels:
      | trader | market id | maintenance | search | initial | release |
      | tt_5_0 | ETH/DEC19 | 20          | 22     | 24      | 28      |
      | tt_5_1 | ETH/DEC19 | 20          | 22     | 24      | 28      |
      | tt_5_2 | ETH/DEC19 | 20          | 22     | 24      | 28      |
      | tt_5_3 | ETH/DEC19 | 20          | 22     | 24      | 28      |
      | tt_5_4 | ETH/DEC19 | 0           | 0      | 0       | 0       |

    # checking balances
    Then the traders should have the following account balances:
      | trader | asset | market id | margin | general |
      | tt_5_0 | BTC   | ETH/DEC19 | 23     | 0       |
      | tt_5_1 | BTC   | ETH/DEC19 | 22     | 0       |
      | tt_5_2 | BTC   | ETH/DEC19 | 21     | 0       |
      | tt_5_3 | BTC   | ETH/DEC19 | 20     | 0       |
      | tt_5_4 | BTC   | ETH/DEC19 | 0      | 0       |
