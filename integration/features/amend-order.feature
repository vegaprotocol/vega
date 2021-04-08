Feature: Amend orders

  Background:
    Given the initial insurance pool balance is "0" for the markets:
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator                  | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-2 | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | market.auction.minimumDuration |
      | 1                              |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Amend rejected for non existing order
# setup accounts
    Given the traders deposit on asset's general account the following amount:
      | trader | asset | amount |
      | myboi  | BTC   | 10000  |
      | aux    | BTC   | 100000 |
      | aux2   | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     |
      | aux     | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux2    | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT  | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    And the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi  | ETH/DEC19 | sell |      1 |     2 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |

# cancel the order, so we cannot edit it.
    And the traders cancel the following orders:
      | trader | reference   |
      | myboi  | myboi-ref-1 |

    Then the traders amend the following orders:
      | trader | reference   | price | size delta | expiresAt | tif     | success |
      | myboi  | myboi-ref-1 | 2     | 3          | 0         | TIF_GTC | false   |

  Scenario: Reduce size success and not loosing position in order book
# setup accounts
    Given the traders deposit on asset's general account the following amount:
      | trader | asset | amount |
      | myboi  | BTC   | 10000  |
      | myboi2 | BTC   | 10000  |
      | myboi3 | BTC   | 10000  |
      | aux    | BTC   | 100000 |
      | aux2   | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     |
      | aux     | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux2    | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT  | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    And the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi  | ETH/DEC19 | sell |      5 |     2 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |
      | myboi2 | ETH/DEC19 | sell |      5 |     2 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-2 |

# reducing size
    Then the traders amend the following orders:
      | trader | reference   | price | size delta | expiresAt | tif     | success |
      | myboi  | myboi-ref-1 | 0     | -2         | 0         | TIF_GTC | true    |

# matching the order now
# this should match with the size 3 order of myboi
    Then the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi3 | ETH/DEC19 | buy  |      3 |     2 |                1 | TYPE_LIMIT | TIF_GTC | myboi-ref-3 |

    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | myboi3 | myboi  | 2     | 3    |

  Scenario: Increase size success and loosing position in order book
# setup accounts
    Given the traders deposit on asset's general account the following amount:
      | trader | asset | amount |
      | myboi  | BTC   | 10000  |
      | myboi2 | BTC   | 10000  |
      | myboi3 | BTC   | 10000  |
      | aux    | BTC   | 100000 |
      | aux2   | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     |
      | aux     | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux2    | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT  | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Then the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi  | ETH/DEC19 | sell |      5 |     2 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |
      | myboi2 | ETH/DEC19 | sell |      5 |     2 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-2 |

# reducing size
    And the traders amend the following orders:
      | trader | reference   | price | size delta | expiresAt | tif     | success |
      | myboi  | myboi-ref-1 | 0     | 3          | 0         | TIF_GTC | true    |

# matching the order now
# this should match with the size 3 order of myboi
    When the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi3 | ETH/DEC19 | buy  |      3 |     2 |                1 | TYPE_LIMIT | TIF_GTC | myboi-ref-3 |
    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | myboi3 | myboi2 |     2 |    3 |

  Scenario: Reduce size success and order cancelled as  < to remaining
# setup accounts
    Given the traders deposit on asset's general account the following amount:
      | trader | asset | amount |
      | myboi  | BTC   | 10000  |
      | myboi2 | BTC   | 10000  |
      | myboi3 | BTC   | 10000  |
      | aux    | BTC   | 100000 |
      | aux2   | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     |
      | aux     | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux2    | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT  | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Then the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi  | ETH/DEC19 | sell |      5 |     2 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |
      | myboi2 | ETH/DEC19 | sell |      5 |     2 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-2 |

# matching the order now
# this will reduce the remaining to 2 so it get cancelled later on
    When the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi3 | ETH/DEC19 | buy  |      3 |     2 |                1 | TYPE_LIMIT | TIF_GTC | myboi-ref-3 |

# reducing size, remaining goes from 2 to -1, this will cancel
    Then the traders amend the following orders:
      | trader | reference   | price | size delta | expiresAt | tif     | success |
      | myboi  | myboi-ref-1 | 0     | -3         | 0         | TIF_GTC | true    |

# check the order status, it should be cancelled
    And the orders should have the following status:
      | trader | reference   | status           |
      | myboi  | myboi-ref-1 | STATUS_CANCELLED |

  Scenario: Amend to invalid tif is rejected
# setup accounts
    Given the traders deposit on asset's general account the following amount:
      | trader | asset | amount |
      | myboi  | BTC   | 10000  |
      | aux    | BTC   | 100000 |
      | aux2   | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     |
      | aux     | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux2    | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT  | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Then the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type  | tif | reference   |
      | myboi  | ETH/DEC19 | sell |      5 |     2 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |


# cannot amend TIF to TIF_FOK so this will be rejected
    Then the traders amend the following orders:
      | trader | reference   | price | size delta | expiresAt | tif     | success |
      | myboi  | myboi-ref-1 | 0     | 0          | 0         | TIF_FOK | false   |

  Scenario: TIF_GTC to TIF_GTT rejected without expiry
# setup accounts
    Given the traders deposit on asset's general account the following amount:
      | trader | asset | amount |
      | myboi  | BTC   | 10000  |
      | aux    | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     |
      | aux     | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT  | TIF_GTC |

    Then the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type  | tif | reference   |
      | myboi  | ETH/DEC19 | sell |      5 |     2 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |

# TIF_GTT rejected because of missing expiresAt
    Then the traders amend the following orders:
      | trader | reference   | price | size delta | expiresAt | tif     | success |
      | myboi  | myboi-ref-1 | 0     | 0          | 0         | TIF_GTT | false   |

  Scenario: TIF_GTC to TIF_GTT with time in the past
# setup accounts
    Given the traders deposit on asset's general account the following amount:
      | trader | asset | amount |
      | myboi  | BTC   | 10000  |
      | aux    | BTC   | 100000 |
      | aux2   | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     |
      | aux     | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux2    | ETH/DEC19 | buy  | 1      | 2     | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT  | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    Then the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type  | tif | reference   |
      | myboi  | ETH/DEC19 | sell |      5 |     2 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |

# reducing size, remaining goes from 2 to -1, this will cancel
    Then the traders amend the following orders:
      | trader | reference   | price | size delta | expiresAt | tif | success |
      | myboi  | myboi-ref-1 |     2 |          0 |     10000 | TIF_GTT | false   |
