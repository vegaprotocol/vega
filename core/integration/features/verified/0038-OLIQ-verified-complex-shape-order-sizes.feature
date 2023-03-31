Feature: Verify the order size is correctly cumulated.

  # Numbers come from 0038-liquidity-provision-order-type.xlsx

  Background:
    Given the log normal risk model named "my-log-normal-risk-model":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.001         | 0.00000190128526884174 | 0  | 0.016 | 2.5   |
    And the markets:
      | id        | quote name | asset | risk model               | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | ETH   | my-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                     | value |
      | market.liquidityProvision.shapes.maxSize | 10    |
      | network.markPriceUpdateMaximumFrequency  | 0s    |

  Scenario: 001: Order from liquidity provision and from normal order submission are correctly cumulated in order book's total size (0038-OLIQ-003, 0038-OLIQ-004, 0038-OLIQ-005)

    Given the parties deposit on asset's general account the following amount:
      | party      | asset | amount       |
      | party1     | ETH   | 10000000     |
      | party2     | ETH   | 10000000     |
      | party-lp-1 | ETH   | 100000000000 |
      | party3     | ETH   | 1000000000   |

    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 1      | 12000007 | 0                | TYPE_LIMIT | TIF_GTC | party1-1  |
      | party2 | ETH/DEC19 | sell | 1      | 12000020 | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party1 | ETH/DEC19 | buy  | 1      | 12000010 | 0                | TYPE_LIMIT | TIF_GFA | party1-2  |
      | party2 | ETH/DEC19 | sell | 1      | 12000010 | 0                | TYPE_LIMIT | TIF_GFA | party2-2  |

    And the parties submit the following liquidity provision:
      | id  | party      | market id | commitment amount | fee | side | pegged reference | proportion | offset | reference | lp type    |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | buy  | MID              | 1          | 10     | lp-1-ref  | submission |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | buy  | MID              | 1          | 9      | lp-1-ref  | submission |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | buy  | MID              | 1          | 8      | lp-1-ref  | submission |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | buy  | MID              | 1          | 7      | lp-1-ref  | submission |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | buy  | MID              | 1          | 6      | lp-1-ref  | submission |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | buy  | MID              | 1          | 5      | lp-1-ref  | submission |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | buy  | MID              | 1          | 4      | lp-1-ref  | submission |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | buy  | MID              | 1          | 3      | lp-1-ref  | submission |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | buy  | MID              | 1          | 2      | lp-1-ref  | submission |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | sell | MID              | 1          | 4      | lp-1-ref  | submission |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | sell | MID              | 1          | 5      | lp-1-ref  | submission |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | sell | MID              | 1          | 6      | lp-1-ref  | submission |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | sell | MID              | 1          | 7      | lp-1-ref  | submission |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | sell | MID              | 1          | 8      | lp-1-ref  | submission |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | sell | MID              | 1          | 9      | lp-1-ref  | submission |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | sell | MID              | 1          | 10     | lp-1-ref  | submission |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "12000010" for the market "ETH/DEC19"

    When the parties place the following orders:
      | party      | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | party-lp-1 | ETH/DEC19 | sell | 50     | 12000013 | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |

    Then the liquidity provisions should have the following states:
      | id  | party      | market    | commitment amount | status        |
      | lp1 | party-lp-1 | ETH/DEC19 | 1000000000        | STATUS_ACTIVE |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    And the orders should have the following states:
      | party      | market id | side | volume | price    | status        |
      | party-lp-1 | ETH/DEC19 | sell | 5      | 12000020 | STATUS_ACTIVE |
      | party-lp-1 | ETH/DEC19 | sell | 5      | 12000019 | STATUS_ACTIVE |
      | party-lp-1 | ETH/DEC19 | sell | 5      | 12000018 | STATUS_ACTIVE |
      | party-lp-1 | ETH/DEC19 | sell | 5      | 12000017 | STATUS_ACTIVE |
      | party-lp-1 | ETH/DEC19 | sell | 5      | 12000016 | STATUS_ACTIVE |
      | party-lp-1 | ETH/DEC19 | sell | 5      | 12000015 | STATUS_ACTIVE |
      | party-lp-1 | ETH/DEC19 | sell | 5      | 12000014 | STATUS_ACTIVE |
      | party-lp-1 | ETH/DEC19 | buy  | 10     | 12000008 | STATUS_ACTIVE |
      | party-lp-1 | ETH/DEC19 | buy  | 10     | 12000007 | STATUS_ACTIVE |
      | party-lp-1 | ETH/DEC19 | buy  | 10     | 12000006 | STATUS_ACTIVE |
      | party-lp-1 | ETH/DEC19 | buy  | 10     | 12000005 | STATUS_ACTIVE |
      | party-lp-1 | ETH/DEC19 | buy  | 10     | 12000004 | STATUS_ACTIVE |
      | party-lp-1 | ETH/DEC19 | buy  | 10     | 12000003 | STATUS_ACTIVE |
      | party-lp-1 | ETH/DEC19 | buy  | 10     | 12000002 | STATUS_ACTIVE |
      | party-lp-1 | ETH/DEC19 | buy  | 10     | 12000001 | STATUS_ACTIVE |
      | party-lp-1 | ETH/DEC19 | buy  | 10     | 12000000 | STATUS_ACTIVE |

    When the parties place the following orders:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | sell | 167    | 12000020 | 0                | TYPE_LIMIT | TIF_GTC | party3-1  |
      | party3 | ETH/DEC19 | sell | 50     | 12000019 | 0                | TYPE_LIMIT | TIF_GTC | party3-2  |
      | party3 | ETH/DEC19 | sell | 50     | 12000018 | 0                | TYPE_LIMIT | TIF_GTC | party3-3  |
      | party3 | ETH/DEC19 | sell | 50     | 12000017 | 0                | TYPE_LIMIT | TIF_GTC | party3-4  |
      | party3 | ETH/DEC19 | sell | 50     | 12000016 | 0                | TYPE_LIMIT | TIF_GTC | party3-5  |
      | party3 | ETH/DEC19 | sell | 50     | 12000015 | 0                | TYPE_LIMIT | TIF_GTC | party3-6  |
      | party3 | ETH/DEC19 | sell | 10     | 12000014 | 0                | TYPE_LIMIT | TIF_GTC | party3-7  |
      | party3 | ETH/DEC19 | buy  | 1      | 12000006 | 0                | TYPE_LIMIT | TIF_GTC | party3-8  |
      | party3 | ETH/DEC19 | buy  | 50     | 12000005 | 0                | TYPE_LIMIT | TIF_GTC | party3-9  |
      | party3 | ETH/DEC19 | buy  | 50     | 12000004 | 0                | TYPE_LIMIT | TIF_GTC | party3-10 |
      | party3 | ETH/DEC19 | buy  | 50     | 12000003 | 0                | TYPE_LIMIT | TIF_GTC | party3-11 |
      | party3 | ETH/DEC19 | buy  | 50     | 12000002 | 0                | TYPE_LIMIT | TIF_GTC | party3-12 |
      | party3 | ETH/DEC19 | buy  | 50     | 12000001 | 0                | TYPE_LIMIT | TIF_GTC | party3-13 |
      | party3 | ETH/DEC19 | buy  | 167    | 12000000 | 0                | TYPE_LIMIT | TIF_GTC | party3-14 |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | volume | price    |
      | sell | 173    | 12000020 |
      | sell | 55     | 12000019 |
      | sell | 55     | 12000018 |
      | sell | 55     | 12000017 |
      | sell | 55     | 12000016 |
      | sell | 55     | 12000015 |
      | sell | 15     | 12000014 |
      | buy  | 10     | 12000008 |
      | buy  | 11     | 12000007 |
      | buy  | 11     | 12000006 |
      | buy  | 60     | 12000005 |
      | buy  | 60     | 12000004 |
      | buy  | 60     | 12000003 |
      | buy  | 60     | 12000002 |
      | buy  | 60     | 12000001 |
      | buy  | 177    | 12000000 |
