Feature: Check penalties for early exit

# For a market with market.liquidity.earlyExitPenalty = 0.25 and 
# total stake = target stake + 40 already, a liquidity provider who reduces their
# commitment by 100 will receive a total of 85 back into their general account
# with 15 transferred into the market's insurance account (40 received without penalty,
# then the remaining 60 receiving a 25% penalty). (0044-LIME-024)

  Background:
    Given the following network parameters are set:
      | name                                                  | value |
      | market.stake.target.timeWindow                        | 24h   |
      | market.stake.target.scalingFactor                     | 1     |
      | market.liquidity.bondPenaltyParameter                 | 1     |
      | market.liquidity.targetstake.triggering.ratio         | 0.1   |
      | network.markPriceUpdateMaximumFrequency               | 0s    |
      | limits.markets.maxPeggedOrders                        | 2     |
      | validators.epoch.length                               | 5s    |
      | market.liquidity.earlyExitPenalty                     | 0.25  |
      | market.liquidity.stakeToCcyVolume                     | 1.0   |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope   | 0.19  |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax     | 1     |

    And the average block duration is "1"
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 60          | 50            | 0.2                    |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 5                 |
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.01        | 0.5                          | 1                             | 1.0                    |
    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0.5                    | 0                         | SLA        |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | party1 | ETH   | 100000000  |
      | party3 | ETH   | 100000000  |
      | party4 | ETH   | 100000000  |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type     |
      | lp1 | party1 | ETH/DEC21 | 1040              | 0.001 | submission  |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC21 | buy  | 1000   | 900   | 0                | TYPE_LIMIT | TIF_GTC | p3b1      |
      | party3 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | p3b2      |
      | party4 | ETH/DEC21 | sell | 1000   | 1100  | 0                | TYPE_LIMIT | TIF_GTC | p4s1      |
      | party4 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | p4s2      |

  Scenario: 001, Check commitment updated are handling in time order
   When the opening auction period ends for market "ETH/DEC21"
   And the auction ends with a traded volume of "10" at a price of "1000"
   Then the market data for the market "ETH/DEC21" should be:
     | mark price | trading mode            | target stake | supplied stake | open interest | best static bid price | static mid price | best static offer price |
     | 1000       | TRADING_MODE_CONTINUOUS | 1000         | 1040           | 10            | 900                   | 1000             | 1100                    |
   # Place LIMIT orders to cover our commitment
   And the parties place the following orders:
     | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
     | party1 | ETH/DEC21 | buy  | 10     | 999   | 0                | TYPE_LIMIT | TIF_GTC | party1-order1 |
     | party1 | ETH/DEC21 | sell | 10     | 1001  | 0                | TYPE_LIMIT | TIF_GTC | party1-order2 |
   When the network moves ahead "1" blocks
   Then the liquidity provisions should have the following states:
     | id  | party  | market    | commitment amount | status           |
     | lp1 | party1 | ETH/DEC21 | 1040              | STATUS_ACTIVE    |
   And the parties should have the following account balances:
     | party  | asset | market id | margin  | general  | bond  |
     | party1 | ETH   | ETH/DEC21 | 1200    | 99997760 | 1040  |    
   And the insurance pool balance should be "0" for the market "ETH/DEC21"



   # Now the party try to reduce their commitment
   And the parties submit the following liquidity provision:
    | id  | party  | market id | commitment amount | fee   | lp type     |
    | lp1 | party1 | ETH/DEC21 | 940               | 0.001 | amendment   |
   When the network moves ahead "7" blocks

   Then the liquidity provisions should have the following states:
     | id  | party  | market    | commitment amount | status           |
     | lp1 | party1 | ETH/DEC21 | 940               | STATUS_ACTIVE    |

   # general account should have 99997760-85 (=99997845)
   And the parties should have the following account balances:
     | party  | asset | market id | margin  | general  | bond  |
     | party1 | ETH   | ETH/DEC21 | 1200    | 99997845 | 940   |    
   And the insurance pool balance should be "15" for the market "ETH/DEC21"

