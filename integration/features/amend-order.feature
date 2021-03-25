Feature: Amend orders

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 | BTC        | BTC   | simple     | 0         | 0         | 0              | 0.016           | 2.0   | 5              | 4              | 3.2           | 0                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Amend rejected for non existing order
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader | asset | amount |
      | myboi  | BTC   | 10000  |
      | aux    | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     |
      | aux     | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT  | TIF_GTC |

    And traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi  | ETH/DEC19 | sell |      1 |     2 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |

# cancel the order, so we cannot edit it.
    And traders cancel the following orders:
      | trader | reference   |
      | myboi  | myboi-ref-1 |

    Then traders amends the following orders reference:
      | trader | reference   | price | sizeDelta | expiresAt | tif     | success |
      | myboi  | myboi-ref-1 | 2     | 3         | 0         | TIF_GTC | false   |

  Scenario: Reduce size success and not loosing position in order book
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader | asset | amount |
      | myboi  | BTC   | 10000  |
      | myboi2 | BTC   | 10000  |
      | myboi3 | BTC   | 10000  |
      | aux    | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     |
      | aux     | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT  | TIF_GTC |

    And traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi  | ETH/DEC19 | sell |      5 |     2 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |
      | myboi2 | ETH/DEC19 | sell |      5 |     2 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-2 |

# reducing size
    Then traders amends the following orders reference:
      | trader | reference   | price | sizeDelta | expiresAt | tif     | success |
      | myboi  | myboi-ref-1 | 0     | -2        | 0         | TIF_GTC | true    |

# matching the order now
# this should match with the size 3 order of myboi
    Then traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi3 | ETH/DEC19 | buy  |      3 |     2 |                1 | TYPE_LIMIT | TIF_GTC | myboi-ref-3 |

    Then the following trades were executed:
      | buyer  | seller | price | size |
      | myboi3 | myboi  | 2     | 3    |

  Scenario: Increase size success and loosing position in order book
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader | asset | amount |
      | myboi  | BTC   | 10000  |
      | myboi2 | BTC   | 10000  |
      | myboi3 | BTC   | 10000  |
      | aux    | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     |
      | aux     | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT  | TIF_GTC |

    Then traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi  | ETH/DEC19 | sell |      5 |     2 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |
      | myboi2 | ETH/DEC19 | sell |      5 |     2 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-2 |

# reducing size
    And traders amends the following orders reference:
      | trader | reference   | price | sizeDelta | expiresAt | tif     | success |
      | myboi  | myboi-ref-1 | 0     | 3         | 0         | TIF_GTC | true    |

# matching the order now
# this should match with the size 3 order of myboi
    When traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi3 | ETH/DEC19 | buy  |      3 |     2 |                1 | TYPE_LIMIT | TIF_GTC | myboi-ref-3 |
    Then the following trades were executed:
      | buyer  | seller | price | size |
      | myboi3 | myboi2 |     2 |    3 |

  Scenario: Reduce size success and order cancelled as  < to remaining
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader | asset | amount |
      | myboi  | BTC   | 10000  |
      | myboi2 | BTC   | 10000  |
      | myboi3 | BTC   | 10000  |
      | aux    | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     |
      | aux     | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT  | TIF_GTC |

    Then traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi  | ETH/DEC19 | sell |      5 |     2 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |
      | myboi2 | ETH/DEC19 | sell |      5 |     2 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-2 |

# matching the order now
# this will reduce the remaining to 2 so it get cancelled later on
    When traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi3 | ETH/DEC19 | buy  |      3 |     2 |                1 | TYPE_LIMIT | TIF_GTC | myboi-ref-3 |

# reducing size, remaining goes from 2 to -1, this will cancel
    Then traders amends the following orders reference:
      | trader | reference   | price | sizeDelta | expiresAt | tif     | success |
      | myboi  | myboi-ref-1 | 0     | -3        | 0         | TIF_GTC | true    |

# check the order status, it should be cancelled
    And verify the status of the order reference:
      | trader | reference   | status           |
      | myboi  | myboi-ref-1 | STATUS_CANCELLED |

  Scenario: Amend to invalid tif is rejected
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader | asset | amount |
      | myboi  | BTC   | 10000  |
      | aux    | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     |
      | aux     | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT  | TIF_GTC |

    Then traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type  | tif | reference   |
      | myboi  | ETH/DEC19 | sell |      5 |     2 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |


# cannot amend TIF to TIF_FOK so this will be rejected
    Then traders amends the following orders reference:
      | trader | reference   | price | sizeDelta | expiresAt | tif     | success |
      | myboi  | myboi-ref-1 | 0     | 0         | 0         | TIF_FOK | false   |

  Scenario: TIF_GTC to TIF_GTT rejected without expiry
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader | asset | amount |
      | myboi  | BTC   | 10000  |
      | aux    | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     |
      | aux     | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT  | TIF_GTC |

    Then traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type  | tif | reference   |
      | myboi  | ETH/DEC19 | sell |      5 |     2 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |

# TIF_GTT rejected because of missing expiresAt
    Then traders amends the following orders reference:
      | trader | reference   | price | sizeDelta | expiresAt | tif     | success |
      | myboi  | myboi-ref-1 | 0     | 0         | 0         | TIF_GTT | false   |

  Scenario: TIF_GTC to TIF_GTT with time in the past
# setup accounts
    Given the traders make the following deposits on asset's general account:
      | trader | asset | amount |
      | myboi  | BTC   | 10000  |
      | aux    | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     |
      | aux     | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT  | TIF_GTC |

    Then traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type  | tif | reference   |
      | myboi  | ETH/DEC19 | sell |      5 |     2 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |

# reducing size, remaining goes from 2 to -1, this will cancel
    Then traders amends the following orders reference:
      | trader | reference   | price | sizeDelta | expiresAt | tif | success |
      | myboi  | myboi-ref-1 |     2 |         0 |     10000 | TIF_GTT | false   |
