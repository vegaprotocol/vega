Feature: Target stake calculation in spot markets

    Tests check the target stake of spot markets, which is calculated as a
    multiple of the maximum supplied stake over a given time window, is
    calculated correctly.

    Tests check both markets using zero decimal places, and markets using 
    non-zero decimal places.

  Background:

    # Initialise the network and register assets
    Given time is updated to "2024-01-01T00:00:00Z"
    And the following network parameters are set:
      | name                                              | value |
      | market.liquidity.earlyExitPenalty                 | 0     |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax | 0     |
    And the following assets are registered:
      | id         | decimal places | quantum |
      | USDT.0.1   | 0              | 1       |
      | BTC.0.1    | 0              | 1       |
      | USDC.2.100 | 2              | 100     |
      | ETH.2.100  | 2              | 100     |
    And the average block duration is "1"

    # Initialise the zero and non-zero decimal places spot markets
    And the spot markets:
      | id       | name     | base asset | quote asset | risk model                    | auction duration | fees         | price monitoring | decimal places | position decimal places | sla params    |
      | BTC/USDT | BTC/USDT | BTC.0.1    | USDT.0.1    | default-log-normal-risk-model | 1                | default-none | default-none     | 0              | 0                       | default-basic |
      | ETH/USDC | BTC/USDT | ETH.2.100  | USDC.2.100  | default-log-normal-risk-model | 1                | default-none | default-none     | 1              | 1                       | default-basic |

    # Deposits for assets with zero decimal places
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

    # Deposits for assets with non-zero decimal places
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


  Scenario Outline: Given a spot market using zero decimals, the target stake should be correctly set to the product of the scaling factor and maximum supplied stake over the window. (0041-TSTK-106)(0041-TSTK-108)

    # Set the target stake time window to two epochs
    Given the following network parameters are set:
      | name                    | value |
      | validators.epoch.length | 10s   |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | scaling factor   | time window |
      | lqm-params | 1.0              | <scaling factor> | 20s         |
    And the spot markets are updated:
      | id       | liquidity monitoring |
      | ETH/USDC | lqm-params           |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type    |
      | lp1 | lp1   | ETH/USDC  | 10000             | 0.02 | submission |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/USDC  | buy  | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/USDC  | sell | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "2" blocks
    Then the market data for the market "ETH/USDC" should be:
      | trading mode            | target stake   | supplied stake |
      | TRADING_MODE_CONTINUOUS | <target stake> | 10000          |

  Examples:
      | scaling factor | target stake |
      | 0              | 0            |
      | 0.5            | 5000         |
      | 1              | 10000        |


  Scenario Outline: Given a spot market using non-zero decimals, the target stake should be correctly set to the product of the scaling factor and maximum supplied stake over the window. (0041-TSTK-107)(0041-TSTK-108)

    # Set the target stake time window to two epochs
    Given the following network parameters are set:
      | name                    | value |
      | validators.epoch.length | 10s   |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | scaling factor   | time window |
      | lqm-params | 1.0              | <scaling factor> | 20s         |
    And the spot markets are updated:
      | id       | liquidity monitoring |
      | BTC/USDT | lqm-params           |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type    |
      | lp1 | lp1   | BTC/USDT  | 10000             | 0.02 | submission |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/USDT  | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/USDT  | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "2" blocks
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake   | supplied stake |
      | TRADING_MODE_CONTINUOUS | <target stake> | 10000          |

  Examples:
      | scaling factor | target stake |
      | 0              | 0            |
      | 0.5            | 5000         |
      | 1              | 10000        |


  Scenario: Given a spot market, if an LP increases or decreases their commitment the target stake should be updated correctly. (0041-TSTK-109)(0041-TSTK-110)(0041-TSTK-112)

    # Set the target stake time window to two epochs
    Given the following network parameters are set:
      | name                    | value |
      | validators.epoch.length | 10s   |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | scaling factor | time window |
      | lqm-params | 1.0              | 1              | 20s         |
    And the spot markets are updated:
      | id       | liquidity monitoring |
      | BTC/USDT | lqm-params           |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type    |
      | lp1 | lp1   | BTC/USDT  | 10000             | 0.02 | submission |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/USDT  | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/USDT  | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "2" blocks
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 10000        | 10000          |

    # LP reduces commitment, supplied and target stake increased immediately (0041-TSTK-109)
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type   |
      | lp1 | lp1   | BTC/USDT  | 20000             | 0.02 | amendment |
    When the network moves ahead "1" blocks
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 20000        | 20000          |

    # LP reduces commitment, supplied and target stake not decreased immediately (0041-TSTK-110)
    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type   |
      | lp1 | lp1   | BTC/USDT  | 10000             | 0.02 | amendment |
    When the network moves ahead "1" blocks
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 20000        | 20000          |

    # LP commitment reduction processed, supplied stake decreases
    Given the network moves ahead "1" epochs
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 20000        | 10000          |

    # Previous supplied stake value dropped, target stake decreases
    Given the network moves ahead "19" blocks
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 20000        | 10000          |
    Given the network moves ahead "1" blocks
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 10000        | 10000          |

    # Increase time window, previous supplied stake NOT included despite being within time window, target stake does not change (0041-TSTK-112)
    Given the liquidity monitoring parameters:
      | name               | triggering ratio | scaling factor | time window |
      | updated-lqm-params | 1.0              | 1.0            | 30s         |
    When the spot markets are updated:
      | id       | liquidity monitoring |
      | BTC/USDT | updated-lqm-params   |
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 10000        | 10000          |



  Scenario Outline: Given a spot market, change of market.stake.target.scalingFactor will immediately change the target stake. (0041-TSTK-111)

    # Set the target stake time window to two epochs
    Given the following network parameters are set:
      | name                    | value |
      | validators.epoch.length | 10s   |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | scaling factor | time window |
      | lqm-params | 1.0              | 1              | 20s         |
    And the spot markets are updated:
      | id       | liquidity monitoring |
      | BTC/USDT | lqm-params           |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type    |
      | lp1 | lp1   | BTC/USDT  | 10000             | 0.02 | submission |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/USDT  | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/USDT  | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "2" blocks
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 10000        | 10000          |

    Given the liquidity monitoring parameters:
      | name               | triggering ratio | scaling factor   | time window |
      | updated-lqm-params | 1.0              | <scaling factor> | 20s         |
    When the spot markets are updated:
      | id       | liquidity monitoring |
      | BTC/USDT | updated-lqm-params   |
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake   | supplied stake |
      | TRADING_MODE_CONTINUOUS | <target stake> | 10000          |

  Examples:
      | scaling factor | target stake |
      | 0              | 0            |
      | 0.5            | 5000         |
      | 1              | 10000        |


  Scenario: Given a spot market, a decrease of time_window will immediately change the length of time window over which the total stake is measured and old records will be dropped hence the target stake will immediately change. (0041-TSTK-113)

    # Set the target stake time window to two epochs
    Given the following network parameters are set:
      | name                    | value |
      | validators.epoch.length | 10s   |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | scaling factor | time window |
      | lqm-params | 1.0              | 1              | 20s         |
    And the spot markets are updated:
      | id       | liquidity monitoring |
      | BTC/USDT | lqm-params           |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type    |
      | lp1 | lp1   | BTC/USDT  | 10000             | 0.02 | submission |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/USDT  | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/USDT  | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "2" blocks
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 10000        | 10000          |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type   |
      | lp1 | lp1   | BTC/USDT  | 5000              | 0.02 | amendment |
    When the network moves ahead "1" epochs
    And the network moves ahead "1" blocks
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 10000        | 5000           |

    # Decrease time window, previous supplied stake dropped, target stake decreeases (0041-TSTK-112)
    Given the liquidity monitoring parameters:
      | name               | triggering ratio | scaling factor | time window |
      | updated-lqm-params | 1.0              | 1.0            | 0s          |
    When the spot markets are updated:
      | id       | liquidity monitoring |
      | BTC/USDT | updated-lqm-params   |
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 5000         | 5000           |
