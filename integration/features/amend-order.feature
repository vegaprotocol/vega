Feature: Amend orders

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/long | tau/short | mu |     r | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode | makerFee | infrastructureFee | liquidityFee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | Prob of trading |
      | ETH/DEC19 | ETH      | BTC       | BTC   |        94 | simple     |          0 |        0 |  0 | 0.016 |   2.0 |              5 |              4 |           3.2 |              42 | 0           | continuous   |        0 |                 0 |            0 |                 0  |                |             |                 | 0.1             |

  Scenario: Amend rejected for non existing order
# setup accounts
    Given the following traders:
      | name  | amount |
      | myboi |  10000 |
    Then I Expect the traders to have new general account:
      | name  | asset |
      | myboi | BTC   |

    Then traders place following orders with references:
      | trader | id        | type | volume | price | resulting trades | type  | tif | reference   |
      | myboi  | ETH/DEC19 | sell |      1 |     1 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |

# cancel the order, so we cannot edit it.
    Then traders cancels the following orders reference:
      | trader | reference    |
      | myboi  | myboi-ref-1  |

    Then traders amends the following orders reference:
      | trader | reference   | price | sizeDelta | expiresAt | tif | success |
      | myboi  | myboi-ref-1 |     2 |         3 |         0 | TIF_GTC | false   |

  Scenario: Reduce size success and not loosing position in order book
# setup accounts
    Given the following traders:
      | name   | amount |
      | myboi  |  10000 |
      | myboi2 |  10000 |
      | myboi3 |  10000 |
    Then I Expect the traders to have new general account:
      | name  | asset |
      | myboi | BTC   |

    Then traders place following orders with references:
      | trader | id        | type | volume | price | resulting trades | type  | tif | reference   |
      | myboi  | ETH/DEC19 | sell |      5 |     1 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |
      | myboi2 | ETH/DEC19 | sell |      5 |     1 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-2 |

# reducing size
    Then traders amends the following orders reference:
      | trader | reference   | price | sizeDelta | expiresAt | tif | success |
      | myboi  | myboi-ref-1 |     0 |        -2 |         0 | TIF_GTC | true    |

# matching the order now
# this should match with the size 3 order of myboi
    Then traders place following orders with references:
      | trader | id        | type | volume | price | resulting trades | type  | tif | reference   |
      | myboi3 | ETH/DEC19 | buy  |      3 |     1 |                1 | TYPE_LIMIT | TIF_GTC | myboi-ref-3 |


# Then the following trades happend
      Then the following trades happened:
        | buyer | seller | price | volume |
        | myboi3 | myboi  |     1 |      3 |

  Scenario: Increase size success and loosing position in order book
# setup accounts
    Given the following traders:
      | name   | amount |
      | myboi  |  10000 |
      | myboi2 |  10000 |
      | myboi3 |  10000 |
    Then I Expect the traders to have new general account:
      | name  | asset |
      | myboi | BTC   |

    Then traders place following orders with references:
      | trader | id        | type | volume | price | resulting trades | type  | tif | reference   |
      | myboi  | ETH/DEC19 | sell |      5 |     1 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |
      | myboi2 | ETH/DEC19 | sell |      5 |     1 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-2 |

# reducing size
    Then traders amends the following orders reference:
      | trader | reference   | price | sizeDelta | expiresAt | tif | success |
      | myboi  | myboi-ref-1 |     0 |        3  |         0 | TIF_GTC | true    |

# matching the order now
# this should match with the size 3 order of myboi
    Then traders place following orders with references:
      | trader | id        | type | volume | price | resulting trades | type  | tif | reference   |
      | myboi3 | ETH/DEC19 | buy  |      3 |     1 |                1 | TYPE_LIMIT | TIF_GTC | myboi-ref-3 |


# Then the following trades happend
      Then the following trades happened:
        | buyer  | seller | price | volume |
        | myboi3 | myboi2 |     1 |      3 |

  Scenario: Reduce size success and order cancelled as  < to remaining
# setup accounts
    Given the following traders:
      | name   | amount |
      | myboi  |  10000 |
      | myboi2 |  10000 |
      | myboi3 |  10000 |
    Then I Expect the traders to have new general account:
      | name  | asset |
      | myboi | BTC   |

    Then traders place following orders with references:
      | trader | id        | type | volume | price | resulting trades | type  | tif | reference   |
      | myboi  | ETH/DEC19 | sell |      5 |     1 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |
      | myboi2 | ETH/DEC19 | sell |      5 |     1 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-2 |

# matching the order now
# this will reduce the remaining to 2 so it get cancelled later on
    Then traders place following orders with references:
      | trader | id        | type | volume | price | resulting trades | type  | tif | reference   |
      | myboi3 | ETH/DEC19 | buy  |      3 |     1 |                1 | TYPE_LIMIT | TIF_GTC | myboi-ref-3 |

# reducing size, remaining goes from 2 to -1, this will cancel
    Then traders amends the following orders reference:
      | trader | reference   | price | sizeDelta | expiresAt | tif | success |
      | myboi  | myboi-ref-1 |     0 |        -3 |         0 | TIF_GTC | true    |

# check the order status, it should be cancelled
    Then verify the status of the order reference:
      | trader | reference   | status           |
      | myboi  | myboi-ref-1 | STATUS_CANCELLED |

  Scenario: Amend to invalid tif is rejected
# setup accounts
    Given the following traders:
      | name   | amount |
      | myboi  |  10000 |
    Then I Expect the traders to have new general account:
      | name  | asset |
      | myboi | BTC   |

    Then traders place following orders with references:
      | trader | id        | type | volume | price | resulting trades | type  | tif | reference   |
      | myboi  | ETH/DEC19 | sell |      5 |     1 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |


# cannot amend TIF to TIF_FOK so this will be rejected
    Then traders amends the following orders reference:
      | trader | reference   | price | sizeDelta | expiresAt | tif | success |
      | myboi  | myboi-ref-1 |     0 |        0  |         0 | TIF_FOK | false |

  Scenario: TIF_GTC to TIF_GTT rejected without expiry
# setup accounts
    Given the following traders:
      | name   | amount |
      | myboi  |  10000 |
    Then I Expect the traders to have new general account:
      | name  | asset |
      | myboi | BTC   |

    Then traders place following orders with references:
      | trader | id        | type | volume | price | resulting trades | type  | tif | reference   |
      | myboi  | ETH/DEC19 | sell |      5 |     1 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |


# TIF_GTT rejected because of missing expiresAt
    Then traders amends the following orders reference:
      | trader | reference   | price | sizeDelta | expiresAt | tif | success |
      | myboi  | myboi-ref-1 |     0 |         0 |         0 | TIF_GTT | false   |

  Scenario: TIF_GTC to TIF_GTT with time in the past
# setup accounts
    Given the following traders:
      | name   | amount |
      | myboi  |  10000 |
    Then I Expect the traders to have new general account:
      | name  | asset |
      | myboi | BTC   |

    Then traders place following orders with references:
      | trader | id        | type | volume | price | resulting trades | type  | tif | reference   |
      | myboi  | ETH/DEC19 | sell |      5 |     1 |                0 | TYPE_LIMIT | TIF_GTC | myboi-ref-1 |


# reducing size, remaining goes from 2 to -1, this will cancel
    Then traders amends the following orders reference:
      | trader | reference   | price | sizeDelta | expiresAt | tif | success |
      | myboi  | myboi-ref-1 |     1 |         0 |     10000 | TIF_GTT | false   |
