Feature: Trader amends his orders

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 | BTC        | BTC   | simple     | 0         | 0         | 0              | 0.016           | 2.0   | 5              | 4              | 3.2           | 0                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Amend rejected for non existing order
    Given the traders make the following deposits on asset's general account:
      | trader | asset | amount |
      | myboi  | BTC   | 10000  |
      | aux    | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
    And traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi  | ETH/DEC19 | sell | 1      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |

# cancel the order, so we cannot edit it.
    And traders cancel the following orders:
      | trader | reference   |
      | myboi  | myboi-ref-1 |
    And traders amend the following orders:
      | trader | reference   | price | size delta | tif     |
      | myboi  | myboi-ref-1 | 2     | 3          | TIF_GTC |
    But the following amendments should be rejected:
      | trader | reference   | error                        |
      | myboi  | myboi-ref-1 | OrderError: Invalid Order ID |

  Scenario: Reduce size success and not loosing position in order book
    Given the traders make the following deposits on asset's general account:
      | trader | asset | amount |
      | myboi  | BTC   | 10000  |
      | myboi2 | BTC   | 10000  |
      | myboi3 | BTC   | 10000  |
      | aux    | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |

    And traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi  | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |
      | myboi2 | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-2 |

# reducing size
    Then traders amend the following orders:
      | trader | reference   | price | size delta | tif     |
      | myboi  | myboi-ref-1 | 0     | -2         | TIF_GTC |
    Then the following amendments should be accepted:
      | trader | reference   |
      | myboi  | myboi-ref-1 |

# matching the order now
# this should match with the size 3 order of myboi
    Then traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi3 | ETH/DEC19 | buy  | 3      | 2     | 1                | TYPE_LIMIT | TIF_GTC | myboi-ref-3 |

    Then the following trades were executed:
      | buyer  | seller | price | size |
      | myboi3 | myboi  | 2     | 3    |

  Scenario: Increase size success and loosing position in order book
    Given the traders make the following deposits on asset's general account:
      | trader | asset | amount |
      | myboi  | BTC   | 10000  |
      | myboi2 | BTC   | 10000  |
      | myboi3 | BTC   | 10000  |
      | aux    | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
    And traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi  | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |
      | myboi2 | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-2 |
    And traders amend the following orders:
      | trader | reference   | price | size delta | tif     | success |
      | myboi  | myboi-ref-1 | 0     | 3          | TIF_GTC | true    |
    Then the following amendments should be accepted:
      | trader | reference   |
      | myboi  | myboi-ref-1 |

# matching the order now
# this should match with the size 3 order of myboi
    When traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi3 | ETH/DEC19 | buy  | 3      | 2     | 1                | TYPE_LIMIT | TIF_GTC | myboi-ref-3 |
    Then the following trades were executed:
      | buyer  | seller | price | size |
      | myboi3 | myboi2 | 2     | 3    |

  Scenario: Reduce size success and order cancelled as  < to remaining
    Given the traders make the following deposits on asset's general account:
      | trader | asset | amount |
      | myboi  | BTC   | 10000  |
      | myboi2 | BTC   | 10000  |
      | myboi3 | BTC   | 10000  |
      | aux    | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
    And traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi  | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |
      | myboi2 | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-2 |

# matching the order now
# this will reduce the remaining to 2 so it get cancelled later on
    When traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi3 | ETH/DEC19 | buy  | 3      | 2     | 1                | TYPE_LIMIT | TIF_GTC | myboi-ref-3 |
    And traders amend the following orders:
      | trader | reference   | price | size delta | tif     |
      | myboi  | myboi-ref-1 | 0     | -3         | TIF_GTC |
    Then the following amendments should be accepted:
      | trader | reference   |
      | myboi  | myboi-ref-1 |
    And verify the status of the order reference:
      | trader | reference   | status           |
      | myboi  | myboi-ref-1 | STATUS_CANCELLED |

  Scenario: Amend to invalid tif is rejected
    Given the traders make the following deposits on asset's general account:
      | trader | asset | amount |
      | myboi  | BTC   | 10000  |
      | aux    | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
    And traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi  | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |
    And traders amend the following orders:
      | trader | reference   | price | size delta | tif     |
      | myboi  | myboi-ref-1 | 0     | 0          | TIF_FOK |
    But the following amendments should be rejected:
      | trader | reference   | error                                      |
      | myboi  | myboi-ref-1 | OrderError: Cannot amend TIF to FOK or IOC |


  Scenario: TIF_GTC to TIF_GTT rejected without expiry
    Given the traders make the following deposits on asset's general account:
      | trader | asset | amount |
      | myboi  | BTC   | 10000  |
      | aux    | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
    And traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi  | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |
    And traders amend the following orders:
      | trader | reference   | price | size delta | tif     |
      | myboi  | myboi-ref-1 | 0     | 0          | TIF_GTT |
    But the following amendments should be rejected:
      | trader | reference   | error                                                           |
      | myboi  | myboi-ref-1 | OrderError: Cannot amend order to GTT without an expiryAt field |


  Scenario: TIF_GTC to TIF_GTT with time in the past
    Given the traders make the following deposits on asset's general account:
      | trader | asset | amount |
      | myboi  | BTC   | 10000  |
      | aux    | BTC   | 100000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     |
      | aux    | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux    | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC |
    And traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | myboi  | ETH/DEC19 | sell | 5      | 2     | 0                | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |
    And traders amend the following orders:
      | trader | reference   | price | size delta | expiration date      | tif     |
      | myboi  | myboi-ref-1 | 2     | 0          | 2019-11-30T00:00:00Z | TIF_GTT |
    But the following amendments should be rejected:
      | trader | reference   | error                                                   |
      | myboi  | myboi-ref-1 | OrderError: ExpiryAt field must not be before CreatedAt |

