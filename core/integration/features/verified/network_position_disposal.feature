Feature: Network disposing position

  Background:

    Given the average block duration is "1"

    Given the following network parameters are set:
      | name                                    | value |
      | market.value.windowLength               | 1h    |
      | network.markPriceUpdateMaximumFrequency | 0s    |
    And the following assets are registered:
      | id  | decimal places | quantum |
      | USD | 0              | 10      |

    Given the liquidation strategies:
      | name             | disposal step | disposal fraction | full disposal size | max fraction consumed |
      | disposal-strat-1 | 5             | 0.2               | 0                  | 0.1                   |
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau           | mu | r | sigma |
      | 0.000001      | 0.00000380258 | 0  | 0 | 1.5   |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 100000  | 0.99        | 3                 |
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 0.85                         | 1                             | 0.5                    |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees         | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor | sla params | liquidation strategy |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | default-none | price-monitoring-1 | default-eth-for-future | 0.001                  | 0                         | SLA        | disposal-strat-1     |


  Scenario: Network takes over distressed position and disposes position over time

    Given the parties deposit on asset's general account the following amount:
      | party       | asset | amount       |
      | lp1         | USD   | 100000000000 |
      | aux1        | USD   | 10000000000  |
      | aux2        | USD   | 10000000000  |
      | atRiskParty | USD   | 180          |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 500000            | 0   | submission |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp1   | ETH/MAR22 | buy  | 1000   | 999   | 0                | TYPE_LIMIT | TIF_GTC | best-bid  |
      | lp1   | ETH/MAR22 | sell | 1000   | 1001  | 0                | TYPE_LIMIT | TIF_GTC | best-ask  |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/MAR22 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/MAR22 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/MAR22"
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            |
      | 1000       | TRADING_MODE_CONTINUOUS |

    Given the parties place the following orders:
      | party       | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1        | ETH/MAR22 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | atRiskParty | ETH/MAR22 | sell | 10     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the network moves ahead "1" blocks

    Then the parties should have the following profit and loss:
      | party       | volume | unrealised pnl | realised pnl |
      | atRiskParty | -10    | 0              | 0            |
    And the parties should have the following margin levels:
      | party       | market id | maintenance | search | initial | release |
      | atRiskParty | ETH/MAR22 | 156         | 171    | 187     | 218     |
    And the parties should have the following account balances:
      | party       | asset | market id | margin | general |
      | atRiskParty | USD   | ETH/MAR22 | 175    | 5       |

    Given the parties amend the following orders:
      | party | reference | price | size delta | tif     |
      | lp1   | best-ask  | 1011  | 0          | TIF_GTC |
      | lp1   | best-bid  | 1009  | 0          | TIF_GTC |
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/MAR22 | buy  | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/MAR22 | sell | 1      | 1010  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the network moves ahead "1" blocks
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            |
      | 1010       | TRADING_MODE_CONTINUOUS |
    Then debug trades

    Then the parties should have the following profit and loss:
      | party       | market id | volume | unrealised pnl | realised pnl |
      | atRiskParty | ETH/MAR22 | 0      | 0              | -180         |
    And the following network trades should be executed:
      | party       | aggressor side | volume |
      | atRiskParty | sell           | 10     |
    And clear trade events

    # Position size is 10, next disposal should be ceil(10*0.2)=ceil(2)=2
    Then the network moves ahead "5" blocks
    And the following network trades should be executed:
      | party | aggressor side | volume |
      | lp1   | buy            | 2      |
    And clear trade events

    # Position size is 8, next disposal should be ceil(8*0.2)=ceil(1.6)=2
    Then the network moves ahead "5" blocks
    Then debug trades
    And the following network trades should be executed:
      | party | aggressor side | volume |
      | lp1   | buy            | 2      |
    And clear trade events

    # Position size is 6, next disposal should be ceil(6*0.2)=ceil(1.4)=2
    Then the network moves ahead "5" blocks
    Then debug trades
    And the following network trades should be executed:
      | party | aggressor side | volume |
      | lp1   | buy            | 2      |
    And clear trade events

    # Position size is 4, next disposal should be ceil(10*0.2)=ceil(2)=2
    Then the network moves ahead "5" blocks
    And the following network trades should be executed:
      | party | aggressor side | volume |
      | lp1   | buy            | 1      |
    And clear trade events

    # Position size is 3, next disposal should be ceil(10*0.2)=ceil(2)=2
    Then the network moves ahead "5" blocks
    And the following network trades should be executed:
      | party | aggressor side | volume |
      | lp1   | buy            | 1      |
    And clear trade events

    # Position size is 2, next disposal should be ceil(10*0.2)=ceil(2)=2
    Then the network moves ahead "5" blocks
    And the following network trades should be executed:
      | party | aggressor side | volume |
      | lp1   | buy            | 1      |
    And clear trade events

    # Position size is 1, next disposal should be ceil(10*0.2)=ceil(2)=2
    Then the network moves ahead "5" blocks
    And the following network trades should be executed:
      | party | aggressor side | volume |
      | lp1   | buy            | 1      |
    And clear trade events








    
    
