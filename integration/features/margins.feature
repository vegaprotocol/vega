Feature: Test trader accounts

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 | ETH        | ETH   | simple     | 0.11      | 0.1       | 0              | 0               | 0     | 1.4            | 1.2            | 1.1           | 1                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: a trader place a new order in the system, margin are calculated
    Given the traders make the following deposits on asset's general account:
      | trader    | asset | amount       |
      | traderGuy | ETH   | 10000        |
      | trader1   | ETH   | 1000000      |
      | trader2   | ETH   | 1000000      |
      | aux       | ETH   | 100000000000 |

  
     # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     | 
      | aux     | ETH/DEC19 | buy  | 1      | 9     | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT  | TIF_GTC | 

    # Trigger an auction to set the mark price
    Then traders place following orders with references:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader1-1 |
      | trader2 | ETH/DEC19 | sell | 1      | 10000 | 0                | TYPE_LIMIT | TIF_GTC | trader2-1 |
      | trader1 | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period for market "ETH/DEC19" ends
    And the market trading mode for the market "ETH/DEC19" is "TRADING_MODE_CONTINUOUS"
    And the mark price for the market "ETH/DEC19" is "1000"
    Then traders cancel the following orders:
      | trader  | reference |
      | trader1 | trader1-1 |
      | trader2 | trader2-1 |

    Then traders place following orders:
      | trader    | market id | side | volume | price | resulting trades | type       | tif     |
      | traderGuy | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    And the market trading mode for the market "ETH/DEC19" is "TRADING_MODE_CONTINUOUS"
    Then the margins levels for the traders are:
      | trader    | market id | maintenance | search | initial | release |
      | traderGuy | ETH/DEC19 | 100         | 110    | 120     | 140     |
    Then traders have the following account balances:
      | trader    | asset | market id | margin | general |
      | traderGuy | ETH   | ETH/DEC19 | 120    | 9880    |

  Scenario: an order is rejected if a trader have insufficient margin
    Given the traders make the following deposits on asset's general account:
      | trader    | asset | amount  |
      | traderGuy | ETH   | 1       |
      | trader1   | ETH   | 1000000 |
      | trader2   | ETH   | 1000000 |

    # Trigger an auction to set the mark price
    Then traders place following orders with references:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader1-1 |
      | trader2 | ETH/DEC19 | sell | 1      | 10000 | 0                | TYPE_LIMIT | TIF_GTC | trader2-1 |
      | trader1 | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "1000"
    Then traders cancel the following orders:
      | trader  | reference |
      | trader1 | trader1-1 |
      | trader2 | trader2-1 |

    Then traders place the following invalid orders:
      | trader    | market id | side | volume | price | error               | type       | tif      |
      | traderGuy | ETH/DEC19 | sell | 1      | 1000  | margin check failed | TYPE_LIMIT |  TIF_GTC |
    Then the following orders are rejected:
      | trader    | id        | reason                          |
      | traderGuy | ETH/DEC19 | ORDER_ERROR_MARGIN_CHECK_FAILED |
