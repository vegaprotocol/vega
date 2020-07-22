Feature: Test mark to market settlement with insurance pool

  Background:
    Given the insurance pool initial balance for the markets is "10000":
    And the executon engine have these markets:
      | name      |baseName | quoteName | asset | markprice | risk model | lamd/long | tau/short | mu | r | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode |
      | ETH/DEC19 |BTC      | ETH       | ETH   | 1000      | simple     | 0.11       | 0.1      | 0  | 0 | 0     | 1.4            | 1.2            | 1.1           | 42              | 0           | continuous   |
  Scenario: If settlement amount > trader’s margin account balance + trader’s general account balance for the asset, the full balance of the trader’s margin account is transferred to the market’s temporary settlement account, the full balance of the trader’s general account for the assets are transferred to the market’s temporary settlement account, the minimum insurance pool account balance for the market & asset, and the remainder, i.e. the difference between the total amount transferred from the trader’s margin + general accounts and the settlement amount, is transferred from the insurance pool account for the market to the temporary settlement account for the market
    Given the following traders:
      | name    | amount |
      | trader1 |    121 |
      | trader2 |  10000 |
      | trader3 |  10000 |
    Then I Expect the traders to have new general account:
      | name    | asset |
      | trader1 | ETH   |
      | trader2 | ETH   |
      | trader3 | ETH   |
    And "trader1" general accounts balance is "121"
    And "trader2" general accounts balance is "10000"
    And "trader3" general accounts balance is "10000"
   And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
    Then traders place following orders:
      | trader  | id         | type | volume | price | resulting trades | type  | tif |
      | trader1 |  ETH/DEC19 | sell |     1  |  1000 |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 |  ETH/DEC19 | buy  |     1  |  1000 |                1 | TYPE_LIMIT | TIF_GTC |
    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin | general |
      | trader1 | ETH   | ETH/DEC19 |    120 |       1 |
      | trader2 | ETH   | ETH/DEC19 |    132 |    9868 |

    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
    Then traders place following orders:
      | trader  | id         | type | volume | price | resulting trades | type  | tif |
      | trader2 |  ETH/DEC19 | buy |     1  |  6000 |                0 | TYPE_LIMIT | TIF_GTC |
    Then I expect the trader to have a margin:
      | trader  | asset |        id | margin | general |
      | trader2 | ETH   | ETH/DEC19 |    264 |    9736 |

    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades | type  | tif |
      | trader3 | ETH/DEC19 | sell |      1 |  5000 |                1 | TYPE_LIMIT | TIF_GTC |
    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin | general |
      | trader1 | ETH   | ETH/DEC19 |      0 |       0 |
      | trader2 | ETH   | ETH/DEC19 |   1584 |   13416 |
      | trader3 | ETH   | ETH/DEC19 |    720 |    9280 |
   And All balances cumulated are worth "30121"
   And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
   And the insurance pool balance is "5121" for the market "ETH/DEC19"

    # Then the following transfers happened:
    #   | from    | to     | fromType            | toType                  | id        | amount | asset |
    #   | trader1 | market | ACCOUNT_TYPE_MARGIN | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 |    240 | ETH   |
    # And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
