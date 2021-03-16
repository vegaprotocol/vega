Feature: Test mark to market settlement

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlement price | auction duration |  maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 |  ETH        | ETH   |  simple     | 0.11      | 0.1       | 0              | 0               | 0     | 1.4            | 1.2            | 1.1           | 42               | 0                |  0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: If settlement amount <= the trader’s margin account balance entire settlement amount is transferred from trader’s margin account to the market’s temporary settlement account
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount |
      | trader1 | ETH   | 10000  |
      | trader2 | ETH   | 10000  |
      | trader3 | ETH   | 10000  |
    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then traders have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 | 120    | 9880    |
      | trader2 | ETH   | ETH/DEC19 | 132    | 9868    |

    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | sell | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
    Then traders have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 | 240    | 9760    |

    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3 | ETH/DEC19 | buy  | 1      | 2000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then traders have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 | 480    | 8520    |
      | trader3 | ETH   | ETH/DEC19 | 264    | 9736    |
      | trader2 | ETH   | ETH/DEC19 | 264    | 10736   |
    Then the following transfers happened:
      | from | to | from account | to account | market id | amount | asset |
      | trader1 | market | ACCOUNT_TYPE_MARGIN | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 240    | ETH   |
    And Cumulated balance for all accounts is worth "30000"
    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM

  Scenario: If settlement amount > trader’s margin account balance  and <= trader's margin account balance + general account balance for the asset, he full balance of the trader’s margin account is transferred to the market’s temporary settlement account the remainder, i.e. difference between the amount transferred from the margin account and the settlement amount, is transferred from the trader’s general account for the asset to the market’s temporary settlement account
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount |
      | trader1 | ETH   | 10000  |
      | trader2 | ETH   | 10000  |
      | trader3 | ETH   | 10000  |
    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then traders have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 | 120    | 9880    |
      | trader2 | ETH   | ETH/DEC19 | 132    | 9868    |

    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | sell | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
    Then traders have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 | 240    | 9760    |

    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3 | ETH/DEC19 | buy  | 1      | 5000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then traders have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 | 1200   | 4800    |
      | trader3 | ETH   | ETH/DEC19 | 660    | 9340    |
      | trader2 | ETH   | ETH/DEC19 | 660    | 13340   |
    Then the following transfers happened:
      | from | to | from account | to account | market id | amount | asset |
      | trader1 | market | ACCOUNT_TYPE_MARGIN | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 240    | ETH   |
    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM

# this part show that funds are moved from margin account general account for trader 3 as he does not have
# enough funds in the margin account
    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3 | ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC19 | sell | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC |
    Then traders have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 | 18     | 15882   |
      | trader3 | ETH   | ETH/DEC19 | 13     | 5037    |
      | trader2 | ETH   | ETH/DEC19 | 7      | 9043    |
    Then the following transfers happened:
      | from | to | from account | to account | market id | amount | asset |
      | trader3 | trader3 | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 660    | ETH   |
      | trader3 | market  | ACCOUNT_TYPE_MARGIN  | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 1320   | ETH   |
    And Cumulated balance for all accounts is worth "30000"

  Scenario: If the mark price hasn’t changed, A trader with no change in open position size has no transfers in or out of their margin account, A trader with no change in open volume
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount |
      | trader1 | ETH   | 10000  |
      | trader2 | ETH   | 10000  |
      | trader3 | ETH   | 10000  |
    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then traders have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 | 120    | 9880    |
      | trader2 | ETH   | ETH/DEC19 | 132    | 9868    |
    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    Then traders have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 | 240    | 9760    |

    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

# here we expect trader 2 to still have the same margin as the previous trade did not change the markprice
    Then traders have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 | 240    | 9760    |
      | trader3 | ETH   | ETH/DEC19 | 132    | 9868    |
      | trader2 | ETH   | ETH/DEC19 | 132    | 9868    |
    And Cumulated balance for all accounts is worth "30000"
    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
