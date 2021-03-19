Feature: Regression test for issue 596

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the execution engine have these markets:
      | name      | quote name | asset |  risk model | lamd/long | tau/short              | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlement price | auction duration |  maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC19 |  BTC       | BTC   |  forward    | 0.001     | 0.00011407711613050422 | 0              | 0.016           | 2.0   | 1.4            | 1.2            | 1.1           | 42               | 1                |  0         | 0                  | 0             | 0                  |                |             |                 | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Traded out position but monies left in margin account
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount  |
      | edd     | BTC   | 10000   |
      | barney  | BTC   | 10000   |
      | chris   | BTC   | 10000   |
      | tamlyn  | BTC   | 10000   |
      | trader1 | BTC   | 1000000 |
      | trader2 | BTC   | 1000000 |
      | aux     | BTC   | 1000    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     | 
      | aux     | ETH/DEC19 | buy  | 1      | 95    | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC19 | sell | 1      | 105   | 0                | TYPE_LIMIT  | TIF_GTC | 

    # Trigger an auction to set the mark price
    Then traders place following orders with references:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "100"

    And the market trading mode for the market "ETH/DEC19" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     |
      | edd    | ETH/DEC19 | sell | 20     | 101   | 0                | TYPE_LIMIT | TIF_GTC |
      | edd    | ETH/DEC19 | sell | 20     | 102   | 0                | TYPE_LIMIT | TIF_GTC |
      | edd    | ETH/DEC19 | sell | 10     | 103   | 0                | TYPE_LIMIT | TIF_GTC |
      | edd    | ETH/DEC19 | sell | 15     | 104   | 0                | TYPE_LIMIT | TIF_GTC |
      | edd    | ETH/DEC19 | sell | 30     | 105   | 0                | TYPE_LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  | 20     | 99    | 0                | TYPE_LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  | 12     | 98    | 0                | TYPE_LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  | 14     | 97    | 0                | TYPE_LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  | 20     | 96    | 0                | TYPE_LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  | 5      | 95    | 0                | TYPE_LIMIT | TIF_GTC |
    Then traders have the following account balances:
      | trader | asset | market id | margin | general |
      | edd    | BTC   | ETH/DEC19 | 848    | 9152    |
      | barney | BTC   | ETH/DEC19 | 594    | 9406    |
    Then traders place following orders:
      | trader | market id | side | volume | price | resulting trades | type        | tif     |
      | chris  | ETH/DEC19 | buy  | 50     | 0     | 3                | TYPE_MARKET | TIF_IOC |
    Then traders have the following account balances:
      | trader | asset | market id | margin | general |
      | edd    | BTC   | ETH/DEC19 | 933    | 9007    |
      | chris  | BTC   | ETH/DEC19 | 790    | 9270    |
      | barney | BTC   | ETH/DEC19 | 594    | 9406    |
    And All balances cumulated are worth "2041000"
# then chris is trading out
    Then traders place following orders:
      | trader | market id | side | volume | price | resulting trades | type        | tif     |
      | chris  | ETH/DEC19 | sell | 50     | 0     | 4                | TYPE_MARKET | TIF_IOC |
    Then traders have the following account balances:
      | trader | asset | market id | margin | general |
      | edd    | BTC   | ETH/DEC19 | 1283   | 9007    |
      | chris  | BTC   | ETH/DEC19 | 0      | 9808    |
      | barney | BTC   | ETH/DEC19 | 630    | 9272    |
    And All balances cumulated are worth "2041000"
# placing new orders to trade out
    Then traders place following orders:
      | trader | market id | side | volume | price | resulting trades | type        | tif     |
      | chris  | ETH/DEC19 | buy  | 5      | 0     | 1                | TYPE_MARKET | TIF_IOC |
# placing order which get cancelled
    Then traders place following orders with references:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference            |
      | chris  | ETH/DEC19 | buy  | 60     | 1     | 0                | TYPE_LIMIT | TIF_GTC | chris-id-1-to-cancel |
# other traders trade together (tamlyn+barney)
    Then traders place following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     |
      | tamlyn | ETH/DEC19 | sell | 12     | 95    | 1                | TYPE_LIMIT | TIF_GTC |
# cancel order
    Then traders cancel the following orders:
      | trader | reference            |
      | chris  | chris-id-1-to-cancel |
