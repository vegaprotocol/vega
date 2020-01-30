Feature: Position resolution case 4

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/short | tau/long | mu |     r | sigma | release factor | initial factor | search factor | settlementPrice |
      | ETH/DEC19 | ETH      | BTC       | BTC   |   9400000 | simple     |          0 |        0 |  0 | 0.016 |   2.0 |              3 |              2 |           1.5 |              42 |

  Scenario: case 4 positions resolution
# setup accounts
    Given the following traders:
      | name             |        amount |
      | sellSideProvider | 1000000000000 |
      | buySideProvider  | 1000000000000 |
      | trader1          |     200000000 |
      | trader2          |    1000000000 |
      | trader3          |     300000000 |
      | trader4          |    1000000000 |
      | designatedLooser |    1160000000 |
    Then I Expect the traders to have new general account:
      | name             | asset |
      | trader1          | BTC   |
      | trader2          | BTC   |
      | trader3          | BTC   |
      | trader4          | BTC   |
      | designatedLooser | BTC   |
      | sellSideProvider | BTC   |
      | buySideProvider  | BTC   |
# insurance pool generation - setup orderbook
    Then traders place following orders with references:
      | trader           | id        | type | volume |    price | resulting trades | type  | tif | reference       |
      | sellSideProvider | ETH/DEC19 | sell |    290 | 15000000 |                0 | LIMIT | GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  |      1 | 14000000 |                0 | LIMIT | GTC | buy-provider-1  |

# insurance pool generation - trade
    Then traders place following orders:
      | trader           | id        | type | volume |    price | resulting trades | type  | tif |
      | designatedLooser | ETH/DEC19 | buy  |    290 | 15000000 |                1 | LIMIT | GTC |

# insurance pool generation - modify order book
    Then traders cancels the following orders reference:
      | trader           | reference       |
      | buySideProvider  | buy-provider-1  |
    Then traders place following orders with references:
      | trader          | id        | type | volume |   price | resulting trades | type  | tif | reference      |
      | buySideProvider | ETH/DEC19 | buy  |      1 | 4000000 |                0 | LIMIT | GTC | buy-provider-2 |

# insurance pool generation - set new mark price (and trigger closeout)
    Then traders place following orders:
      | trader           | id        | type | volume |    price | resulting trades | type  | tif |
      | sellSideProvider | ETH/DEC19 | sell |      1 | 12000000 |                0 | LIMIT | GTC |
      | buySideProvider  | ETH/DEC19 | buy  |      1 | 12000000 |                1 | LIMIT | GTC |

# check positions
    Then position API produce the following:
      | trader           | volume | unrealisedPNL | realisedPNL |
      | designatedLooser |      0 |             0 |  -870000000 |

# checking margins
    Then I expect the trader to have a margin:
      | trader           | asset | id        | margin | general |
      | designatedLooser | BTC   | ETH/DEC19 |      0 |       0 |

# then we make sure the insurance pool collected the funds
    And the insurance pool balance is "290000000" for the market "ETH/DEC19"
