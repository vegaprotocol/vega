Feature: Target stake implementation ../spec/0041-target-stake.md

# Market risk parameters and assets don't really matter.
# We need to track open interst i.e. sum of all long positions across the parties and how they change over time

  Background:
    Given the network parameter target_stake_time_window is 7 days
    Given the network parameter target_stake_scaling_factor is 1.5
    Given the markets starts on "2021-03-01T00:00:00Z" and expires on "2021-12-31T23:59:59Z"
    And the execution engine has these markets:
      | name      | baseName | quoteName | asset | markprice | risk model | lamd/long | tau/short | mu | r  | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode | makerFee | infrastructureFee | liquidityFee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | Prob of trading |
      | ETH/DEC21 | ETH      | BTC       | BTC   | 100       | simple     | 0.1       | 0.1       | -1 | -1 | -1    | 1.4            | 1.2            | 1.1           | 100             | 0           | continuous   | 0.00025  | 0.0005            | 0.001        |                 0  |                |             |                 | 0.1             |

# Above, it says mark price but really I don't mind if we start
# with an opening auction as long as at start of the scenario
# no-one has any open positions in the market.
# So if we want to start with an auction, trade volume 1, then close out the position.

# T0 + 8 days so whatever open interest was there after the auction
# this is now out of the time window.
    Then the time is updated to "2021-03-08T00:00:00Z"

  Scenario:
    # setup accounts
    Given the following taders:
      | name  | amount    |
      | tt_0  | 100000000 |
      | tt_1  | 100000000 |
      | tt_2  | 100000000 |
      | tt_3  | 100000000 |

    Then I Expect the traders to have new general account:
      | name  | asset |
      | tt_0  | BTC   |
      | tt_1  | BTC   |
      | tt_2  | BTC   |
      | tt_3  | BTC    |

    # put some volume on the book so that others can increase their
    # positions and close out if needed too
    Then traders place following orders with references:
      | trader | id        | type | volume | price | resulting trades | type        | tif     | reference |
      | tt_0   | ETH/DEC21 | buy  | 1000   | 90    | 0                | TYPE_LIMIT  | TIF_GTC | tt_0_0    |
      | tt_0   | ETH/DEC21 | sell | 1000   | 110   | 0                | TYPE_LIMIT  | TIF_GTC | tt_0_1    |

    # nothing should have traded, we have mark price set apriori or
    # due to auction closing.
    And the mark price for the market "ETH/DEC21" is "100"

    # Traders 1, 2, 3 go long
    Then traders place following orders with references:
      | trader | id        | type | volume | price | resulting trades | type        | tif     | reference |
      | tt_1   | ETH/DEC21 | buy  | 10     | 110   | 1                | TYPE_LIMIT  | TIF_GTC | tt_1_0    |
      | tt_2   | ETH/DEC21 | buy  | 20     | 110   | 1                | TYPE_LIMIT  | TIF_GTC | tt_2_0    |
      | tt_3   | ETH/DEC21 | buy  | 30     | 110   | 1                | TYPE_LIMIT  | TIF_GTC | tt_2_0    |

    # So now traders 1,2,3 are long 10+20+30 = 60.
    And the mark price for the market "ETH/DEC21" is "110"
    And the max_oi for the market "ETH/DEC21" is "60"

    # Target stake is mark_price x max_oi x target_stake_scaling_factor x rf_short
    # rf_short should have been set above to 0.1
    # target_stake = 110 x 60 x 1.5 x 0.1
    And the target_stake for the market "ETH/DEC21" is "990"

    # T0 + 8 days + 1 hour
    Then the time is updated to "2021-03-08T00:01:00Z"

    # Trader 3 closes out 20
      | tt_3   | ETH/DEC21 | sell  | 20     | 90   | 1                | TYPE_LIMIT  | TIF_GTC | tt_2_1    |

    And the mark price for the market "ETH/DEC21" is "90"

    # the maximum oi over the last 7 days is still unchanged
    And the max_oi for the market "ETH/DEC21" is "60"
    # target_stake = 90 x 60 x 1.5 x 0.1
    And the target_stake for the market "ETH/DEC21" is "810"

    # T0 + 15 days + 2 hour
    # so now the peak of 60 should have passed from window
    Then the time is updated to "2021-03-15T00:02:00Z"

    And the mark price for the market "ETH/DEC21" is "90"
    And the max_oi for the market "ETH/DEC21" is "40"
    # target_stake = 90 x 40 x 1.5 x 0.1
    And the target_stake for the market "ETH/DEC21" is "540"