Feature: Test margin release on order cancel

  Background:

    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 5              |
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | decimal places | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 2              | 1e6                    | 1e6                       |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  @MarginRelease
  Scenario: a party place a new order in the system, margin are updated, the order is closed, margin is 0ed (0003-MTMK-0013)
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount          |
      | partyGuy | ETH   | 10000000        |
      | party1   | ETH   | 1000000000      |
      | party2   | ETH   | 1000000000      |
      | aux      | ETH   | 100000000000000 |
      | lpprov   | ETH   | 100000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000000       | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000000       | 0.1 | sell | ASK              | 50         | 100    | submission |


    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 9     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | party1-1  |
      | party2 | ETH/DEC19 | sell | 1      | 10000 | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party1 | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | party1-2  |
      | party2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | party2-2  |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | partyGuy | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |

    Then the parties should have the following margin levels:
      | party    | market id | maintenance | search | initial | release |
      | partyGuy | ETH/DEC19 | 100000      | 110000 | 120000  | 140000  |

    And the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | partyGuy | ETH   | ETH/DEC19 | 120000 | 9880000 |

    When the parties cancel the following orders:
      | party    | reference |
      | partyGuy | ref-1     |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general  |
      | partyGuy | ETH   | ETH/DEC19 | 0      | 10000000 |


  @MarginRelease
  Scenario: a party place a new market order in the system, order, trade, party margin is updated, then place an GTC order which will trade, margin is 0ed
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount          |
      | partyGuy | ETH   | 10000000        |
      | party1   | ETH   | 1000000000      |
      | party2   | ETH   | 1000000000      |
      | aux      | ETH   | 100000000000000 |
      | lpprov   | ETH   | 100000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000000       | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000000       | 0.1 | sell | ASK              | 50         | 100    | submission |


    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 9     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | party1-1  |
      | party1 | ETH/DEC19 | buy  | 10     | 990   | 0                | TYPE_LIMIT | TIF_GTC | party1-1  |
      | party2 | ETH/DEC19 | sell | 1      | 10000 | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party2 | ETH/DEC19 | sell | 10     | 1010  | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party1 | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | party1-2  |
      | party2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | party2-2  |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type        | tif     | reference |
      | partyGuy | ETH/DEC19 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC | ref-1     |


    Then the parties should have the following margin levels:
      | party    | market id | maintenance | search | initial | release |
      | partyGuy | ETH/DEC19 | 119000      | 130900 | 142800  | 166600  |

    And the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | partyGuy | ETH   | ETH/DEC19 | 142800 | 9758200 |

    Then the parties should have the following profit and loss:
      | party    | volume | unrealised pnl | realised pnl |
      | partyGuy | -1     | 0              | 0            |

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | partyGuy | ETH/DEC19 | buy  | 1      | 1005  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    Then the parties should have the following margin levels:
      | party    | market id | maintenance | search | initial | release |
      | partyGuy | ETH/DEC19 | 119000      | 130900 | 142800  | 166600  |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 1      | 1005  | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    Then the parties should have the following margin levels:
      | party    | market id | maintenance | search | initial | release |
      | partyGuy | ETH/DEC19 | 0           | 0      | 0       | 0       |

    And the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | partyGuy | ETH   | ETH/DEC19 | 0      | 9886000 |


  @MarginRelease
  Scenario: a party place a new order in the system, party is closed out with only potential position, margin is 0ed
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount          |
      | partyGuy | ETH   | 120000          |
      | party1   | ETH   | 1000000000      |
      | party2   | ETH   | 1000000000      |
      | aux      | ETH   | 100000000000000 |
      | lpprov   | ETH   | 100000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000000       | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000000       | 0.1 | sell | ASK              | 50         | 100    | submission |


    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 9     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | party1-1  |
      | party2 | ETH/DEC19 | sell | 1      | 10000 | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party1 | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | party1-2  |
      | party2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | party2-2  |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | partyGuy | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |

    Then the parties should have the following margin levels:
      | party    | market id | maintenance | search | initial | release |
      | partyGuy | ETH/DEC19 | 100000      | 110000 | 120000  | 140000  |

    And the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | partyGuy | ETH   | ETH/DEC19 | 120000 | 0       |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 1      | 500   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | buy  | 1      | 500   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the mark price should be "500" for the market "ETH/DEC19"

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | partyGuy | ETH   | ETH/DEC19 | 60000  | 60000   |

  @MarginRelease
  Scenario: a party place a new order in the system, party is closing position via closeout of other parties
    Given the parties deposit on asset's general account the following amount:
      | party        | asset | amount          |
      | partyGuy     | ETH   | 120000          |
      | partyGuyGood | ETH   | 1000000000      |
      | party1       | ETH   | 1000000000      |
      | party2       | ETH   | 1000000000      |
      | aux          | ETH   | 100000000000000 |
      | lpprov       | ETH   | 100000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000000       | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000000       | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 9     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | party1-1  |
      | party2 | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party1 | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | party1-2  |
      | party2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | party2-2  |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"

    When the parties place the following orders with ticks:
      | party        | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | partyGuy     | ETH/DEC19 | sell | 1      | 9999  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | partyGuyGood | ETH/DEC19 | sell | 1      | 10000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |

    Then the parties should have the following margin levels:
      | party    | market id | maintenance | search | initial | release |
      | partyGuy | ETH/DEC19 | 100000      | 110000 | 120000  | 140000  |

    When the parties place the following orders with ticks:
      | party        | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | partyGuyGood | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party2-2  |
      | party1       | ETH/DEC19 | sell | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | party1-2  |

    Then the parties should have the following profit and loss:
      | party        | volume | unrealised pnl | realised pnl |
      | partyGuyGood | 1      | 0              | 0            |

    And the parties should have the following account balances:
      | party        | asset | market id | margin  | general   |
      | partyGuy     | ETH   | ETH/DEC19 | 120000  | 0         |
      | partyGuyGood | ETH   | ETH/DEC19 | 1320000 | 998680000 |

    # this will trade with party guy
    # which is going to get him distressed
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC19 | buy  | 1      | 9999  | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # now the margins should be back to 0, but it's actually
    # not being released when the party is used in the distressed flow
    Then the parties should have the following account balances:
      | party        | asset | market id | margin | general    |
      | partyGuy     | ETH   | ETH/DEC19 | 0      | 0          |
      | partyGuyGood | ETH   | ETH/DEC19 | 0      | 1008999000 |

    # TODO: FIX THIS
    # partyGuyGood should have a margin of 0 here.
    # Position is actuall 0, and the party have no potential position
    # so we just have collateral stuck in the margin account
    Then the parties should have the following profit and loss:
      | party        | volume | unrealised pnl | realised pnl |
      | partyGuyGood | 0      | 0              | 8999000      |
