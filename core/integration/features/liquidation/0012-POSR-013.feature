Feature: 0012-POSR-013 Gradual release of position.

  Background:
    Given the oracle spec for settlement data filtering data from "0xCAFECAFE1" named "ethDec21Oracle":
      | property         | type         | binding         |
      | prices.BTC.value | TYPE_INTEGER | settlement data |
    And the oracle spec for trading termination filtering data from "0xCAFECAFE1" named "ethDec21Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator                  | auction duration | fees         | price monitoring | data source config | linear slippage factor | quadratic slippage factor | sla params      | liquidation strategy |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-2 | default-overkill-margin-calculator | 1                | default-none | default-none     | ethDec21Oracle     | 0.9145                 | 0                         | default-futures | AC-013-strat         |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |


  @LiquidationAC
  Scenario: 0012-POSR-013 based on verified-positions-resolution-1.feature
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount        |
      | sellSideProvider | BTC   | 1000000000000 |
      | buySideProvider  | BTC   | 1000000000000 |
      | desginatedLoser  | BTC   | 11600         |
      | aux              | BTC   | 1000000000000 |
      | aux2             | BTC   | 1000000000000 |
      | bulkSeller       | BTC   | 9999999999999 |
      | bulkBuyer        | BTC   | 9999999999999 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 10     | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "150" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # insurance pool generation - setup orderbook
    When the parties place the following orders with ticks:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 290    | 150   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |

    # insurance pool generation - trade
    When the parties place the following orders with ticks:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | desginatedLoser | ETH/DEC19 | buy  | 280    | 150   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |
    And the network moves ahead "1" blocks

    Then the parties should have the following account balances:
      | party           | asset | market id | margin | general |
      | desginatedLoser | BTC   | ETH/DEC19 | 0      | 0       |

    And the parties should have the following margin levels:
      | party           | market id | maintenance | search | initial | release |
      | desginatedLoser | ETH/DEC19 | 0           | 0      | 0       | 0       |

    Then the parties should have the following profit and loss:
      | party            | volume | unrealised pnl | realised pnl |
      | aux              | 1      | 0              | 0            |
      | aux2             | -1     | 0              | 0            |
      | sellSideProvider | -280   | 0              | 0            |
      | buySideProvider  | 0      | 0              | 0            |
      | desginatedLoser  | 0      | 0              | -11600       |
      | network          | 280    | 0              | 0            |
    # Now that the network has the position as per AC (280 long), ensure the volume on the book is correct
    # The buy volume should always be 10,000. Place sell orders to match to avoid increased margin requirement
    # as a result of an unknown exit price.
    # Current book:
    # SELL orders:
    # 	| Party            | Volume | Remaining | Price |
	#   | sellSideProvider | 290    | 10        | 150   |
	#   | aux              | 10     | 10        | 2000  |
    # BUY orders:
    #    | Party           | Volume | Remaining | Price |
	#    | buySideProvider | 1      | 1         | 140   |
	#    | aux             | 10     | 10        | 1     |
    # First bring both sides to 100, then increase gradually to avoid margin issues, both sides will have 10k volume on the book after this step
    When the parties place the following orders with ticks:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 80     | 150   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-2 |
      | buySideProvider  | ETH/DEC19 | buy  | 89     | 140   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2  |
      | bulkBuyer        | ETH/DEC19 | buy  | 200    | 145   | 0                | TYPE_LIMIT | TIF_GTC | bbuy-1          |
      | bulkSeller       | ETH/DEC19 | sell | 200    | 150   | 0                | TYPE_LIMIT | TIF_GTC | bsell-1         |
      | bulkBuyer        | ETH/DEC19 | buy  | 400    | 145   | 0                | TYPE_LIMIT | TIF_GTC | bbuy-2          |
      | bulkSeller       | ETH/DEC19 | sell | 400    | 150   | 0                | TYPE_LIMIT | TIF_GTC | bsell-2         |
      | bulkBuyer        | ETH/DEC19 | buy  | 800    | 145   | 0                | TYPE_LIMIT | TIF_GTC | bbuy-3          |
      | bulkSeller       | ETH/DEC19 | sell | 800    | 150   | 0                | TYPE_LIMIT | TIF_GTC | bsell-3         |
      | bulkBuyer        | ETH/DEC19 | buy  | 1500   | 145   | 0                | TYPE_LIMIT | TIF_GTC | bbuy-4          |
      | bulkSeller       | ETH/DEC19 | sell | 1500   | 150   | 0                | TYPE_LIMIT | TIF_GTC | bsell-4         |
      | bulkBuyer        | ETH/DEC19 | buy  | 7000   | 145   | 0                | TYPE_LIMIT | TIF_GTC | bbuy-5          |
      | bulkSeller       | ETH/DEC19 | sell | 7000   | 150   | 0                | TYPE_LIMIT | TIF_GTC | bsell-5         |
    # Move network forwards 10 blocks to have the network reduce its position
    And the network moves ahead "10" blocks
    Then the following trades should be executed:
      | buyer     | price | size | seller  |
      | bulkBuyer | 145   | 100  | network |
    Then the parties should have the following profit and loss:
      | party            | volume | unrealised pnl | realised pnl |
      | aux              | 1      | 0              | 0            |
      | aux2             | -1     | 0              | 0            |
      | sellSideProvider | -280   | 0              | 0            |
      | buySideProvider  | 0      | 0              | 0            |
      | desginatedLoser  | 0      | 0              | -11600       |
      | network          | 180    | 0              | -500         |
      | bulkBuyer        | 100    | 500            | 0            |
    # Restore the volume on the book, use a different party just because
    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | aux   | ETH/DEC19 | buy  | 100    | 140   | 0                | TYPE_LIMIT | TIF_GTC | aux-provider-2 |
    # Next release of position
    And the network moves ahead "10" blocks
    Then the following trades should be executed:
      | buyer     | price | size | seller  |
      | bulkBuyer | 145   | 90   | network |
    Then the parties should have the following profit and loss:
      | party            | volume | unrealised pnl | realised pnl |
      | aux              | 1      | 0              | 0            |
      | aux2             | -1     | 0              | 0            |
      | sellSideProvider | -280   | 0              | 0            |
      | buySideProvider  | 0      | 0              | 0            |
      | desginatedLoser  | 0      | 0              | -11600       |
      | network          | 90     | 0              | -950         |
      | bulkBuyer        | 190    | 950            | 0            |
    # Restore the volume on the book, use a different party just because
    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | aux2  | ETH/DEC19 | buy  | 90     | 140   | 0                | TYPE_LIMIT | TIF_GTC | aux-provider-3 |

    # Next release of position
    And the network moves ahead "10" blocks
    ## We provided the volume on the book in separate orders, the trades will reflect this
    ## so in this case, the trade for 45 is split across 2 orders. This does show that this batch/trade
    ## is different to the next, making all network trades unique in this scenario.
    Then the following trades should be executed:
      | buyer     | price | size | seller  |
      | bulkBuyer | 145   | 10   | network |
      | bulkBuyer | 145   | 35   | network |
    Then the parties should have the following profit and loss:
      | party            | volume | unrealised pnl | realised pnl |
      | aux              | 1      | 0              | 0            |
      | aux2             | -1     | 0              | 0            |
      | sellSideProvider | -280   | 0              | 0            |
      | buySideProvider  | 0      | 0              | 0            |
      | desginatedLoser  | 0      | 0              | -11600       |
      | network          | 45     | 0              | -1175        |
      | bulkBuyer        | 235    | 1175           | 0            |
    # Restore the volume on the book, use a different party just because
    When the parties place the following orders with ticks:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | buySideProvider | ETH/DEC19 | buy  | 45     | 140   | 0                | TYPE_LIMIT | TIF_GTC | aux-provider-3 |
    # Last release of position
    And the network moves ahead "10" blocks
    Then the following trades should be executed:
      | buyer     | price | size | seller  |
      | bulkBuyer | 145   | 45   | network |
    Then the parties should have the following profit and loss:
      | party            | volume | unrealised pnl | realised pnl |
      | aux              | 1      | 0              | 0            |
      | aux2             | -1     | 0              | 0            |
      | sellSideProvider | -280   | 0              | 0            |
      | buySideProvider  | 0      | 0              | 0            |
      | desginatedLoser  | 0      | 0              | -11600       |
      | network          | 0      | 0              | -1400        |
      | bulkBuyer        | 280    | 1400           | 0            |
