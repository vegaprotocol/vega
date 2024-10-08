Feature: FCAP liquidations

    vega-market-sim fuzz testing shows parties being liquidated after
    opening positions with market orders.

    Test replicates behaviour and shows network liquidating party.

  Background:

    # Initialise the network and register the assets
    Given the average block duration is "1"
    And the following network parameters are set:
      | name                                    | value |
      | market.fee.factors.makerFee             | 0     |
      | market.fee.factors.infrastructureFee    | 0     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

    And the following assets are registered:
      | id       | decimal places | quantum |
      | USD-1-10 | 0              | 1       |

    # Initialise the parties and deposit assets
    Given the parties deposit on asset's general account the following amount:
      | party | asset    | amount       |
      | lp    | USD-1-10 | 100000000000 |
      | aux1  | USD-1-10 | 100000000000 |
      | aux2  | USD-1-10 | 100000000000 |

    # Setup the FCAP market in continuous trading
    Given the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 3600    | 0.99        | 5                 |
    Given the markets:
      | id               | quote name | asset    | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | decimal places | position decimal places | max price cap | binary | fully collateralised |
      | FCAP/USD-1-10    | ETH        | USD-1-10 | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e-3                   | 0                         | default-futures | 0              | 0                       | 100           | true   | true                 |
      | FCAP-PM/USD-1-10 | ETH        | USD-1-10 | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | price-monitoring | default-eth-for-future | 1e-3                   | 0                         | default-futures | 0              | 0                       | 100           | true   | true                 |
    And the parties submit the following liquidity provision:
      | id  | party | market id     | commitment amount | fee | lp type    |
      | lp1 | lp    | FCAP/USD-1-10 | 1000000           | 0   | submission |
    And the parties place the following orders:
      | party | market id     | side | volume | price | resulting trades | type       | tif     |
      | aux1  | FCAP/USD-1-10 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | FCAP/USD-1-10 | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties submit the following liquidity provision:
      | id  | party | market id        | commitment amount | fee | lp type    |
      | lp1 | lp    | FCAP-PM/USD-1-10 | 1000000           | 0   | submission |
    And the parties place the following orders:
      | party | market id        | side | volume | price | resulting trades | type       | tif     |
      | aux1  | FCAP-PM/USD-1-10 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | FCAP-PM/USD-1-10 | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "2" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "FCAP/USD-1-10"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "FCAP-PM/USD-1-10"



  @CappedF @NoPerp
  Scenario: Party opens a short position with a market order and is liquidiated at the next mark to market.

    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 5s    |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset    | amount |
      | trader | USD-1-10 | 50     |

    Given the parties place the following orders:
      | party  | market id     | side | volume | price | resulting trades | type        | tif     | error               |
      | aux1   | FCAP/USD-1-10 | sell | 1      | 60    | 0                | TYPE_LIMIT  | TIF_GTC |                     |
      | trader | FCAP/USD-1-10 | buy  | 1      | 0     | 0                | TYPE_MARKET | TIF_FOK | margin check failed |
    And the parties should have the following account balances:
      | party  | asset    | market id     | margin | general |
      | trader | USD-1-10 | FCAP/USD-1-10 | 0      | 50      |


  @CappedF @NoPerp
  Scenario: Party place a limit order with not enough to cover fes

    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 5s    |
      | market.fee.factors.infrastructureFee    | 0.1   |


    Given the parties deposit on asset's general account the following amount:
      | party  | asset    | amount |
      | trader | USD-1-10 | 50     |

    Given the parties place the following orders:
      | party  | market id        | side | volume | price | resulting trades | type       | tif     |
      | trader | FCAP-PM/USD-1-10 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    And the parties should have the following account balances:
      | party  | asset    | market id        | margin | general |
      | trader | USD-1-10 | FCAP-PM/USD-1-10 | 50     | 0       |

    And the market data for the market "FCAP-PM/USD-1-10" should be:
      | mark price | trading mode            | horizon | min bound | max bound |
      | 50         | TRADING_MODE_CONTINUOUS | 3600    | 48        | 52        |
    Given the parties place the following orders:
      | party | market id        | side | volume | price | resulting trades | type       | tif     | reference          |
      | aux1  | FCAP-PM/USD-1-10 | buy  | 1      | 60    | 0                | TYPE_LIMIT | TIF_GTC | auction-order-aux1 |
      | aux2  | FCAP-PM/USD-1-10 | sell | 1      | 60    | 0                | TYPE_LIMIT | TIF_GTC | auction-order-aux2 |
    When the network moves ahead "1" blocks
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "FCAP-PM/USD-1-10"
    Then the parties cancel the following orders:
      | party | reference          |
      | aux1  | auction-order-aux1 |
      | aux2  | auction-order-aux2 |
    Given the parties place the following orders:
      | party | market id        | side | volume | price | resulting trades | type       | tif     |
      | aux1  | FCAP-PM/USD-1-10 | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
    Then the network moves ahead "10" blocks
    Then debug trades
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "FCAP-PM/USD-1-10"

