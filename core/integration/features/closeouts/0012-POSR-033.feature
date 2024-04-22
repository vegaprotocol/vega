Feature: Covers 0012-POSR-033

  Background:
    # disposal strategy every 5 seconds, 20% until 10 or less, max 10% of the book used, slippage is set to 10 so price range is always wide enough
    Given the liquidation strategies:
      | name             | disposal step | disposal fraction | full disposal size | max fraction consumed | disposal slippage range |
      | disposal-strat-1 | 5             | 0.2               | 10                 | 0.5                   | 0.1                     |
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | liquidation strategy |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-4 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.25                   | 0                         | default-futures | disposal-strat-1     |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 2     |

  Scenario: When calculating the available volume, volume outside the disposal price range should not be considered.
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

    # place orders and generate trades, do not progress time
    When the parties place the following orders with ticks:
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

    # some time passes, position network still hasn't been closed/disposed of
    When the network moves ahead "1" blocks
    Then the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl |
      | tt_4    | 4      | -200           | 0            |
      | tt_5    | 0      | 0              | -100         |
      | tt_6    | -4     | 200            | -27          |
      | tt_10   | 26     | 0              | 0            |
      | tt_11   | -30    | 200            | -65          |
      | network | 4      | 0              | 0            |
    # clear trade events to ensure that we are not picking up a closeout trade that happened before the disposal step expires
    And clear trade events
    # move mid price + create an order within the 10% range, max fraction is 0.5, so we expect half of the volume to trade
    # The book has much, much move volume, but we only check the volume in the available range
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | tt_6  | ETH/DEC19 | sell | 2      | 150   | 0                | TYPE_LIMIT | TIF_GTC | tt_10-2   |
      | tt_10 | ETH/DEC19 | buy  | 4      | 126   | 0                | TYPE_LIMIT | TIF_GTC | tt_10-2   |



    # some time passes, now the network  disposes of its position
    When the network moves ahead "5" blocks
    Then the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl |
      | tt_4    | 4      | -200           | 0            |
      | tt_5    | 0      | 0              | -100         |
      | tt_6    | -4     | 200            | -27          |
      | tt_10   | 28     | -52            | 0            |
      | tt_11   | -30    | 200            | -65          |
      | network | 2      | 0              | 52           |
    # Disposal fraction == 0.5, only a half of which trades
    And the following network trades should be executed:
      | party | aggressor side | volume |
      | tt_10 | sell           | 2      |


    # Next closeout check -> the network trades again, but only consumes half again, so we expect a volume of 1
    When the network moves ahead "6" blocks
    Then the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl |
      | tt_4    | 4      | -200           | 0            |
      | tt_5    | 0      | 0              | -100         |
      | tt_6    | -4     | 200            | -27          |
      | tt_10   | 29     | -78            | 0            |
      | tt_11   | -30    | 200            | -65          |
      | network | 1      | 0              | 78           |
    # Disposal fraction == 0.5, only a half of which trades
    And the following network trades should be executed:
      | party | aggressor side | volume |
      | tt_10 | sell           | 1      |

