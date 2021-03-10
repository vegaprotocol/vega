Feature: Set up a market, with an opening auction, then uncross the book. Make sure opening auction can end if we have remaingin volume in the uncrossing range


  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long              | tau/short              | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlement price | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC20 | ETH        | ETH   | simple     | 0.08628781058136630000 | 0.09370922348428490000 | -1             | -1              | -1    | 1.4            | 1.2            | 1.1           | 100              | 1                | 0.004     | 0.001              | 0.3           | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 100   |

  Scenario: set up 2 traders with balance
    # setup accounts
    Given the following traders:
      | name    | amount     |
      | trader1 | 1000000000 |
      | trader2 | 1000000000 |
      | trader3 | 1000000000 |
    Then I Expect the traders to have new general account:
      | name    | asset |
      | trader1 | ETH   |
      | trader2 | ETH   |
      | trader3 | ETH   |

    # place orders and generate trades - slippage 100
    Then traders place following orders with references:
      | trader  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | trader2 | ETH/DEC20 | buy  | 1      | 9500000  | 0                | TYPE_LIMIT | TIF_GTC | t2-b-1    |
      | trader1 | ETH/DEC20 | buy  | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-1    |
      | trader2 | ETH/DEC20 | sell | 2      | 10000000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |

    Then the opening auction period for market "ETH/DEC20" ends

    And executed trades:
      | buyer   | price    | size | seller  |
      | trader1 | 10000000 | 1    | trader2 |
    And the mark price for the market "ETH/DEC20" is "10000000"
