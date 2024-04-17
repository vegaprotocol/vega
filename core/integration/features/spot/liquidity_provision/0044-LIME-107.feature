Feature: Spot market SLA

  Scenario: 001 0044-LIME-107
    Given time is updated to "2023-07-20T00:00:00Z"

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0         | 0                  |
    Given the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |
    And the fees configuration named "fees-config-4":
      | maker fee | infrastructure fee | liquidity fee method | liquidity fee constant |
      | 0.0004    | 0.001              | METHOD_MARGINAL_COST | 0.01                   |

    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 3600    | 0.999       | 300               |

    And the liquidity sla params named "SLA-1":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 0.6                          | 2                             | 0.2                    |

    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 1              |
      | BTC | 1              |

    And the following network parameters are set:
      | name                                                | value |
      | network.markPriceUpdateMaximumFrequency             | 2s    |
      | market.liquidity.earlyExitPenalty                   | 0.25  |
      | market.liquidity.bondPenaltyParameter               | 0.2   |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0.15  |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax   | 0.3   |
      | market.liquidity.maximumLiquidityFeeFactorLevel     | 0.4   |
      | validators.epoch.length                             | 4s    |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | sla params |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-4 | price-monitoring-1 | SLA-1      |
    And the following network parameters are set:
      | name                                             | value |
      | market.liquidity.providersFeeCalculationTimeStep | 1s    |
      | market.liquidity.stakeToCcyVolume                | 1     |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 10000  |
      | party2 | BTC   | 500    |
      | lp1    | ETH   | 4000   |
      | lp1    | BTC   | 60     |
      | lp2    | ETH   | 4000   |
      | lp2    | BTC   | 60     |
      | lp3    | ETH   | 4000   |
      | lp3    | BTC   | 60     |

    And the average block duration is "1"

    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 0.2              | 20s         | 0.8            |

    When the spot markets are updated:
      | id      | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | BTC/ETH | updated-lqm-params   | 0.5                    | 0.5                       |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | BTC/ETH   | 3000              | 0.1 | submission |
      | lp2 | lp2   | BTC/ETH   | 3000              | 0.1 | submission |

    Then the network moves ahead "1" blocks
    And the network treasury balance should be "0" for the asset "ETH"
    And the global insurance pool balance should be "0" for the asset "ETH"
    And the global insurance pool balance should be "0" for the asset "BTC"
    And the party "lp1" lp liquidity fee account balance should be "0" for the market "BTC/ETH"
    Then the party "lp1" lp liquidity bond account balance should be "3000" for the market "BTC/ETH"

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode                 | auction trigger         | target stake | supplied stake | open interest |
      | 0          | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING | 4800         | 6000           | 0             |

    # place orders and generate trades
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    | only |
      | party1 | BTC/ETH   | buy  | 6      | 8     | 0                | TYPE_LIMIT | TIF_GTC | party-order5 |      |
      | party1 | BTC/ETH   | buy  | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC | party-order3 |      |
      | party2 | BTC/ETH   | sell | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC | party-order4 |      |
      | party2 | BTC/ETH   | sell | 6      | 24    | 0                | TYPE_LIMIT | TIF_GTC | party-order6 |      |

    When the network moves ahead "2" blocks

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 15    | 1    | party2 |

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 15         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 4800         | 6000           | 0             |

    Then "lp1" should have general account balance of "1000" for asset "ETH"
    Then "lp2" should have general account balance of "1000" for asset "ETH"

    #0044-LIME-108:If a liquidity provider with an active liquidity provision at the start of an epoch amends the fee level associated to this commitment during the epoch, this change will only take effect at the end of the epoch.
    #0044-LIME-109:If a liquidity provider with an active liquidity provision at the start of an epoch increases their liquidity provision staked commitment during the epoch
    #0044-LIME-110:the protocol will increase the bond to the new level if they have sufficient collateral in the settlement asset of the market to meet new commitment amount

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type   | error                                            |
      | lp1 | lp1   | BTC/ETH   | 20000             | 0.2 | amendment | commitment submission rejected, not enough stake |
    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type   |
      | lp1 | lp1   | BTC/ETH   | 2000              | 0.2 | amendment |
      | lp2 | lp2   | BTC/ETH   | 4000              | 0.4 | amendment |

    Then the party "lp1" lp liquidity bond account balance should be "3000" for the market "BTC/ETH"
    Then the party "lp2" lp liquidity bond account balance should be "4000" for the market "BTC/ETH"
    And the liquidity fee factor should be "0.1" for the market "BTC/ETH"

    Then the network moves ahead "5" blocks
    #0044-LIME-107:Lp reduces LP commitment and got slashed during the epoch, and the lower (slashed) LP bond stake will be retained
    Then the party "lp1" lp liquidity bond account balance should be "1550" for the market "BTC/ETH"
    #0044-LIME-111:at the end of the current epoch rewards / penalties are evaluated based on the balance of the bond account at start of epoch
    Then the party "lp2" lp liquidity bond account balance should be "3550" for the market "BTC/ETH"
    And the liquidity fee factor should be "0.4" for the market "BTC/ETH"

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 15         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 5600         | 5100           | 0             |

    Then the following transfers should happen:
      | from | to     | from account      | to account                    | market id | amount | asset |
      | lp1  | market | ACCOUNT_TYPE_BOND | ACCOUNT_TYPE_NETWORK_TREASURY | BTC/ETH   | 450    | ETH   |
      | lp2  | market | ACCOUNT_TYPE_BOND | ACCOUNT_TYPE_NETWORK_TREASURY | BTC/ETH   | 450    | ETH   |
    Then the network moves ahead "1" blocks

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp3 | lp3   | BTC/ETH   | 3000              | 0.1 | submission |
    Then the party "lp3" lp liquidity bond account balance should be "3000" for the market "BTC/ETH"
    Then "lp3" should have general account balance of "1000" for asset "ETH"
    #Then the network moves ahead "1" blocks

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type   |
      | lp3 | lp3   | BTC/ETH   | 2000              | 0.1 | amendment |
      
    #0044-LIME-112:A liquidity provider who reduces their liquidity provision such that the total stake on the market is still above the target stake
    Then the party "lp3" lp liquidity bond account balance should be "2000" for the market "BTC/ETH"
    Then "lp3" should have general account balance of "2000" for asset "ETH"

