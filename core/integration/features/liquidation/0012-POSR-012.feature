Feature: 0012-POSR-012 Update the liquidation strategy through market update

  Background:
    Given the liquidation strategies:
      | name             | disposal step | disposal fraction | full disposal size | max fraction consumed | disposal slippage range |
      | slow-liquidation | 100           | 0.2               | 1                  | 0.2                   | 0.5                     |
      | fast-liquidation | 10            | 0.1               | 20                 | 0.05                  | 0.5                     |

    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | liquidation strategy |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-4 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.25                   | 0                         | default-futures | slow-liquidation     |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 2     |

  @NoPerp @LiquidationUpdate
  Scenario: Update liquidation strategy through market update
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | tt_4   | BTC   | 500000    |
      | tt_5   | BTC   | 100       |
      | tt_6   | BTC   | 100000000 |
      | tt_10  | BTC   | 10000000  |
      | tt_11  | BTC   | 10000000  |
      | tt_aux | BTC   | 100000000 |
      | t2_aux | BTC   | 100000000 |
      | lpprov | BTC   | 100000000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | tt_aux | ETH/DEC19 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1   |
      | tt_aux | ETH/DEC19 | sell | 1      | 200   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1   |
      | t2_aux | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-2   |
      | tt_aux | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-2   |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | MID              | 50         | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | MID              | 50         | 100    |
    Then the opening auction period ends for market "ETH/DEC19"

    # place orders and generate trades
    When the parties place the following orders "1" blocks apart:
      | party | market id | side | volume | price | resulting trades | type        | tif     | reference | expires in |
      | tt_10 | ETH/DEC19 | buy  | 5      | 100   | 0                | TYPE_LIMIT  | TIF_GTT | tt_10-1   | 3600       |
      | tt_11 | ETH/DEC19 | sell | 5      | 100   | 1                | TYPE_LIMIT  | TIF_GTT | tt_11-1   | 3600       |
      | tt_4  | ETH/DEC19 | buy  | 2      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_4-1    |            |
      | tt_4  | ETH/DEC19 | buy  | 2      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_4-2    |            |
      | tt_5  | ETH/DEC19 | buy  | 2      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_5-1    |            |
      | tt_6  | ETH/DEC19 | sell | 2      | 150   | 1                | TYPE_LIMIT  | TIF_GTC | tt_6-1    |            |
      | tt_5  | ETH/DEC19 | buy  | 2      | 150   | 0                | TYPE_LIMIT  | TIF_GTC | tt_5-2    |            |
      | tt_6  | ETH/DEC19 | sell | 2      | 150   | 1                | TYPE_LIMIT  | TIF_GTC | tt_6-2    |            |
      | tt_10 | ETH/DEC19 | buy  | 25     | 100   | 0                | TYPE_LIMIT  | TIF_GTC | tt_10-2   |            |
      | tt_11 | ETH/DEC19 | sell | 25     | 0     | 3                | TYPE_MARKET | TIF_FOK | tt_11-2   |            |
    And the network moves ahead "1" blocks

    And the mark price should be "100" for the market "ETH/DEC19"

    # checking margins
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general |
      | tt_5  | BTC   | ETH/DEC19 | 0      | 0       |

    # then we make sure the insurance pool collected the funds
    And the insurance pool balance should be "0" for the market "ETH/DEC19"

    #check positions
    Then the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl |
      | tt_4    | 4      | -200           | 0            |
      | tt_5    | 0      | 0              | -100         |
      | tt_6    | -4     | 200            | -27          |
      | tt_10   | 26     | 0              | 0            |
      | tt_11   | -30    | 200            | -65          |
      | network | 4      | 0              | 0            |
    And the following trades should be executed:
      | buyer   | price | size | seller |
      | network | 100   | 4    | tt_5   |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | tt_10 | ETH/DEC19 | buy  | 50     | 100   | 0                | TYPE_LIMIT | TIF_GTC | tt_10-n   |
    And the network moves ahead "101" blocks
    Then the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl |
      | tt_4    | 4      | -200           | 0            |
      | tt_5    | 0      | 0              | -100         |
      | tt_6    | -4     | 200            | -27          |
      | tt_10   | 27     | 0              | 0            |
      | tt_11   | -30    | 200            | -65          |
      | network | 3      | 0              | 0            |
    # Network trades with good party for a size of 1
    And the following trades should be executed:
      | buyer | price | size | seller  |
      | tt_10 | 100   | 1    | network |
    # Now update the market
    When the markets are updated:
      | id        | linear slippage factor | quadratic slippage factor | liquidation strategy |
      | ETH/DEC19 | 0.25                   | 0                         | fast-liquidation     |
    # Now the network should dispose of its entire position
    When the network moves ahead "11" blocks
    Then the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl |
      | tt_4    | 4      | -200           | 0            |
      | tt_5    | 0      | 0              | -100         |
      | tt_6    | -4     | 200            | -27          |
      | tt_10   | 30     | 0              | 0            |
      | tt_11   | -30    | 200            | -65          |
      | network | 0      | 0              | 0            |
    # Network has been closed out entirely now
    And the following trades should be executed:
      | buyer | price | size | seller  |
      | tt_10 | 100   | 3    | network |
