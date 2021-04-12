Feature: Target stake

# Market risk parameters and assets don't really matter.
# We need to track open interest i.e. sum of all long positions across the parties and how they change over time

  Background:
    Given the following network parameters are set:
      | name                              | value |
      | market.stake.target.timeWindow    | 7     |
      | market.stake.target.scalingFactor | 1.5   |

    And the markets start on "2021-03-01T00:00:00Z" and expire on "2021-12-31T23:59:59Z"

    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | -1          | -11           | 0.1                    |

    And the log normal risk model named "log-normal-risk-model-1":
    | risk aversion | tau                    | mu | r     | sigma |
    | 0.000001      | 0.00011407711613050422 | 0  | -1 | -1   |

    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee | liquidity fee |
      | 0.00025   | 0.0005             | 0.001         |

    And the margin calculator named "margin-calculator-1":
    | search factor | initial factor | release factor |
    | 1.1           | 1.2            | 1.4            |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | fees          | price monitoring    | oracle config          |
      | ETH/DEC21 | BTC        | BTC   | simple-risk-model-1 | default-margin-calculator | fees-config-1 | default-none        | default-eth-for-future |
    
# Above, it says mark price but really I don't mind if we start
# with an opening auction as long as at start of the scenario
# no-one has any open positions in the market.
# So if we want to start with an auction, trade volume 1, then close out the position.

# T0 + 8 days so whatever open interest was there after the auction
# this is now out of the time window.
    And time is updated to "2021-03-08T00:00:00Z"

  Scenario:
    # setup accounts
    Given the traders deposit on asset's general account the following amount:
      | trader | asset | amount    |
      | tt_0   | BTC   | 100000000 |
      | tt_1   | BTC   | 100000000 |
      | tt_2   | BTC   | 100000000 |
      | tt_3   | BTC   | 100000000 |

#    Then I Expect the traders to have new general account:
#      | name  | asset |
#      | tt_0  | BTC   |
#      | tt_1  | BTC   |
#      | tt_2  | BTC   |
#      | tt_3  | BTC    |

    # put some volume on the book so that others can increase their
    # positions and close out if needed too
    When traders place the following orders:
      | trader | id        | type | volume | price | resulting trades | type        | tif     | reference |
      | tt_0   | ETH/DEC21 | buy  | 1000   | 90    | 0                | TYPE_LIMIT  | TIF_GTC | tt_0_0    |
      | tt_0   | ETH/DEC21 | sell | 1000   | 110   | 0                | TYPE_LIMIT  | TIF_GTC | tt_0_1    |

    # nothing should have traded, we have mark price set apriori or
    # due to auction closing.
    Then the mark price should be "100" for the market "ETH/DEC21"

    # Traders 1, 2, 3 go long
    When traders place the following orders:
      | trader | id        | type | volume | price | resulting trades | type        | tif     | reference |
      | tt_1   | ETH/DEC21 | buy  | 10     | 110   | 1                | TYPE_LIMIT  | TIF_GTC | tt_1_0    |
      | tt_2   | ETH/DEC21 | buy  | 20     | 110   | 1                | TYPE_LIMIT  | TIF_GTC | tt_2_0    |
      | tt_3   | ETH/DEC21 | buy  | 30     | 110   | 1                | TYPE_LIMIT  | TIF_GTC | tt_2_0    |

    # So now traders 1,2,3 are long 10+20+30 = 60.
    Then the mark price should be "110" for the market "ETH/DEC21"
    And the max_oi for the market "ETH/DEC21" is "60"

    # Target stake is mark_price x max_oi x target_stake_scaling_factor x rf_short
    # rf_short should have been set above to 0.1
    # target_stake = 110 x 60 x 1.5 x 0.1
    And the target_stake for the market "ETH/DEC21" is "990"

    # T0 + 8 days + 1 hour
    Then time is updated to "2021-03-08T00:01:00Z"

    # Trader 3 closes out 20
      | tt_3   | ETH/DEC21 | sell  | 20     | 90   | 1                | TYPE_LIMIT  | TIF_GTC | tt_2_1    |

    And the mark price should be "90" for the market "ETH/DEC21"

    # the maximum oi over the last 7 days is still unchanged
    And the max_oi for the market "ETH/DEC21" is "60"
    # target_stake = 90 x 60 x 1.5 x 0.1
    And the target_stake for the market "ETH/DEC21" is "810"

    # T0 + 15 days + 2 hour
    # so now the peak of 60 should have passed from window
    Then time is updated to "2021-03-15T00:02:00Z"

    And the mark price should be "90" for the market "ETH/DEC21"
    And the max_oi for the market "ETH/DEC21" is "40"
    # target_stake = 90 x 40 x 1.5 x 0.1
    And the target_stake for the market "ETH/DEC21" is "540"