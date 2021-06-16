Feature: Test margin release on order cancel

  Background:

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: a trader place a new order in the system, margin are updated, the order is closed, margin is 0ed
    Given the traders deposit on asset's general account the following amount:
      | trader    | asset | amount       |
      | traderGuy | ETH   | 10000        |
      | trader1   | ETH   | 1000000      |
      | trader2   | ETH   | 1000000      |
      | aux       | ETH   | 100000000000 |


     # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     |
      | aux     | ETH/DEC19 | buy  | 1      | 9     | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT  | TIF_GTC |

    # Trigger an auction to set the mark price
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader1-1 |
      | trader2 | ETH/DEC19 | sell | 1      | 10000 | 0                | TYPE_LIMIT | TIF_GTC | trader2-1 |
      | trader1 | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"

    When the traders place the following orders:
      | trader    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | traderGuy | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |

    Then the traders should have the following margin levels:
      | trader    | market id | maintenance | search | initial | release |
      | traderGuy | ETH/DEC19 | 100         | 110    | 120     | 140     |

    And the traders should have the following account balances:
      | trader    | asset | market id | margin | general |
      | traderGuy | ETH   | ETH/DEC19 | 120    | 9880    |

    When the traders cancel the following orders:
      | trader    | reference |
      | traderGuy | ref-1     |

    Then the traders should have the following account balances:
      | trader    | asset | market id | margin | general |
      | traderGuy | ETH   | ETH/DEC19 |      0 | 10000   |


  Scenario: a trader place a new market order in the system, order, trade, party margin is updated, then place an GTC order which will trade, margin is 0ed
    Given the traders deposit on asset's general account the following amount:
      | trader    | asset | amount       |
      | traderGuy | ETH   | 10000        |
      | trader1   | ETH   | 1000000      |
      | trader2   | ETH   | 1000000      |
      | aux       | ETH   | 100000000000 |


     # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     |
      | aux     | ETH/DEC19 | buy  | 1      | 9     | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT  | TIF_GTC |

    # Trigger an auction to set the mark price
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader1-1 |
      | trader1 | ETH/DEC19 | buy  | 10     | 990   | 0                | TYPE_LIMIT | TIF_GTC | trader1-1 |
      | trader2 | ETH/DEC19 | sell | 1      | 10000 | 0                | TYPE_LIMIT | TIF_GTC | trader2-1 |
      | trader2 | ETH/DEC19 | sell | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | trader2-1 |
      | trader1 | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"

    When the traders place the following orders:
      | trader    | market id | side | volume | price | resulting trades | type        | tif     | reference |
      | traderGuy | ETH/DEC19 | sell | 1      |     0 | 0                | TYPE_MARKET | TIF_IOC | ref-1     |

    Then the traders should have the following margin levels:
      | trader    | market id | maintenance | search | initial | release |
      | traderGuy | ETH/DEC19 | 119         | 130    | 142     | 166     |

    And the traders should have the following account balances:
      | trader    | asset | market id | margin | general |
      | traderGuy | ETH   | ETH/DEC19 | 142    | 9858    |

    Then the traders should have the following profit and loss:
      | trader    | volume | unrealised pnl | realised pnl |
      | traderGuy | -1     | 0              | 0            |

    When the traders place the following orders:
      | trader    | market id | side | volume | price | resulting trades | type        | tif     | reference |
      | traderGuy | ETH/DEC19 | buy  | 1      |  1005 | 0                | TYPE_LIMIT  | TIF_GTC | ref-2     |

    Then the traders should have the following margin levels:
      | trader    | market id | maintenance | search | initial | release |
      | traderGuy | ETH/DEC19 | 119         | 130    | 142     | 166     |

    When the traders place the following orders:
      | trader    | market id | side | volume | price | resulting trades | type        | tif     | reference |
      | trader1   | ETH/DEC19 | sell  | 1     |  1005 | 0                | TYPE_LIMIT  | TIF_GTC | ref-2     |

    Then the traders should have the following margin levels:
      | trader    | market id | maintenance | search | initial | release |
      | traderGuy | ETH/DEC19 | 0           | 0      | 0       | 0       |

    And the traders should have the following account balances:
      | trader    | asset | market id | margin | general |
      | traderGuy | ETH   | ETH/DEC19 | 0      | 9985    |


  Scenario: a trader place a new order in the system, trader is closed out with only potential position, margin is 0ed
    Given the traders deposit on asset's general account the following amount:
      | trader    | asset | amount       |
      | traderGuy | ETH   | 120          |
      | trader1   | ETH   | 1000000      |
      | trader2   | ETH   | 1000000      |
      | aux       | ETH   | 100000000000 |


     # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     |
      | aux     | ETH/DEC19 | buy  | 1      | 9     | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT  | TIF_GTC |

    # Trigger an auction to set the mark price
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader1-1 |
      | trader2 | ETH/DEC19 | sell | 1      | 10000 | 0                | TYPE_LIMIT | TIF_GTC | trader2-1 |
      | trader1 | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"

    When the traders place the following orders:
      | trader    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | traderGuy | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |

    Then the traders should have the following margin levels:
      | trader    | market id | maintenance | search | initial | release |
      | traderGuy | ETH/DEC19 | 100         | 110    | 120     | 140     |

    And the traders should have the following account balances:
      | trader    | asset | market id | margin | general |
      | traderGuy | ETH   | ETH/DEC19 | 120    | 0       |

    When the traders place the following orders:
      | trader    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1   | ETH/DEC19 | sell | 1      | 500   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2   | ETH/DEC19 | buy  | 1      | 500   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the mark price should be "500" for the market "ETH/DEC19"

    Then the traders should have the following account balances:
      | trader    | asset | market id | margin | general |
      | traderGuy | ETH   | ETH/DEC19 | 120    | 0       |

  Scenario: a trader place a new order in the system, trader is closing position via closeout of other traders
    Given the traders deposit on asset's general account the following amount:
      | trader        | asset | amount       |
      | traderGuy     | ETH   | 120          |
      | traderGuyGood | ETH   | 1000000      |
      | trader1       | ETH   | 1000000      |
      | trader2       | ETH   | 1000000      |
      | aux           | ETH   | 100000000000 |

     # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     |
      | aux     | ETH/DEC19 | buy  | 1      | 9     | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT  | TIF_GTC |

    # Trigger an auction to set the mark price
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader1-1 |
      | trader2 | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC | trader2-1 |
      | trader1 | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"

    When the traders place the following orders:
      | trader        | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | traderGuy     | ETH/DEC19 | sell | 1      | 9999  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | traderGuyGood | ETH/DEC19 | sell | 1      | 10000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |

    Then the traders should have the following margin levels:
      | trader    | market id | maintenance | search | initial | release |
      | traderGuy | ETH/DEC19 | 100         | 110    | 120     | 140     |

    When the traders place the following orders:
      | trader        | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | traderGuyGood | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | trader2-2 |
      | trader1       | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | trader1-2 |

    And debug orders
    Then debug trades

    Then the traders should have the following profit and loss:
      | trader        | volume | unrealised pnl | realised pnl |
      | traderGuyGood | 1      | 0              | 0            |

    And the traders should have the following account balances:
      | trader        | asset | market id | margin | general |
      | traderGuy     | ETH   | ETH/DEC19 | 120    | 0       |
      | traderGuyGood | ETH   | ETH/DEC19 | 1320   | 998680  |

    # this will trade with trader guy
    # which is going to get him distressed
    When the traders place the following orders:
      | trader    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader2   | ETH/DEC19 | buy  | 1      | 9999  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # now the margins should be back to 0, but it's actually
    # not being released when the party is used in the distressed flow
    Then the traders should have the following account balances:
      | trader        | asset | market id | margin | general |
      | traderGuy     | ETH   | ETH/DEC19 |   0    | 0       |
      | traderGuyGood | ETH   | ETH/DEC19 | 13306  | 995693  |

    # Position is actuall 0, and the party have no potential position
    # so we just have collateral stuck in the margin account
    Then the traders should have the following profit and loss:
      | trader        | volume | unrealised pnl | realised pnl |
      | traderGuyGood | 0      | 0              | 9000         |
