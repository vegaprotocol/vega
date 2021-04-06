Feature: Verify the order size is correctly cumulated.

  Background:
    Given the initial insurance pool balance is "0" for the markets:
    And the log normal risk model named "my-log-normal-risk-model":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.001         | 0.00000190128526884174 | 0  | 0.016 | 2.5   |
    And the markets:
      | id        | quote name | asset | risk model               | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | ETH        | ETH   | my-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Order from liquidity provision and from normal order submission are correctly cumulated in order book's total size.

    Given the traders deposit on asset's general account the following amount:
      | trader      | asset | amount       |
      | trader1     | ETH   | 10000000     |
      | trader2     | ETH   | 10000000     |
      | trader-lp-1 | ETH   | 100000000000 |
      | trader3     | ETH   | 1000000000   |

    # Trigger an auction to set the mark price
    When the traders place the following orders:
      | trader  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 12000007 | 0                | TYPE_LIMIT | TIF_GTC | trader1-1 |
      | trader2 | ETH/DEC19 | sell | 1      | 12000020 | 0                | TYPE_LIMIT | TIF_GTC | trader2-1 |
      | trader1 | ETH/DEC19 | buy  | 1      | 12000010 | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 12000010 | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "12000010" for the market "ETH/DEC19"

    Then the traders place the following orders:
      | trader      | market id | side | volume | price     | resulting trades | type       | tif     | reference |
      | trader-lp-1 | ETH/DEC19 | sell | 50      | 12000013 | 0                | TYPE_LIMIT | TIF_GTC | trader2-1 |

    Then debug market data for "ETH/DEC19"

    When the traders submit the following liquidity provision:
      | id  | party       | market id | commitment amount | fee | order side | order reference | order proportion | order offset |reference |
      | lp1 | trader-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | buy        | MID             | 1                | -10          | lp-1-ref |
      | lp1 | trader-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | buy        | MID             | 1                | -9           | lp-1-ref |
      | lp1 | trader-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | buy        | MID             | 1                | -8           | lp-1-ref |
      | lp1 | trader-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | buy        | MID             | 1                | -7           | lp-1-ref |
      | lp1 | trader-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | buy        | MID             | 1                | -6           | lp-1-ref |
      | lp1 | trader-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | buy        | MID             | 1                | -5           | lp-1-ref |
      | lp1 | trader-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | buy        | MID             | 1                | -4           | lp-1-ref |
      | lp1 | trader-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | buy        | MID             | 1                | -3           | lp-1-ref |
      | lp1 | trader-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | buy        | MID             | 1                | -2           | lp-1-ref |
      | lp1 | trader-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | sell       | MID             | 1                | 4            | lp-1-ref |
      | lp1 | trader-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | sell       | MID             | 1                | 5            | lp-1-ref |
      | lp1 | trader-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | sell       | MID             | 1                | 6            | lp-1-ref |
      | lp1 | trader-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | sell       | MID             | 1                | 7            | lp-1-ref |
      | lp1 | trader-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | sell       | MID             | 1                | 8            | lp-1-ref |
      | lp1 | trader-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | sell       | MID             | 1                | 9            | lp-1-ref |
      | lp1 | trader-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | sell       | MID             | 1                | 10           | lp-1-ref |

    Then I see the LP events:
      | id  | party       | market    | commitment amount | status        |
      | lp1 | trader-lp-1 | ETH/DEC19 | 1000000000        | STATUS_ACTIVE |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    And I see the following order events:
      | trader      | market id | side | volume | reference | offset | price    | status        |
      | trader-lp-1 | ETH/DEC19 | sell | 17     |           | 0      | 12000014 | STATUS_ACTIVE |
      | trader-lp-1 | ETH/DEC19 | sell | 17     |           | 0      | 12000015 | STATUS_ACTIVE |
      | trader-lp-1 | ETH/DEC19 | sell | 17     |           | 0      | 12000016 | STATUS_ACTIVE |
      | trader-lp-1 | ETH/DEC19 | sell | 17     |           | 0      | 12000017 | STATUS_ACTIVE |
      | trader-lp-1 | ETH/DEC19 | sell | 17     |           | 0      | 12000018 | STATUS_ACTIVE |
      | trader-lp-1 | ETH/DEC19 | sell | 17     |           | 0      | 12000019 | STATUS_ACTIVE |
      | trader-lp-1 | ETH/DEC19 | sell | 17     |           | 0      | 12000020 | STATUS_ACTIVE |
      | trader-lp-1 | ETH/DEC19 | buy  | 19     |           | 0      | 12000008 | STATUS_ACTIVE |
      | trader-lp-1 | ETH/DEC19 | buy  | 19     |           | 0      | 12000007 | STATUS_ACTIVE |
      | trader-lp-1 | ETH/DEC19 | buy  | 19     |           | 0      | 12000006 | STATUS_ACTIVE |
      | trader-lp-1 | ETH/DEC19 | buy  | 19     |           | 0      | 12000005 | STATUS_ACTIVE |
      | trader-lp-1 | ETH/DEC19 | buy  | 19     |           | 0      | 12000004 | STATUS_ACTIVE |
      | trader-lp-1 | ETH/DEC19 | buy  | 19     |           | 0      | 12000003 | STATUS_ACTIVE |
      | trader-lp-1 | ETH/DEC19 | buy  | 19     |           | 0      | 12000002 | STATUS_ACTIVE |
      | trader-lp-1 | ETH/DEC19 | buy  | 19     |           | 0      | 12000001 | STATUS_ACTIVE |
      | trader-lp-1 | ETH/DEC19 | buy  | 19     |           | 0      | 12000000 | STATUS_ACTIVE |

    When the traders place the following orders:
      | trader  | market id | side | volume | price    | resulting trades | type       | tif     | reference  |
      | trader3 | ETH/DEC19 | sell | 167    | 12000020 | 0                | TYPE_LIMIT | TIF_GTC | trader3-1  |
      | trader3 | ETH/DEC19 | sell | 50     | 12000019 | 0                | TYPE_LIMIT | TIF_GTC | trader3-2  |
      | trader3 | ETH/DEC19 | sell | 50     | 12000018 | 0                | TYPE_LIMIT | TIF_GTC | trader3-3  |
      | trader3 | ETH/DEC19 | sell | 50     | 12000017 | 0                | TYPE_LIMIT | TIF_GTC | trader3-4  |
      | trader3 | ETH/DEC19 | sell | 50     | 12000016 | 0                | TYPE_LIMIT | TIF_GTC | trader3-5  |
      | trader3 | ETH/DEC19 | sell | 50     | 12000015 | 0                | TYPE_LIMIT | TIF_GTC | trader3-6  |
      | trader3 | ETH/DEC19 | sell | 10     | 12000014 | 0                | TYPE_LIMIT | TIF_GTC | trader3-7  |
      | trader3 | ETH/DEC19 | buy  | 1      | 12000006 | 0                | TYPE_LIMIT | TIF_GTC | trader3-8  |
      | trader3 | ETH/DEC19 | buy  | 50     | 12000005 | 0                | TYPE_LIMIT | TIF_GTC | trader3-9  |
      | trader3 | ETH/DEC19 | buy  | 50     | 12000004 | 0                | TYPE_LIMIT | TIF_GTC | trader3-10 |
      | trader3 | ETH/DEC19 | buy  | 50     | 12000003 | 0                | TYPE_LIMIT | TIF_GTC | trader3-11 |
      | trader3 | ETH/DEC19 | buy  | 50     | 12000002 | 0                | TYPE_LIMIT | TIF_GTC | trader3-12 |
      | trader3 | ETH/DEC19 | buy  | 50     | 12000001 | 0                | TYPE_LIMIT | TIF_GTC | trader3-13 |
      | trader3 | ETH/DEC19 | buy  | 167    | 12000000 | 0                | TYPE_LIMIT | TIF_GTC | trader3-14 |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price    | volume |
      | sell | 12000020 | 185    |
      | sell | 12000019 | 67     |
      | sell | 12000018 | 67     |
      | sell | 12000017 | 67     |
      | sell | 12000016 | 67     |
      | sell | 12000015 | 67     |
      | sell | 12000014 | 27     |
      | sell | 12000013 | 50     |
      | buy  | 12000008 | 19     |
      | buy  | 12000007 | 20     | # +1 here for the order used to set the midprice
      | buy  | 12000006 | 20     |
      | buy  | 12000005 | 69     |
      | buy  | 12000004 | 69     |
      | buy  | 12000003 | 69     |
      | buy  | 12000002 | 69     |
      | buy  | 12000001 | 69     |
      | buy  | 12000000 | 186    |
