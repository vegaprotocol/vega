Feature: Test mark to market settlement with insurance pool

  Background:
    Given the insurance pool initial balance for the markets is "10000":
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 | ETH        | ETH   | simple     | 0.11      | 0.1       | 0              | 0               | 0     | 1.4            | 1.2            | 1.1           | 1                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And the following network parameters are set:
      | market.auction.minimumDuration |
      | 1                              |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: If settlement amount > trader’s margin account balance + trader’s general account balance for the asset, the full balance of the trader’s margin account is transferred to the market’s temporary settlement account, the full balance of the trader’s general account for the assets are transferred to the market’s temporary settlement account, the minimum insurance pool account balance for the market & asset, and the remainder, i.e. the difference between the total amount transferred from the trader’s margin + general accounts and the settlement amount, is transferred from the insurance pool account for the market to the temporary settlement account for the market
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount |
      | trader1 | ETH   | 5122   |
      | trader2 | ETH   | 10000  |
      | trader3 | ETH   | 10000  |
      | aux     | ETH   | 10000  |
      | aux2    | ETH   | 10000  |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     | reference |
      | aux     | ETH/DEC19 | buy  | 1      | 999    | 0                | TYPE_LIMIT  | TIF_GTC | ref-1     |
      | aux     | ETH/DEC19 | sell | 1      | 6001   | 0                | TYPE_LIMIT  | TIF_GTC | ref-2     |
      | aux2    | ETH/DEC19 | buy  | 1      | 1000   | 0                | TYPE_LIMIT  | TIF_GTC | ref-3     |
      | aux     | ETH/DEC19 | sell | 1      | 1000   | 0                | TYPE_LIMIT  | TIF_GTC | ref-4     |
    Then the opening auction period for market "ETH/DEC19" ends
    And the trading mode for the market "ETH/DEC19" is "TRADING_MODE_CONTINUOUS"

    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
    Then traders have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 |   5122 | 0       |
      | trader2 | ETH   | ETH/DEC19 |   132  | 9868    |

    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader2 | ETH/DEC19 | buy  | 1      | 6000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then traders have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader2 | ETH   | ETH/DEC19 |  265   |    9735 |

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC19 | sell | 1      | 5000  | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then traders have the following account balances:
      | trader  | asset | market id | margin  | general |
      | trader1 | ETH   | ETH/DEC19 |    0    |    0    |
      | trader2 | ETH   | ETH/DEC19 |    13586|    1414 |
      | trader3 | ETH   | ETH/DEC19 |    721  |    9279 |

    And Cumulated balance for all accounts is worth "55122"
    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
    And the insurance pool balance is "10122" for the market "ETH/DEC19"
