Feature: test margin during amending orders

  Background:

    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.bondPenaltyParameter         | 0.2   |
      | market.liquidity.targetstake.triggering.ratio | 0.1   |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e6                    | 1e6                       |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party0 | USD   | 500000    |
      | party1 | USD   | 100000000 |
      | party2 | USD   | 100000000 |
      | party3 | USD   | 100000000 |
      | party4 | USD   | 100000000 |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  @ExcessAmend
  @MTMDelta
  Scenario: 001, reduce order size, 0011-MARA-004

    Given the average block duration is "1"

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/MAR22 | 50000             | 0.001 | sell | ASK              | 500        | 20     | submission |
      | lp1 | party0 | ETH/MAR22 | 50000             | 0.001 | buy  | BID              | 500        | 20     | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/MAR22 | buy  | 20     | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party2 | ETH/MAR22 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party3 | ETH/MAR22 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |
      | party4 | ETH/MAR22 | sell | 20     | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-4 |

    When the opening auction period ends for market "ETH/MAR22"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 0.1
    And the insurance pool balance should be "0" for the market "ETH/MAR22"
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 1000      | 1000      | 35569        | 50000          | 10            |

    # check the requried balances
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  |
      | party1 | USD   | ETH/MAR22 | 19218  | 99980782 |
      | party4 | USD   | ETH/MAR22 | 93902  | 99906098 |

    #margin for party4: 20*1000*3.5569036=71139

    #check the margin levels
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/MAR22 | 16015       | 17616  | 19218   | 22421   |
      | party4 | ETH/MAR22 | 71139       | 78252  | 85366   | 99594   |

    Then the parties amend the following orders:
      | party  | reference  | price | size delta | tif     |
      | party1 | buy-ref-1  | 900   | -18        | TIF_GTC |
      | party4 | sell-ref-4 | 1100  | 20         | TIF_GTC |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  |
      | party1 | USD   | ETH/MAR22 | 1922   | 99998078 |
      | party4 | USD   | ETH/MAR22 | 170732 | 99829268 |
    #check the margin levels
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/MAR22 | 1602        | 1762   | 1922    | 2242    |
      | party4 | ETH/MAR22 | 142277      | 156504 | 170732  | 199187  |

    And the parties cancel the following orders:
      | party  | reference  |
      | party1 | buy-ref-1  |
      | party4 | sell-ref-4 |

    And the network moves ahead "1" blocks

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general   |
      | party1 | USD   | ETH/MAR22 | 0      | 100000000 |
      | party4 | USD   | ETH/MAR22 | 0      | 100000000 |
