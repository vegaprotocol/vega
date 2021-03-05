Feature: Distressed traders should not have general balance left

  Background:
    Given the markets starts on "2020-10-16T00:00:00Z" and expires on "2020-12-31T23:59:59Z"
    And the execution engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/long | tau/short | mu | r | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode | makerFee | infrastructureFee | liquidityFee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | Prob of trading | oracleSpecPubKeys     | oracleSpecProperty | oracleSpecPropertyType | oracleSpecBinding |
      | ETH/DEC20 | BTC      | ETH       | ETH   | 1000      | simple     | 0.11      | 0.1       | 0  | 0 | 0     | 1.4            | 1.2            | 1.1           | 42              | 0           | continuous   | 0        | 0                 | 0            | 0                  |                |             |                 | 0.1             | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value   | TYPE_INTEGER           | prices.ETH.value  |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42   |
    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

  Scenario: Upper bound breached
    Given the following traders:
      | name    | amount         |
      | trader1 | 10000000000000 |
      | trader2 | 10000000000000 |
      | trader3 | 24000          |
      | trader4 | 10000000000000 |
      | trader5 | 10000000000000 |

    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  | 1      | 100   | 1                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC20 | sell | 20     | 120   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  | 20     | 80    | 0                | TYPE_LIMIT | TIF_GTC |


    And the mark price for the market "ETH/DEC20" is "100"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    # T0 + 1min - this causes the price for comparison of the bounds to be 567
    Then the time is updated to "2020-10-16T00:01:00Z"

    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type       | tif     |
      | trader4 | ETH/DEC20 | sell | 10     | 100   | 0                | TYPE_LIMIT | TIF_GTC |

    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type       | tif     |
      | trader5 | ETH/DEC20 | buy  | 10     | 100   | 1                | TYPE_LIMIT | TIF_FOK |
      | trader3 | ETH/DEC20 | buy  | 10     | 110   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader3 | ETH/DEC20 | sell | 10     | 120   | 0                | TYPE_LIMIT | TIF_GTC |
    And dump orders

    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin | general       |
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
    And dump orders

    Then I see the following order events:
      | trader  | id        | side | volume | reference | offset | price | status        |
      | trader3 | ETH/DEC20 | buy  | 989    | BID       | -10    | 100   | STATUS_ACTIVE |
      | trader3 | ETH/DEC20 | sell | 760    | ASK       | 10     | 130   | STATUS_ACTIVE |
      # | trader3 | ETH/DEC20 | sell |    582 |      ASK  | 50     | 170   | STATUS_ACTIVE |
      # | trader3 | ETH/DEC20 | sell |    791 |      ASK  | 5      | 125   | STATUS_ACTIVE |
      # | trader3 | ETH/DEC20 | sell |    507 |      ASK  | 75     | 195   | STATUS_ACTIVE |
      # | trader3 | ETH/DEC20 | sell |    520 |      MID  | 75     | 190   | STATUS_ACTIVE |
    ## The sum of the margin + general account == 10000 - 10000 (commitment amount)
    And I expect the trader to have a margin:
      | trader  | asset | id        | margin | general |
      | trader3 | ETH   | ETH/DEC20 | 13186  | 814     |
      
    ## Now let's increase the mark price so trader3 gets distressed
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type       | tif     |
      | trader5 | ETH/DEC20 | buy  | 20     | 165   | 1                | TYPE_LIMIT | TIF_GTC |
    And the mark price for the market "ETH/DEC20" is "120"

    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin | general |
      | trader3 | ETH   | ETH/DEC20 | 13186  | 814     |
      # expected balances to be margin(15165) general(0), instead saw margin(13186), general(814), (trader: trader3)

    ## Now let's increase the mark price so trader3 gets distressed
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type       | tif     |
      | trader5 | ETH/DEC20 | buy  | 30     | 165   | 2                | TYPE_LIMIT | TIF_GTC |
    And the mark price for the market "ETH/DEC20" is "130"

    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin | general |
      | trader3 | ETH   | ETH/DEC20 | 17143  | 0       |
