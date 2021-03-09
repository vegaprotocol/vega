Feature: Set up a market, with an opening auction, then uncross the book


  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | mark price | risk model | lamd/long              | tau/short              | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlement price | auction duration |  maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC20 | ETH       | ETH        |  100        | simple     | 0.08628781058136630000 | 0.09370922348428490000 | -1             | -1              | -1    | 1.4            | 1.2            | 1.1           | 100              | 1                |  0.004     | 0.001              | 0.3           | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 100   |

  Scenario: set up 2 traders with balance
    # setup accounts
    Given the following traders:
      | name    | amount     |
      | trader1 | 1000000000 |
      | trader2 | 1000000000 |
      | trader3 | 1000000000 |
    Then I Expect the traders to have new general account:
      | name    | asset |
      | trader1 | ETH   |
      | trader2 | ETH   |
      | trader3 | ETH   |

    # place orders and generate trades - slippage 100
    Then traders place following orders with references:
      | trader  | id        | type | volume | price    | resulting trades | type        | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 10500000 | 0                | TYPE_LIMIT  | TIF_GTC | t1-s-1    |
      | trader2 | ETH/DEC20 | buy  | 1      |  9500000 | 0                | TYPE_LIMIT  | TIF_GTC | t2-b-1    |
      | trader1 | ETH/DEC20 | buy  | 1      | 10000000 | 0                | TYPE_LIMIT  | TIF_GFA | t1-b-1    |
      | trader2 | ETH/DEC20 | sell | 1      | 10000000 | 0                | TYPE_LIMIT  | TIF_GFA | t2-s-1    |

    Then the opening auction period for market "ETH/DEC20" ends

    And executed trades:
      | buyer   | price    | size | seller  |
      | trader1 | 10000000 | 1    | trader2 |
    And the mark price for the market "ETH/DEC20" is "10000000"

    Then traders place following orders with references:
      | trader  | id        | type | volume | price    | resulting trades | type        | tif     | reference |
      | trader1 | ETH/DEC20 | buy  | 1      | 10000000 | 0                | TYPE_LIMIT  | TIF_GTC | post-oa-1 |
      | trader3 | ETH/DEC20 | sell | 1      | 10000000 | 1                | TYPE_LIMIT  | TIF_GTC | post-oa-2 |

    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin  | general   |
      | trader3 | ETH   | ETH/DEC20 | 1724511 | 995225489 |
    And dump transfers
    And the following transfers happened:
      | from    | to      | fromType                | toType                           | id        | amount  | asset |
      | trader3 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC20 |   40000 | ETH   |
      | trader3 |         | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           |   10000 | ETH   |
      | trader3 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC20 | 3000000 | ETH   |
      | market  | trader1 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC20 |   40000 | ETH   |
    And clear transfer events

    # Amend orders to set slippage to 120
    Then traders amends the following orders reference:
      | trader  | reference   | price    | sizeDelta | expiresAt | tif     | success |
      | trader1 | t1-s-1      | 12500000 |        0  |         0 | TIF_GTC | true    |
      | trader2 | t2-b-1      | 10500000 |        0  |         0 | TIF_GTC | true    |

    Then traders place following orders with references:
      | trader  | id        | type | volume | price    | resulting trades | type        | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 12000000 | 0                | TYPE_LIMIT  | TIF_GTC | t1-s-2    |
      | trader2 | ETH/DEC20 | buy  | 1      | 12000000 | 1                | TYPE_LIMIT  | TIF_GTC | t2-b-3    |

    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin  | general   |
      | trader3 | ETH   | ETH/DEC20 | 1949413 | 993000587 |

    # MTM loss + margin low
    And the following transfers happened:
      | from    | to      | fromType                | toType                           | id        | amount  | asset |
      | trader3 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_SETTLEMENT          | ETH/DEC20 |  275489 | ETH   |
      | trader3 | trader3 | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_MARGIN              | ETH/DEC20 | 1949413 | ETH   |
    And clear transfer events

    # Amend orders to set slippage to 140
    Then traders amends the following orders reference:
      | trader  | reference   | price    | sizeDelta | expiresAt | tif     | success |
      | trader1 | t1-s-1      | 14500000 |        0  |         0 | TIF_GTC | true    |
      | trader2 | t2-b-1      | 13500000 |        0  |         0 | TIF_GTC | true    |

    Then traders place following orders with references:
      | trader  | id        | type | volume | price    | resulting trades | type        | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 14000000 | 0                | TYPE_LIMIT  | TIF_GTC | t1-s-3    |
      | trader2 | ETH/DEC20 | buy  | 1      | 14000000 | 1                | TYPE_LIMIT  | TIF_GTC | t2-b-4    |

    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin  | general   |
      | trader3 | ETH   | ETH/DEC20 | 2174316 | 990775684 |

    # Check MTM Loss transfer happened
    And the following transfers happened:
      | from    | to      | fromType                | toType                           | id        | amount  | asset |
      | trader3 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_SETTLEMENT          | ETH/DEC20 |   50587 | ETH   |
    And clear transfer events

    # Amend orders to set slippage to 160
    Then traders amends the following orders reference:
      | trader  | reference   | price    | sizeDelta | expiresAt | tif     | success |
      | trader1 | t1-s-1      | 16500000 |        0  |         0 | TIF_GTC | true    |
      | trader2 | t2-b-1      | 15500000 |        0  |         0 | TIF_GTC | true    |

    Then traders place following orders with references:
      | trader  | id        | type | volume | price    | resulting trades | type        | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 16000000 | 0                | TYPE_LIMIT  | TIF_GTC | t1-s-4    |
      | trader2 | ETH/DEC20 | buy  | 1      | 16000000 | 1                | TYPE_LIMIT  | TIF_GTC | t2-b-5    |

    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin  | general   |
      | trader3 | ETH   | ETH/DEC20 | 2399217 | 988550783 |

    # Check MTM Loss transfer happened
    And the following transfers happened:
      | from    | to      | fromType                | toType                           | id        | amount  | asset |
      | trader3 | market  | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT          | ETH/DEC20 | 2000000 | ETH   |
      | trader3 | trader3 | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_MARGIN              | ETH/DEC20 | 2224901 | ETH   |
    And clear transfer events

    # Amend orders to set slippage to 180
    Then traders amends the following orders reference:
      | trader  | reference   | price    | sizeDelta | expiresAt | tif     | success |
      | trader1 | t1-s-1      | 18500000 |        0  |         0 | TIF_GTC | true    |
      | trader2 | t2-b-1      | 17500000 |        0  |         0 | TIF_GTC | true    |

    Then traders place following orders with references:
      | trader  | id        | type | volume | price    | resulting trades | type        | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 18000000 | 0                | TYPE_LIMIT  | TIF_GTC | t1-s-3    |
      | trader2 | ETH/DEC20 | buy  | 1      | 18000000 | 1                | TYPE_LIMIT  | TIF_GTC | t2-b-6    |

    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin  | general   |
      | trader3 | ETH   | ETH/DEC20 | 2624120 | 986325880 |

    # Check MTM Loss transfer happened
    And the following transfers happened:
      | from    | to      | fromType                | toType                           | id        | amount  | asset |
      | trader3 | market  | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT          | ETH/DEC20 | 2000000 | ETH   |
      | trader3 | trader3 | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_MARGIN              | ETH/DEC20 | 2224903 | ETH   |
    And clear transfer events

    # Amend orders to set slippage to 140
    # Amending prices down, so amend buy order first, so it doesn't uncross with the lowered sell order
    Then traders amends the following orders reference:
      | trader  | reference   | price    | sizeDelta | expiresAt | tif     | success |
      | trader2 | t2-b-1      | 13500000 |        0  |         0 | TIF_GTC | true    |
      | trader1 | t1-s-1      | 14500000 |        0  |         0 | TIF_GTC | true    |

    Then traders place following orders with references:
      | trader  | id        | type | volume | price    | resulting trades | type        | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 14000000 | 0                | TYPE_LIMIT  | TIF_GTC | t1-s-4    |
      | trader2 | ETH/DEC20 | buy  | 1      | 14000000 | 1                | TYPE_LIMIT  | TIF_GTC | t2-b-7    |

    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin  | general   |
      | trader3 | ETH   | ETH/DEC20 | 2174316 | 990775684 |

    # Check MTM Loss transfer happened
    And the following transfers happened:
      | from    | to      | fromType                | toType                           | id        | amount  | asset |
      | market  | trader3 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN              | ETH/DEC20 | 4000000 | ETH   |
      | trader3 | trader3 | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_GENERAL             | ETH/DEC20 | 4449804 | ETH   |
    And clear transfer events

    # Amend orders to set slippage to 120
    # Amending prices down, so amend buy order first, so it doesn't uncross with the lowered sell order
    Then traders amends the following orders reference:
      | trader  | reference   | price    | sizeDelta | expiresAt | tif     | success |
      | trader2 | t2-b-1      | 11500000 |        0  |         0 | TIF_GTC | true    |
      | trader1 | t1-s-1      | 12500000 |        0  |         0 | TIF_GTC | true    |

    Then traders place following orders with references:
      | trader  | id        | type | volume | price    | resulting trades | type        | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 12000000 | 0                | TYPE_LIMIT  | TIF_GTC | t1-s-5    |
      | trader2 | ETH/DEC20 | buy  | 1      | 12000000 | 1                | TYPE_LIMIT  | TIF_GTC | t2-b-8    |

    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin  | general   |
      | trader3 | ETH   | ETH/DEC20 | 1949413 | 993000587 |

    # Check MTM Loss transfer happened
    And the following transfers happened:
      | from    | to      | fromType                | toType                           | id        | amount  | asset |
      | market  | trader3 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN              | ETH/DEC20 | 2000000 | ETH   |
      | trader3 | trader3 | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_GENERAL             | ETH/DEC20 | 2224903 | ETH   |
    And clear transfer events

    # Amend orders to set slippage to 110
    # Amending prices down, so amend buy order first, so it doesn't uncross with the lowered sell order
    Then traders amends the following orders reference:
      | trader  | reference   | price    | sizeDelta | expiresAt | tif     | success |
      | trader2 | t2-b-1      | 10500000 |        0  |         0 | TIF_GTC | true    |
      | trader1 | t1-s-1      | 11500000 |        0  |         0 | TIF_GTC | true    |

    Then traders place following orders with references:
      | trader  | id        | type | volume | price    | resulting trades | type        | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 11000000 | 0                | TYPE_LIMIT  | TIF_GTC | t1-s-6    |
      | trader2 | ETH/DEC20 | buy  | 1      | 11000000 | 1                | TYPE_LIMIT  | TIF_GTC | t2-b-9    |

    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin  | general   |
      | trader3 | ETH   | ETH/DEC20 | 1836962 | 994113038 |

    # Check MTM Loss transfer happened
    And the following transfers happened:
      | from    | to      | fromType                | toType                           | id        | amount  | asset |
      | market  | trader3 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN              | ETH/DEC20 | 1000000 | ETH   |
      | trader3 | trader3 | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_GENERAL             | ETH/DEC20 | 1112451 | ETH   |
    And clear transfer events

    # Amend orders to set slippage to 100
    # Amending prices down, so amend buy order first, so it doesn't uncross with the lowered sell order
    Then traders amends the following orders reference:
      | trader  | reference   | price    | sizeDelta | expiresAt | tif     | success |
      | trader2 | t2-b-1      | 9500000  |        0  |         0 | TIF_GTC | true    |
      | trader1 | t1-s-1      | 10500000 |        0  |         0 | TIF_GTC | true    |

    Then traders place following orders with references:
      | trader  | id        | type | volume | price    | resulting trades | type        | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 10000000 | 0                | TYPE_LIMIT  | TIF_GTC | t1-s-7    |
      | trader2 | ETH/DEC20 | buy  | 1      | 10000000 | 1                | TYPE_LIMIT  | TIF_GTC | t2-b-10   |

    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin  | general   |
      | trader3 | ETH   | ETH/DEC20 | 1724511 | 995225489 |

    # Check MTM Loss transfer happened
    And the following transfers happened:
      | from    | to      | fromType                | toType                           | id        | amount  | asset |
      | market  | trader3 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN              | ETH/DEC20 | 1000000 | ETH   |
      | trader3 | trader3 | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_GENERAL             | ETH/DEC20 | 1112451 | ETH   |
    And clear transfer events

    Then traders place following orders with references:
      | trader  | id        | type | volume | price    | resulting trades | type        | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 10000000 | 0                | TYPE_LIMIT  | TIF_GTC | post-oa-3 |
      | trader3 | ETH/DEC20 | buy  | 1      | 10000000 | 1                | TYPE_LIMIT  | TIF_GTC | post-oa-4 |

    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin  | general   |
      | trader3 | ETH   | ETH/DEC20 | 0       | 993900000 |

    And the following transfers happened:
      | from    | to      | fromType                | toType                           | id        | amount  | asset |
      | trader3 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC20 |   40000 | ETH   |
      | trader3 |         | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           |   10000 | ETH   |
      | trader3 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC20 | 3000000 | ETH   |
      | market  | trader1 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC20 |   40000 | ETH   |
    And clear transfer events
