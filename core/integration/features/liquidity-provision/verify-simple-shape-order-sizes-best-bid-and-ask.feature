Feature: Verify the order size is correctly cumulated.

  Background:
    Given the log normal risk model named "my-log-normal-risk-model":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.001         | 0.00000190128526884174 | 0  | 0.016 | 2.5   |
    And the markets:
      | id        | quote name | asset | risk model               | margin calculator         | auction duration | fees         | price monitoring | data source config          |
      | ETH/DEC19 | ETH        | ETH   | my-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: Order from liquidity provision and from normal order submission are correctly cumulated in order book's total size.

    Given the parties deposit on asset's general account the following amount:
      | party      | asset | amount       |
      | party1     | ETH   | 10000000     |
      | party2     | ETH   | 10000000     |
      | party-lp-1 | ETH   | 100000000000 |
      | party3     | ETH   | 1000000000   |
      | lpprov     | ETH   | 1000000000   |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 1      | 11999999 | 0                | TYPE_LIMIT | TIF_GTC | party1-1 |
      | party2 | ETH/DEC19 | sell | 1      | 12000001 | 0                | TYPE_LIMIT | TIF_GTC | party2-1 |
      | party1 | ETH/DEC19 | buy  | 1      | 12000000 | 0                | TYPE_LIMIT | TIF_GFA | party1-2 |
      | party2 | ETH/DEC19 | sell | 1      | 12000000 | 0                | TYPE_LIMIT | TIF_GFA | party2-2 |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "12000000" for the market "ETH/DEC19"

    When the parties submit the following liquidity provision:
      | id  | party      | market id | commitment amount | fee | side | pegged reference | proportion | offset | reference | lp type    |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | buy  | BID              | 1          | 9      | lp-1-ref  | submission |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | sell | ASK              | 1          | 9      | lp-1-ref  | submission |
    Then the liquidity provisions should have the following states:
      | id  | party      | market    | commitment amount | status        |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000        | STATUS_ACTIVE |
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the orders should have the following states:
      | party      | market id | side | volume | price    | status        |
      | party-lp-1 | ETH/DEC19 | buy  | 167    | 11999990 | STATUS_ACTIVE |
      | party-lp-1 | ETH/DEC19 | sell | 167    | 12000010 | STATUS_ACTIVE |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | buy  | 167    | 11999990 | 0                | TYPE_LIMIT | TIF_GTC | party3-1 |
      | party3 | ETH/DEC19 | sell | 167    | 12000010 | 0                | TYPE_LIMIT | TIF_GTC | party3-2 |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price    | volume |
      | buy  | 11999990 | 334    |
      | sell | 12000010 | 334    |
