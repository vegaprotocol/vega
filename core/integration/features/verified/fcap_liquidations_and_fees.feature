Feature: FCAP liquidations

    Checking interactions between parties paying fees when barely having
    enough funds to cover positions on FCAP markets.

  Background:

    # Initialise the network and register the assets
    Given the average block duration is "1"
    And the following network parameters are set:
      | name                                    | value |
      | market.fee.factors.makerFee             | 0.1   |
      | market.fee.factors.infrastructureFee    | 0.1   |
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
      | FCAP-PM/USD-1-10 | ETH        | USD-1-10 | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | price-monitoring | default-eth-for-future | 1e-3                   | 0                         | default-futures | 0              | 0                       | 100           | true   | true                 |
    And the parties submit the following liquidity provision:
      | id  | party | market id        | commitment amount | fee | lp type    |
      | lp1 | lp    | FCAP-PM/USD-1-10 | 1000000           | 0   | submission |
    And the parties place the following orders:
      | party | market id        | side | volume | price | resulting trades | type       | tif     |
      | aux1  | FCAP-PM/USD-1-10 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | FCAP-PM/USD-1-10 | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "2" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "FCAP-PM/USD-1-10"


  Scenario: Party places a limit order with not enough funds to cover fes

    Given the parties deposit on asset's general account the following amount:
      | party  | asset    | amount |
      | trader | USD-1-10 | 50     |

    Given the parties place the following orders:
      | party  | market id        | side | volume | price | resulting trades | type       | tif     |
      | aux1   | FCAP-PM/USD-1-10 | sell | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | trader | FCAP-PM/USD-1-10 | buy  | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the following trades should be executed:
      | buyer   | price | size | seller |
      | trader  | 50    | 1    | aux1   |
      | network | 50    | 1    | trader |


  Scenario: Party places a limit order which requires no fees in continuous trading but later requires fees in auction

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
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "FCAP-PM/USD-1-10"
    And the following trades should be executed:
      | buyer   | price | size | seller |
      | trader  | 50    | 1    | aux1   |
      | network | 50    | 1    | trader |


