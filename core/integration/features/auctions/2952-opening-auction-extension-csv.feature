Feature: Set up a market, with an opening auction, then uncross the book

  Background:
    Given the simple risk model named "my-simple-risk-model":
      | long                   | short                  | max move up | min move down | probability of trading |
      | 0.08628781058136630000 | 0.09370922348428490000 | -1          | -1            | 0.2                    |
    And the fees configuration named "my-fees-config":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the markets:
      | id        | quote name | asset | risk model           | margin calculator         | auction duration | fees           | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC20 | ETH        | ETH   | my-simple-risk-model | default-margin-calculator | 1                | my-fees-config | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: set up 2 parties with balance
    Given the parties deposit on asset's general account the following amount:
      | party     | asset | amount       |
      | party1    | ETH   | 1000000000   |
      | party2    | ETH   | 1000000000   |
      | party3    | ETH   | 1000000000   |
      | auxiliary | ETH   | 100000000000 |
      | party-lp  | ETH   | 100000000000 |
    And the parties submit the following liquidity provision:
      | id  | party    | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party-lp | ETH/DEC20 | 30000000          | 0.3 | buy  | BID              | 50         | 10     | submission |
      | lp1 | party-lp | ETH/DEC20 | 30000000          | 0.3 | sell | ASK              | 50         | 10     | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    And the parties place the following orders:
      | party     | market id | side | volume | price     | resulting trades | type       | tif     |
      | auxiliary | ETH/DEC20 | buy  | 1      | 1         | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary | ETH/DEC20 | sell | 1      | 100000000 | 0                | TYPE_LIMIT | TIF_GTC |

    # place orders and generate trades - slippage 100
    When the parties place the following orders:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 10500000 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-1    |
      | party2 | ETH/DEC20 | buy  | 1      | 9500000  | 0                | TYPE_LIMIT | TIF_GTC | t2-b-1    |
      | party1 | ETH/DEC20 | buy  | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-1    |
      | party2 | ETH/DEC20 | sell | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GFA | t2-s-1    |
    And the opening auction period ends for market "ETH/DEC20"
    Then the following trades should be executed:
      | buyer  | price    | size | seller |
      | party1 | 10000000 | 1    | party2 |
    And the mark price should be "10000000" for the market "ETH/DEC20"

    When the parties place the following orders "1" blocks apart:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | buy  | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GTC | post-oa-1 |
      | party3 | ETH/DEC20 | sell | 1      | 10000000 | 1                | TYPE_LIMIT | TIF_GTC | post-oa-2 |
    Then the following trades should be executed:
      | buyer  | price    | size | seller |
      | party1 | 10000000 | 1    | party3 |
    And the parties should have the following account balances:
      | party  | asset | market id | margin  | general   |
      | party3 | ETH   | ETH/DEC20 | 1724511 | 995225489 |
    And the following transfers should happen:
      | from   | to     | from account            | to account                       | market id | amount  | asset |
      | party3 | market | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC20 | 40000   | ETH   |
      | party3 |        | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 10000   | ETH   |
      | party3 | market | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC20 | 3000000 | ETH   |
      | market | party1 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC20 | 40000   | ETH   |

    # Amend orders to set slippage to 120
    When the parties amend the following orders:
      | party  | reference | price    | size delta | tif     |
      | party1 | t1-s-1    | 12500000 | 0          | TIF_GTC |
      | party2 | t2-b-1    | 10500000 | 0          | TIF_GTC |

    When the parties place the following orders "1" blocks apart:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 12000000 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-2    |
      | party2 | ETH/DEC20 | buy  | 1      | 12000000 | 1                | TYPE_LIMIT | TIF_GTC | t2-b-3    |
    Then the following transfers should happen:
      | from   | to     | from account         | to account              | market id | amount  | asset |
      | party3 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC20 | 275489  | ETH   |
      | party3 | party3 | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_MARGIN     | ETH/DEC20 | 1949413 | ETH   |
    And the parties should have the following account balances:
      | party  | asset | market id | margin  | general   |
      | party3 | ETH   | ETH/DEC20 | 1949413 | 993000587 |
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search  | initial | release |
      | party3 | ETH/DEC20 | 1624511     | 1786962 | 1949413 | 2274315 |

    #maitenance_margin_party3: 1*(12500000-12000000)+1*12000000*0.09370922348428490000=1624511

    # MTM loss + margin low

    # Amend orders to set slippage to 140
    When the parties amend the following orders:
      | party  | reference | price    | size delta | tif     |
      | party1 | t1-s-1    | 14500000 | 0          | TIF_GTC |
      | party2 | t2-b-1    | 13500000 | 0          | TIF_GTC |
    #Then debug detailed orderbook volumes for market "ETH/DEC20"

    When the parties place the following orders "1" blocks apart:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 14000000 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-3    |
      | party2 | ETH/DEC20 | buy  | 1      | 14000000 | 1                | TYPE_LIMIT | TIF_GTC | t2-b-4    |
    # Check MTM Loss transfer happened
    Then the following transfers should happen:
      | from   | to     | from account        | to account              | market id | amount  | asset |
      | party3 | market | ACCOUNT_TYPE_MARGIN | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC20 | 1949413 | ETH   |
    And the parties should have the following account balances:
      | party  | asset | market id | margin  | general   |
      | party3 | ETH   | ETH/DEC20 | 2174316 | 990775684 |


    # Amend orders to set slippage to 160
    When the parties amend the following orders:
      | party  | reference | price    | size delta | tif     |
      | party1 | t1-s-1    | 16500000 | 0          | TIF_GTC |
      | party2 | t2-b-1    | 15500000 | 0          | TIF_GTC |
    Then the network moves ahead "1" blocks

    When the parties place the following orders "1" blocks apart:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 16000000 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-4    |
      | party2 | ETH/DEC20 | buy  | 1      | 16000000 | 1                | TYPE_LIMIT | TIF_GTC | t2-b-5    |
    # Check MTM Loss transfer happened
    Then the following transfers should happen:
      | from   | to     | from account         | to account              | market id | amount  | asset |
      | party3 | market | ACCOUNT_TYPE_MARGIN  | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC20 | 2000000 | ETH   |
      | party3 | party3 | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_MARGIN     | ETH/DEC20 | 2224901 | ETH   |
    And the parties should have the following account balances:
      | party  | asset | market id | margin  | general   |
      | party3 | ETH   | ETH/DEC20 | 2399217 | 988550783 |

    # Amend orders to set slippage to 180
    When the parties amend the following orders:
      | party  | reference | price    | size delta | tif     |
      | party1 | t1-s-1    | 18500000 | 0          | TIF_GTC |
      | party2 | t2-b-1    | 17500000 | 0          | TIF_GTC |
    Then the network moves ahead "1" blocks

    When the parties place the following orders "1" blocks apart:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 18000000 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-3    |
      | party2 | ETH/DEC20 | buy  | 1      | 18000000 | 1                | TYPE_LIMIT | TIF_GTC | t2-b-6    |
    # Check MTM Loss transfer happened
    Then the following transfers should happen:
      | from   | to     | from account         | to account              | market id | amount  | asset |
      | party3 | market | ACCOUNT_TYPE_MARGIN  | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC20 | 2000000 | ETH   |
      | party3 | party3 | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_MARGIN     | ETH/DEC20 | 2224903 | ETH   |
    And the parties should have the following account balances:
      | party  | asset | market id | margin  | general   |
      | party3 | ETH   | ETH/DEC20 | 2624120 | 986325880 |

    # Amend orders to set slippage to 140
    # Amending prices down, so amend buy order first, so it doesn't uncross with the lowered sell order
    When the parties amend the following orders:
      | party  | reference | price    | size delta | tif     |
      | party2 | t2-b-1    | 13500000 | 0          | TIF_GTC |
      | party1 | t1-s-1    | 14500000 | 0          | TIF_GTC |

    When the parties place the following orders "1" blocks apart:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 14000000 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-4    |
      | party2 | ETH/DEC20 | buy  | 1      | 14000000 | 1                | TYPE_LIMIT | TIF_GTC | t2-b-7    |
    # Check MTM Loss transfer happened
    #  4449804
    Then the following transfers should happen:
      | from   | to     | from account            | to account           | market id | amount  | asset |
      | market | party3 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN  | ETH/DEC20 | 4000000 | ETH   |
      | party3 | party3 | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_GENERAL | ETH/DEC20 | 4449804 | ETH   |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general   |
      | party3 | ETH   | ETH/DEC20 | 2174316 | 990775684 |

    # Amend orders to set slippage to 120
    # Amending prices down, so amend buy order first, so it doesn't uncross with the lowered sell order
    When the parties amend the following orders:
      | party  | reference | price    | size delta | tif     |
      | party2 | t2-b-1    | 11500000 | 0          | TIF_GTC |
      | party1 | t1-s-1    | 12500000 | 0          | TIF_GTC |
    And the parties place the following orders "1" blocks apart:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 12000000 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-5    |
      | party2 | ETH/DEC20 | buy  | 1      | 12000000 | 1                | TYPE_LIMIT | TIF_GTC | t2-b-8    |
    # Check MTM Loss transfer happened
    Then the following transfers should happen:
      | from   | to     | from account            | to account           | market id | amount  | asset |
      | market | party3 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN  | ETH/DEC20 | 2000000 | ETH   |
      | party3 | party3 | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_GENERAL | ETH/DEC20 | 2224903 | ETH   |
    And the parties should have the following account balances:
      | party  | asset | market id | margin  | general   |
      | party3 | ETH   | ETH/DEC20 | 1949413 | 993000587 |


    # Amend orders to set slippage to 110
    # Amending prices down, so amend buy order first, so it doesn't uncross with the lowered sell order
    When the parties amend the following orders:
      | party  | reference | price    | size delta | tif     |
      | party2 | t2-b-1    | 10500000 | 0          | TIF_GTC |
      | party1 | t1-s-1    | 11500000 | 0          | TIF_GTC |
    And the parties place the following orders "1" blocks apart:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 11000000 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-6    |
      | party2 | ETH/DEC20 | buy  | 1      | 11000000 | 1                | TYPE_LIMIT | TIF_GTC | t2-b-9    |
    # Check MTM Loss transfer happened
    Then the following transfers should happen:
      | from   | to     | from account            | to account           | market id | amount  | asset |
      | market | party3 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN  | ETH/DEC20 | 1000000 | ETH   |
      | party3 | party3 | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_GENERAL | ETH/DEC20 | 1112451 | ETH   |
    And the parties should have the following account balances:
      | party  | asset | market id | margin  | general   |
      | party3 | ETH   | ETH/DEC20 | 1836962 | 994113038 |

    # Amend orders to set slippage to 100
    # Amending prices down, so amend buy order first, so it doesn't uncross with the lowered sell order
    When the parties amend the following orders:
      | party  | reference | price    | size delta | tif     |
      | party2 | t2-b-1    | 9500000  | 0          | TIF_GTC |
      | party1 | t1-s-1    | 10500000 | 0          | TIF_GTC |
    And the parties place the following orders "1" blocks apart:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-7    |
      | party2 | ETH/DEC20 | buy  | 1      | 10000000 | 1                | TYPE_LIMIT | TIF_GTC | t2-b-10   |
    # Check MTM Loss transfer happened
    Then the following transfers should happen:
      | from   | to     | from account            | to account           | market id | amount  | asset |
      | market | party3 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN  | ETH/DEC20 | 1000000 | ETH   |
      | party3 | party3 | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_GENERAL | ETH/DEC20 | 1112451 | ETH   |
    And the parties should have the following account balances:
      | party  | asset | market id | margin  | general   |
      | party3 | ETH   | ETH/DEC20 | 1724511 | 995225489 |

    When the parties place the following orders "1" blocks apart:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GTC | post-oa-3 |
      | party3 | ETH/DEC20 | buy  | 1      | 10000000 | 1                | TYPE_LIMIT | TIF_GTC | post-oa-4 |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general   |
      | party3 | ETH   | ETH/DEC20 | 0      | 993900000 |
    And the following transfers should happen:
      | from   | to     | from account            | to account                       | market id | amount  | asset |
      | party3 | market | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC20 | 40000   | ETH   |
      | party3 |        | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 10000   | ETH   |
      | party3 | market | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC20 | 3000000 | ETH   |
      | market | party1 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC20 | 40000   | ETH   |
