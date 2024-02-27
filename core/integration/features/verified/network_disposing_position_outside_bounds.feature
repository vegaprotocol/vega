Feature: Disposing position outside bounds

  # A network should be able to dispose it's position against any orders outside price monitoring bounds but not orders outside the liquidity price range.

  Background:

    # Configure the network
    Given the average block duration is "1"
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 2     |
    And the following assets are registered:
      | id      | decimal places | quantum |
      | USD.0.1 | 0              | 1       |

    # Configure the markets
    Given the liquidation strategies:
      | name                | disposal step | disposal fraction | full disposal size | max fraction consumed |
      | liquidation-strat-1 | 1             | 0.5               | 0                  | 1                     |
      | liquidation-strat-2 | 1             | 1                 | 0                  | 0.5                   |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 6200    | 0.99        | 5                 |
    And the markets:
      | id        | quote name | asset    | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | liquidation strategy | sla params    |
      | ETH/MAR22 | ETH        | USD.0.10 | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | price-monitoring | default-eth-for-future | 0.001                  | 0                         | liquidation-strat-1  | default-basic |
      | ETH/MAR23 | ETH        | USD.0.10 | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | price-monitoring | default-eth-for-future | 0.001                  | 0                         | liquidation-strat-2  | default-basic |

  Scenario: Network considers volume outside price-monitoring bounds as avaliable to dispose against when calculating max consumption (0012-POSR-019)(0012-POSR-020)(0012-POSR-025)(0012-POSR-029)
    # Orderbook setup such that LPs post orders outside of price-monitoring bounds using limit ormal orders and iceberg orders. The avaliable volume should include volume outside bounds and the full size of the iceberg orders.

    # Market configuiration
    Given the liquidity sla params named "sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 100         | 0.6                          | 1                             | 1.0                    |
    When the markets are updated:
      | id        | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR23 | 1e-3                   | 0                         | sla-params |

    # Setup the market
    Given the initial insurance pool balance is "10000" for all the markets
    And the parties deposit on asset's general account the following amount:
      | party | asset    | amount       |
      | lp1   | USD.0.10 | 100000000000 |
      | aux1  | USD.0.10 | 10000000000  |
      | aux2  | USD.0.10 | 10000000000  |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 500000            | 0   | submission |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp1   | ETH/MAR23 | buy  | 5      | 180   | 0                | TYPE_LIMIT | TIF_GTC | best-bid  |
      | lp1   | ETH/MAR23 | sell | 5      | 210   | 0                | TYPE_LIMIT | TIF_GTC | best-ask  |
    And the parties place the following pegged iceberg orders:
      | party | market id | side | volume | resulting trades | type       | tif     | peak size | minimum visible size | pegged reference | offset |
      | lp1   | ETH/MAR23 | buy  | 15     | 0                | TYPE_LIMIT | TIF_GTC | 1         | 1                    | BID              | 1      |
      | lp1   | ETH/MAR23 | sell | 15     | 0                | TYPE_LIMIT | TIF_GTC | 1         | 1                    | ASK              | 1      |
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/MAR23 | buy  | 1      | 200   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/MAR23 | sell | 1      | 200   | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/MAR23"

    # atRiskPary opens a long position
    Given the parties deposit on asset's general account the following amount:
      | party       | asset    | amount |
      | atRiskParty | USD.0.10 | 1700   |
    And the parties place the following orders:
      | party       | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1        | ETH/MAR23 | sell | 100    | 200   | 0                | TYPE_LIMIT | TIF_GTC |
      | atRiskParty | ETH/MAR23 | buy  | 100    | 200   | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the parties should have the following profit and loss:
      | party       | volume | unrealised pnl | realised pnl |
      | atRiskParty | 100    | 0              | 0            |
    And the parties should have the following margin levels:
      | party       | market id | maintenance | search | initial | release |
      | atRiskParty | ETH/MAR23 | 1412        | 1553   | 1694    | 1976    |
    And the parties should have the following account balances:
      | party       | asset    | market id | margin | general |
      | atRiskParty | USD.0.10 | ETH/MAR23 | 1670   | 30      |

    # Market moves against atRiskParty whom is liquidated
    Given the parties amend the following orders:
      | party | reference | price | size delta | tif     |
      | lp1   | best-bid  | 180   | 0          | TIF_GTC |
      | lp1   | best-ask  | 220   | 0          | TIF_GTC |
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/MAR23 | buy  | 1      | 190   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/MAR23 | sell | 1      | 190   | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the mark price should be "190" for the market "ETH/MAR23"
    And the parties should have the following profit and loss:
      | party       | volume | unrealised pnl | realised pnl |
      | atRiskParty | 0      | 0              | -1700        |
      | network     | 100    | 0              | 0            |

    # Network cannot dispose of its position outside of price monitoring bounds
    Given the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | horizon | min bound | max bound |
      | 190        | TRADING_MODE_CONTINUOUS | 6200    | 186       | 214       |
    When the network moves ahead "1" blocks
    And the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl |
      | network | 100     | 0             | 0            |

  @me
  Scenario: Network does not consider volume outside liquidity range as avaliable to dispose against when calculating max consumption (0012-POSR-019)(0012-POSR-021)(0012-POSR-025)
    # Orderbook setup such that LPs post a mix of limit orders within and outside liquidity range. Only the volume inside the liquidity price range should be considered.

    # Market configuiration
    Given the liquidity sla params named "sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.11        | 0.6                          | 1                             | 1.0                    |
    When the markets are updated:
      | id        | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR23 | 1e-3                   | 0                         | sla-params |

    # Setup the market
    Given the initial insurance pool balance is "10000" for all the markets
    And the parties deposit on asset's general account the following amount:
      | party | asset    | amount       |
      | lp1   | USD.0.10 | 100000000000 |
      | aux1  | USD.0.10 | 10000000000  |
      | aux2  | USD.0.10 | 10000000000  |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | ETH/MAR23 | 500000            | 0   | submission |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp1   | ETH/MAR23 | buy  | 10     | 180   | 0                | TYPE_LIMIT | TIF_GTC | best-bid  |
      | lp1   | ETH/MAR23 | sell | 10     | 210   | 0                | TYPE_LIMIT | TIF_GTC | best-ask  |
    And the parties place the following pegged iceberg orders:
      | party | market id | side | volume | resulting trades | type       | tif     | peak size | minimum visible size | pegged reference | offset |
      | lp1   | ETH/MAR23 | buy  | 10     | 0                | TYPE_LIMIT | TIF_GTC | 1         | 1                    | BID              | 10     |
      | lp1   | ETH/MAR23 | sell | 10     | 0                | TYPE_LIMIT | TIF_GTC | 1         | 1                    | ASK              | 10     |
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/MAR23 | buy  | 1      | 200   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/MAR23 | sell | 1      | 200   | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/MAR23"

    # atRiskPary opens a long position
    Given the parties deposit on asset's general account the following amount:
      | party       | asset    | amount |
      | atRiskParty | USD.0.10 | 1700   |
    And the parties place the following orders:
      | party       | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1        | ETH/MAR23 | sell | 100    | 200   | 0                | TYPE_LIMIT | TIF_GTC |
      | atRiskParty | ETH/MAR23 | buy  | 100    | 200   | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the parties should have the following profit and loss:
      | party       | volume | unrealised pnl | realised pnl |
      | atRiskParty | 100    | 0              | 0            |
    And the parties should have the following margin levels:
      | party       | market id | maintenance | search | initial | release |
      | atRiskParty | ETH/MAR23 | 1412        | 1553   | 1694    | 1976    |
    And the parties should have the following account balances:
      | party       | asset    | market id | margin | general |
      | atRiskParty | USD.0.10 | ETH/MAR23 | 1670   | 30      |

    # Market moves against atRiskParty whom is liquidated
    Given the parties amend the following orders:
      | party | reference | price | size delta | tif     |
      | lp1   | best-bid  | 186   | 0          | TIF_GTC |
      | lp1   | best-ask  | 220   | 0          | TIF_GTC |
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/MAR23 | buy  | 1      | 190   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/MAR23 | sell | 1      | 190   | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the mark price should be "190" for the market "ETH/MAR23"
    And the parties should have the following profit and loss:
      | party       | volume | unrealised pnl | realised pnl |
      | atRiskParty | 0      | 0              | -1700        |
      | network     | 100    | 0              | 0            |

    # Network only able do dispose volume against orders inside liquidity range
    Given the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | static mid price |
      | 190        | TRADING_MODE_CONTINUOUS | 203              |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer | price | size | seller  |
      | lp1   | 186   | 5    | network |
    And the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl |
      | network | 95     | 0              | -20          |

  Scenario: Volume on the book within liquidity price range but outside price monitoring bounds, network able to dispose position (0012-POSR-026)(0012-POSR-030)

    # Market configuiration
    Given the liquidity sla params named "sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 100         | 0.6                          | 1                             | 1.0                    |
    When the markets are updated:
      | id        | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | 1e-3                   | 0                         | sla-params |

    # Setup the market
    Given the initial insurance pool balance is "10000" for all the markets
    And the parties deposit on asset's general account the following amount:
      | party | asset    | amount       |
      | lp1   | USD.0.10 | 100000000000 |
      | aux1  | USD.0.10 | 10000000000  |
      | aux2  | USD.0.10 | 10000000000  |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 500000            | 0   | submission |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp1   | ETH/MAR22 | buy  | 1000   | 180   | 0                | TYPE_LIMIT | TIF_GTC | best-bid  |
      | lp1   | ETH/MAR22 | sell | 1000   | 210   | 0                | TYPE_LIMIT | TIF_GTC | best-ask  |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/MAR22 | buy  | 1      | 200   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/MAR22 | sell | 1      | 200   | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/MAR22"


    # atRiskPary opens a long position
    Given the parties deposit on asset's general account the following amount:
      | party       | asset    | amount |
      | atRiskParty | USD.0.10 | 20     |
    And the parties place the following orders:
      | party       | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1        | ETH/MAR22 | sell | 1      | 200   | 0                | TYPE_LIMIT | TIF_GTC |
      | atRiskParty | ETH/MAR22 | buy  | 1      | 200   | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the parties should have the following profit and loss:
      | party       | volume | unrealised pnl | realised pnl |
      | atRiskParty | 1      | 0              | 0            |
    And the parties should have the following margin levels:
      | party       | market id | maintenance | search | initial | release |
      | atRiskParty | ETH/MAR22 | 15          | 16     | 18      | 21      |
    And the parties should have the following account balances:
      | party       | asset    | market id | margin | general |
      | atRiskParty | USD.0.10 | ETH/MAR22 | 16     | 4       |

    # Market moves against atRiskParty whom is liquidated
    Given the parties amend the following orders:
      | party | reference | price | size delta | tif     |
      | lp1   | best-bid  | 180   | 0          | TIF_GTC |
      | lp1   | best-ask  | 220   | 0          | TIF_GTC |
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/MAR22 | buy  | 1      | 190   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/MAR22 | sell | 1      | 190   | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the mark price should be "190" for the market "ETH/MAR22"
    And the parties should have the following profit and loss:
      | party       | volume | unrealised pnl | realised pnl |
      | atRiskParty | 0      | 0              | -20          |
      | network     | 1      | 0              | 0            |

    # Network doesn't trade because it would be outside of price bounds
    Given the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound |
      | 190        | TRADING_MODE_CONTINUOUS | 6200    | 186       | 214       |
    When the network moves ahead "1" blocks
    And the parties should have the following profit and loss:
      | party       | volume | unrealised pnl | realised pnl |
      | network     | 1      | 0              | 0            |
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound |
      | 190        | TRADING_MODE_CONTINUOUS | 6200    | 186       | 214       |


  Scenario: Volume on the book outside liquidity price range, network unable to dispose position (0012-POSR-027)
    
    # Market configuiration
    Given the liquidity sla params named "sla-params":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.05        | 0.6                          | 1                             | 1.0                    |
    When the markets are updated:
      | id        | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | 1e-3                   | 0                         | sla-params |

    # Setup the market
    Given the initial insurance pool balance is "10000" for all the markets
    And the parties deposit on asset's general account the following amount:
      | party | asset    | amount       |
      | lp1   | USD.0.10 | 100000000000 |
      | aux1  | USD.0.10 | 10000000000  |
      | aux2  | USD.0.10 | 10000000000  |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 500000            | 0   | submission |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp1   | ETH/MAR22 | buy  | 1000   | 180   | 0                | TYPE_LIMIT | TIF_GTC | best-bid  |
      | lp1   | ETH/MAR22 | sell | 1000   | 210   | 0                | TYPE_LIMIT | TIF_GTC | best-ask  |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/MAR22 | buy  | 1      | 200   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/MAR22 | sell | 1      | 200   | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/MAR22"


    # atRiskPary opens a long position
    Given the parties deposit on asset's general account the following amount:
      | party       | asset    | amount |
      | atRiskParty | USD.0.10 | 20     |
    And the parties place the following orders:
      | party       | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1        | ETH/MAR22 | sell | 1      | 200   | 0                | TYPE_LIMIT | TIF_GTC |
      | atRiskParty | ETH/MAR22 | buy  | 1      | 200   | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the parties should have the following profit and loss:
      | party       | volume | unrealised pnl | realised pnl |
      | atRiskParty | 1      | 0              | 0            |
    And the parties should have the following margin levels:
      | party       | market id | maintenance | search | initial | release |
      | atRiskParty | ETH/MAR22 | 15          | 16     | 18      | 21      |
    And the parties should have the following account balances:
      | party       | asset    | market id | margin | general |
      | atRiskParty | USD.0.10 | ETH/MAR22 | 16     | 4       |

    # Market moves against atRiskParty whom is liquidated
    Given the parties amend the following orders:
      | party | reference | price | size delta | tif     |
      | lp1   | best-bid  | 180   | 0          | TIF_GTC |
      | lp1   | best-ask  | 220   | 0          | TIF_GTC |
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/MAR22 | buy  | 1      | 190   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/MAR22 | sell | 1      | 190   | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the mark price should be "190" for the market "ETH/MAR22"
    And the parties should have the following profit and loss:
      | party       | volume | unrealised pnl | realised pnl |
      | atRiskParty | 0      | 0              | -20          |
      | network     | 1      | 0              | 0            |

    # Network cannot disposes its position with trades outside liquidity price range
    Given the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | static mid price |
      | 190        | TRADING_MODE_CONTINUOUS | 200              |
    When the network moves ahead "1" blocks
    Then the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl |
      | network | 1      | 0              | 0            |