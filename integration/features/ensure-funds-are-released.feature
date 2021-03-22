Feature: Test margins releases on position = 0

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 | BTC        | BTC   | simple     | 0.2       | 0.1       | 0              | 0.016           | 2.0   | 5              | 4              | 3.2           | 1                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: No margin left for fok order as first order
    Given the traders make the following deposits on asset's general account:
      | trader    | asset | amount     |
      | traderGuy | BTC   | 1000000000 |
      | trader1   | BTC   | 1000000    |
      | trader2   | BTC   | 1000000    |
      | aux       | BTC   | 100000    |

  # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     | 
      | aux     | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC19 | sell | 1      | 15001 | 0                | TYPE_LIMIT  | TIF_GTC | 

    # Trigger an auction to set the mark price
    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader1-1 |
      | trader2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | trader2-1 |
      | trader1 | ETH/DEC19 | buy  | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "94"
    Then traders cancel the following orders:
      | trader  | reference |
      | trader1 | trader1-1 |
      | trader2 | trader2-1 |

    When traders place the following orders:
      | trader    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | traderGuy | ETH/DEC19 | buy  | 13     | 15000 | 0                | TYPE_LIMIT | TIF_FOK | ref-1     |
    Then traders have the following account balances:
      | trader    | asset | market id | margin | general    |
      | traderGuy | BTC   | ETH/DEC19 | 0      | 1000000000 |

  Scenario: No margin left for wash trade
    Given the traders make the following deposits on asset's general account:
      | trader    | asset | amount     |
      | traderGuy | BTC   | 1000000000 |
      | trader1   | BTC   | 1000000    |
      | trader2   | BTC   | 1000000    |
      | aux       | BTC   | 100000    |

  # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     | 
      | aux     | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC19 | sell | 1      | 15001 | 0                | TYPE_LIMIT  | TIF_GTC | 

    # Trigger an auction to set the mark price
    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader1-1 |
      | trader2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | trader2-1 |
      | trader1 | ETH/DEC19 | buy  | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "94"
    Then traders cancel the following orders:
      | trader  | reference |
      | trader1 | trader1-1 |
      | trader2 | trader2-1 |

    When traders place the following orders:
      | trader    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | traderGuy | ETH/DEC19 | buy  | 13     | 15000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then traders have the following account balances:
      | trader    | asset | market id | margin | general   |
      | traderGuy | BTC   | ETH/DEC19 | 980    | 999999020 |

# now we place an order which would wash trade and see
    When traders place the following orders:
      | trader    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | traderGuy | ETH/DEC19 | sell | 13     | 15000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |

# checking margins, should have the margins required for the current order
    Then traders have the following account balances:
      | trader    | asset | market id | margin | general   |
      | traderGuy | BTC   | ETH/DEC19 | 980    | 999999020 |

  Scenario: No margin left after cancelling order and getting back to 0 position
    Given the traders make the following deposits on asset's general account:
      | trader    | asset | amount     |
      | traderGuy | BTC   | 1000000000 |
      | trader1   | BTC   | 1000000    |
      | trader2   | BTC   | 1000000    |
      | aux       | BTC   | 100000    |

  # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     | 
      | aux     | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC19 | sell | 1      | 15001 | 0                | TYPE_LIMIT  | TIF_GTC | 

    # Trigger an auction to set the mark price
    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader1-1 |
      | trader2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | trader2-1 |
      | trader1 | ETH/DEC19 | buy  | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "94"
    Then traders cancel the following orders:
      | trader  | reference |
      | trader1 | trader1-1 |
      | trader2 | trader2-1 |

    When traders place the following orders:
      | trader    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | traderGuy | ETH/DEC19 | buy  | 13     | 15000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |

    Then traders have the following account balances:
      | trader    | asset | market id | margin | general   |
      | traderGuy | BTC   | ETH/DEC19 | 980    | 999999020 |

    Then traders cancel the following orders:
      | trader    | reference |
      | traderGuy | ref-1     |

    Then traders have the following account balances:
      | trader    | asset | market id | margin | general    |
      | traderGuy | BTC   | ETH/DEC19 | 0      | 1000000000 |

  Scenario: No margin left for wash trade after cancelling first order
    Given the traders make the following deposits on asset's general account:
      | trader    | asset | amount     |
      | traderGuy | BTC   | 1000000000 |
      | trader1   | BTC   | 1000000    |
      | trader2   | BTC   | 1000000    |
      | aux       | BTC   | 100000    |

  # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     | 
      | aux     | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC19 | sell | 1      | 15001 | 0                | TYPE_LIMIT  | TIF_GTC | 

    # Trigger an auction to set the mark price
    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader1-1 |
      | trader2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | trader2-1 |
      | trader1 | ETH/DEC19 | buy  | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "94"
    Then traders cancel the following orders:
      | trader  | reference |
      | trader1 | trader1-1 |
      | trader2 | trader2-1 |

    When traders place the following orders:
      | trader    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | traderGuy | ETH/DEC19 | buy  | 13     | 15000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |

    Then traders have the following account balances:
      | trader    | asset | market id | margin | general   |
      | traderGuy | BTC   | ETH/DEC19 | 980    | 999999020 |

# now we place an order which would wash trade and see
    When traders place the following orders:
      | trader    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | traderGuy | ETH/DEC19 | sell | 13     | 15000 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

# checking margins, should have the margins required for the current order
    Then traders have the following account balances:
      | trader    | asset | market id | margin | general   |
      | traderGuy | BTC   | ETH/DEC19 | 980    | 999999020 |

# cancel the first order
    Then traders cancel the following orders:
      | trader    | reference |
      | traderGuy | ref-1     |

    Then traders have the following account balances:
      | trader    | asset | market id | margin | general    |
      | traderGuy | BTC   | ETH/DEC19 | 0      | 1000000000 |
