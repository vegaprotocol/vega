Feature: Position esolution case 3

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/long | tau/short | mu |     r | sigma | release factor | initial factor | search factor | settlementPrice |
      | ETH/DEC19 | ETH      | BTC       | BTC   |        94 | simple     |          0 |        0 |  0 | 0.016 |   2.0 |              5 |              4 |           3.2 |              42 |

  Scenario: https://docs.google.com/spreadsheets/d/1D433fpt7FUCk04dZ9FHDVy-4hA6Bw_a2
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
      | sellSideProvider | ETH/DEC19 | sell |    290 |   150 |                0 | LIMIT | GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  |      1 |   140 |                0 | LIMIT | GTC | buy-provider-1  |

# insurance pool generation - trade
    Then traders place following orders:
      | trader           | id        | type | volume | price | resulting trades | type  | tif |
      | designatedLooser | ETH/DEC19 | buy  |    290 |   150 |                1 | LIMIT | GTC |

    Then the margins levels for the traders are:
      | trader           | id        | maintenance | search | initial | release |
      | designatedLooser | ETH/DEC19 |        2900 |   9280 |   11600 |   14500 |

# insurance pool generation - modify order book
    Then traders cancels the following orders reference:
      | trader           | reference       |
      | buySideProvider  | buy-provider-1  |
    Then traders place following orders with references:
      | trader          | id        | type | volume | price | resulting trades | type  | tif | reference      |
      | buySideProvider | ETH/DEC19 | buy  |    400 |    40 |                0 | LIMIT | GTC | buy-provider-2 |

# check the trader accounts
    Then I expect the trader to have a margin:
      | trader           | asset | id        | margin | general |
      | designatedLooser | BTC   | ETH/DEC19 |  11600 |     400 |

# insurance pool generation - set new mark price (and trigger closeout)
    Then traders place following orders:
      | trader           | id        | type | volume | price | resulting trades | type  | tif |
      | sellSideProvider | ETH/DEC19 | sell |      1 |   120 |                0 | LIMIT | GTC |
      | buySideProvider  | ETH/DEC19 | buy  |      1 |   120 |                1 | LIMIT | GTC |

# check positions
    Then position API produce the following:
     | trader           | volume | unrealisedPNL | realisedPNL |
     | designatedLooser |      0 |             0 |      -12000 |

# checking margins
    Then I expect the trader to have a margin:
      | trader           | asset | id        | margin | general |
      | designatedLooser | BTC   | ETH/DEC19 |      0 |       0 |

# then we make sure the insurance pool collected the funds
    And the insurance pool balance is "3300" for the market "ETH/DEC19"
