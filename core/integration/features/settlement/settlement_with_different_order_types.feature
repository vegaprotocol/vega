Feature: Test mark to market settlement with periodicity, takes the first scenario from mark_to_market_settlement_neg_pdp

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | position decimal places | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | -3                      | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |

  Scenario: S001 - with order type GTC only
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 5s    |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | ETH   | 10000000  |
      | party2 | ETH   | 10000000  |
      | party3 | ETH   | 10000000  |
      | aux    | ETH   | 100000000 |
      | aux2   | ETH   | 100000000 |
      | lpprov | ETH   | 100000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 10000000          | 0.001 | buy  | BID              | 50         | 1      | submission |
      | lp1 | lpprov | ETH/DEC19 | 10000000          | 0.001 | sell | ASK              | 50         | 1      | submission |
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

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC19 | 120000 | 9880000 |
      | party2 | ETH   | ETH/DEC19 | 132000 | 9867000 |

    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC19 | sell | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general |
      | party1 | ETH   | ETH/DEC19 | 5041200 | 4958800 |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | sell | 3      | 3000  | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "6" blocks
    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general |
      | party1 | ETH   | ETH/DEC19 | 1440000 | 8560000 |
      | party2 | ETH   | ETH/DEC19 | 1273200 | 8725800 |
      | party3 | ETH   | ETH/DEC19 | 360000  | 9640000 |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | buy  | 1      | 2000  | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "6" blocks
    ## Now mark to market, the mark price should be 2,000 at this point, dramatically changing the balances
    ## The interval is set to 5s, so 6 blocks should do the trick

    And the following transfers should happen:
      | from   | to     | from account            | to account              | market id | amount  | asset |
      | party3 | party3 | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 360000  | ETH   |
      | party3 | party3 | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 2245200 | ETH   |
      | aux    | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 1000000 | ETH   |
      | party1 | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 1000000 | ETH   |
      | market | aux2   | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 1000000 | ETH   |
      | market | party2 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 1000000 | ETH   |

    And the cumulated balance for all accounts should be worth "330000000"
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC19"

  Scenario: S002 - Test that market settlement cashflows only depend on parties positions and is independent of what order types (GTC, IOC, IOK) there are on the book. (0024-OSTA-009)
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 5s    |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | ETH   | 10000000  |
      | party2 | ETH   | 10000000  |
      | party3 | ETH   | 10000000  |
      | aux    | ETH   | 100000000 |
      | aux2   | ETH   | 100000000 |
      | lpprov | ETH   | 100000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 10000000          | 0.001 | buy  | BID              | 50         | 1      | submission |
      | lp1 | lpprov | ETH/DEC19 | 10000000          | 0.001 | sell | ASK              | 50         | 1      | submission |
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

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | expires in |
      | party1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTT | 3600       |
      | party2 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTT | 3600       |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC19 | 120000 | 9880000 |
      | party2 | ETH   | ETH/DEC19 | 132000 | 9867000 |

    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | expires in |
      | party1 | ETH/DEC19 | sell | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTT | 3600       |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general |
      | party1 | ETH   | ETH/DEC19 | 5041200 | 4958800 |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | expires in |
      | party3 | ETH/DEC19 | sell | 3      | 3000  | 0                | TYPE_LIMIT | TIF_GTT | 3600       |

    When the network moves ahead "6" blocks
    Then the parties should have the following account balances:
      | party  | asset | market id | margin  | general |
      | party1 | ETH   | ETH/DEC19 | 1440000 | 8560000 |
      | party2 | ETH   | ETH/DEC19 | 1273200 | 8725800 |
      | party3 | ETH   | ETH/DEC19 | 360000  | 9640000 |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | expires in |
      | party3 | ETH/DEC19 | buy  | 1      | 2000  | 1                | TYPE_LIMIT | TIF_GTT | 3600       |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | sell | 1      | 2000  | 0                | TYPE_LIMIT | TIF_IOC |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC19 | sell | 1      | 2000  | 0                | TYPE_LIMIT | TIF_FOK |
    Then the market data for the market "ETH/DEC19" should be:
      | mark price | last traded price | trading mode            | 
      | 1000       | 2000              | TRADING_MODE_CONTINUOUS |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | initial |
      | party3 | ETH/DEC19 | 1061000     | 1273200 |

    When the network moves ahead "6" blocks
    ## Now mark to market, the mark price should be 2,000 at this point, dramatically changing the balances
    ## The interval is set to 5s, so 6 blocks should do the trick

    Then the market data for the market "ETH/DEC19" should be:
      | mark price | last traded price | trading mode            | 
      | 2000       | 2000              | TRADING_MODE_CONTINUOUS |
    And the following transfers should happen:
      | from   | to     | from account            | to account              | market id | amount  | asset |
      | party3 | party3 | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 360000  | ETH   |
      | party3 | party3 | ACCOUNT_TYPE_GENERAL    | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 1332000 | ETH   |
      | aux    | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 1000000 | ETH   |
      | party1 | market | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/DEC19 | 1000000 | ETH   |
      | market | aux2   | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 1000000 | ETH   |
      | market | party2 | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/DEC19 | 1000000 | ETH   |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | initial |
      | party3 | ETH/DEC19 | 2171000     | 2605200 |
    And the parties should have the following account balances:
      | party  | asset | market id | margin  | general |
      | party3 | ETH   | ETH/DEC19 | 2605200 | 7392800 |

    And the cumulated balance for all accounts should be worth "330000000"
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC19"
