Feature: Set up a market, with an opening auction, then uncross the book


  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/long | tau/short | mu | r  | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode | makerFee | infrastructureFee | liquidityFee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | Prob of trading |
      | ETH/DEC20 | ETH      | ETH       | ETH   | 100       | simple     | 0.1       | 0.1       | -1 | -1 | -1    | 1.4            | 1.2            | 1.1           | 100             | 1           | continuous   |     0.04 |              0.01 |          0.3 |                 0  |                |             |                 | 0.1             |

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

    # place orders and generate trades
    Then traders place following orders with references:
      | trader  | id        | type | volume | price    | resulting trades | type        | tif     | reference |
      | trader2 | ETH/DEC20 | buy  | 1      |  9500000 | 0                | TYPE_LIMIT  | TIF_GTC | t1-b-2    |
      | trader1 | ETH/DEC20 | sell | 1      | 10500000 | 0                | TYPE_LIMIT  | TIF_GTC | t2-s-2    |
      | trader1 | ETH/DEC20 | buy  | 1      | 10000000 | 0                | TYPE_LIMIT  | TIF_GFA | t1-b-3    |
      | trader2 | ETH/DEC20 | sell | 1      | 10000000 | 0                | TYPE_LIMIT  | TIF_GFA | t2-s-3    |

    Then the opening auction period for market "ETH/DEC20" ends
    ## We're seeing these events twice for some reason
    And executed trades:
      | buyer   | price    | size | seller  |
      | trader1 | 10000000 | 1    | trader2 |
    And the mark price for the market "ETH/DEC20" is "10000000"

    Then traders place following orders with references:
      | trader  | id        | type | volume | price    | resulting trades | type        | tif     | reference |
      | trader1 | ETH/DEC20 | buy  | 1      | 10000000 | 0                | TYPE_LIMIT  | TIF_GTC | post-oa-1 |
      | trader3 | ETH/DEC20 | sell | 1      | 10000000 | 1                | TYPE_LIMIT  | TIF_GTC | post-oa-2 |

    Then I expect the trader to have a margin:
      | trader  | asset | id        | margin  | general   |
      # Expected (as per CSV):
      # | trader3 | ETH   | ETH/DEC20 | 1724511 | 995225489 |
      # Without fees (current fees are best guess):
      # | trader3 | ETH   | ETH/DEC20 | 1800000 | 998200000 |
      # With best-guess fees:
      | trader3 | ETH   | ETH/DEC20 | 1800000 | 994700000 |
    And dump transfers
    # And the following transfers happened:
    Then the following transfers happened:
      | from    | to      | fromType                | toType                           | id        | amount  | asset |
      | trader3 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC20 |  400000 | ETH   |
      | trader3 |         | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           |  100000 | ETH   |
      | trader3 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC20 | 3000000 | ETH   |
      | market  | trader1 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC20 |  400000 | ETH   |
