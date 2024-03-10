Feature: Iceberg orders in isolated margin mode

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.5                    | 0                         | default-futures |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 1500  |
    Given the average block duration is "1"

  @iceberg
  Scenario: 001 Iceberg order submission with valid TIF's
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/DEC19 | isolated margin | 0.15          |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       | isolated margin | 0.15          | 0     |

    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | only |
      | party1 | ETH/DEC19 | buy  | 100    | 10    | 0                | TYPE_LIMIT | TIF_GTC | 10        | 5                    | post |

    Then the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party1 | ETH/DEC19 | buy  | 10             | 10    | STATUS_ACTIVE | 90              |

    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | expires in | peak size | minimum visible size | only |
      | party2 | ETH/DEC19 | buy  | 100    | 10    | 0                | TYPE_LIMIT | TIF_GTT | 3600       | 8         | 4                    | post |

    Then the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party2 | ETH/DEC19 | buy  | 8              | 10    | STATUS_ACTIVE | 92              |

  @iceberg
  Scenario: 002 An iceberg order with either an ordinary can be submitted, and iceberg order with pegged limit price will be rejected
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/DEC19 | isolated margin | 0.15          |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       | isolated margin | 0.15          | 0     |

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | best-bid  |
      | party2 | ETH/DEC19 | sell | 1      | 20    | 0                | TYPE_LIMIT | TIF_GTC | best-ask  |
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | reference        |
      | party1 | ETH/DEC19 | buy  | 10     | 5     | 0                | TYPE_LIMIT | TIF_GTC | 3         | 1                    | ordinary-iceberg |
    And the parties place the following pegged iceberg orders:
      | party  | market id | side | volume | resulting trades | type       | tif     | peak size | minimum visible size | pegged reference | offset | reference      | error              |
      | party1 | ETH/DEC19 | buy  | 10     | 0                | TYPE_LIMIT | TIF_GTC | 2         | 1                    | BID              | 1      | pegged-iceberg | invalid OrderError |
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | buy  | 5     | 3      |
      | buy  | 9     | 0      |
      | buy  | 10    | 1      |

    # Move best-bid and check pegged iceberg order is re-priced
    When the parties amend the following orders:
      | party  | reference | price | size delta | tif     |
      | party2 | best-bid  | 9     | 0          | TIF_GTC |
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | buy  | 5     | 3      |
      | buy  | 8     | 0      |
      | buy  | 9     | 1      |

  @iceberg
  Scenario: 003 Iceberg order margin calculation
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/DEC19 | isolated margin | 0.15          |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       | isolated margin | 0.15          | 0     |

    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | only | reference       |
      | party1 | ETH/DEC19 | buy  | 100    | 10    | 0                | TYPE_LIMIT | TIF_GTC | 10        | 5                    | post | iceberg-order-1 |

    Then the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party1 | ETH/DEC19 | buy  | 10             | 10    | STATUS_ACTIVE | 90              |

    #order margin level: 10*100*0.15=150
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       | isolated margin | 0.15          | 150   |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | BTC   | ETH/DEC19 | 0      | 9850    |

    # And another party places a normal limit order for the same price and quantity, then the same margin should be taken
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference            |
      | party2 | ETH/DEC19 | buy  | 100    | 10    | 0                | TYPE_LIMIT | TIF_GTC | normal-limit-order-1 |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party2 | BTC   | ETH/DEC19 | 26     | 9974    |

    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party2 | ETH/DEC19 | isolated margin | 0.15          |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party2 | ETH/DEC19 | 0           | 0      | 0       | 0       | isolated margin | 0.15          | 150   |

    # Now we cancel the iceberg order
    Then the parties cancel the following orders:
      | party  | reference       |
      | party1 | iceberg-order-1 |

    # And the margin taken for the iceberg order is released
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | BTC   | ETH/DEC19 | 0      | 10000   |

    # Now we cancel the normal limit order
    Then the parties cancel the following orders:
      | party  | reference            |
      | party2 | normal-limit-order-1 |

    # And the margin taken for the normal limit order is released
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party2 | BTC   | ETH/DEC19 | 0      | 10000   |

  @iceberg
  Scenario: 004 iceberg basic refresh
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size |
      | party1 | ETH/DEC19 | buy  | 100    | 10    | 0                | TYPE_LIMIT | TIF_GTC | 10        | 5                    |

    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/DEC19 | isolated margin | 0.15          |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       | isolated margin | 0.15          | 150   |

    Then the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party1 | ETH/DEC19 | buy  | 10             | 10    | STATUS_ACTIVE | 90              |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC19 | sell | 6      | 10    | 1                | TYPE_LIMIT | TIF_GTC |

    Then the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party1 | ETH/DEC19 | buy  | 10             | 10    | STATUS_ACTIVE | 84              |

    #order margin: 10*94*0.15=141
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/DEC19 | 8           | 0      | 9       | 0       | isolated margin | 0.15          | 141   |

  @iceberg
  Scenario: 005 Iceberg order trading during auction uncrossing
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |

    Given the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    | peak size | minimum visible size |
      | party1 | ETH/DEC19 | buy  | 10     | 2     | 0                | TYPE_LIMIT | TIF_GTC | this-order-1 | 2         | 1                    |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC19 | buy  | 8      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    | peak size | minimum visible size |
      | party3 | ETH/DEC19 | sell | 10     | 2     | 0                | TYPE_LIMIT | TIF_GTC | this-order-1 | 2         | 1                    |

    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/DEC19 | isolated margin | 0.15          |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       | isolated margin | 0.15          | 3     |
    And the opening auction period ends for market "ETH/DEC19"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    # Check only the display volume of party1 is filled and is refreshed at the back of the que
    And the following trades should be executed:
      | buyer  | seller | price | size |
      | party1 | party3 | 2     | 2    |
    # Check the remaining volume of party3s iceberg is filled in a single trade with party2
    And the following trades should be executed:
      | buyer  | seller | price | size |
      | party2 | party3 | 2     | 8    |

  @iceberg
  @margin
  Scenario: 006 Iceberg increase size success and not losing position in order book
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/DEC19 | isolated margin | 0.15          |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       | isolated margin | 0.15          | 0     |

    And the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    | peak size | minimum visible size |
      | party1 | ETH/DEC19 | sell | 50     | 2     | 0                | TYPE_LIMIT | TIF_GTC | this-order-1 | 2         | 1                    |
      | party2 | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | this-order-2 | 2         | 1                    |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin |
      | party1 | BTC   | ETH/DEC19 | 0      | 9985    | 15           |

    # increasing size
    Then the parties amend the following orders:
      | party  | reference    | price | size delta | tif     |
      | party1 | this-order-1 | 2     | 50         | TIF_GTC |

    # the visible is the same and only the reserve amount has increased
    Then the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party1 | ETH/DEC19 | sell | 2              | 2     | STATUS_ACTIVE | 98              |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin |
      | party1 | BTC   | ETH/DEC19 | 0      | 9970    | 30           |

    # matching the order now
    # this should match with the size 2 order of party1
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | buy  | 2      | 2     | 1                | TYPE_LIMIT | TIF_GTC | party3    |

    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party3 | party1 | 2     | 2    |

  @iceberg
  Scenario: 007 Iceberg decrease size success and not losing position in order book
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |

    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/DEC19 | isolated margin | 0.15          |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       | isolated margin | 0.15          | 0     |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    And the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    | peak size | minimum visible size |
      | party1 | ETH/DEC19 | sell | 100    | 2     | 0                | TYPE_LIMIT | TIF_GTC | this-order-1 | 2         | 1                    |
      | party2 | ETH/DEC19 | sell | 100    | 2     | 0                | TYPE_LIMIT | TIF_GTC | this-order-2 | 2         | 1                    |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin |
      | party1 | BTC   | ETH/DEC19 | 0      | 9970    | 30           |

    # decreasing size
    Then the parties amend the following orders:
      | party  | reference    | price | size delta | tif     |
      | party1 | this-order-1 | 2     | -50        | TIF_GTC |

    # the visible is the same and only the reserve amount has decreased
    Then the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party1 | ETH/DEC19 | sell | 2              | 2     | STATUS_ACTIVE | 48              |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin |
      | party1 | BTC   | ETH/DEC19 | 0      | 9985    | 15           |

    # matching the order now
    # this should match with the size 2 order of party1
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | buy  | 2      | 2     | 1                | TYPE_LIMIT | TIF_GTC | party3    |

    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party3 | party1 | 2     | 2    |

  @iceberg
  Scenario: 008  Iceberg amend price reenters aggressively
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/DEC19 | isolated margin | 0.15          |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       | isolated margin | 0.15          | 0     |
    And the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    | peak size | minimum visible size |
      | party1 | ETH/DEC19 | sell | 16     | 5     | 0                | TYPE_LIMIT | TIF_GTC | this-order-1 | 5         | 1                    |
      | party2 | ETH/DEC19 | buy  | 10     | 2     | 0                | TYPE_LIMIT | TIF_GTC | this-order-2 | 2         | 1                    |

    # amend the buy order so that it will cross with the other iceberg
    Then the parties amend the following orders:
      | party  | reference    | price | size delta | tif     |
      | party2 | this-order-2 | 5     | 0          | TIF_GTC |

    # the amended iceberg will trade aggressively and be fully consumed
    Then the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status         | reserved volume |
      | party1 | ETH/DEC19 | sell | 5              | 5     | STATUS_STOPPED | 1               |
      | party2 | ETH/DEC19 | buy  | 0              | 5     | STATUS_FILLED  | 0               |

  @margin
  @iceberg
  Scenario: 009 Cancelling an active iceberg order
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/DEC19 | isolated margin | 0.15          |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       | isolated margin | 0.15          | 0     |

    Given the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | reference |
      | party1 | ETH/DEC19 | buy  | 100    | 5     | 0                | TYPE_LIMIT | TIF_GTC | 2         | 1                    | iceberg   |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin |
      | party1 | BTC   | ETH/DEC19 | 0      | 9925    | 75           |
    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | buy  | 5     | 2      |
    When the parties cancel the following orders:
      | party  | reference |
      | party1 | iceberg   |
    # The order should be cancelled
    Then the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status           | reserved volume | reference |
      | party1 | ETH/DEC19 | buy  | 2              | 5     | STATUS_CANCELLED | 98              | iceberg   |
    # The margin released
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin |
      | party1 | BTC   | ETH/DEC19 | 0      | 10000   | 0            |
    # And the order book updated
    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | buy  | 5     | 0      |

  @iceberg
  Scenario: 010 An aggressive iceberg order crosses an order with volume > iceberg volume
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount      |
      | party1 | BTC   | 10000000000 |
      | party2 | BTC   | 10000       |
      | aux    | BTC   | 100000      |
      | aux2   | BTC   | 100000      |
      | lpprov | BTC   | 90000000    |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    | p-1       |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    | p-2       |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 20     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/DEC19 | isolated margin | 0.15          |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       | isolated margin | 0.15          | 0     |

    When the network moves ahead "1" blocks
    And the orders should have the following status:
      | party  | reference | status        |
      | lpprov | p-1       | STATUS_PARKED |
      | lpprov | p-2       | STATUS_ACTIVE |

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC19 | sell | 15     | 5     | 0                | TYPE_LIMIT | TIF_GTC |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 105   | 2      |
      | sell | 5     | 15     |

    #requried position margin: 10*5*0.15 = 7.5
    #maintenance margin: 10*min(5, 2*1e6)+10*0.1*2=52
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | error               |
      | party1 | ETH/DEC19 | buy  | 10     | 5     | 1                | TYPE_LIMIT | TIF_GTC | 2         | 1                    | margin check failed |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general     | order margin |
      | party1 | BTC   | ETH/DEC19 | 0      | 10000000000 | 0            |

  @iceberg
  Scenario: 011 An aggressive iceberg order crosses an order with volume < iceberg volume
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/DEC19 | isolated margin | 0.15          |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       | isolated margin | 0.15          | 0     |

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC19 | sell | 10     | 5     | 0                | TYPE_LIMIT | TIF_GTC |
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | error               |
      | party1 | ETH/DEC19 | buy  | 15     | 5     | 1                | TYPE_LIMIT | TIF_GTC | 2         | 1                    | margin check failed |

  @iceberg
  Scenario: 012 A passive iceberg order (the only order at the price level) crosses an order with volume > iceberg volume
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/DEC19 | isolated margin | 0.15          |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       | isolated margin | 0.15          | 0     |

    Given the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size |
      | party1 | ETH/DEC19 | buy  | 10     | 5     | 0                | TYPE_LIMIT | TIF_GTC | 2         | 1                    |
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC19 | sell | 15     | 5     | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party1 | party2 | 5     | 10   |
    And the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party1 | ETH/DEC19 | buy  | 0              | 5     | STATUS_FILLED | 0               |

  @iceberg
  Scenario: 013 A passive iceberg order (one of multiple orders at the price level) crosses an order with volume > iceberg volume
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | party4 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |
    And the markets are updated:
        | id          | linear slippage factor |
        | ETH/DEC19   | 0                      |
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |

    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/DEC19 | isolated margin | 0.15          |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       | isolated margin | 0.15          | 0     |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Given the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size |
      | party1 | ETH/DEC19 | buy  | 10     | 5     | 0                | TYPE_LIMIT | TIF_GTC | 2         | 1                    |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC19 | buy  | 7      | 5     | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC19 | buy  | 7      | 5     | 0                | TYPE_LIMIT | TIF_GTC |
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/DEC19 | sell | 15     | 5     | 3                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party1 | party4 | 5     | 2    |
      | party2 | party4 | 5     | 7    |
      | party3 | party4 | 5     | 6    |
    And the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status        | reserved volume |
      | party1 | ETH/DEC19 | buy  | 2              | 5     | STATUS_ACTIVE | 6               |

  @iceberg
  Scenario: 014 An aggressive iceberg order crosses orders where the cumulative volume > iceberg volume
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | party4 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |

    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/DEC19 | isolated margin | 0.15          |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       | isolated margin | 0.15          | 0     |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC19 | sell | 30     | 5     | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC19 | sell | 40     | 5     | 0                | TYPE_LIMIT | TIF_GTC |
      | party4 | ETH/DEC19 | sell | 50     | 5     | 0                | TYPE_LIMIT | TIF_GTC |
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | error               |
      | party1 | ETH/DEC19 | buy  | 100    | 5     | 3                | TYPE_LIMIT | TIF_GTC | 2         | 1                    | margin check failed |

  @iceberg
  Scenario: 015 An aggressive iceberg order crosses orders where the cumulative volume < iceberg volume
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | party4 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/DEC19 | isolated margin | 0.15          |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       | isolated margin | 0.15          | 0     |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC19 | sell | 30     | 5     | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC19 | sell | 40     | 5     | 0                | TYPE_LIMIT | TIF_GTC |
    When the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | error               |
      | party1 | ETH/DEC19 | buy  | 100    | 5     | 2                | TYPE_LIMIT | TIF_GTC | 2         | 1                    | margin check failed |

  @iceberg
  Scenario: 016 Amended order trades with iceberg order triggering a refresh
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | party4 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/DEC19 | isolated margin | 0.15          |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       | isolated margin | 0.15          | 0     |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Given the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size |
      | party1 | ETH/DEC19 | sell | 10     | 5     | 0                | TYPE_LIMIT | TIF_GTC | 2         | 1                    |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | party2 | ETH/DEC19 | buy  | 7      | 4     | 0                | TYPE_LIMIT | TIF_GTC | order-to-amend |
      | party3 | ETH/DEC19 | sell | 5      | 5     | 0                | TYPE_LIMIT | TIF_GTC |                |
    When the parties amend the following orders:
      | party  | reference      | price | size delta | tif     |
      | party2 | order-to-amend | 5     | 0          | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party2 | party1 | 5     | 2    |
      | party2 | party3 | 5     | 5    |
    And the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status         | reserved volume |
      | party1 | ETH/DEC19 | sell | 2              | 5     | STATUS_STOPPED | 6               |

  @iceberg
  Scenario: 017 Attempting to wash trade with iceberg orders
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | party4 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |

    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |

    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/DEC19 | isolated margin | 0.15          |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       | isolated margin | 0.15          | 0     |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC19 | sell | 5      | 5     | 0                | TYPE_LIMIT | TIF_GTC |
      | party3 | ETH/DEC19 | sell | 5      | 5     | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | reference |
      | party1 | ETH/DEC19 | sell | 5      | 5     | 0                | TYPE_LIMIT | TIF_GTC | 2         | 1                    | iceberg   |
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error               |
      | party1 | ETH/DEC19 | buy  | 20     | 5     | 0                | TYPE_LIMIT | TIF_GTC | normal    | margin check failed |

    And the iceberg orders should have the following states:
      | party  | market id | reference | side | visible volume | price | status        | reserved volume |
      | party1 | ETH/DEC19 | iceberg   | sell | 2              | 5     | STATUS_ACTIVE | 3               |

  @iceberg
  Scenario: 018 An order matches multiple icebergs at the same level where the order volume < cumulative iceberg display volume
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | party4 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |

    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/DEC19 | isolated margin | 0.15          |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       | isolated margin | 0.15          | 0     |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Given the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | reference |
      | party1 | ETH/DEC19 | sell | 200    | 5     | 0                | TYPE_LIMIT | TIF_GTC | 2         | 1                    | iceberg   |
      | party2 | ETH/DEC19 | sell | 100    | 5     | 0                | TYPE_LIMIT | TIF_GTC | 2         | 1                    | iceberg2  |
      | party3 | ETH/DEC19 | sell | 100    | 5     | 0                | TYPE_LIMIT | TIF_GTC | 2         | 1                    | iceberg3  |
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/DEC19 | buy  | 300    | 5     | 3                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party4 | party1 | 5     | 150  |
      | party4 | party2 | 5     | 75   |
      | party4 | party3 | 5     | 75   |
    And the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status         | reserved volume | reference |
      | party1 | ETH/DEC19 | sell | 2              | 5     | STATUS_STOPPED | 48              | iceberg   |
      | party2 | ETH/DEC19 | sell | 2              | 5     | STATUS_ACTIVE  | 23              | iceberg2  |
      | party3 | ETH/DEC19 | sell | 2              | 5     | STATUS_ACTIVE  | 23              | iceberg3  |

  @iceberg
  Scenario: 019 An order matches multiple icebergs at the same level where the order volume > cumulative iceberg display volume
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | party4 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |

    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/DEC19 | isolated margin | 0.15          |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       | isolated margin | 0.15          | 0     |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Given the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | peak size | minimum visible size | reference |
      | party1 | ETH/DEC19 | sell | 200    | 5     | 0                | TYPE_LIMIT | TIF_GTC | 2         | 1                    | iceberg   |
      | party2 | ETH/DEC19 | sell | 100    | 5     | 0                | TYPE_LIMIT | TIF_GTC | 2         | 1                    | iceberg   |
      | party3 | ETH/DEC19 | sell | 100    | 5     | 0                | TYPE_LIMIT | TIF_GTC | 2         | 1                    | iceberg   |
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party4 | ETH/DEC19 | buy  | 600    | 5     | 3                | TYPE_LIMIT | TIF_GTC |
    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party4 | party1 | 5     | 200  |
      | party4 | party2 | 5     | 100  |
      | party4 | party3 | 5     | 100  |

  @iceberg
  Scenario: 020 Pegged orders are not re-priced as price-levels are consumed
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | party1 | BTC   | 10000    |
      | party2 | BTC   | 10000    |
      | party3 | BTC   | 10000    |
      | aux    | BTC   | 100000   |
      | aux2   | BTC   | 100000   |
      | lpprov | BTC   | 90000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |

    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50     | 100    |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/DEC19 | isolated margin | 0.15          |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/DEC19 | 0           | 0      | 0       | 0       | isolated margin | 0.15          | 0     |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC19 | buy  | 10     | 5     | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | sell | 1      | 20    | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | sell | 10     | 25    | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties place the following pegged iceberg orders:
      | party  | market id | side | volume | resulting trades | type       | tif     | peak size | minimum visible size | pegged reference | offset | error              |
      | party1 | ETH/DEC19 | buy  | 10     | 0                | TYPE_LIMIT | TIF_GTC | 2         | 1                    | BID              | 1      | invalid OrderError |
    And the parties place the following pegged orders:
      | party  | market id | side | volume | pegged reference | offset | error              |
      | party1 | ETH/DEC19 | buy  | 1      | BID              | 2      | invalid OrderError |
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | sell | 12     | 8     | 1                | TYPE_LIMIT | TIF_GTC |
    # Check pegged ordinary and iceberg orders are not re-priced as the best-bid price-level is consumed
    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party2 | party3 | 10    | 1    |

