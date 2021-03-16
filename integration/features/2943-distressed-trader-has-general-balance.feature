Feature: Distressed traders should not have general balance left

  Background:
    Given the markets start on "2020-10-16T00:00:00Z" and expire on "2020-12-31T23:59:59Z"
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlement price | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC20 | ETH        | ETH   | simple     | 0.11      | 0.1       | 0              | 0               | 0     | 1.4            | 1.2            | 1.1           | 42               | 0                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |
    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

  Scenario: Upper bound breached
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount         |
      | trader1 | ETH   | 10000000000000 |
      | trader2 | ETH   | 10000000000000 |
      | trader3 | ETH   | 24000          |
      | trader4 | ETH   | 10000000000000 |
      | trader5 | ETH   | 10000000000000 |

    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  | 1      | 100   | 1                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC20 | sell | 20     | 120   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  | 20     | 80    | 0                | TYPE_LIMIT | TIF_GTC |


    And the mark price for the market "ETH/DEC20" is "100"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    # T0 + 1min - this causes the price for comparison of the bounds to be 567
    Then time is updated to "2020-10-16T00:01:00Z"

    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader4 | ETH/DEC20 | sell | 10     | 100   | 0                | TYPE_LIMIT | TIF_GTC |

    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader5 | ETH/DEC20 | buy  | 10     | 100   | 1                | TYPE_LIMIT | TIF_FOK |
      | trader3 | ETH/DEC20 | buy  | 10     | 110   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3 | ETH/DEC20 | sell | 10     | 120   | 0                | TYPE_LIMIT | TIF_GTC |

    Then traders have the following account balances:
      | trader  | asset | market id | margin | general       |
      | trader4 | ETH   | ETH/DEC20 | 360    | 9999999999640 |
      | trader5 | ETH   | ETH/DEC20 | 372    | 9999999999628 |
    And clear order events
    Then the trader submits LP:
      | id  | party   | market    | commitment amount | fee | order side | order reference | order proportion | order offset |
      | lp1 | trader3 | ETH/DEC20 | 10000             | 0.1 | buy        | BID             | 10               | -10          |
      | lp1 | trader3 | ETH/DEC20 | 10000             | 0.1 | sell       | ASK             | 10               | 10           |
    Then I see the LP events:
      | id  | party   | market    | commitment amount |
      | lp1 | trader3 | ETH/DEC20 | 10000             |

    Then I see the following order events:
      | trader  | market id | side | volume | reference | offset | price | status        |
      | trader3 | ETH/DEC20 | buy  | 989    | BID       | -10    | 100   | STATUS_ACTIVE |
      | trader3 | ETH/DEC20 | sell | 760    | ASK       | 10     | 130   | STATUS_ACTIVE |
    ## The sum of the margin + general account == 10000 - 10000 (commitment amount)
    Then traders have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader3 | ETH   | ETH/DEC20 | 13186  | 814     |

    ## Now let's increase the mark price so trader3 gets distressed
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader5 | ETH/DEC20 | buy  | 20     | 165   | 1                | TYPE_LIMIT | TIF_GTC |
    And the mark price for the market "ETH/DEC20" is "120"

    Then traders have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader3 | ETH   | ETH/DEC20 | 13186  | 814     |
      # expected balances to be margin(15165) general(0), instead saw margin(13186), general(814), (trader: trader3)

    ## Now let's increase the mark price so trader3 gets distressed
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader5 | ETH/DEC20 | buy  | 30     | 165   | 2                | TYPE_LIMIT | TIF_GTC |
    And the mark price for the market "ETH/DEC20" is "130"

    Then traders have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader3 | ETH   | ETH/DEC20 | 17143  | 0       |
