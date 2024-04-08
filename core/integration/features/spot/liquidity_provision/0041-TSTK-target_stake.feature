Feature: Target stake calculation in spot markets

  Background:

    # Initialise the network and registr assets
    Given time is updated to "2024-01-01T00:00:00Z"
    And the following network parameters are set:
      | name                                             | value |
      | validators.epoch.length                          | 10m   |
      | market.liquidity.providersFeeCalculationTimeStep | 10s   |
      # | market.liquidity.earlyExitPenalty                | 0.25 |
      # | market.liquidity.sla.nonPerformanceBondPenaltyMax | 0.1   |
    And the following assets are registered:
      | id        | decimal places | quantum |
      | BTC.0.1   | 0              | 1       |
      | USDT.0.1  | 0              | 1       |
      | ETH.1.10  | 2              | 100     |
      | USDT.C.10 | 2              | 100     |
    And the average block duration is "1"

    # Initialise the zero and non-zero decimal places spot markets
    And the spot markets:
      | id       | name     | base asset | quote asset | risk model                    | auction duration | fees         | price monitoring | decimal places | position decimal places | sla params    |
      | BTC/USDT | BTC/USDT | BTC.0.1    | USDT.0.1    | default-log-normal-risk-model | 1                | default-none | default-none     | 0              | 0                       | default-basic |
      | ETH/USDC | BTC/USDT | ETH.2.100  | USDC.2.100  | default-log-normal-risk-model | 1                | default-none | default-none     | 1              | 1                       | default-basic |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 1.0              | 5s          | 0.9            |
    When the spot markets are updated:
      | id       | liquidity monitoring |
      | BTC/USDT | lqm-params           |
      | ETH/USDC | lqm-params           |

    # Depositis for assets with zero decimal places
    Given the parties deposit on asset's general account the following amount:
      | party | asset    | amount   |
      | lp1   | USDT.0.1 | 10000000 |
      | lp2   | USDT.0.1 | 10000000 |
      | aux1  | USDT.0.1 | 10000000 |
      | aux2  | USDT.0.1 | 10000000 |
    Given the parties deposit on asset's general account the following amount:
      | party | asset   | amount |
      | lp1   | BTC.0.1 | 10000  |
      | lp2   | BTC.0.1 | 10000  |
      | aux1  | BTC.0.1 | 10000  |
      | aux2  | BTC.0.1 | 10000  |

    # Depositis for assets with non-zero decimal places
    Given the parties deposit on asset's general account the following amount:
      | party | asset      | amount     |
      | lp1   | USDC.2.100 | 1000000000 |
      | lp2   | USDC.2.100 | 1000000000 |
      | aux1  | USDC.2.100 | 1000000000 |
      | aux2  | USDC.2.100 | 1000000000 |
    Given the parties deposit on asset's general account the following amount:
      | party | asset     | amount  |
      | lp1   | ETH.2.100 | 1000000 |
      | lp2   | ETH.2.100 | 1000000 |
      | aux1  | ETH.2.100 | 1000000 |
      | aux2  | ETH.2.100 | 1000000 |


  Scenario: For a spot market, given a scaling_factor=1 and zero asset decimals. The target stake should be set to the maximum total supplied stake over the previous time_window. (0041-TSTK-106)

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type    |
      | lp1 | lp1   | BTC/USDT  | 10000             | 0.02 | submission |
      | lp2 | lp2   | BTC/USDT  | 10000             | 0.02 | submission |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/USDT  | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/USDT  | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "2" blocks
    And the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 18000        | 20000          |


  Scenario: For a spot market, given a scaling_factor=1 and non-zero asset decimals. The target stake should be set to the maximum total supplied stake over the previous time_window. (0041-TSTK-107)

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type    |
      | lp1 | lp1   | ETH/USDC  | 1000000           | 0.02 | submission |
      | lp2 | lp2   | ETH/USDC  | 1000000           | 0.02 | submission |

    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/USDC  | buy  | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USDC  | sell | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |

    Then the network moves ahead "2" blocks
    And the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 1800000      | 2000000        |