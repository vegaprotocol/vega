Feature: When the network party holds a non-zero position and there are not enough funds in market's insurance pool
         to meet the mark-to-market payment the network's position is unaffected and loss socialisation is applied. (0012-POSR-010)

  Background:
    Given the liquidation strategies:
      | name           | disposal step | disposal fraction | full disposal size | max fraction consumed | disposal slippage range |
      | disposal-strat | 1000          | 1.0               | 1000               | 1.0                   | 0.1                     |

    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator                  | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | liquidation strategy |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-2 | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.25                   | 0                         | default-futures | disposal-strat       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  @NoPerp
  Scenario: Implement trade and order network. Covers both 0012-POSR-012 and 0012-POSR-011
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount        |
      | sellSideProvider | BTC   | 1000000000000 |
      | buySideProvider  | BTC   | 1000000000000 |
      | designatedLoser  | BTC   | 120           |
      | aux              | BTC   | 1000000000000 |
      | aux2             | BTC   | 1000000000000 |

    # insurance pool generation - setup orderbook
    When the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC19 | sell | 290    | 150   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC19 | buy  | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |
      | aux              | ETH/DEC19 | sell | 100    | 159   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1         |
      | aux              | ETH/DEC19 | sell | 1      | 149   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-2         |
      | aux2             | ETH/DEC19 | buy  | 1      | 149   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1         |
      | aux2             | ETH/DEC19 | buy  | 100    | 140   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-2         |
    Then the opening auction period ends for market "ETH/DEC19"
    When the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    And the mark price should be "149" for the market "ETH/DEC19"

    Then the parties should have the following account balances:
      | party           | asset | market id | general |
      | designatedLoser | BTC   | ETH/DEC19 | 120     |

    # insurance pool generation - trade
    When the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | designatedLoser  | ETH/DEC19 | buy  | 250    | 150   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    When the network moves ahead "1" blocks

    And the mark price should be "150" for the market "ETH/DEC19"

    # The designatedLoser will be closed out and have all their funds moved to the insurance pool
    Then the parties should have the following account balances:
      | party            | asset | market id | general       | margin      |
      | designatedLoser  | BTC   | ETH/DEC19 | 0             | 0           |
      | aux              | BTC   | ETH/DEC19 | 999999999848  | 151         |
      | aux2             | BTC   | ETH/DEC19 | 999999999848  | 153         |
      | sellSideProvider | BTC   | ETH/DEC19 | 999999962500  | 37500       |
      | buySideProvider  | BTC   | ETH/DEC19 | 1000000000000 | 0           |

    And the insurance pool balance should be "120" for the market "ETH/DEC19"    

    # check the network trades happened and the network party holds the volume
    Then the following network trades should be executed:
      | party           | aggressor side | volume |
      | designatedLoser | buy            | 250    |

    And the parties should have the following profit and loss:
      | party            | volume | unrealised pnl | realised pnl |
      | aux2             | 1      | 1              | 0            |
      | aux              | -1     | -1             | 0            |
      | sellSideProvider | -250   | 0              | 0            |
      | designatedLoser  | 0      | 0              | -120         |
      | network          | 250    | 0              | 0            |

    # Move the mark price to make our network trader lose money
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 145   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | sell | 1      | 145   | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks

    And the mark price should be "145" for the market "ETH/DEC19"

    # The insurance pool will be depleted due to the price move
    And the insurance pool balance should be "0" for the market "ETH/DEC19"    

    # Check that funds have not been fully paid (lose socialisation has taken place)
    # aux is short and should gain 6 but only gets 1 (will lose 5),
    # sellSideProvider is short and should gain 1250 but only gets 124 (will lose 1126)
    # Total distributed is 125 when it should have been 1255
    Then the parties should have the following account balances:
      | party            | asset | market id | general       | margin      |
      | aux              | BTC   | ETH/DEC19 | 1000000000000 | 0           |
      | aux2             | BTC   | ETH/DEC19 | 999999999996  | 0           |
      | sellSideProvider | BTC   | ETH/DEC19 | 999999962500  | 37624       |
      | buySideProvider  | BTC   | ETH/DEC19 | 1000000000000 | 0           |
    # This part explicitly covers 0012-POSR-011:
    # The insurance pool balance is zero, so the network does not meet the required margin balance
    # Despite this, it can maintain its position, and loss socialisation kicks in during the MTM settlement.
    And the parties should have the following profit and loss:
      | party            | volume | unrealised pnl | realised pnl |
      | aux2             | 0      | 0              | -4           |
      | aux              | 0      | 0              | 0            |
      | sellSideProvider | -250   | 1250           | -1126        |
      | designatedLoser  | 0      | 0              | -120         |
      | network          | 250    | -1250          | 0            |
    And the parties should have the following margin levels:
      | party   | market id | maintenance |
      | network | ETH/DEC19 | 9063        |
    And the insurance pool balance should be "0" for the market "ETH/DEC19"    
