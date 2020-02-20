Feature: 691 - order book cleanup (removing confiscated orders) on liquidation

  Background:
    Given the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/short | tau/long | mu |     r | sigma | release factor | initial factor | search factor | settlementPrice |
      | ETH/DEC19 | ETH      | BTC       | BTC   |        94 | simple     |          0 |        0 |  0 | 0.016 |   2.0 |              5 |              4 |           3.2 |              10 |
    And the following traders:
      | name             |        amount |
      | sellSideProvider | 1000000000000 |
      | theTrader        |            10 |

  Scenario: trader places unmatched order and creates a position. Then trader places very expensive order and gets liquidated and previous orders should be removed
    Given I Expect the traders to have new general account:
      | name             | asset |
      | sellSideProvider | BTC   |
      | theTrader        | BTC   |

# putting orders that must be removed on liquidation
    Then traders place following orders:
      | trader           | id        | type | volume | price | resulting trades | type  | tif |
      | theTrader        | ETH/DEC19 | buy  |     10 |    10 |                0 | LIMIT | GTC |
      | theTrader        | ETH/DEC19 | buy  |      7 |    10 |                0 | LIMIT | GTC |

# putting orders that will trigger liquidation for theTrader
    Then traders place following orders:
      | trader           | id        | type | volume | price | resulting trades | type  | tif |
      | sellSideProvider | ETH/DEC19 | sell |     10 |   500 |                0 | LIMIT | GTC |
      | theTrader        | ETH/DEC19 | buy  |     10 |   500 |                1 | LIMIT | GTC |
    And the mark price for the market "ETH/DEC19" is "500"

# check positions - should be liquidated
#   Then position API produce the following:
#     | trader    | volume | unrealisedPNL | realisedPNL |
#     | theTrader |      0 |             0 |           0 |

# checking margins - should be empty
#    Then I expect the trader to have a margin:
#      | trader    | asset | id        | margin | general |
#      | theTrader | BTC   | ETH/DEC19 |      0 |       0 |

# putting orders that should not be matched against old orders theTrader put before
    Then traders place following orders:
      | trader           | id        | type | volume | price | resulting trades | type  | tif |
      | sellSideProvider | ETH/DEC19 | sell |    100 |    10 |                0 | LIMIT | GTC |