Feature: Set up a market, with an opening auction, then uncross the book. Make sure opening auction can end if we have remaingin volume in the uncrossing range


  Background:

    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 5              |
    And the simple risk model named "my-simple-risk-model":
      | long                   | short                  | max move up | min move down | probability of trading |
      | 0.08628781058136630000 | 0.09370922348428490000 | -1          | -1            | 0.2                    |
    And the fees configuration named "my-fees-config":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the markets:
      | id        | quote name | asset | risk model           | margin calculator         | auction duration | fees           | price monitoring | data source config     | decimal places | linear slippage factor | quadratic slippage factor |
      | ETH/DEC20 | ETH        | ETH   | my-simple-risk-model | default-margin-calculator | 1                | my-fees-config | default-none     | default-eth-for-future | 2              | 1e6                    | 1e6                       |

  Scenario: set up 2 parties with balance
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount        |
      | party1 | ETH   | 1000000000000 |
      | party2 | ETH   | 1000000000000 |
      | party3 | ETH   | 1000000000000 |
      | lpprov | ETH   | 1000000000000 |

    # place orders and generate trades - slippage 100
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 937000000         | 0.1 | buy  | MID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC20 | 937000000         | 0.1 | sell | MID              | 50         | 100    | submission |
    And the parties place the following orders:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC20 | buy  | 1      | 950000  | 0                | TYPE_LIMIT | TIF_GTC | t2-b-1    |
      | party1 | ETH/DEC20 | buy  | 1      | 1000000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-1    |
      | party2 | ETH/DEC20 | sell | 2      | 1000000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |
    Then the market data for the market "ETH/DEC20" should be:
      | trading mode                 | supplied stake | target stake |
      | TRADING_MODE_OPENING_AUCTION | 937000000      | 937000000    |

    When the opening auction period ends for market "ETH/DEC20"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the following trades should be executed:
      | buyer  | price   | size | seller |
      | party1 | 1000000 | 1    | party2 |
    And the mark price should be "1000000" for the market "ETH/DEC20"

