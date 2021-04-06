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
      | trader1 | ETH/DEC19 | buy  | 1      | 11999999 | 0                | TYPE_LIMIT | TIF_GTC | trader1-1 |
      | trader2 | ETH/DEC19 | sell | 1      | 12000001 | 0                | TYPE_LIMIT | TIF_GTC | trader2-1 |
      | trader1 | ETH/DEC19 | buy  | 1      | 12000000 | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 12000000 | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "12000000" for the market "ETH/DEC19"

    Then debug market data for "ETH/DEC19"


    When the traders submit the following liquidity provision:
      | id  | party       | market id | commitment amount | fee | order side | order reference | order proportion | order offset |reference |
      | lp1 | trader-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | buy        | BID             | 1                | -9          | lp-1-ref |
      | lp1 | trader-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | sell       | ASK             | 1                | 9           | lp-1-ref |
    Then I see the LP events:
      | id  | party       | market    | commitment amount | status        |
      | lp1 | trader-lp-1 | ETH/DEC19 | 1000000000        | STATUS_ACTIVE |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    And I see the following order events:
      | trader      | market id | side | volume | reference | offset | price    | status        |
      | trader-lp-1 | ETH/DEC19 | buy  | 167    |           | 0      | 11999990 | STATUS_ACTIVE |
      | trader-lp-1 | ETH/DEC19 | sell | 167    |           | 0      | 12000010 | STATUS_ACTIVE |

    When the traders place the following orders:
      | trader  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC19 | buy  | 167    | 11999990 | 0                | TYPE_LIMIT | TIF_GTC | trader3-1 |
      | trader3 | ETH/DEC19 | sell | 167    | 12000010 | 0                | TYPE_LIMIT | TIF_GTC | trader3-2 |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price    | volume |
      | buy  | 11999990 | 334    |
      | sell | 12000010 | 334    |
