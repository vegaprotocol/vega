Feature: Test margins releases on position = 0

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                | margin calculator                  | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: No margin left for fok order as first order (0011-MARA-003)
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount     |
      | partyGuy | BTC   | 1000000000 |
      | party1   | BTC   | 1000000    |
      | party2   | BTC   | 1000000    |
      | aux      | BTC   | 100000     |
      | lpprov   | BTC   | 100000     |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | sell | ASK              | 50         | 10     | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 15001 | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | party1-1  |
      | party2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party1 | ETH/DEC19 | buy  | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | party1-2  |
      | party2 | ETH/DEC19 | sell | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | party2-2  |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "94" for the market "ETH/DEC19"
    Then the parties cancel the following orders:
      | party  | reference |
      | party1 | party1-1  |
      | party2 | party2-1  |

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | partyGuy | ETH/DEC19 | buy  | 13     | 15000 | 0                | TYPE_LIMIT | TIF_FOK | ref-1     |
    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general    |
      | partyGuy | BTC   | ETH/DEC19 | 0      | 1000000000 |

  Scenario: No margin left for wash trade (0011-MARA-003)
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount     |
      | partyGuy | BTC   | 1000000000 |
      | party1   | BTC   | 1000000    |
      | party2   | BTC   | 1000000    |
      | aux      | BTC   | 100000     |
      | lpprov   | BTC   | 100000     |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | sell | ASK              | 50         | 10     | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 15001 | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | party1-1  |
      | party2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party1 | ETH/DEC19 | buy  | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | party1-2  |
      | party2 | ETH/DEC19 | sell | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | party2-2  |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "94" for the market "ETH/DEC19"
    Then the parties cancel the following orders:
      | party  | reference |
      | party1 | party1-1  |
      | party2 | party2-1  |

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | partyGuy | ETH/DEC19 | buy  | 13     | 15000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general   |
      | partyGuy | BTC   | ETH/DEC19 | 980    | 999999020 |

    # now we place an order which would wash trade and see
    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | partyGuy | ETH/DEC19 | sell | 13     | 15000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |

    # checking margins, should have the margins required for the current order
    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general   |
      | partyGuy | BTC   | ETH/DEC19 | 980    | 999999020 |

  Scenario: No margin left after cancelling order and getting back to 0 position (0011-MARA-003)
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount     |
      | partyGuy | BTC   | 1000000000 |
      | party1   | BTC   | 1000000    |
      | party2   | BTC   | 1000000    |
      | aux      | BTC   | 100000     |
      | lpprov   | BTC   | 100000     |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | sell | ASK              | 50         | 10     | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 15001 | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | party1-1  |
      | party2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party1 | ETH/DEC19 | buy  | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | party1-2  |
      | party2 | ETH/DEC19 | sell | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | party2-2  |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "94" for the market "ETH/DEC19"
    Then the parties cancel the following orders:
      | party  | reference |
      | party1 | party1-1  |
      | party2 | party2-1  |

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | partyGuy | ETH/DEC19 | buy  | 13     | 15000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general   |
      | partyGuy | BTC   | ETH/DEC19 | 980    | 999999020 |

    Then the parties cancel the following orders:
      | party    | reference |
      | partyGuy | ref-1     |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general    |
      | partyGuy | BTC   | ETH/DEC19 | 0      | 1000000000 |

  Scenario: No margin left for wash trade after cancelling first order (0011-MARA-003)
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount     |
      | partyGuy | BTC   | 1000000000 |
      | party1   | BTC   | 1000000    |
      | party2   | BTC   | 1000000    |
      | aux      | BTC   | 100000     |
      | lpprov   | BTC   | 100000     |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | sell | ASK              | 50         | 10     | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 15001 | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | party1-1  |
      | party2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party1 | ETH/DEC19 | buy  | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | party1-2  |
      | party2 | ETH/DEC19 | sell | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | party2-2  |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "94" for the market "ETH/DEC19"
    Then the parties cancel the following orders:
      | party  | reference |
      | party1 | party1-1  |
      | party2 | party2-1  |

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | partyGuy | ETH/DEC19 | buy  | 13     | 15000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general   |
      | partyGuy | BTC   | ETH/DEC19 | 980    | 999999020 |

    # now we place an order which would wash trade and see
    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | partyGuy | ETH/DEC19 | sell | 13     | 15000 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    # checking margins, should have the margins required for the current order
    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general   |
      | partyGuy | BTC   | ETH/DEC19 | 980    | 999999020 |

    # cancel the first order
    Then the parties cancel the following orders:
      | party    | reference |
      | partyGuy | ref-1     |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general    |
      | partyGuy | BTC   | ETH/DEC19 | 0      | 1000000000 |