# then chris is trading out
    Then traders place following orders:
      | trader | market id | side | volume | price | resulting trades | type        | tif     |
      | chris  | ETH/DEC19 | sell | 5      | 0     | 2                | TYPE_MARKET | TIF_IOC |
    Then traders have the following account balances:
      | trader | asset | market id | margin | general |
      | chris  | BTC   | ETH/DEC19 | 0      | 9767    |
    And All balances cumulated are worth "2041000"

  Scenario: Traded out position but monies left in margin account if trade which trade out do not update the markprice
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount  |
      | edd     | BTC   | 10000   |
      | barney  | BTC   | 10000   |
      | chris   | BTC   | 10000   |
      | tamlyn  | BTC   | 10000   |
      | trader1 | BTC   | 1000000 |
      | trader2 | BTC   | 1000000 |
      | aux     | BTC   | 1000    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     | 
      | aux     | ETH/DEC19 | buy  | 1      | 95    | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC19 | sell | 1      | 105   | 0                | TYPE_LIMIT  | TIF_GTC | 

    # Trigger an auction to set the mark price
    Then traders place following orders with references:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader1-2 |
      | trader2 | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GFA | trader2-2 |
    Then the opening auction period for market "ETH/DEC19" ends
    And the mark price for the market "ETH/DEC19" is "100"

    And the market trading mode for the market "ETH/DEC19" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     |
      | edd    | ETH/DEC19 | sell | 20     | 101   | 0                | TYPE_LIMIT | TIF_GTC |
      | edd    | ETH/DEC19 | sell | 20     | 102   | 0                | TYPE_LIMIT | TIF_GTC |
      | edd    | ETH/DEC19 | sell | 10     | 103   | 0                | TYPE_LIMIT | TIF_GTC |
      | edd    | ETH/DEC19 | sell | 15     | 104   | 0                | TYPE_LIMIT | TIF_GTC |
      | edd    | ETH/DEC19 | sell | 30     | 105   | 0                | TYPE_LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  | 20     | 99    | 0                | TYPE_LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  | 12     | 98    | 0                | TYPE_LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  | 14     | 97    | 0                | TYPE_LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  | 20     | 96    | 0                | TYPE_LIMIT | TIF_GTC |
      | barney | ETH/DEC19 | buy  | 5      | 95    | 0                | TYPE_LIMIT | TIF_GTC |
    Then traders have the following account balances:
      | trader | asset | market id | margin | general |
      | edd    | BTC   | ETH/DEC19 | 848    | 9152    |
      | barney | BTC   | ETH/DEC19 | 594    | 9406    |
    Then traders place following orders:
      | trader | market id | side | volume | price | resulting trades | type        | tif     |
      | chris  | ETH/DEC19 | buy  | 50     | 0     | 3                | TYPE_MARKET | TIF_IOC |
    Then traders have the following account balances:
      | trader | asset | market id | margin | general |
      | edd    | BTC   | ETH/DEC19 | 933    | 9007    |
      | chris  | BTC   | ETH/DEC19 | 790    | 9270    |
      | barney | BTC   | ETH/DEC19 | 594    | 9406    |
    And All balances cumulated are worth "2041000"
# then chris is trading out
    Then traders place following orders:
      | trader | market id | side | volume | price | resulting trades | type        | tif     |
      | chris  | ETH/DEC19 | sell | 50     | 0     | 4                | TYPE_MARKET | TIF_IOC |
    Then traders have the following account balances:
      | trader | asset | market id | margin | general |
      | edd    | BTC   | ETH/DEC19 | 1283   | 9007    |
      | chris  | BTC   | ETH/DEC19 | 0      | 9808    |
      | barney | BTC   | ETH/DEC19 | 630    | 9272    |
    And All balances cumulated are worth "2041000"
# placing new orders to trade out
    Then traders place following orders:
      | trader | market id | side | volume | price | resulting trades | type        | tif     |
      | chris  | ETH/DEC19 | buy  | 5      | 0     | 1                | TYPE_MARKET | TIF_IOC |
# placing order which get cancelled
    Then traders place following orders with references:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference            |
      | chris  | ETH/DEC19 | buy  | 60     | 1     | 0                | TYPE_LIMIT | TIF_GTC | chris-id-1-to-cancel |
# other traders trade together (tamlyn+barney)
    Then traders place following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     |
      | tamlyn | ETH/DEC19 | sell | 3      | 95    | 1                | TYPE_LIMIT | TIF_GTC |
# cancel order
    Then traders cancel the following orders:
      | trader | reference            |
      | chris  | chris-id-1-to-cancel |
# then chris is trading out
    Then traders place following orders:
      | trader | market id | side | volume | price | resulting trades | type        | tif     |
      | chris  | ETH/DEC19 | sell | 5      | 0     | 1                | TYPE_MARKET | TIF_IOC |
    Then traders have the following account balances:
      | trader | asset | market id | margin | general |
      | chris  | BTC   | ETH/DEC19 | 0      | 9768    |
    And All balances cumulated are worth "2041000"
