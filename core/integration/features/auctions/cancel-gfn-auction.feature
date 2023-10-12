Feature: When moving into auction, all GFN orders are cancelled

  Background:
    Given the price monitoring named "my-price-monitoring":
      | horizon | probability | auction extension |
      | 60      | 0.95        | 240               |
      | 600     | 0.99        | 360               |
    And the price monitoring named "my-price-monitoring-2":
      | horizon | probability | auction extension |
      | 60      | 0.95        | 240               |
      | 120     | 0.99        | 360               |
    And the log normal risk model named "my-log-normal-risk-model":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.000001      | 0.00011407711613050422 | 0  | 0.016 | 2.0   |
    And the markets:
      | id        | quote name | asset | risk model                    | margin calculator         | auction duration | fees         | price monitoring      | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC20 | ETH        | ETH   | default-log-normal-risk-model | default-margin-calculator | 60               | default-none | my-price-monitoring   | default-eth-for-future | 0.01                   | 0                         | default-futures |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 60    |
      | limits.markets.maxPeggedOrders | 2     |


  @GFNCancel
  Scenario: replicates test_GFN_OrdersCancelledIntoAuction ST - covers 0068-MATC-027 and 0026-AUCT-015
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | ETH   | 100000000000 |
      | party2 | ETH   | 100000000000 |
      | party3 | ETH   | 100000000000 |
      | party4 | ETH   | 100000000000 |
      | aux    | ETH   | 100000000000 |
      | aux2   | ETH   | 100000000000 |
      | lpprov | ETH   | 100000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lpprov | ETH/DEC20 | 2         | 1                    | buy  | BID              | 50         | 100    |
      | lpprov | ETH/DEC20 | 2         | 1                    | sell | ASK              | 50         | 100    |

    When the parties place the following orders:
      | party | market id | side | volume | price   | resulting trades | type       | tif     |
      | aux   | ETH/DEC20 | buy  | 1      | 1       | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 200000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC20 | buy  | 1      | 1000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 1000000 | 0                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "59" blocks
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux2  | ETH/DEC20 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC20"

    And the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 60      | 995       | 1005      | 7434         | 90000000       | 10            |
      | 1000       | TRADING_MODE_CONTINUOUS | 600     | 978       | 1022      | 7434         | 90000000       | 10            |

    # place GFN orders outside of bounds so we can trigger auction without those order uncrossing
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC20 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GFN |
      | party2 | ETH/DEC20 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GFN |
      | party4 | ETH/DEC20 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the order book should have the following volumes for market "ETH/DEC20":
      | side | volume | price   |
      | buy  | 1      | 1       |
      | buy  | 2      | 900     |
      | buy  | 1      | 990     |
      | buy  | 1      | 1000    |
      | sell | 2      | 1010    |
      | sell | 2      | 1110    |
      | sell | 1      | 200000  |
      | sell | 1      | 1000000 |
    And the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 60      | 995       | 1005      | 7434         | 90000000       | 10            |
      | 1000       | TRADING_MODE_CONTINUOUS | 600     | 978       | 1022      | 7434         | 90000000       | 10            |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party3 | ETH/DEC20 | buy  | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"
    ## The GFN orders will should be removed from the book (buy at 990, sell at 1010)
    ## the 2 GTC orders that triggered the price auction, however, remain
    And the order book should have the following volumes for market "ETH/DEC20":
      | side | volume | price   |
      | buy  | 1      | 1       |
      | buy  | 0      | 900     |
      | buy  | 0      | 990     |
      | buy  | 1      | 1000    |
      | buy  | 1      | 1010    |
      | sell | 1      | 1010    |
      | sell | 0      | 1110    |
      | sell | 1      | 200000  |
      | sell | 1      | 1000000 |
