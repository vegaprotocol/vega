Feature: Test trader accounts

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/short | tau/long | mu | r | sigma | release factor | initial factor | search factor | settlementPrice |
      | ETH/DEC19 | BTC      | ETH       | ETH   |        94 | simple     |        0.1 |      0.2 |  0 | 0 |     0 |              5 |              4 |           3.2 |              94 |
    And the following traders:
      | name       | amount |
      | trader1    | 1000000000  |
      | sellSideMM | 1000000000  |
      | buySideMM  | 1000000000  |
    # setting mark price
    And traders place following orders:
      | trader     | market id | type | volume | price | resulting trades | type  | tif |
      | sellSideMM | ETH/DEC19 | sell |      1 |   10300000 |                0 | LIMIT | GTC |
      | buySideMM  | ETH/DEC19 |  buy |      1 |   10300000 |                1 | LIMIT | GTC |
    # setting order book
    And traders place following orders:
      | trader     | market id | type | volume | price | resulting trades | type  | tif |
      | sellSideMM | ETH/DEC19 | sell |    100 |   25000000 |                0 | LIMIT | GTC |
      | sellSideMM | ETH/DEC19 | sell |     11 |   14000000 |                0 | LIMIT | GTC |
      | sellSideMM | ETH/DEC19 | sell |      2 |   11200000 |                0 | LIMIT | GTC |
      | buySideMM  | ETH/DEC19 |  buy |      1 |   10000000 |                0 | LIMIT | GTC |
      | buySideMM  | ETH/DEC19 |  buy |      3 |    9600000 |                0 | LIMIT | GTC |
      | buySideMM  | ETH/DEC19 |  buy |     15 |    9000000 |                0 | LIMIT | GTC |
      | buySideMM  | ETH/DEC19 |  buy |     50 |    8700000 |                0 | LIMIT | GTC |


  Scenario: trader places riskier long
    Given I Expect the traders to have new general account:
      | name    | asset |
      | trader1 | ETH   |
    # no margin account created for trader1, just general account
    And "trader1" have only one account per asset
    # placing test order
    Then traders place following orders:
      | trader     | market id | type | volume | price | resulting trades | type  | tif |
      | trader1    | ETH/DEC19 |  buy |     13 |   15000000 |                2 | LIMIT | GTC |
    And "trader1" general account for asset "ETH" balance is "683999968"
    #And executed trades:
    #  |  buyer  | price | size |       seller |
    #  | trader1 |   112 |    2 |   sellSideMM |
    #  | trader1 |   140 |   11 |   sellSideMM |

    # checking margins
    Then I expect the trader to have a margin:
      | trader  | asset | market id |    margin |   general |
      | trader1 | ETH   | ETH/DEC19 | 321600032 | 683999968 |
    And the margins levels for the traders are:
      | trader  | market id | maintenance |    search |   initial |   release |
      | trader1 | ETH/DEC19 |    80400008 | 257280025 | 321600032 | 402000040 |
