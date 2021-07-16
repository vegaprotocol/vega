Feature: Test margins releases on position = 0

  Background:

    And the markets:
      | id        | quote name | asset | risk model                | margin calculator                  | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |

  Scenario: No margin left for fok order as first order
    Given the traders deposit on asset's general account the following amount:
      | trader    | asset | amount     |
      | traderGuy | BTC   | 1000000000 |
      | trader1   | BTC   | 1000000    |
      | trader2   | BTC   | 1000000    |
      | aux       | BTC   | 100000     |

  # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 1      | 15001 | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader1-1 |
      | trader2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | trader2-1 |
      | trader1 | ETH/DEC19 | buy  | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "94" for the market "ETH/DEC19"
    Then the traders cancel the following orders:
      | trader  | reference |
      | trader1 | trader1-1 |
      | trader2 | trader2-1 |

    When the traders place the following orders:
      | trader    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | traderGuy | ETH/DEC19 | buy  | 13     | 15000 | 0                | TYPE_LIMIT | TIF_FOK | ref-1     |
    Then the traders should have the following account balances:
      | trader    | asset | market id | margin | general    |
      | traderGuy | BTC   | ETH/DEC19 | 0      | 1000000000 |

  Scenario: No margin left for wash trade
    Given the traders deposit on asset's general account the following amount:
      | trader    | asset | amount     |
      | traderGuy | BTC   | 1000000000 |
      | trader1   | BTC   | 1000000    |
      | trader2   | BTC   | 1000000    |
      | aux       | BTC   | 100000     |

  # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 1      | 15001 | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader1-1 |
      | trader2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | trader2-1 |
      | trader1 | ETH/DEC19 | buy  | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "94" for the market "ETH/DEC19"
    Then the traders cancel the following orders:
      | trader  | reference |
      | trader1 | trader1-1 |
      | trader2 | trader2-1 |

    When the traders place the following orders:
      | trader    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | traderGuy | ETH/DEC19 | buy  | 13     | 15000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the traders should have the following account balances:
      | trader    | asset | market id | margin | general   |
      | traderGuy | BTC   | ETH/DEC19 | 980    | 999999020 |

# now we place an order which would wash trade and see
    When the traders place the following orders:
      | trader    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | traderGuy | ETH/DEC19 | sell | 13     | 15000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |

# checking margins, should have the margins required for the current order
    Then the traders should have the following account balances:
      | trader    | asset | market id | margin | general   |
      | traderGuy | BTC   | ETH/DEC19 | 980    | 999999020 |

  Scenario: No margin left after cancelling order and getting back to 0 position
    Given the traders deposit on asset's general account the following amount:
      | trader    | asset | amount     |
      | traderGuy | BTC   | 1000000000 |
      | trader1   | BTC   | 1000000    |
      | trader2   | BTC   | 1000000    |
      | aux       | BTC   | 100000     |

  # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 1      | 15001 | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader1-1 |
      | trader2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | trader2-1 |
      | trader1 | ETH/DEC19 | buy  | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "94" for the market "ETH/DEC19"
    Then the traders cancel the following orders:
      | trader  | reference |
      | trader1 | trader1-1 |
      | trader2 | trader2-1 |

    When the traders place the following orders:
      | trader    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | traderGuy | ETH/DEC19 | buy  | 13     | 15000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |

    Then the traders should have the following account balances:
      | trader    | asset | market id | margin | general   |
      | traderGuy | BTC   | ETH/DEC19 | 980    | 999999020 |

    Then the traders cancel the following orders:
      | trader    | reference |
      | traderGuy | ref-1     |

    Then the traders should have the following account balances:
      | trader    | asset | market id | margin | general    |
      | traderGuy | BTC   | ETH/DEC19 | 0      | 1000000000 |

  Scenario: No margin left for wash trade after cancelling first order
    Given the traders deposit on asset's general account the following amount:
      | trader    | asset | amount     |
      | traderGuy | BTC   | 1000000000 |
      | trader1   | BTC   | 1000000    |
      | trader2   | BTC   | 1000000    |
      | aux       | BTC   | 100000     |

  # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 1      | 15001 | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader1-1 |
      | trader2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | trader2-1 |
      | trader1 | ETH/DEC19 | buy  | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "94" for the market "ETH/DEC19"
    Then the traders cancel the following orders:
      | trader  | reference |
      | trader1 | trader1-1 |
      | trader2 | trader2-1 |

    When the traders place the following orders:
      | trader    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | traderGuy | ETH/DEC19 | buy  | 13     | 15000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |

    Then the traders should have the following account balances:
      | trader    | asset | market id | margin | general   |
      | traderGuy | BTC   | ETH/DEC19 | 980    | 999999020 |

# now we place an order which would wash trade and see
    When the traders place the following orders:
      | trader    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | traderGuy | ETH/DEC19 | sell | 13     | 15000 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

# checking margins, should have the margins required for the current order
    Then the traders should have the following account balances:
      | trader    | asset | market id | margin | general   |
      | traderGuy | BTC   | ETH/DEC19 | 980    | 999999020 |

# cancel the first order
    Then the traders cancel the following orders:
      | trader    | reference |
      | traderGuy | ref-1     |

    Then the traders should have the following account balances:
      | trader    | asset | market id | margin | general    |
      | traderGuy | BTC   | ETH/DEC19 | 0      | 1000000000 |
