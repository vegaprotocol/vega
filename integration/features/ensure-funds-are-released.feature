Feature: Test margins releases on position = 0

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/long | tau/short | mu |     r | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode | makerFee | infrastructureFee | liquidityFee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | Prob of trading |
      | ETH/DEC19 | ETH      | BTC       | BTC   |        94 | simple     |       0.2 |       0.1 |  0 | 0.016 |   2.0 |              5 |              4 |           3.2 |              42 |           0 | continuous   |        0 |                 0 |            0 |                 0  |                |             |                 | 0.1             |

  Scenario: No margin left for fok order as first order
# setup accounts
    Given the following traders:
      | name             |     amount |
      | traderGuy        | 1000000000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | traderGuy        | BTC   |

# setup previous mark price
    Then traders place following orders:
      | trader           | id        | type | volume |    price | resulting trades | type  | tif |

    Then traders place following orders:
      | trader    | id        | type | volume | price | resulting trades | type       | tif     |
      | traderGuy | ETH/DEC19 | buy  |     13 | 15000 |                0 | TYPE_LIMIT | TIF_FOK |

# checking margins
    Then I expect the trader to have a margin:
      | trader    | asset | id        | margin |   general |
      | traderGuy | BTC   | ETH/DEC19 |      0 | 1000000000 |

  Scenario: No margin left for wash trade
# setup accounts
    Given the following traders:
      | name             |     amount |
      | traderGuy        | 1000000000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | traderGuy        | BTC   |

# setup previous mark price
    Then traders place following orders:
      | trader           | id        | type | volume |    price | resulting trades | type  | tif |

    Then traders place following orders:
      | trader    | id        | type | volume | price | resulting trades | type       | tif     |
      | traderGuy | ETH/DEC19 | buy  |     13 | 15000 |                0 | TYPE_LIMIT | TIF_GTC |

# checking margins
    Then I expect the trader to have a margin:
      | trader    | asset | id        | margin |   general |
      | traderGuy | BTC   | ETH/DEC19 |      980 | 999999020 |

# now we place an order which would wash trade and see
    Then traders place following orders:
      | trader    | id        | type | volume | price | resulting trades | type       | tif     |
      | traderGuy | ETH/DEC19 | sell |     13 | 15000 |                0 | TYPE_LIMIT | TIF_GTC |

# checking margins, should have the margins required for the current order
    Then I expect the trader to have a margin:
      | trader    | asset | id        | margin |   general |
      | traderGuy | BTC   | ETH/DEC19 |    980 | 999999020 |

  Scenario: No margin left after cancelling order and getting back to 0 position
# setup accounts
    Given the following traders:
      | name             |     amount |
      | traderGuy        | 1000000000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | traderGuy        | BTC   |


    Then traders place following orders with references:
      | trader    | id        | type | volume | price | resulting trades | type       | tif     | reference |
      | traderGuy | ETH/DEC19 | buy  |     13 | 15000 |                0 | TYPE_LIMIT | TIF_GTC | ref-1 |

# checking margins
    Then I expect the trader to have a margin:
      | trader    | asset | id        | margin |   general |
      | traderGuy | BTC   | ETH/DEC19 |      980 | 999999020 |

# cancel the order
    Then traders cancels the following orders reference:
      | trader    | reference |
      | traderGuy | ref-1     |

    Then I expect the trader to have a margin:
      | trader    | asset | id        | margin |   general |
      | traderGuy | BTC   | ETH/DEC19 |      0 | 1000000000 |

  Scenario: No margin left for wash trade after cancelling first order
# setup accounts
    Given the following traders:
      | name             |     amount |
      | traderGuy        | 1000000000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | traderGuy        | BTC   |

    Then traders place following orders with references:
      | trader    | id        | type | volume | price | resulting trades | type       | tif     | reference |
      | traderGuy | ETH/DEC19 | buy  |     13 | 15000 |                0 | TYPE_LIMIT | TIF_GTC | ref-1     |

# checking margins
    Then I expect the trader to have a margin:
      | trader    | asset | id        | margin |   general |
      | traderGuy | BTC   | ETH/DEC19 |      980 | 999999020 |

# now we place an order which would wash trade and see
    Then traders place following orders:
      | trader    | id        | type | volume | price | resulting trades | type       | tif     |
      | traderGuy | ETH/DEC19 | sell |     13 | 15000 |                0 | TYPE_LIMIT | TIF_GTC |

# checking margins, should have the margins required for the current order
    Then I expect the trader to have a margin:
      | trader    | asset | id        | margin |   general |
      | traderGuy | BTC   | ETH/DEC19 |    980 | 999999020 |

# cancel the first order
    Then traders cancels the following orders reference:
      | trader    | reference |
      | traderGuy | ref-1     |

    Then I expect the trader to have a margin:
      | trader    | asset | id        | margin |   general |
      | traderGuy | BTC   | ETH/DEC19 |      0 | 1000000000 |
