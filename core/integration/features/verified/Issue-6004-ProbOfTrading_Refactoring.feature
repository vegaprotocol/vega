Feature: test probability of trading used in LP vol when best bid/ask is changing

  Background:

    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 10000   | 0.99        | 300               |
    And the following network parameters are set:
      | name                                          | value |
      | market.liquidity.bondPenaltyParameter       | 0.2   |
      | market.liquidity.stakeToCcyVolume           | 1.0   |
      | network.markPriceUpdateMaximumFrequency       | 0s    |
      | limits.markets.maxPeggedOrders                | 2     |
    Given the liquidity monitoring parameters:
            | name               | triggering ratio | time window | scaling factor |
            | lqm-params         | 0.1              | 24h         | 1.0            |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model              | margin calculator         | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/MAR22 | ETH        | USD   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | default-none     | default-eth-for-future | 1e-2                    | 0                         | default-futures |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount      |
      | party0 | USD   | 5000000000  |
      | party1 | USD   | 10000000000 |
      | party2 | USD   | 10000000000 |
      | party3 | USD   | 10000000000 |

    And the average block duration is "1"