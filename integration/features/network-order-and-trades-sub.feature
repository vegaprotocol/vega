Feature: Ensure network trader are generated

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/long | tau/short | mu |     r | sigma | release factor | initial factor | search factor | settlementPrice |
      | ETH/DEC19 | ETH      | BTC       | BTC   |        94 | simple     |          0 |        0 |  0 | 0.016 |   2.0 |              5 |              4 |           3.2 |              42 |

  Scenario: Implement trade and order network
# setup accounts
    Given the following traders:
      | name             |        amount |
      | sellSideProvider | 1000000000000 |
      | buySideProvider  | 1000000000000 |
      | designatedLooser |         12000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | designatedLooser | BTC   |
      | sellSideProvider | BTC   |
      | buySideProvider  | BTC   |

# insurance pool generation - setup orderbook
    Then traders place following orders with references:
      | trader           | id        | type | volume | price | resulting trades | type  | tif | reference       |
      | sellSideProvider | ETH/DEC19 | sell |    290 |   150 |                0 | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  |      1 |   140 |                0 | TYPE_LIMIT | TIF_GTC | buy-provider-1  |

# insurance pool generation - trade
    Then traders place following orders:
      | trader           | id        | type | volume | price | resulting trades | type  | tif |
      | designatedLooser | ETH/DEC19 | buy  |    290 |   150 |                1 | TYPE_LIMIT | TIF_GTC |

# insurance pool generation - modify order book
    Then traders cancels the following orders reference:
      | trader           | reference       |
      | buySideProvider  | buy-provider-1  |
    Then traders place following orders with references:
      | trader          | id        | type | volume | price | resulting trades | type  | tif | reference      |
      | buySideProvider | ETH/DEC19 | buy  |    400 |    40 |                0 | TYPE_LIMIT | TIF_GTC | buy-provider-2 |

# insurance pool generation - set new mark price (and trigger closeout)
    Then traders place following orders:
      | trader           | id        | type | volume | price | resulting trades | type  | tif |
      | sellSideProvider | ETH/DEC19 | sell |      1 |   120 |                0 | TYPE_LIMIT | TIF_GTC |
      | buySideProvider  | ETH/DEC19 | buy  |      1 |   120 |                1 | TYPE_LIMIT | TIF_GTC |

# check the network trade happened
    Then the following network trades happened:
      | trader           | aggressor side | volume |
      | designatedLooser | buy            |    290 |
      | buySideProvider  | sell           |    290 |
