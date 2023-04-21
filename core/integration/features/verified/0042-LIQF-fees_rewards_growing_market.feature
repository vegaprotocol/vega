Feature:

  Background:

    Given the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 500         | 500           | 0.1                    |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |
    And the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 1h    |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 2.5   |
      | market.liquidity.targetstake.triggering.ratio       | 0     |
      | market.liquidity.providers.fee.distributionTimeStep | 10m   |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/MAR22 | USD        | USD   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 0.5                    | 0                         |


  @FeeRound
  Scenario: 001 2 LPs joining at start, unequal commitments. Checking calculation of equity-like-shares and liquidity-fee-distribution in a market with small growth (0042-LIQF-008)

    # Scenario has 6 market periods:

    # - 0th period (bootstrap period): no LP changes, no trades
    # - 1st period: 1 LPs decrease commitment, some trades occur
    # - 2nd period: 1 LPs increase commitment, some trades occur
    # - 3rd period: 2 LPs decrease commitment, some trades occur
    # - 4th period: 2 LPs increase commitment, some trades occur
    # - 5th period: 1 LPs decrease commitment, 1 LPs increase commitment, some trades occur


    # Scenario moves ahead to next market period by:

    # - moving ahead "1" blocks to trigger the next liquidity distribution
    # - moving ahead "1" blocks to trigger the next market period


    # Following checks occur in each market where trades:

    # - Check transfers from the price taker to the market-liquidity-pool are correct
    # - Check accumulated-liquidity-fees are non-zero and correct
    # - Check equity-like-shares are correct
    # - Check transfers from the market-liquidity-pool to the liquidity-providers are correct
    # - Check accumulated-liquidity-fees are zero

    Given the average block duration is "1801"

    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | lp1    | USD   | 100000000 |
      | lp2    | USD   | 100000000 |
      | party1 | USD   | 100000    |
      | party2 | USD   | 100000    |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | buy  | MID              | 3          | 1      | submission |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | sell | MID              | 3          | 1      |            |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 46000             | 0.002 | buy  | MID              | 3          | 1      | submission |
      | lp2 | lp2   | ETH/MAR22 | 46000             | 0.002 | sell | MID              | 3          | 1      |            |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |


    # 0th period (bootstrap period): no LP changes, no trades
    Then the opening auction period ends for market "ETH/MAR22"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 20   | party2 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 500       | 1500      | 5000         | 50000          | 20            |

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.08              | 4000                    |
      | lp2   | 0.92              | 50000                   |

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"


    When the network moves ahead "2" blocks:

    # -------------------------------------------------------------------------------------------------------------------
    # -------------------------------------------------------------------------------------------------------------------


    # 1st period: 1 LPs decrease commitment, positive growth:
    # -------------------------------------------------------------------------------------------------------------------

    # Check equity-like-shares before liquidity amendment
    Given the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.08              | 4000                    |
      | lp2   | 0.92              | 50000                   |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | buy  | MID              | 3          | 1      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | sell | MID              | 3          | 1      | amendment |

    # Confirm equity-like-shares updated immediately after liquidity amendment
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0612244897959184 | 4000                    |
      | lp2   | 0.9387755102040816 | 50000                   |

    # -------------------------------------------------------------------------------------------------------------------

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 21     | 1001  | 1                | TYPE_LIMIT | TIF_GTC |


    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1001  | 21   | lp2    |

    # CALCULATION:
    # liquidity_fee = ceil(volume * price * liquidity_fee_factor) = ceil(21 * 1001 * 0.002) = 43

    And the following transfers should happen:
      | from   | to     | from account         | to account                  | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY | ETH/MAR22 | 43     | USD   |

    And the accumulated liquidity fees should be "43" for the market "ETH/MAR22"

    And the market data for the market "ETH/MAR22" should be:
      | mark price | last traded price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | 1001              | TRADING_MODE_CONTINUOUS | 1       | 500       | 1500      | 10250        | 49000          | 41            |

    # -------------------------------------------------------------------------------------------------------------------

    # Check equity-like-shares before network moves forward
    When the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0612244897959184 | 4000                    |
      | lp2   | 0.9387755102040816 | 50000                   |

    # Trigger next liquidity fee distribution without triggering next market period
    And the network moves ahead "1" blocks:

    # Confirm equity-like-shares are unchanged by network moving forwards (as new market period not entered)
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0612244897959184 | 4000                    |
      | lp2   | 0.9387755102040816 | 50000                   |

    And the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 2      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 40     | USD   |

    And the accumulated liquidity fees should be "1" for the market "ETH/MAR22"

    # -------------------------------------------------------------------------------------------------------------------

    # Check equity-like-shares before network moves forward
    When the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0612244897959184 | 4000                    |
      | lp2   | 0.9387755102040816 | 50000                   |

    # Trigger entry into next market period
    And the network moves ahead "1" blocks:

    # Confirm equity-like-shares are unchanged by the network moving forwards (as virtual-stakes scaled by same factor, r)
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0612244897959184 | 4000                    |
      | lp2   | 0.9387755102040816 | 50000                   |

    # -------------------------------------------------------------------------------------------------------------------
    # -------------------------------------------------------------------------------------------------------------------


    # 2nd period: 1 LPs increase commitment, positive growth:
    # -------------------------------------------------------------------------------------------------------------------

    # Check equity-like-shares before liquidity amendment
    Given the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0612244897959184 | 4000                    |
      | lp2   | 0.9387755102040816 | 50000                   |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | buy  | MID              | 3          | 1      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | sell | MID              | 3          | 1      |           |

    # Confirm equity-like-shares updated immediately after liquidity amendment
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0795418016037822 | 15812.68125             |
      | lp2   | 0.9204581983962178 | 50000                   |

    # -------------------------------------------------------------------------------------------------------------------

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 22     | 1001  | 1                | TYPE_LIMIT | TIF_GTC |

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1001  | 22   | lp2    |

    # CALCULATION:
    # liquidity_fee = ceil(volume * price * liquidity_fee_factor) =  ceil(22 * 1001 * 0.002) = 45

    And the following transfers should happen:
      | from   | to     | from account         | to account                  | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY | ETH/MAR22 | 45     | USD   |

    And the accumulated liquidity fees should be "46" for the market "ETH/MAR22"

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001       | TRADING_MODE_CONTINUOUS | 1       | 502       | 1500      | 15765        | 50000          | 63            |

    # -------------------------------------------------------------------------------------------------------------------

    # Check equity-like-shares before network moves forward
    Given the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0795418016037822 | 15812.68125             |
      | lp2   | 0.9204581983962178 | 50000                   |

    # Trigger next liquidity fee distribution without triggering next period
    When the network moves ahead "1" blocks:

    # Confirm equity-like-shares are unchanged by network moving forwards (as new market period not entered)
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0795418016037822 | 15812.68125             |
      | lp2   | 0.9204581983962178 | 50000                   |

    And the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 3      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 42     | USD   |

    And the accumulated liquidity fees should be "1" for the market "ETH/MAR22"

    # -------------------------------------------------------------------------------------------------------------------

    # Check equity-like-shares before network moves forward
    When the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0795418016037822 | 15812.68125             |
      | lp2   | 0.9204581983962178 | 50000                   |

    # Trigger entry into next market period
    And the network moves ahead "1" blocks:

    # Confirm equity-like-shares are unchanged by the network moving forwards (as virtual-stakes scaled by same factor, r)
    When the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0795418016037822 | 15812.68125             |
      | lp2   | 0.9204581983962178 | 50000                   |

    # ----------------------------------------------------------------------------------
    # ----------------------------------------------------------------------------------


    # 3rd period: 2 LPs decrease commitment, positive growth:
    # ----------------------------------------------------------------------------------

    # Check equity-like-shares before liquidity amendment
    Given the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0795418016037822 | 15812.68125             |
      | lp2   | 0.9204581983962178 | 50000                   |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | buy  | MID              | 3          | 1      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | sell | MID              | 3          | 1      |           |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp2 | lp2   | ETH/MAR22 | 45000             | 0.002 | buy  | MID              | 3          | 1      | amendment |
      | lp2 | lp2   | ETH/MAR22 | 45000             | 0.002 | sell | MID              | 3          | 1      |           |

    # Confirm equity-like-shares updated immediately after liquidity amendment
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0621352630754675 | 15812.68125             |
      | lp2   | 0.9378647369245325 | 50000                   |

    # ----------------------------------------------------------------------------------

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 23     | 1001  | 2                | TYPE_LIMIT | TIF_GTC |

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1001  | 3    | lp1    |
      | party1 | 1001  | 21   | lp2    |

    # CALCULATION:
    # liquidity_fee = ceil(volume * price * liquidity_fee_factor) =  ceil(2 * 1001 * 0.002) + ceil(18 * 1001 * 0.002) = 49

    When the following transfers should happen:
      | from   | to     | from account         | to account                  | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY | ETH/MAR22 | 48     | USD   |

    Then the accumulated liquidity fees should be "49" for the market "ETH/MAR22"

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001       | TRADING_MODE_CONTINUOUS | 1       | 502       | 1500      | 21521        | 48000          | 86            |

    # -------------------------------------------------------------------------------------------------------------------

    # Check equity-like-shares before the network moves forward
    Given the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0621352630754675 | 15812.68125             |
      | lp2   | 0.9378647369245325 | 50000                   |

    # Trigger next liquidity fee distribution without triggering next period
    When the network moves ahead "1" blocks:

    # Confirm equity-like-shares are unchanged by network moving forwards (as new market period not entered)
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0621352630754675 | 15812.68125             |
      | lp2   | 0.9378647369245325 | 50000                   |

    And the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 3      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 45     | USD   |

    And the accumulated liquidity fees should be "1" for the market "ETH/MAR22"

    # -------------------------------------------------------------------------------------------------------------------

    # Check equity-like-shares before network moves forward
    Given the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0621352630754675 | 15812.68125             |
      | lp2   | 0.9378647369245325 | 50000                   |

    # Trigger entry into next market period
    When the network moves ahead "1" blocks:

    # Confirm equity-like-shares are unchanged by the network moving forwards (as virtual-stakes scaled by same factor, r)
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0621352630754675 | 15812.68125             |
      | lp2   | 0.9378647369245325 | 50000                   |

    # -------------------------------------------------------------------------------------------------------------------
    # -------------------------------------------------------------------------------------------------------------------


    # 4nd period: 2 LPs increase commitment, positive growth:
    # -------------------------------------------------------------------------------------------------------------------

    # Check equity-like-shares before liquidity amendment
    Given the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0621352630754675 | 15812.68125             |
      | lp2   | 0.9378647369245325 | 50000                   |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | buy  | MID              | 3          | 1      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | sell | MID              | 3          | 1      |           |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp2 | lp2   | ETH/MAR22 | 46000             | 0.002 | buy  | MID              | 3          | 1      | amendment |
      | lp2 | lp2   | ETH/MAR22 | 46000             | 0.002 | sell | MID              | 3          | 1      |           |

    # Confirm equity-like-shares updated immediately after liquidity amendment
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0784675524761938 | 25014.390259105092498   |
      | lp2   | 0.9215324475238062 | 50078.6851584004428259  |

    # -------------------------------------------------------------------------------------------------------------------

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 24     | 1001  | 2                | TYPE_LIMIT | TIF_GTC |

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1001  | 3    | lp1    |
      | party1 | 1001  | 21   | lp2    |

    # CALCULATION:
    # liquidity_fee = ceil(volume * price * liquidity_fee_factor) =  ceil(3 * 1001 * 0.002) + ceil(3 * 1001 * 0.002) = 50

    When the following transfers should happen:
      | from   | to     | from account         | to account                  | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY | ETH/MAR22 | 50     | USD   |

    Then the accumulated liquidity fees should be "51" for the market "ETH/MAR22"

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001       | TRADING_MODE_CONTINUOUS | 1       | 502       | 1500      | 27527        | 50000          | 110           |

    # -------------------------------------------------------------------------------------------------------------------

    # Check equity-like-shares before network moves forward
    Given the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0784675524761938 | 25014.390259105092498   |
      | lp2   | 0.9215324475238062 | 50078.6851584004428259  |

    # Trigger next liquidity fee distribution without triggering next period
    When the network moves ahead "1" blocks:

    # Confirm equity-like-shares are unchanged by network moving forwards (as new market period not entered)
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0784675524761938 | 25014.390259105092498   |
      | lp2   | 0.9215324475238062 | 50078.6851584004428259  |

    And the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 4      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 46     | USD   |

    And the accumulated liquidity fees should be "1" for the market "ETH/MAR22"

    # -------------------------------------------------------------------------------------------------------------------

    # Check equity-like-shares before network moves forward
    Given the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0784675524761938 | 25014.390259105092498   |
      | lp2   | 0.9215324475238062 | 50078.6851584004428259  |

    # Trigger entry into next market period
    When the network moves ahead "1" blocks:

    # Confirm equity-like-shares are unchanged by the network moving forwards (as virtual-stakes scaled by same factor, r)
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0784675524761938 | 25014.390259105092498   |
      | lp2   | 0.9215324475238062 | 50078.6851584004428259  |

    # -------------------------------------------------------------------------------------------------------------------
    # -------------------------------------------------------------------------------------------------------------------


    # 5th period: 1 LPs decrease commitment 1 LPs increase commitment, some trades occur
    # -------------------------------------------------------------------------------------------------------------------

    # Check equity-like-shares before liquidity amendment
    Given the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0784675524761938 | 25014.390259105092498   |
      | lp2   | 0.9215324475238062 | 50078.6851584004428259  |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | buy  | MID              | 3          | 1      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | sell | MID              | 3          | 1      | amendment |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp2 | lp2   | ETH/MAR22 | 47000             | 0.002 | buy  | BID              | 1          | 2      | amendment |
      | lp2 | lp2   | ETH/MAR22 | 47000             | 0.002 | buy  | MID              | 3          | 1      | amendment |
      | lp2 | lp2   | ETH/MAR22 | 47000             | 0.002 | sell | ASK              | 1          | 2      | amendment |
      | lp2 | lp2   | ETH/MAR22 | 47000             | 0.002 | sell | MID              | 3          | 1      | amendment |

    # Confirm equity-like-shares updated immediately after liquidity amendment
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0589326838400726 | 25014.390259105092498   |
      | lp2   | 0.9410673161599274 | 50178.9876096722076451  |

    # -------------------------------------------------------------------------------------------------------------------

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 25     | 1001  | 2                | TYPE_LIMIT | TIF_GTC |

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1001  | 3    | lp1    |
      | party1 | 1001  | 22   | lp2    |

    # liquidity_fee = ceil(volume * price * liquidity_fee_factor) =  ceil(3 * 1001 * 0.002) + ceil(3 * 1001 * 0.002) = 52

    When the following transfers should happen:
      | from   | to     | from account         | to account                  | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY | ETH/MAR22 | 52     | USD   |

    # CALCULATION:
    Then the accumulated liquidity fees should be "53" for the market "ETH/MAR22"

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001       | TRADING_MODE_CONTINUOUS | 1       | 502       | 1500      | 33783        | 50000          | 135           |

    # -------------------------------------------------------------------------------------------------------------------

    # Check equity-like-shares before network moves forward
    Given the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0589326838400726 | 25014.390259105092498   |
      | lp2   | 0.9410673161599274 | 50178.9876096722076451  |

    # Trigger next liquidity fee distribution without triggering next period
    When the network moves ahead "1" blocks:

    # Confirm equity-like-shares are unchanged by network moving forwards (as new market period not entered)
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0589326838400726 | 25014.390259105092498   |
      | lp2   | 0.9410673161599274 | 50178.9876096722076451  |

    And the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 3      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 49     | USD   |

    Then the accumulated liquidity fees should be "1" for the market "ETH/MAR22"

    # -------------------------------------------------------------------------------------------------------------------

    # Check equity-like-shares before network moves forward
    Given the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0589326838400726 | 25014.390259105092498   |
      | lp2   | 0.9410673161599274 | 50178.9876096722076451  |

    # Trigger entry into next market period
    When the network moves ahead "1" blocks:

    # Confirm equity-like-shares are unchanged by the network moving forwards (as virtual-stakes scaled by same factor, r)
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.0589326838400726 | 25014.390259105092498   |
      | lp2   | 0.9410673161599274 | 50178.9876096722076451  |

  # -------------------------------------------------------------------------------------------------------------------
  # -------------------------------------------------------------------------------------------------------------------

  @FeeRound
  Scenario: 002  Checks ELS calculations for a market which grows and then shrinks.

    Given the average block duration is "1801"

    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | lp1    | USD   | 100000000 |
      | lp2    | USD   | 100000000 |
      | lp3    | USD   | 100000000 |
      | party1 | USD   | 100000    |
      | party2 | USD   | 100000    |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 25000             | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | lp1   | ETH/MAR22 | 25000             | 0.001 | buy  | MID              | 3          | 1      | amendment  |
      | lp1 | lp1   | ETH/MAR22 | 25000             | 0.001 | sell | ASK              | 1          | 2      | amendment  |
      | lp1 | lp1   | ETH/MAR22 | 25000             | 0.001 | sell | MID              | 3          | 1      | amendment  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 25000             | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/MAR22 | 25000             | 0.001 | buy  | MID              | 3          | 1      | amendment  |
      | lp2 | lp2   | ETH/MAR22 | 25000             | 0.001 | sell | ASK              | 1          | 2      | amendment  |
      | lp2 | lp2   | ETH/MAR22 | 25000             | 0.001 | sell | MID              | 3          | 1      | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |


    # 0th period (bootstrap period): no LP changes, no trades
    Then the opening auction period ends for market "ETH/MAR22"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 10   | party2 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 500       | 1500      | 2500         | 50000          | 10            |

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.5               | 25000                   |
      | lp2   | 0.5               | 50000                   |

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"


    When the network moves ahead "2" blocks:

    # -------------------------------------------------------------------------------------------------------------------
    # -------------------------------------------------------------------------------------------------------------------


    # 1st period: Positive growth (all LP virtual-stake values greater than their physical-stake values)
    # -------------------------------------------------------------------------------------------------------------------

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 20     | 1001  | 2                | TYPE_LIMIT | TIF_GTC |

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1001  | 19   | lp1    |
      | party1 | 1001  | 1    | lp2    |

    # CALCULATION:
    # liquidity_fee = trades * ceil(volume/trades * price * liquidity_fee_factor) = ceil(19 * 1001 * 0.001) + ceil(1 * 1001 * 0.001) = 22

    And the following transfers should happen:
      | from   | to     | from account         | to account                  | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY | ETH/MAR22 | 22     | USD   |

    And the accumulated liquidity fees should be "22" for the market "ETH/MAR22"

    And the market data for the market "ETH/MAR22" should be:
      | mark price | last traded price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | 1001              | TRADING_MODE_CONTINUOUS | 1       | 500       | 1500      | 7500         | 50000          | 30            |

    # -------------------------------------------------------------------------------------------------------------------

    # Trigger next liquidity fee distribution without triggering next market period
    When the network moves ahead "1" blocks:

    # Confirm equity-like-shares are unchanged by network moving forwards (as new market period not entered)
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.5               | 25000                   |
      | lp2   | 0.5               | 50000                   |

    And the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 11     | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 11     | USD   |

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

    # -------------------------------------------------------------------------------------------------------------------

    # Trigger entry into next market period
    When the network moves ahead "1" blocks:

    # Confirm equity-like-shares are unchanged by the network moving forwards (as virtual-stakes scaled by same factor, r)
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.5               | 25000                   |
      | lp2   | 0.5               | 50000                   |

    # -------------------------------------------------------------------------------------------------------------------
    # -------------------------------------------------------------------------------------------------------------------


    # 2nd period: Positive growth (all LP virtual-stake values greater than their physical-stake values)
    # -------------------------------------------------------------------------------------------------------------------

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp3   | ETH/MAR22 | 25000             | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | lp3   | ETH/MAR22 | 25000             | 0.001 | buy  | MID              | 3          | 1      | amendment  |
      | lp1 | lp3   | ETH/MAR22 | 25000             | 0.001 | sell | ASK              | 1          | 2      | amendment  |
      | lp1 | lp3   | ETH/MAR22 | 25000             | 0.001 | sell | MID              | 3          | 1      | amendment  |

    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.3750624687656172 | 25000                   |
      | lp2   | 0.3750624687656172 | 50000                   |
      | lp3   | 0.2498750624687656 | 100050                  |

    # -------------------------------------------------------------------------------------------------------------------

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 30     | 1001  | 2                | TYPE_LIMIT | TIF_GTC |

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1001  | 19   | lp2    |
      | party1 | 1001  | 11   | lp1    |

    # CALCULATION:
    # liquidity_fee = trades * ceil(volume/trades * price * liquidity_fee_factor) = ceil(19 * 1001 * 0.001) + ceil(11 * 1001 * 0.001) = 32

    And the following transfers should happen:
      | from   | to     | from account         | to account                  | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY | ETH/MAR22 | 32     | USD   |

    And the accumulated liquidity fees should be "32" for the market "ETH/MAR22"

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001       | TRADING_MODE_CONTINUOUS | 1       | 502       | 1500      | 15015        | 75000          | 60            |

    # -------------------------------------------------------------------------------------------------------------------

    # Trigger next liquidity fee distribution without triggering next period
    When the network moves ahead "1" blocks:

    # Confirm equity-like-shares are unchanged by network moving forwards (as new market period not entered)
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.3750624687656172 | 25000                   |
      | lp2   | 0.3750624687656172 | 50000                   |
      | lp3   | 0.2498750624687656 | 100050                  |

    # lp3 just joined so liquidity score this period is virtually 0, hence no rewards
    And the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 16     | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 16     | USD   |
      #| market | lp3 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 0      | USD   |

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

    # -------------------------------------------------------------------------------------------------------------------

    # Trigger entry into next market period
    And the network moves ahead "1" blocks:

    # Confirm equity-like-shares are unchanged by the network moving forwards (as virtual-stakes scaled by same factor, r)
    When the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.3750624687656172 | 25000                   |
      | lp2   | 0.3750624687656172 | 50000                   |
      | lp3   | 0.2498750624687656 | 100050                  |

    # -------------------------------------------------------------------------------------------------------------------
    # -------------------------------------------------------------------------------------------------------------------

    # 3rd period: Negative growth (all LP virtual-stake values greater than their physical-stake values)
    # -------------------------------------------------------------------------------------------------------------------

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 10     | 1001  | 1                | TYPE_LIMIT | TIF_GTC |

    # CALCULATION:
    # liquidity_fee = trades * ceil(volume/trades * price * liquidity_fee_factor) =  1 * ceil(10/1 * 1001 * 0.001) = ceil(10.001) = 11

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1001  | 10   | lp3    |

    And the following transfers should happen:
      | from   | to     | from account         | to account                  | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY | ETH/MAR22 | 11     | USD   |

    And the accumulated liquidity fees should be "11" for the market "ETH/MAR22"

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001       | TRADING_MODE_CONTINUOUS | 1       | 502       | 1500      | 17517        | 75000          | 70            |

    # -------------------------------------------------------------------------------------------------------------------

    # Trigger next liquidity fee distribution without triggering next period
    When the network moves ahead "1" blocks:

    # Confirm equity-like-shares are unchanged by network moving forwards (as new market period not entered)
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.3750624687656172 | 25000                   |
      | lp2   | 0.3750624687656172 | 50000                   |
      | lp3   | 0.2498750624687656 | 100050                  |

    And the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 4      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 4      | USD   |
      | market | lp3 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 2      | USD   |

    And the accumulated liquidity fees should be "1" for the market "ETH/MAR22"

    # -------------------------------------------------------------------------------------------------------------------

    # Trigger entry into next market period
    And the network moves ahead "1" blocks:

    # Confirm equity-like-shares are unchanged by the network moving forwards (as virtual-stakes scaled by same factor, r)
    When the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.3750624687656172 | 25000                   |
      | lp2   | 0.3750624687656172 | 50000                   |
      | lp3   | 0.2498750624687656 | 100050                  |

    # -------------------------------------------------------------------------------------------------------------------
    # -------------------------------------------------------------------------------------------------------------------

    # 4th period: Negative growth (not all LP virtual-stake values greater than their physical-stake values)
    # -------------------------------------------------------------------------------------------------------------------

    # Trigger next liquidity fee distribution without triggering next period
    When the network moves ahead "1" blocks:

    # Confirm equity-like-shares are unchanged by network moving forwards (as new market period not entered)
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.3750624687656172 | 25000                   |
      | lp2   | 0.3750624687656172 | 50000                   |
      | lp3   | 0.2498750624687656 | 100050                  |

    And the accumulated liquidity fees should be "1" for the market "ETH/MAR22"

    # -------------------------------------------------------------------------------------------------------------------

    # Trigger entry into next market period
    And the network moves ahead "1" blocks:

    # Confirm equity-like-shares are unchanged by the network moving forwards (as virtual-stakes scaled by same factor, r)
    When the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.3685041026719966 | 25000                   |
      | lp2   | 0.3685041026719966 | 50000                   |
      | lp3   | 0.2629917946560067 | 100050                  |

    # -------------------------------------------------------------------------------------------------------------------
    # -------------------------------------------------------------------------------------------------------------------

    # 5th period: Negative growth (not all LP virtual-stake values greater than their physical-stake values)
    # -------------------------------------------------------------------------------------------------------------------

    # Trigger next liquidity fee distribution without triggering next period
    When the network moves ahead "1" blocks:

    # Confirm equity-like-shares are unchanged by network moving forwards (as new market period not entered)
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.3685041026719966 | 25000                   |
      | lp2   | 0.3685041026719966 | 50000                   |
      | lp3   | 0.2629917946560067 | 100050                  |

    And the accumulated liquidity fees should be "1" for the market "ETH/MAR22"

    # -------------------------------------------------------------------------------------------------------------------

    # Trigger entry into next market period
    And the network moves ahead "1" blocks:

    # Confirm equity-like-shares are unchanged by the network moving forwards (as virtual-stakes scaled by same factor, r)
    When the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.3500899460323806 | 25000                   |
      | lp2   | 0.3500899460323806 | 50000                   |
      | lp3   | 0.2998201079352388 | 100050                  |

    # -------------------------------------------------------------------------------------------------------------------
    # -------------------------------------------------------------------------------------------------------------------

    # 6th period: Negative growth (not all LP virtual-stake values greater than their physical-stake values)
    # -------------------------------------------------------------------------------------------------------------------

    # Trigger next liquidity fee distribution without triggering next period
    When the network moves ahead "1" blocks:

    # Confirm equity-like-shares are unchanged by network moving forwards (as new market period not entered)
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.3500899460323806 | 25000                   |
      | lp2   | 0.3500899460323806 | 50000                   |
      | lp3   | 0.2998201079352388 | 100050                  |

    And the accumulated liquidity fees should be "1" for the market "ETH/MAR22"

    # -------------------------------------------------------------------------------------------------------------------

    # Trigger entry into next market period
    And the network moves ahead "1" blocks:

    # Confirm equity-like-shares are unchanged by the network moving forwards (as virtual-stakes scaled by same factor, r)
    When the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.3334285170378831 | 25000                   |
      | lp2   | 0.3334285170378831 | 50000                   |
      | lp3   | 0.3331429659242338 | 100050                  |

    # -------------------------------------------------------------------------------------------------------------------
    # -------------------------------------------------------------------------------------------------------------------

    # 7th period: Negative growth (all LP virtual-stake equal to their physical-stake values)
    # -------------------------------------------------------------------------------------------------------------------

    # Trigger next liquidity fee distribution without triggering next period
    When the network moves ahead "1" blocks:

    # Confirm equity-like-shares are unchanged by network moving forwards (as new market period not entered)
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.3334285170378831 | 25000                   |
      | lp2   | 0.3334285170378831 | 50000                   |
      | lp3   | 0.3331429659242338 | 100050                  |

    And the accumulated liquidity fees should be "1" for the market "ETH/MAR22"

    # -------------------------------------------------------------------------------------------------------------------

    # Trigger entry into next market period
    And the network moves ahead "1" blocks:

    # Confirm equity-like-shares are unchanged by the network moving forwards (as virtual-stakes scaled by same factor, r)
    When the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.3333333333333333 | 25000                   |
      | lp2   | 0.3333333333333333 | 50000                   |
      | lp3   | 0.3333333333333333 | 100050                  |

# -------------------------------------------------------------------------------------------------------------------
# -------------------------------------------------------------------------------------------------------------------
