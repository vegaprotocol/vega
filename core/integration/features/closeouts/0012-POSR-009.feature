Feature: When a party is distressed and gets closed out the network's position gets modified to reflect that it's now the network party that holds that volume. (0012-POSR-009)

  Background:
    Given the liquidation strategies:
      | name           | disposal step | disposal fraction | full disposal size | max fraction consumed | disposal slippage range |
      | disposal-strat | 10            | 1.0               | 1000               | 1.0                   | 0.1                     |

    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator                  | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | liquidation strategy |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-2 | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e3                    | 1e3                       | default-futures | disposal-strat       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: Implement trade and order network
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount        |
      | sellSideProvider | BTC   | 1000000000000 |
      | buySideProvider  | BTC   | 1000000000000 |
      | designatedLoser  | BTC   | 12000         |
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

    Then the parties should have the following account balances:
      | party           | asset | market id | general |
      | designatedLoser | BTC   | ETH/DEC19 | 12000   |

    # insurance pool generation - trade
    When the parties place the following orders "1" blocks apart:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | designatedLoser  | ETH/DEC19 | buy  | 105    | 150   | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    # The designatedLoser will be closed out and have all their funds moved to the insurance pool
    Then the parties should have the following account balances:
      | party           | asset | market id | general | margin |
      | designatedLoser | BTC   | ETH/DEC19 | 0       | 0      |

    And the global insurance pool balance should be "0" for the asset "BTC"
    And the insurance pool balance should be "12000" for the market "ETH/DEC19"    

    # check the network trades happened and the network party holds the volume
    Then the following network trades should be executed:
      | party           | aggressor side | volume |
      | designatedLoser | buy            | 105    |
