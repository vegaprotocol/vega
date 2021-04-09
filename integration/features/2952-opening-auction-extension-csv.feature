Feature: Set up a market, with an opening auction, then uncross the book

  Background:

    And the simple risk model named "my-simple-risk-model":
      | long                   | short                  | max move up | min move down | probability of trading |
      | 0.08628781058136630000 | 0.09370922348428490000 | -1          | -1            | 0.1                    |
    And the fees configuration named "my-fees-config":
      | maker fee | infrastructure fee | liquidity fee |
      | 0.004     | 0.001              | 0.3           |
    And the markets:
      | id        | quote name | asset | risk model           | margin calculator         | auction duration | fees           | price monitoring | oracle config          |
      | ETH/DEC20 | ETH        | ETH   | my-simple-risk-model | default-margin-calculator | 1                | my-fees-config | default-none     | default-eth-for-future |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 100   |

  Scenario: set up 2 traders with balance
    Given the traders deposit on asset's general account the following amount:
      | trader    | asset | amount       |
      | trader1   | ETH   | 1000000000   |
      | trader2   | ETH   | 1000000000   |
      | trader3   | ETH   | 1000000000   |
      | auxiliary | ETH   | 100000000000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader     | market id | side | volume | price      | resulting trades | type        | tif     |
      | auxiliary  | ETH/DEC20 | buy  | 1      | 1          | 0                | TYPE_LIMIT  | TIF_GTC |
      | auxiliary  | ETH/DEC20 | sell | 1      | 100000000  | 0                | TYPE_LIMIT  | TIF_GTC |

    # place orders and generate trades - slippage 100
    When the traders place the following orders:
      | trader  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 10500000 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-1    |
      | trader2 | ETH/DEC20 | buy  | 1      | 9500000  | 0                | TYPE_LIMIT | TIF_GTC | t2-b-1    |
      | trader1 | ETH/DEC20 | buy  | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-1    |
      | trader2 | ETH/DEC20 | sell | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GFA | t2-s-1    |
    And the opening auction period ends for market "ETH/DEC20"
    Then the following trades should be executed:
      | buyer   | price    | size | seller  |
      | trader1 | 10000000 | 1    | trader2 |
    And the mark price should be "10000000" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | buy  | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GTC | post-oa-1 |
      | trader3 | ETH/DEC20 | sell | 1      | 10000000 | 1                | TYPE_LIMIT | TIF_GTC | post-oa-2 |
    Then the traders should have the following account balances:
      | trader  | asset | market id | margin  | general   |
      | trader3 | ETH   | ETH/DEC20 | 1724511 | 995225489 |
    And the following transfers should happen:
      | from    | to      | from account            | to account                       | market id | amount  | asset |
      | trader3 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC20 | 40000   | ETH   |
      | trader3 |         | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 10000   | ETH   |
      | trader3 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC20 | 3000000 | ETH   |
      | market  | trader1 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC20 | 40000   | ETH   |
    And clear transfer events

    # Amend orders to set slippage to 120
    When the traders amend the following orders:
      | trader  | reference | price    | size delta | tif     |
      | trader1 | t1-s-1    | 12500000 | 0          | TIF_GTC |
      | trader2 | t2-b-1    | 10500000 | 0          | TIF_GTC |
    Then the following amendments should be accepted:
      | trader  | reference |
      | trader1 | t1-s-1    |
      | trader2 | t2-s-1    |

    When the traders place the following orders:
      | trader  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 12000000 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-2    |
      | trader2 | ETH/DEC20 | buy  | 1      | 12000000 | 1                | TYPE_LIMIT | TIF_GTC | t2-b-3    |
    Then the traders should have the following account balances:
      | trader  | asset | market id | margin  | general   |
      | trader3 | ETH   | ETH/DEC20 | 1949413 | 993000587 |
    # MTM loss + margin low
    And the following transfers should happen:
      | from    | to      | from account         | to account              | market id | amount  | asset |
      | trader3 | market  | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC20 | 275489  | ETH   |
      | trader3 | trader3 | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_MARGIN     | ETH/DEC20 | 1949413 | ETH   |
    And clear transfer events

    # Amend orders to set slippage to 140
    When the traders amend the following orders:
      | trader  | reference | price    | size delta | tif     |
      | trader1 | t1-s-1    | 14500000 | 0          | TIF_GTC |
      | trader2 | t2-b-1    | 13500000 | 0          | TIF_GTC |
    Then the following amendments should be accepted:
      | trader  | reference |
      | trader1 | t1-s-1    |
      | trader2 | t2-s-1    |
    When the traders place the following orders:
      | trader  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 14000000 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-3    |
      | trader2 | ETH/DEC20 | buy  | 1      | 14000000 | 1                | TYPE_LIMIT | TIF_GTC | t2-b-4    |

    Then the traders should have the following account balances:
      | trader  | asset | market id | margin  | general   |
      | trader3 | ETH   | ETH/DEC20 | 2174316 | 990775684 |

    # Check MTM Loss transfer happened
    And the following transfers should happen:
      | from    | to     | from account         | to account              | market id | amount | asset |
      | trader3 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC20 | 50587  | ETH   |
    And clear transfer events

    # Amend orders to set slippage to 160
    When the traders amend the following orders:
      | trader  | reference | price    | size delta | tif     |
      | trader1 | t1-s-1    | 16500000 | 0          | TIF_GTC |
      | trader2 | t2-b-1    | 15500000 | 0          | TIF_GTC |
    Then the following amendments should be accepted:
      | trader  | reference |
      | trader1 | t1-s-1    |
      | trader2 | t2-s-1    |


    When the traders place the following orders:
      | trader  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 16000000 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-4    |
      | trader2 | ETH/DEC20 | buy  | 1      | 16000000 | 1                | TYPE_LIMIT | TIF_GTC | t2-b-5    |
    Then the traders should have the following account balances:
      | trader  | asset | market id | margin  | general   |
      | trader3 | ETH   | ETH/DEC20 | 2399217 | 988550783 |
    # Check MTM Loss transfer happened
    And the following transfers should happen:
      | from    | to      | from account         | to account              | market id | amount  | asset |
      | trader3 | market  | ACCOUNT_TYPE_MARGIN  | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC20 | 2000000 | ETH   |
      | trader3 | trader3 | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_MARGIN     | ETH/DEC20 | 2224901 | ETH   |
    And clear transfer events

    # Amend orders to set slippage to 180
    When the traders amend the following orders:
      | trader  | reference | price    | size delta | tif     |
      | trader1 | t1-s-1    | 18500000 | 0          | TIF_GTC |
      | trader2 | t2-b-1    | 17500000 | 0          | TIF_GTC |
    Then the following amendments should be accepted:
      | trader  | reference |
      | trader1 | t1-s-1    |
      | trader2 | t2-s-1    |

    When the traders place the following orders:
      | trader  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 18000000 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-3    |
      | trader2 | ETH/DEC20 | buy  | 1      | 18000000 | 1                | TYPE_LIMIT | TIF_GTC | t2-b-6    |
    Then the traders should have the following account balances:
      | trader  | asset | market id | margin  | general   |
      | trader3 | ETH   | ETH/DEC20 | 2624120 | 986325880 |
    # Check MTM Loss transfer happened
    And the following transfers should happen:
      | from    | to      | from account         | to account              | market id | amount  | asset |
      | trader3 | market  | ACCOUNT_TYPE_MARGIN  | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC20 | 2000000 | ETH   |
      | trader3 | trader3 | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_MARGIN     | ETH/DEC20 | 2224903 | ETH   |
    And clear transfer events

    # Amend orders to set slippage to 140
    # Amending prices down, so amend buy order first, so it doesn't uncross with the lowered sell order
    When the traders amend the following orders:
      | trader  | reference | price    | size delta | tif     |
      | trader2 | t2-b-1    | 13500000 | 0          | TIF_GTC |
      | trader1 | t1-s-1    | 14500000 | 0          | TIF_GTC |
    Then the following amendments should be accepted:
      | trader  | reference |
      | trader1 | t1-s-1    |
      | trader2 | t2-s-1    |

    When the traders place the following orders:
      | trader  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 14000000 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-4    |
      | trader2 | ETH/DEC20 | buy  | 1      | 14000000 | 1                | TYPE_LIMIT | TIF_GTC | t2-b-7    |
    Then the traders should have the following account balances:
      | trader  | asset | market id | margin  | general   |
      | trader3 | ETH   | ETH/DEC20 | 2174316 | 990775684 |
    # Check MTM Loss transfer happened
    And the following transfers should happen:
      | from    | to      | from account            | to account           | market id | amount  | asset |
      | market  | trader3 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN  | ETH/DEC20 | 4000000 | ETH   |
      | trader3 | trader3 | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_GENERAL | ETH/DEC20 | 4449804 | ETH   |
    And clear transfer events

    # Amend orders to set slippage to 120
    # Amending prices down, so amend buy order first, so it doesn't uncross with the lowered sell order
    When the traders amend the following orders:
      | trader  | reference | price    | size delta | tif     |
      | trader2 | t2-b-1    | 11500000 | 0          | TIF_GTC |
      | trader1 | t1-s-1    | 12500000 | 0          | TIF_GTC |
    Then the following amendments should be accepted:
      | trader  | reference |
      | trader1 | t1-s-1    |
      | trader2 | t2-s-1    |

    When the traders place the following orders:
      | trader  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 12000000 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-5    |
      | trader2 | ETH/DEC20 | buy  | 1      | 12000000 | 1                | TYPE_LIMIT | TIF_GTC | t2-b-8    |
    Then the traders should have the following account balances:
      | trader  | asset | market id | margin  | general   |
      | trader3 | ETH   | ETH/DEC20 | 1949413 | 993000587 |
    # Check MTM Loss transfer happened
    And the following transfers should happen:
      | from    | to      | from account            | to account           | market id | amount  | asset |
      | market  | trader3 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN  | ETH/DEC20 | 2000000 | ETH   |
      | trader3 | trader3 | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_GENERAL | ETH/DEC20 | 2224903 | ETH   |
    And clear transfer events

    # Amend orders to set slippage to 110
    # Amending prices down, so amend buy order first, so it doesn't uncross with the lowered sell order
    When the traders amend the following orders:
      | trader  | reference | price    | size delta | tif     |
      | trader2 | t2-b-1    | 10500000 | 0          | TIF_GTC |
      | trader1 | t1-s-1    | 11500000 | 0          | TIF_GTC |
    Then the following amendments should be accepted:
      | trader  | reference |
      | trader1 | t1-s-1    |
      | trader2 | t2-s-1    |

    When the traders place the following orders:
      | trader  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 11000000 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-6    |
      | trader2 | ETH/DEC20 | buy  | 1      | 11000000 | 1                | TYPE_LIMIT | TIF_GTC | t2-b-9    |
    Then the traders should have the following account balances:
      | trader  | asset | market id | margin  | general   |
      | trader3 | ETH   | ETH/DEC20 | 1836962 | 994113038 |
    # Check MTM Loss transfer happened
    And the following transfers should happen:
      | from    | to      | from account            | to account           | market id | amount  | asset |
      | market  | trader3 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN  | ETH/DEC20 | 1000000 | ETH   |
      | trader3 | trader3 | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_GENERAL | ETH/DEC20 | 1112451 | ETH   |
    And clear transfer events

    # Amend orders to set slippage to 100
    # Amending prices down, so amend buy order first, so it doesn't uncross with the lowered sell order
    When the traders amend the following orders:
      | trader  | reference | price    | size delta | tif     |
      | trader2 | t2-b-1    | 9500000  | 0          | TIF_GTC |
      | trader1 | t1-s-1    | 10500000 | 0          | TIF_GTC |
    Then the following amendments should be accepted:
      | trader  | reference |
      | trader1 | t1-s-1    |
      | trader2 | t2-s-1    |

    When the traders place the following orders:
      | trader  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GTC | t1-s-7    |
      | trader2 | ETH/DEC20 | buy  | 1      | 10000000 | 1                | TYPE_LIMIT | TIF_GTC | t2-b-10   |
    Then the traders should have the following account balances:
      | trader  | asset | market id | margin  | general   |
      | trader3 | ETH   | ETH/DEC20 | 1724511 | 995225489 |
    # Check MTM Loss transfer happened
    And the following transfers should happen:
      | from | to | from account | to account | market id | amount | asset |
      | market  | trader3 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN  | ETH/DEC20 | 1000000 | ETH   |
      | trader3 | trader3 | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_GENERAL | ETH/DEC20 | 1112451 | ETH   |
    And clear transfer events

    When the traders place the following orders:
      | trader  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GTC | post-oa-3 |
      | trader3 | ETH/DEC20 | buy  | 1      | 10000000 | 1                | TYPE_LIMIT | TIF_GTC | post-oa-4 |
    Then the traders should have the following account balances:
      | trader  | asset | market id | margin | general   |
      | trader3 | ETH   | ETH/DEC20 | 0      | 993900000 |
    And the following transfers should happen:
      | from | to | from account | to account | market id | amount | asset |
      | trader3 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_MAKER          | ETH/DEC20 | 40000   | ETH   |
      | trader3 |         | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_INFRASTRUCTURE |           | 10000   | ETH   |
      | trader3 | market  | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_FEES_LIQUIDITY      | ETH/DEC20 | 3000000 | ETH   |
      | market  | trader1 | ACCOUNT_TYPE_FEES_MAKER | ACCOUNT_TYPE_GENERAL             | ETH/DEC20 | 40000   | ETH   |
    And clear transfer events
