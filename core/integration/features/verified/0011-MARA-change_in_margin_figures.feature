Feature: Test party accounts

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: 001: A party place a new order in the system, margin are calculated (0011-MARA-001, 0011-MARA-002)
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount       |
      | partyGuy | ETH   | 1000         |
      | party1   | ETH   | 1000000      |
      | party2   | ETH   | 1000000      |
      | aux      | ETH   | 100000000000 |
      | lpprov   | ETH   | 100000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | sell | ASK              | 50         | 10     | submission |
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
    Then the parties cancel the following orders:
      | party  | reference |
      | party1 | party1-1  |
      | party2 | party2-1  |

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | partyGuy | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the parties should have the following margin levels:
      | party    | market id | maintenance | search | initial | release |
      | partyGuy | ETH/DEC19 | 100         | 110    | 120     | 140     |
    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | partyGuy | ETH   | ETH/DEC19 | 120    | 880     |

    # Amending order such that margin check goes through and margin account is topped up successfully
    When the parties amend the following orders:
      | party    | reference | price | size delta | tif     |
      | partyGuy | ref-1     | 1000  | 2          | TIF_GTC |

    Then the parties should have the following margin levels:
      | party    | market id | maintenance | search | initial | release |
      | partyGuy | ETH/DEC19 | 300         | 330    | 360     | 420     |

    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | partyGuy | ETH   | ETH/DEC19 | 360    | 640     |

    # Amending order such that margin check fails and amend order gets rejected
    When the parties amend the following orders:
      | party    | reference | price | size delta | tif     | error               |
      | partyGuy | ref-1     | 1000  | 8          | TIF_GTC | margin check failed |

    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 1000  | 3      |

  Scenario: 002: An order is rejected if a party have insufficient margin (0011-MARA-002)
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount  |
      | partyGuy | ETH   | 1       |
      | party1   | ETH   | 1000000 |
      | party2   | ETH   | 1000000 |
      | lpprov   | ETH   | 1000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | sell | ASK              | 50         | 10     | submission |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | party1-1  |
      | party2 | ETH/DEC19 | sell | 1      | 10000 | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party1 | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | party1-2  |
      | party2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | party2-2  |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"
    Then the parties cancel the following orders:
      | party  | reference |
      | party1 | party1-1  |
      | party2 | party2-1  |

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | type       | tif     | reference | error               |
      | partyGuy | ETH/DEC19 | sell | 1      | 1000  | TYPE_LIMIT | TIF_GTC | party1-1  | margin check failed |
    Then the following orders should be rejected:
      | party    | market id | reason                          |
      | partyGuy | ETH/DEC19 | ORDER_ERROR_MARGIN_CHECK_FAILED |
