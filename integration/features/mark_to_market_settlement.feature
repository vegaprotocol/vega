Feature: Test mark to market settlement

  Background:
    Given the executon engine have these markets:
      | name      |baseName | quoteName | asset | markprice | risk model | lamd/short | tau/long | mu | r | sigma | release factor | initial factor | search factor |
      | ETH/DEC19 |BTC      | ETH       | ETH   | 1000      | simple     | 0.11       | 0.1      | 0  | 0 | 0     | 1.4            | 1.2            | 1.1           |

  Scenario: If settlement amount <= the trader’s margin account balance entire settlement amount is transferred from trader’s margin account to the market’s temporary settlement account
    Given the following traders:
      | name    | amount |
      | trader1 |  10000 |
      | trader2 |  10000 |
      | trader3 |  10000 |
    Then I Expect the traders to have new general account:
      | name    | asset |
      | trader1 | ETH   |
      | trader2 | ETH   |
      | trader3 | ETH   |
    And "trader1" general accounts balance is "10000"
    And "trader2" general accounts balance is "10000"
    And "trader3" general accounts balance is "10000"
    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
    Then traders place following orders:
      | trader  | id         | type | volume | price | resulting trades |
      | trader1 |  ETH/DEC19 | sell |     1  |  1000 |                0 |
      | trader2 |  ETH/DEC19 | buy  |     1  |  1000 |                1 |
    Then I expect the trader to have a margin:
      | trader  | asset |        id | margin | general |
      | trader1 | ETH   | ETH/DEC19 |    120 |    9880 |
      | trader2 | ETH   | ETH/DEC19 |    132 |    9868 |

    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
    Then traders place following orders:
      | trader  | id         | type | volume | price | resulting trades |
      | trader1 |  ETH/DEC19 | sell |     1  |  2000 |                0 |
    Then I expect the trader to have a margin:
      | trader  | asset |        id | margin | general |
      | trader1 | ETH   | ETH/DEC19 |    240 |    9760 |

    Then traders place following orders:
      | trader  | id         | type | volume | price | resulting trades |
      | trader3 |  ETH/DEC19 | buy  |     1  |  2000 |                1 |
    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin | general |
      | trader1 | ETH   | ETH/DEC19 |    480 |    9280 |
      | trader3 | ETH   | ETH/DEC19 |    132 |    9868 |
      | trader2 | ETH   | ETH/DEC19 |    308 |    9932 |
    Then the following transfers happend:
      | from    | to     | fromType | toType     | id        | amount | asset |
      | trader1 | market | MARGIN   | SETTLEMENT | ETH/DEC19 |    240 | ETH   |
    And All balances cumulated are worth "30000"
    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM

  Scenario: If settlement amount > trader’s margin account balance  and <= trader's margin account balance + general account balance for the asset, he full balance of the trader’s margin account is transferred to the market’s temporary settlement account the remainder, i.e. difference between the amount transferred from the margin account and the settlement amount, is transferred from the trader’s general account for the asset to the market’s temporary settlement account
    Given the following traders:
      | name    | amount |
      | trader1 |  10000 |
      | trader2 |  10000 |
      | trader3 |  10000 |
    Then I Expect the traders to have new general account:
      | name    | asset |
      | trader1 | ETH   |
      | trader2 | ETH   |
      | trader3 | ETH   |
    And "trader1" general accounts balance is "10000"
    And "trader2" general accounts balance is "10000"
    And "trader3" general accounts balance is "10000"
    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
    Then traders place following orders:
      | trader  | id         | type | volume | price | resulting trades |
      | trader1 |  ETH/DEC19 | sell |     1  |  1000 |                0 |
      | trader2 |  ETH/DEC19 | buy  |     1  |  1000 |                1 |
    Then I expect the trader to have a margin:
      | trader  | asset |        id | margin | general |
      | trader1 | ETH   | ETH/DEC19 |    120 |    9880 |
      | trader2 | ETH   | ETH/DEC19 |    132 |    9868 |

    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
    Then traders place following orders:
      | trader  | id         | type | volume | price | resulting trades |
      | trader1 |  ETH/DEC19 | sell |     1  |  5000 |                0 |
    Then I expect the trader to have a margin:
      | trader  | asset |        id | margin | general |
      | trader1 | ETH   | ETH/DEC19 |    240 |    9760 |

    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades |
      | trader3 | ETH/DEC19 | buy  |      1 |  5000 |                1 |
    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin | general |
      | trader1 | ETH   | ETH/DEC19 |   1200 |    8560 |
      | trader3 | ETH   | ETH/DEC19 |    132 |    9868 |
      | trader2 | ETH   | ETH/DEC19 |    660 |    9580 |
    Then the following transfers happend:
      | from    | to     | fromType | toType     | id        | amount | asset |
      | trader1 | market | MARGIN   | SETTLEMENT | ETH/DEC19 |    240 | ETH   |
    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM

