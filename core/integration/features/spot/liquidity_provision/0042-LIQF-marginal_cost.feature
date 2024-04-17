Feature: Marginal cost liquidity fee selection for spot markets

  Background:

    # Initialise the network and register assets
    Given time is updated to "2024-01-01T00:00:00Z"
    And the following network parameters are set:
      | name                                              | value |
      | market.liquidity.earlyExitPenalty                 | 0     |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax | 0     |
    And the following assets are registered:
      | id       | decimal places | quantum |
      | USDT.0.1 | 0              | 1       |
      | BTC.0.1  | 0              | 1       |
    And the average block duration is "1"

    # Initialise the zero and non-zero decimal places spot markets
    And the spot markets:
      | id       | name     | base asset | quote asset | risk model                    | auction duration | fees         | price monitoring | decimal places | position decimal places | sla params    |
      | BTC/USDT | BTC/USDT | BTC.0.1    | USDT.0.1    | default-log-normal-risk-model | 1                | default-none | default-none     | 0              | 0                       | default-basic |

    # Deposits for assets with zero decimal places
    Given the parties deposit on asset's general account the following amount:
      | party | asset    | amount   |
      | lp1   | USDT.0.1 | 10000000 |
      | lp2   | USDT.0.1 | 10000000 |
      | lp3   | USDT.0.1 | 10000000 |
      | aux1  | USDT.0.1 | 10000000 |
      | aux2  | USDT.0.1 | 10000000 |
    Given the parties deposit on asset's general account the following amount:
      | party | asset   | amount |
      | aux1  | BTC.0.1 | 10000  |
      | aux2  | BTC.0.1 | 10000  |


  Scenario: An LP joining a market that is below the target stake with a higher fee bid than the current fee: their fee is used (0042-LIQF-073)

    # Set the target stake time window to five epochs
    Given the following network parameters are set:
      | name                    | value |
      | validators.epoch.length | 10s   |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | scaling factor | time window |
      | lqm-params | 1.0              | 1              | 50s         |
    And the spot markets are updated:
      | id       | liquidity monitoring |
      | BTC/USDT | lqm-params           |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type    |
      | lp1 | lp1   | BTC/USDT  | 10000             | 0.01 | submission |
      | lp2 | lp2   | BTC/USDT  | 10000             | 0.02 | submission |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/USDT  | buy  | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/USDT  | sell | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "2" blocks
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 20000        | 20000          |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type   |
      | lp1 | lp1   | BTC/USDT  | 1000              | 0.01 | amendment |
      | lp2 | lp2   | BTC/USDT  | 1000              | 0.02 | amendment |
    When the network moves ahead "1" epochs
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 20000        | 2000           |
    And the liquidity fee factor should be "0.02" for the market "BTC/USDT"

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type    |
      | lp3 | lp3   | BTC/USDT  | 1000              | 0.03 | submission |
    When the network moves ahead "1" epochs
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 20000        | 3000           |
    And the liquidity fee factor should be "0.03" for the market "BTC/USDT"


  Scenario: An LP joining a market that is below the target stake with a lower fee bid than the current fee: fee doesn't change (0042-LIQF-074)

    # Set the target stake time window to five epochs
    Given the following network parameters are set:
      | name                    | value |
      | validators.epoch.length | 10s   |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | scaling factor | time window |
      | lqm-params | 1.0              | 1              | 50s         |
    And the spot markets are updated:
      | id       | liquidity monitoring |
      | BTC/USDT | lqm-params           |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type    |
      | lp1 | lp1   | BTC/USDT  | 10000             | 0.01 | submission |
      | lp2 | lp2   | BTC/USDT  | 10000             | 0.02 | submission |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/USDT  | buy  | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/USDT  | sell | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "2" blocks
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 20000        | 20000          |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type   |
      | lp1 | lp1   | BTC/USDT  | 1000              | 0.01 | amendment |
      | lp2 | lp2   | BTC/USDT  | 1000              | 0.02 | amendment |
    When the network moves ahead "1" epochs
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 20000        | 2000           |
    And the liquidity fee factor should be "0.02" for the market "BTC/USDT"

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp3 | lp3   | BTC/USDT  | 1000              | 0   | submission |
    When the network moves ahead "1" epochs
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 20000        | 3000           |
    And the liquidity fee factor should be "0.02" for the market "BTC/USDT"


  Scenario: An LP joining a market that is above the target stake with a sufficiently large commitment to push ALL higher bids above the target stake and a lower fee bid than the current fee: their fee is used (0042-LIQF-075)
  
    # Set the target stake time window to five epochs
    Given the following network parameters are set:
      | name                    | value |
      | validators.epoch.length | 10s   |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | scaling factor | time window |
      | lqm-params | 1.0              | 0.5            | 50s         |
    And the spot markets are updated:
      | id       | liquidity monitoring |
      | BTC/USDT | lqm-params           |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type    |
      | lp1 | lp1   | BTC/USDT  | 10000             | 0.01 | submission |
      | lp2 | lp2   | BTC/USDT  | 10000             | 0.02 | submission |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/USDT  | buy  | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/USDT  | sell | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "2" blocks
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 10000        | 20000          |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp3 | lp3   | BTC/USDT  | 30000             | 0   | submission |
    When the network moves ahead "1" epochs
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 25000        | 50000          |
    And the liquidity fee factor should be "0" for the market "BTC/USDT"


  Scenario: An LP joining a market that is above the target stake with a commitment not large enough to push any higher bids above the target stake, and a lower fee bid than the current fee: the fee doesn't change (0042-LIQF-076)
  
    # Set the target stake time window to five epochs
    Given the following network parameters are set:
      | name                    | value |
      | validators.epoch.length | 10s   |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | scaling factor | time window |
      | lqm-params | 1.0              | 0.8            | 50s         |
    And the spot markets are updated:
      | id       | liquidity monitoring |
      | BTC/USDT | lqm-params           |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type    |
      | lp1 | lp1   | BTC/USDT  | 10000             | 0.01 | submission |
      | lp2 | lp2   | BTC/USDT  | 10000             | 0.02 | submission |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/USDT  | buy  | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/USDT  | sell | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "2" blocks
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 16000        | 20000          |
    And the liquidity fee factor should be "0.02" for the market "BTC/USDT"

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp3 | lp3   | BTC/USDT  | 10000             | 0   | submission |
    When the network moves ahead "1" epochs
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 24000        | 30000          |
    And the liquidity fee factor should be "0.02" for the market "BTC/USDT"


  Scenario: An LP joining a market that is above the target stake with a commitment large enough to push one of two higher bids above the target stake, and a lower fee bid than the current fee: the fee changes to the other lower bid (0042-LIQF-077)
  
    # Set the target stake time window to five epochs
    Given the following network parameters are set:
      | name                    | value |
      | validators.epoch.length | 10s   |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | scaling factor | time window |
      | lqm-params | 1.0              | 0.6            | 50s         |
    And the spot markets are updated:
      | id       | liquidity monitoring |
      | BTC/USDT | lqm-params           |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type    |
      | lp1 | lp1   | BTC/USDT  | 10000             | 0.01 | submission |
      | lp2 | lp2   | BTC/USDT  | 10000             | 0.02 | submission |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/USDT  | buy  | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/USDT  | sell | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "2" blocks
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 12000        | 20000          |
    And the liquidity fee factor should be "0.02" for the market "BTC/USDT"

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp3 | lp3   | BTC/USDT  | 10000             | 0   | submission |
    When the network moves ahead "1" epochs
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 18000        | 30000          |
    And the liquidity fee factor should be "0.01" for the market "BTC/USDT"


  Scenario: An LP joining a market that is above the target stake with a commitment large enough to push one of two higher bids above the target stake, and a higher fee bid than the current fee: the fee doesn't change (0042-LIQF-078)
  
    # Set the target stake time window to five epochs
    Given the following network parameters are set:
      | name                    | value |
      | validators.epoch.length | 10s   |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | scaling factor | time window |
      | lqm-params | 1.0              | 0.6            | 50s         |
    And the spot markets are updated:
      | id       | liquidity monitoring |
      | BTC/USDT | lqm-params           |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type    |
      | lp1 | lp1   | BTC/USDT  | 10000             | 0.01 | submission |
      | lp2 | lp2   | BTC/USDT  | 10000             | 0.02 | submission |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/USDT  | buy  | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/USDT  | sell | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "2" blocks
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 12000        | 20000          |
    And the liquidity fee factor should be "0.02" for the market "BTC/USDT"

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type    |
      | lp3 | lp3   | BTC/USDT  | 10000             | 0.03 | submission |
    When the network moves ahead "1" epochs
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 18000        | 30000          |
    And the liquidity fee factor should be "0.02" for the market "BTC/USDT"


  Scenario: An LP leaves a market that is above target stake when their fee bid is currently being used: fee changes to fee bid by the LP who takes their place in the bidding order (0042-LIQF-079)
  
    # Set the target stake time window to five epochs
    Given the following network parameters are set:
      | name                    | value |
      | validators.epoch.length | 10s   |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | scaling factor | time window |
      | lqm-params | 1.0              | 1              | 50s         |
    And the spot markets are updated:
      | id       | liquidity monitoring |
      | BTC/USDT | lqm-params           |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type    |
      | lp1 | lp1   | BTC/USDT  | 10000             | 0.01 | submission |
      | lp2 | lp2   | BTC/USDT  | 10000             | 0.02 | submission |
      | lp3 | lp3   | BTC/USDT  | 10000             | 0.03 | submission |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/USDT  | buy  | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/USDT  | sell | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "2" blocks
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 30000        | 30000          |
    And the liquidity fee factor should be "0.03" for the market "BTC/USDT"

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type      |
      | lp3 | lp3   | BTC/USDT  | 10000             | 0.03 | cancellation |
    When the network moves ahead "1" epochs
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 30000        | 20000          |
    And the liquidity fee factor should be "0.02" for the market "BTC/USDT"


  Scenario: An LP leaves a market that is above target stake when their fee bid is lower than the one currently being used and their commitment size changes the LP that meets the target stake: fee changes to fee bid by the LP that is now at the place in the bid order to provide the target stake  (0042-LIQF-080)
  
    # Set the target stake time window to five epochs
    Given the following network parameters are set:
      | name                    | value |
      | validators.epoch.length | 10s   |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | scaling factor | time window |
      | lqm-params | 1.0              | 0.5            | 50s         |
    And the spot markets are updated:
      | id       | liquidity monitoring |
      | BTC/USDT | lqm-params           |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type    |
      | lp1 | lp1   | BTC/USDT  | 10000             | 0.01 | submission |
      | lp2 | lp2   | BTC/USDT  | 10000             | 0.02 | submission |
      | lp3 | lp3   | BTC/USDT  | 10000             | 0.03 | submission |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/USDT  | buy  | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/USDT  | sell | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "2" blocks
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 15000        | 30000          |
    And the liquidity fee factor should be "0.02" for the market "BTC/USDT"

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type      |
      | lp1 | lp1   | BTC/USDT  | 10000             | 0.01 | cancellation |
    When the network moves ahead "1" epochs
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 15000        | 20000          |
    And the liquidity fee factor should be "0.03" for the market "BTC/USDT"


  Scenario: An LP leaves a market that is above target stake when their fee bid is lower than the one currently being used. The loss of their commitment doesn't change which LP meets the target stake: fee doesn't change (0042-LIQF-081)
  
    # Set the target stake time window to five epochs
    Given the following network parameters are set:
      | name                    | value |
      | validators.epoch.length | 10s   |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | scaling factor | time window |
      | lqm-params | 1.0              | 0.4            | 50s         |
    And the spot markets are updated:
      | id       | liquidity monitoring |
      | BTC/USDT | lqm-params           |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type    |
      | lp1 | lp1   | BTC/USDT  | 2000              | 0.01 | submission |
      | lp2 | lp2   | BTC/USDT  | 14000             | 0.02 | submission |
      | lp3 | lp3   | BTC/USDT  | 14000             | 0.03 | submission |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/USDT  | buy  | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/USDT  | sell | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "2" blocks
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 12000        | 30000          |
    And the liquidity fee factor should be "0.02" for the market "BTC/USDT"

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type      |
      | lp1 | lp1   | BTC/USDT  | 2000              | 0.01 | cancellation |
    When the network moves ahead "1" epochs
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 12000        | 28000          |
    And the liquidity fee factor should be "0.02" for the market "BTC/USDT"


  Scenario: Given the fee setting method is marginal cost. An LP leaves a spot market that is above target stake when their fee bid is higher than the one currently being used: fee doesn't change (0042-LIQF-106)

    # Set the target stake time window to five epochs
    Given the following network parameters are set:
      | name                    | value |
      | validators.epoch.length | 10s   |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | scaling factor | time window |
      | lqm-params | 1.0              | 0.4            | 50s         |
    And the spot markets are updated:
      | id       | liquidity monitoring |
      | BTC/USDT | lqm-params           |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type    |
      | lp1 | lp1   | BTC/USDT  | 2000              | 0.01 | submission |
      | lp2 | lp2   | BTC/USDT  | 14000             | 0.02 | submission |
      | lp3 | lp3   | BTC/USDT  | 14000             | 0.03 | submission |

    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | BTC/USDT  | buy  | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | BTC/USDT  | sell | 10     | 10000 | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "2" blocks
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 12000        | 30000          |
    And the liquidity fee factor should be "0.02" for the market "BTC/USDT"

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee  | lp type      |
      | lp3 | lp3   | BTC/USDT  | 14000             | 0.01 | cancellation |
    When the network moves ahead "1" epochs
    Then the market data for the market "BTC/USDT" should be:
      | trading mode            | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | 12000        | 16000          |
    And the liquidity fee factor should be "0.02" for the market "BTC/USDT"