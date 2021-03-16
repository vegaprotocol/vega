Feature: test bugfix 614 for margin calculations

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlement price | auction duration |  maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 |  ETH        | ETH   |  simple     | 0.2       | 0.1       | 0              | 0               | 0     | 5              | 4              | 3.2           | 100              | 1                |  0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 100   |

  Scenario: CASE-1: Trader submits long order that will trade - new formula & high exit price
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount  |
      | chris   | ETH   | 10000   |
      | edd     | ETH   | 10000   |
      | barney  | ETH   | 10000   |
      | rebecca | ETH   | 10000   |
      | tamlyn  | ETH   | 10000   |
      | trader1 | ETH   | 1000000 |
      | trader2 | ETH   | 1000000 |

    # Trigger an auction to set the mark price
    Then traders place following orders with references:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | trader1-1 |
      | trader2 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | trader2-1 |
      | trader1 | ETH/DEC19 | buy  | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 94    | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "94"
    Then traders cancel the following orders:
      | trader  | reference |
      | trader1 | trader1-1 |
      | trader2 | trader2-1 |

    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | chris   | ETH/DEC19 | sell | 100    | 250   | 0                | TYPE_LIMIT | TIF_GTC |
      | edd     | ETH/DEC19 | sell | 11     | 140   | 0                | TYPE_LIMIT | TIF_GTC |
      | barney  | ETH/DEC19 | sell | 2      | 112   | 0                | TYPE_LIMIT | TIF_GTC |
      | barney  | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | edd     | ETH/DEC19 | buy  | 3      | 96    | 0                | TYPE_LIMIT | TIF_GTC |
      | chris   | ETH/DEC19 | buy  | 15     | 90    | 0                | TYPE_LIMIT | TIF_GTC |
      | rebecca | ETH/DEC19 | buy  | 50     | 87    | 0                | TYPE_LIMIT | TIF_GTC |
      # this is now the actual trader that we are testing
    Then traders place following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     |
      | tamlyn | ETH/DEC19 | buy  | 13     | 150   | 2                | TYPE_LIMIT | TIF_GTC |
    Then the margins levels for the traders are:
      | trader | market id | maintenance | search | initial | release |
      | tamlyn | ETH/DEC19 | 988         | 3161   | 3952    | 4940    |
    Then traders have the following account balances:
      | trader  | asset | market id | margin | general |
      | tamlyn  | ETH   | ETH/DEC19 | 3952   | 6104    |
      | chris   | ETH   | ETH/DEC19 | 3760   | 6240    |
      | edd     | ETH   | ETH/DEC19 | 5456   | 4544    |
      | barney  | ETH   | ETH/DEC19 | 992    | 8952    |
      | rebecca | ETH   | ETH/DEC19 | 3760   | 6240    |
    And Cumulated balance for all accounts is worth "2050000"
