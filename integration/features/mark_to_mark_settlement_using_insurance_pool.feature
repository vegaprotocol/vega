Feature: Test mark to market settlement with insurance pool

  Background:
    Given the insurance pool initial balance for the markets is "10000":
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 | ETH        | ETH   | simple     | 0.11      | 0.1       | 0              | 0               | 0     | 1.4            | 1.2            | 1.1           | 0                | 0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: If settlement amount > trader’s margin account balance + trader’s general account balance for the asset, the full balance of the trader’s margin account is transferred to the market’s temporary settlement account, the full balance of the trader’s general account for the assets are transferred to the market’s temporary settlement account, the minimum insurance pool account balance for the market & asset, and the remainder, i.e. the difference between the total amount transferred from the trader’s margin + general accounts and the settlement amount, is transferred from the insurance pool account for the market to the temporary settlement account for the market
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount |
      | trader1 | ETH   | 121    |
      | trader2 | ETH   | 10000  |
      | trader3 | ETH   | 10000  |
      | aux     | ETH   | 10000  |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     | 
      | aux     | ETH/DEC19 | buy  | 1      | 999   | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC19 | sell | 1      | 6001  | 0                | TYPE_LIMIT  | TIF_GTC | 

    And the market trading mode for the market "ETH/DEC19" is "TRADING_MODE_CONTINUOUS"   

    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin | general |
      | trader1 | ETH   | ETH/DEC19 |   5122 |       0 |
      | trader2 | ETH   | ETH/DEC19 |    133 |    9867 |

    And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader2 | ETH/DEC19 | buy  | 1      | 6000  | 0                | TYPE_LIMIT | TIF_GTC |
    Then I expect the trader to have a margin:
      | trader  | asset |        id | margin | general |
      | trader2 | ETH   | ETH/DEC19 |    265 |    9735 |

    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader3 | ETH/DEC19 | sell | 1      | 5000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin | general |
      | trader1 | ETH   | ETH/DEC19 |      0 |       0 |
      | trader2 | ETH   | ETH/DEC19 |  13586 |    1414 |
      | trader3 | ETH   | ETH/DEC19 |    721 |    9279 |

   And All balances cumulated are worth "45122"
   And the settlement account balance is "0" for the market "ETH/DEC19" before MTM
   And the insurance pool balance is "10122" for the market "ETH/DEC19"