# this part show that funds are moved from margin account general account for trader 3 as he does not have
# enough funds in the margin account
    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades |
      | trader3 | ETH/DEC19 | buy  |      1 |    50 |                0 |
      | trader1 | ETH/DEC19 | sell |      1 |    50 |                1 |
    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin | general |
      | trader1 | ETH   | ETH/DEC19 |     21 |   11719 |
      | trader3 | ETH   | ETH/DEC19 |     13 |    8667 |
      | trader2 | ETH   | ETH/DEC19 |      6 |    9574 |
    Then the following transfers happend:
      | from    | to      | fromType | toType     | id        | amount | asset |
      | trader3 | trader3 | GENERAL  | MARGIN     | ETH/DEC19 |   1188 | ETH   |
      | trader3 | market  | MARGIN   | SETTLEMENT | ETH/DEC19 |   1320 | ETH   |
    And All balances cumulated are worth "30000"

  Scenario: If the mark price hasn’t changed, A trader with no change in open position size has no transfers in or out of their margin account, A trader with no change in open volume
    Given the following traders:
      | name    | amount |
      | trader1 |  10000 |
      | trader2 |  10000 |
      | trader3 |  10000 |
    Then I Expect the traders to have new general account:
      | name    | asset |
      | trader1 | ETH   |
      | trader2 | ETH   |
      | trader3 | ETH   |
    And "trader1" general accounts balance is "10000"
    And "trader2" general accounts balance is "10000"
    And "trader3" general accounts balance is "10000"
    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
    Then traders place following orders:
      | trader  | id         | type | volume | price | resulting trades |
      | trader1 |  ETH/DEC19 | sell |     1  |  1000 |                0 |
      | trader2 |  ETH/DEC19 | buy  |     1  |  1000 |                1 |
    Then I expect the trader to have a margin:
      | trader  | asset |        id | margin | general |
      | trader1 | ETH   | ETH/DEC19 |    120 |    9880 |
      | trader2 | ETH   | ETH/DEC19 |    132 |    9868 |
    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
    Then traders place following orders:
      | trader  | id         | type | volume | price | resulting trades |
      | trader1 |  ETH/DEC19 | sell |     1  |  1000 |                0 |
    Then I expect the trader to have a margin:
      | trader  | asset |        id | margin | general |
      | trader1 | ETH   | ETH/DEC19 |    240 |    9760 |

    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades |
      | trader3 | ETH/DEC19 | buy  |      1 |  1000 |                1 |

# here we expect trader 2 to still have the same margin as the previous trade did not change the markprice
    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin | general |
      | trader1 | ETH   | ETH/DEC19 |    240 |    9760 |
      | trader3 | ETH   | ETH/DEC19 |    132 |    9868 |
      | trader2 | ETH   | ETH/DEC19 |    132 |    9868 |
    And All balances cumulated are worth "30000"
    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM

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
      | trader  | id         | type | volume | price | resulting trades |
      | trader1 |  ETH/DEC19 | sell |     1  |  1000 |                0 |
      | trader2 |  ETH/DEC19 | buy  |     1  |  1000 |                1 |
    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin | general |
      | trader1 | ETH   | ETH/DEC19 |    120 |       1 |
      | trader2 | ETH   | ETH/DEC19 |    132 |    9868 |

    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
    Then traders place following orders:
      | trader  | id         | type | volume | price | resulting trades |
      | trader2 |  ETH/DEC19 | buy |     1  |  6000 |                0 |
    Then I expect the trader to have a margin:
      | trader  | asset |        id | margin | general |
      | trader2 | ETH   | ETH/DEC19 |    264 |    9736 |

    Then traders place following orders:
      | trader  | id        | type | volume | price | resulting trades |
      | trader3 | ETH/DEC19 | sell |      1 |  5000 |                1 |
    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin | general |
      | trader1 | ETH   | ETH/DEC19 |      0 |       0 |
      | trader2 | ETH   | ETH/DEC19 |   1320 |    8800 |
      | trader3 | ETH   | ETH/DEC19 |    120 |    9880 |
   And All balances cumulated are worth "20120"

    # Then the following transfers happend:
    #   | from    | to     | fromType | toType     | id        | amount | asset |
    #   | trader1 | market | MARGIN   | SETTLEMENT | ETH/DEC19 |    240 | ETH   |
    # And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
