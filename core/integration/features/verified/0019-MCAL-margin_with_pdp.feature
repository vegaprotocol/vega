Feature: Test party accounts margins with fractional orders - the test is identical to margin.feature just with 2 position decimal places and all volumes are scaled by 10^2, to demonstrate that margins are calculated correctly (0019-MCAL-008)

  Background:

    Given the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | position decimal places | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 2                       | 1e6                    | 1e6                       |

  Scenario: a party place a new order in the system, margin are calculated
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount       |
      | partyGuy | ETH   | 10000        |
      | party1   | ETH   | 1000000      |
      | party2   | ETH   | 1000000      |
      | aux      | ETH   | 100000000000 |
      | lpprov   | ETH   | 1000000000   |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 900000000         | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | lpprov | ETH/DEC19 | 900000000         | 0.1 | sell | ASK              | 50         | 10     | submission |


    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 100    | 9     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 100    | 10001 | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 100    | 10    | 0                | TYPE_LIMIT | TIF_GTC | party1-1  |
      | party2 | ETH/DEC19 | sell | 100    | 10000 | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party1 | ETH/DEC19 | buy  | 100    | 1000  | 0                | TYPE_LIMIT | TIF_GFA | party1-2  |
      | party2 | ETH/DEC19 | sell | 100    | 1000  | 0                | TYPE_LIMIT | TIF_GFA | party2-2  |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"
    Then the parties cancel the following orders:
      | party  | reference |
      | party1 | party1-1  |
      | party2 | party2-1  |

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | partyGuy | ETH/DEC19 | sell | 100    | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the parties should have the following margin levels:
      | party    | market id | maintenance | search | initial | release |
      | partyGuy | ETH/DEC19 | 100         | 110    | 120     | 140     |
    Then the parties should have the following account balances:
      | party    | asset | market id | margin | general |
      | partyGuy | ETH   | ETH/DEC19 | 120    | 9880    |

  Scenario: an order is rejected if a party have insufficient margin
    Given the parties deposit on asset's general account the following amount:
      | party    | asset | amount     |
      | partyGuy | ETH   | 1          |
      | party1   | ETH   | 1000000    |
      | party2   | ETH   | 1000000    |
      | lpprov   | ETH   | 1000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 900000000         | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | lpprov | ETH/DEC19 | 900000000         | 0.1 | sell | ASK              | 50         | 10     | submission |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 100    | 10    | 0                | TYPE_LIMIT | TIF_GTC | party1-1  |
      | party2 | ETH/DEC19 | sell | 100    | 10000 | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party1 | ETH/DEC19 | buy  | 100    | 1000  | 0                | TYPE_LIMIT | TIF_GFA | party1-2  |
      | party2 | ETH/DEC19 | sell | 100    | 1000  | 0                | TYPE_LIMIT | TIF_GFA | party2-2  |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "1000" for the market "ETH/DEC19"
    Then the parties cancel the following orders:
      | party  | reference |
      | party1 | party1-1  |
      | party2 | party2-1  |

    When the parties place the following orders with ticks:
      | party    | market id | side | volume | price | type       | tif     | reference | error               |
      | partyGuy | ETH/DEC19 | sell | 100    | 1000  | TYPE_LIMIT | TIF_GTC | party1-1  | margin check failed |
    Then the following orders should be rejected:
      | party    | market id | reason                          |
      | partyGuy | ETH/DEC19 | ORDER_ERROR_MARGIN_CHECK_FAILED |
