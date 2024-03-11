Feature: Test mark to market settlement

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | position decimal places | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | -3                      | 4                      | 0                         | default-futures |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 2     |

  Scenario: If settlement amount <= the party’s margin account balance entire settlement amount is transferred from party’s margin account to the market’s temporary settlement account
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | ETH   | 10000000  |
      | party2 | ETH   | 10000000  |
      | party3 | ETH   | 10000000  |
      | aux    | ETH   | 100000000 |
      | aux2   | ETH   | 100000000 |
      | lpprov | ETH   | 100000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 10000000          | 0.001 | submission |
      | lp1 | lpprov | ETH/DEC19 | 10000000          | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50         | 1      |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50         | 1      |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 49    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 5001  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    And the market data for the market "ETH/DEC19" should be:
      | target stake | supplied stake |
      | 1100000      | 10000000       |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | buy  | 1      | 2000  | 1                | TYPE_LIMIT | TIF_GTC |

    Then the following transfers should happen:
      | from   | to     | from account        | to account              | market id | amount  | asset |
      | party1 | market | ACCOUNT_TYPE_MARGIN | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 1000000 | ETH   |
    And the cumulated balance for all accounts should be worth "330000000"
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"

  Scenario: If settlement amount > party’s margin account balance  and <= party's margin account balance + general account balance for the asset, he full balance of the party’s margin account is transferred to the market’s temporary settlement account the remainder, i.e. difference between the amount transferred from the margin account and the settlement amount, is transferred from the party’s general account for the asset to the market’s temporary settlement account
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | ETH   | 10000000  |
      | party2 | ETH   | 10000000  |
      | party3 | ETH   | 10000000  |
      | aux    | ETH   | 100000000 |
      | aux2   | ETH   | 100000000 |
      | lpprov | ETH   | 100000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 10000000          | 0.001 | submission |
      | lp1 | lpprov | ETH/DEC19 | 10000000          | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50         | 1      |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50         | 1      |
 
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 5001  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general |
      | party1 | ETH   | ETH/DEC19 | 4920000 | 5080000 |
      | party2 | ETH   | ETH/DEC19 | 4932000 | 5067000 |

    And the accumulated liquidity fees should be "1000" for the market "ETH/DEC19"
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 1      | 5000  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general |
      | party1 | ETH   | ETH/DEC19 | 5040000 | 4960000 |

    # update linear slippage factor more in line with what book-based slippage used to be
    And the markets are updated:
        | id          | linear slippage factor |
        | ETH/DEC19   | 0.35                  |
   
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | buy  | 1      | 5000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the following transfers should happen:
      | from   | to     | from account        | to account              | market id | amount  | asset |
      | party1 | market | ACCOUNT_TYPE_MARGIN | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 4000000 | ETH   |
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
  
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC19 | sell | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    Then the following transfers should happen:
      | from   | to     | from account          | to account              | market id | amount  | asset |
      | party3 | party3 | ACCOUNT_TYPE_GENERAL  | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 660000  | ETH   |
      | party3 | market | ACCOUNT_TYPE_MARGIN   | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 3420000 | ETH   |
      | party3 | market | ACCOUNT_TYPE_GENERAL  | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 581000 | ETH   |
    And the cumulated balance for all accounts should be worth "330000000"



