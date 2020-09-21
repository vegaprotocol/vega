Feature: test for issue xxxx

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/long | tau/short              | mu | r     | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode | makerFee | infrastructureFee | liquidityFee |
      | ETH/DEC19 | BTC      | ETH       | ETH   | 10000     | forward    | 0.001     | 0.00011407711613050422 | 0  | 0.016 | 1.5   | 1.4            | 1.2            | 1.1           | 10000           | 0           | continuous   | 0        | 0                 | 0            |

  Scenario: a trader place a new order in the system, margin are calculated, then the order is stopped, the margin is released
    Given the following traders:
      | name    | amount   |
      | trader1 | 10000000 |
      | trader2 | 10000000 |
      | trader3 | 10000000 |
      | trader4 | 10000000 |
      | trader5 | 10000000 |
      | trader6 | 10000000 |
      | trader7 | 10000000 |
      | trader8 | 10000000 |
      | trader9 | 10000000 |

    Then I Expect the traders to have new general account:
      | name    | asset |
      | trader1 | ETH   |
      | trader2 | ETH   |
      | trader3 | ETH   |
      | trader4 | ETH   |
      | trader5 | ETH   |
      | trader6 | ETH   |
      | trader7 | ETH   |
      | trader8 | ETH   |
      | trader9 | ETH   |


    #test_ActiveGTCOrder
    And the mark price for the market "ETH/DEC19" is "10000"
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | buy  | 21     | 22    | 0                | TYPE_LIMIT | TIF_GTC |
    And the mark price for the market "ETH/DEC19" is "10000"
    # test_RejectedGTC_MarginCheckFail
    # TODO: Implement option to test if order rejected, for now just uncommnet the lines below and satisfy yourself that it fails.
    # Then traders place following orders:
    #   | trader  | id        | type | volume       | price  | resulting trades | type       | tif     |
    #   | trader1 | ETH/DEC19 | buy  | 500000000000 | 220000 | 0                | TYPE_LIMIT | TIF_GTC |
    #test_FilledGTCOrder
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type       | tif     |
      | trader5 | ETH/DEC19 | buy  | 19     | 25    | 0                | TYPE_LIMIT | TIF_GTC |
      | trader6 | ETH/DEC19 | sell | 19     | 25    | 1                | TYPE_LIMIT | TIF_GTC |
    And the mark price for the market "ETH/DEC19" is "25"
    # test_PartiallyFilledGTC
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type       | tif     |
      | trader5 | ETH/DEC19 | buy  | 30     | 33    | 0                | TYPE_LIMIT | TIF_GTC |
      | trader6 | ETH/DEC19 | sell | 20     | 33    | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following trades happened:
      | buyer   | seller  | price | volume |
      | trader5 | trader6 | 33    | 20     |
    And the mark price for the market "ETH/DEC19" is "33"
    # test_FilledGTTOrder
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type       | tif     |
      | trader3 | ETH/DEC19 | buy  | 50     | 38    | 0                | TYPE_LIMIT | TIF_GTT |
      | trader4 | ETH/DEC19 | sell | 50     | 38    | 1                | TYPE_LIMIT | TIF_GTT |
    And the mark price for the market "ETH/DEC19" is "38"
    # test_ExpiredGTTOrder - Just checks that the order expires
    #test_PartiallyFilledGTTOrder
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type       | tif     |
      | trader3 | ETH/DEC19 | buy  | 50     | 20    | 0                | TYPE_LIMIT | TIF_GTT |
      | trader4 | ETH/DEC19 | sell | 50     | 20    | 1                | TYPE_LIMIT | TIF_GTT |
    And the mark price for the market "ETH/DEC19" is "20"
    Then the following trades happened:
      | buyer   | seller  | price | volume |
      | trader3 | trader4 | 20    | 19     |

# Then the margins levels for the traders are:
#   | trader  | id        | maintenance | search | initial | release |
#   | traderC | ETH/DEC19 | 2632        | 2895   | 3158    | 3684    |
# Then I expect the trader to have a margin:
#   | trader  | asset | id        | margin | general |
#   | traderA | ETH   | ETH/DEC19 | 3158   | 42      |
