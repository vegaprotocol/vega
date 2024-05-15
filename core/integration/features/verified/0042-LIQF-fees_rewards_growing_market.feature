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
      | name                                             | value |
      | market.value.windowLength                        | 5m    |
      | network.markPriceUpdateMaximumFrequency          | 0s    |
      | limits.markets.maxPeggedOrders                   | 8     |
      | validators.epoch.length                          | 10s   |
      | market.liquidity.providersFeeCalculationTimeStep | 1s    |
      | market.liquidity.equityLikeShareFeeFraction      | 1     |
    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | lqm-params         | 0.0              | 24h         | 2.5            |
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | USD        | USD   | lqm-params           | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring | default-eth-for-future | 0.5                    | 0                         | SLA        |


  @FeeRound
  Scenario: 001 2 LPs joining at start, unequal commitments. Checking calculation of equity-like-shares and liquidity-fee-distribution in a market with small growth (0042-LIQF-008)

    # Scenario has 6 market periods:

    # - 0th period (bootstrap period): no LP changes, no trades
    # - 1st period: 1 LPs decrease commitment, some trades occur
    # - 2nd period: 1 LPs increase commitment, some trades occur
    # - 3rd period: 2 LPs decrease commitment, some trades occur
    # - 4th period: 2 LPs increase commitment, some trades occur
    # - 5th period: 1 LPs decrease commitment, 1 LPs increase commitment, some trades occur

    Given the average block duration is "1"

    Given time is updated to "2019-11-30T00:00:00Z"

    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | lp1    | USD   | 100000000 |
      | lp2    | USD   | 100000000 |
      | party1 | USD   | 100000    |
      | party2 | USD   | 100000    |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference |
      | lp1   | ETH/MAR22 | 50        | 40                   | buy  | MID              | 100    | 1      | lp1-bids  |
      | lp1   | ETH/MAR22 | 50        | 40                   | sell | MID              | 100    | 1      | lp1-asks  |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 46000             | 0.002 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference |
      | lp2   | ETH/MAR22 | 500       | 460                  | buy  | MID              | 1000   | 1      | lp2-bids  |
      | lp2   | ETH/MAR22 | 500       | 460                  | sell | MID              | 1000   | 1      | lp2-asks  |

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
      | party | equity like share | average entry valuation | virtual stake          |
      | lp1   | 0.08              | 4000                    | 4000.0000000000000000  |
      | lp2   | 0.92              | 50000                   | 46000.0000000000000000 |

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

    Given time is updated to "2019-11-30T00:05:01Z"

    # -------------------------------------------------------------------------------------------------------------------
    # -------------------------------------------------------------------------------------------------------------------


    # 1st period: 1 LPs decrease commitment, positive growth:
    # -------------------------------------------------------------------------------------------------------------------

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type   |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | amendment |

    When the network moves ahead "1" blocks

    And time is updated to "2019-11-30T00:05:12Z"

    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation | virtual stake          |
      | lp1   | 0.0612244897959184 | 4000                    | 3000.0000000000000000  |
      | lp2   | 0.9387755102040816 | 50000                   | 46000.0000000000000000 |

    # -------------------------------------------------------------------------------------------------------------------

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 21     | 1001  | 1                | TYPE_LIMIT | TIF_GTC |

    And the following transfers should happen:
      | from   | to     | from account         | to account                  | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY | ETH/MAR22 | 43     | USD   |

    And the accumulated liquidity fees should be "43" for the market "ETH/MAR22"

    When the network moves ahead "6" blocks:

    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 2      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 40     | USD   |

    And the accumulated liquidity fees should be "1" for the market "ETH/MAR22"

    # -------------------------------------------------------------------------------------------------------------------

    # Trigger entry into next market period
    Given time is updated to "2019-11-30T00:10:15Z"

    # Confirm equity-like-shares are unchanged by the network moving forwards (as virtual-stakes scaled by same factor, r)
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation | virtual stake          |
      | lp1   | 0.0612244897959184 | 4000                    | 3076.5750000000000000  |
      | lp2   | 0.9387755102040816 | 50000                   | 47174.1500000000000000 |

    # -------------------------------------------------------------------------------------------------------------------
    # -------------------------------------------------------------------------------------------------------------------


    # 2nd period: 1 LPs increase commitment, positive growth:
    # -------------------------------------------------------------------------------------------------------------------

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type   |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | amendment |

    When the network moves ahead "1" blocks

    And time is updated to "2019-11-30T00:10:25Z"

    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation | virtual stake          |
      | lp1   | 0.0795418016037822 | 15812.68125             | 4076.5750000000000000  |
      | lp2   | 0.9204581983962178 | 50000                   | 47174.1500000000000000 |

    # -------------------------------------------------------------------------------------------------------------------

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 22     | 1001  | 1                | TYPE_LIMIT | TIF_GTC |

    And the following transfers should happen:
      | from   | to     | from account         | to account                  | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY | ETH/MAR22 | 45     | USD   |

    And the accumulated liquidity fees should be "46" for the market "ETH/MAR22"

    When the network moves ahead "6" blocks

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001       | TRADING_MODE_CONTINUOUS | 1       | 502       | 1500      | 15765        | 50000          | 63            |

    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 3      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 42     | USD   |

    And the accumulated liquidity fees should be "1" for the market "ETH/MAR22"

    # -------------------------------------------------------------------------------------------------------------------

    # Trigger entry into next market period
    Given time is updated to "2019-11-30T00:15:15Z"

    # Confirm equity-like-shares are unchanged by the network moving forwards (as virtual-stakes scaled by same factor, r)
    When the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation | virtual stake          |
      | lp1   | 0.0795418016037822 | 15812.68125             | 4076.5750000000000000  |
      | lp2   | 0.9204581983962178 | 50000                   | 47174.1500000000000000 |

    # ----------------------------------------------------------------------------------
    # ----------------------------------------------------------------------------------


    # 3rd period: 2 LPs decrease commitment, positive growth:
    # ----------------------------------------------------------------------------------

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type   |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | amendment |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type   |
      | lp2 | lp2   | ETH/MAR22 | 45000             | 0.002 | amendment |

    When the network moves ahead "1" blocks

    And time is updated to "2019-11-30T00:15:31Z"

    # Confirm equity-like-shares updated after liquidity amendment
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation | virtual stake          |
      | lp1   | 0.0621352630754675 | 15812.68125             | 3132.5359904073524062  |
      | lp2   | 0.9378647369245325 | 50000                   | 47282.2500000000015443 |

    # ----------------------------------------------------------------------------------

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 23     | 1001  | 1                | TYPE_LIMIT | TIF_GTC |

    And the following transfers should happen:
      | from   | to     | from account         | to account                  | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY | ETH/MAR22 | 47     | USD   |

    And the accumulated liquidity fees should be "48" for the market "ETH/MAR22"

    When the network moves ahead "6" blocks

    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 2      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 45     | USD   |

    And the accumulated liquidity fees should be "1" for the market "ETH/MAR22"

    # -------------------------------------------------------------------------------------------------------------------

    # Trigger entry into next market period
    Given time is updated to "2019-11-30T00:20:15Z"

    # Confirm equity-like-shares are unchanged by the network moving forwards (as virtual-stakes scaled by same factor, r)
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation | virtual stake          |
      | lp1   | 0.0621352630754675 | 15812.68125             | 3132.5359904073524062  |
      | lp2   | 0.9378647369245325 | 50000                   | 47282.2500000000015443 |

    # -------------------------------------------------------------------------------------------------------------------
    # -------------------------------------------------------------------------------------------------------------------


    # 4nd period: 2 LPs increase commitment, positive growth:
    # -------------------------------------------------------------------------------------------------------------------

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type   |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | amendment |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type   |
      | lp2 | lp2   | ETH/MAR22 | 46000             | 0.002 | amendment |

    When the network moves ahead "1" blocks

    And time is updated to "2019-11-30T00:25:16Z"

    # Confirm equity-like-shares updated after liquidity amendment
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation | virtual stake          |
      | lp1   | 0.0788429431184488 | 24713.2074351018384876  | 4231.2886745998665238  |
      | lp2   | 0.9211570568815512 | 50052.4953476175511728  | 49436.0213881794964683 |

    # -------------------------------------------------------------------------------------------------------------------

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 24     | 1001  | 1                | TYPE_LIMIT | TIF_GTC |

    And the following transfers should happen:
      | from   | to     | from account         | to account                  | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY | ETH/MAR22 | 49     | USD   |

    And the accumulated liquidity fees should be "50" for the market "ETH/MAR22"

    When the network moves ahead "6" blocks

    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 3      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 46     | USD   |

    And the accumulated liquidity fees should be "1" for the market "ETH/MAR22"

    # -------------------------------------------------------------------------------------------------------------------

    # Trigger entry into next market period
    Given time is updated to "2019-11-30T00:30:31Z"

    # Confirm equity-like-shares are unchanged by the network moving forwards (as virtual-stakes scaled by same factor, r)
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation | virtual stake          |
      | lp1   | 0.0788429431184488 | 24713.2074351018384876  | 4329.9102566560480206  |
      | lp2   | 0.9211570568815512 | 50052.4953476175511728  | 50588.2610519803921571 |

    # -------------------------------------------------------------------------------------------------------------------
    # -------------------------------------------------------------------------------------------------------------------


    # 5th period: 1 LPs decrease commitment 1 LPs increase commitment, some trades occur
    # -------------------------------------------------------------------------------------------------------------------

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type   |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | amendment |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type   |
      | lp2 | lp2   | ETH/MAR22 | 47000             | 0.002 | amendment |

    When the network moves ahead "1" blocks

    And time is updated to "2019-11-30T00:35:34Z"

    # Confirm equity-like-shares updated after liquidity amendment
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation | virtual stake          |
      | lp1   | 0.06              | 24713.2074351018384876  | 3000.0000000000000000  |
      | lp2   | 0.94              | 50154.2655262740379175  | 47000.0000000000000000 |

    # -------------------------------------------------------------------------------------------------------------------

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 25     | 1001  | 1                | TYPE_LIMIT | TIF_GTC |

    And the following transfers should happen:
      | from   | to     | from account         | to account                  | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY | ETH/MAR22 | 51     | USD   |

    And the accumulated liquidity fees should be "52" for the market "ETH/MAR22"

    When the network moves ahead "6" blocks

    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 3      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 48     | USD   |

    And the accumulated liquidity fees should be "1" for the market "ETH/MAR22"

    # -------------------------------------------------------------------------------------------------------------------

    # Trigger entry into next market period
    Given time is updated to "2019-11-30T00:40:38Z"

    # Confirm equity-like-shares are unchanged by the network moving forwards (as virtual-stakes scaled by same factor, r)
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation | virtual stake          |
      | lp1   | 0.06              | 24713.2074351018384876  | 3155.9503263563577000  |
      | lp2   | 0.94              | 50154.2655262740379175  | 49443.2217795829373000 |

  # -------------------------------------------------------------------------------------------------------------------
  # -------------------------------------------------------------------------------------------------------------------

  @FeeRound
  Scenario: 002  Checks ELS calculations for a market which grows and then shrinks.

    Given the average block duration is "1"

    Given time is updated to "2019-11-30T00:00:00Z"

    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | lp1    | USD   | 100000000 |
      | lp2    | USD   | 100000000 |
      | lp3    | USD   | 100000000 |
      | party1 | USD   | 100000    |
      | party2 | USD   | 100000    |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 25000             | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference |
      | lp1   | ETH/MAR22 | 50        | 40                   | buy  | MID              | 100    | 1      | lp1-bids  |
      | lp1   | ETH/MAR22 | 50        | 40                   | sell | MID              | 100    | 1      | lp1-asks  |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 25000             | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference |
      | lp2   | ETH/MAR22 | 50        | 40                   | buy  | MID              | 100    | 1      | lp2-bids  |
      | lp2   | ETH/MAR22 | 50        | 40                   | sell | MID              | 100    | 1      | lp2-asks  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |


    # 0th period (bootstrap period): no LP changes, no trades
    Then the opening auction period ends for market "ETH/MAR22"

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 500       | 1500      | 2500         | 50000          | 10            |

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation | virtual stake          |
      | lp1   | 0.5               | 25000                   | 25000.0000000000000000 |
      | lp2   | 0.5               | 50000                   | 25000.0000000000000000 |

    Given time is updated to "2019-11-30T00:05:03Z"

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation | virtual stake          |
      | lp1   | 0.5               | 25000                   | 25000.0000000000000000 |
      | lp2   | 0.5               | 50000                   | 25000.0000000000000000 |

    # -------------------------------------------------------------------------------------------------------------------
    # -------------------------------------------------------------------------------------------------------------------


    # 1st period: Positive growth (all LP virtual-stake values greater than their physical-stake values)
    # -------------------------------------------------------------------------------------------------------------------

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 20     | 1001  | 1                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "1" blocks:

    # Trigger entry into next market period
    Given time is updated to "2019-11-30T00:10:04Z"

    # Confirm equity-like-shares are unchanged by the network moving forwards (as virtual-stakes scaled by same factor, r)
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation | virtual stake          |
      | lp1   | 0.5               | 25000                   | 37525.0000000000000000 |
      | lp2   | 0.5               | 50000                   | 37525.0000000000000000 |

    # -------------------------------------------------------------------------------------------------------------------
    # -------------------------------------------------------------------------------------------------------------------


    # 2nd period: Positive growth (all LP virtual-stake values greater than their physical-stake values)
    # -------------------------------------------------------------------------------------------------------------------

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp3   | ETH/MAR22 | 25000             | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference |
      | lp3   | ETH/MAR22 | 50        | 40                   | buy  | MID              | 100    | 1      | lp3-bids  |
      | lp3   | ETH/MAR22 | 50        | 40                   | sell | MID              | 100    | 1      | lp3-asks  |

    When the network moves ahead "1" blocks
    And time is updated to "2019-11-30T00:10:24Z"
 
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation | virtual stake          |
      | lp1   | 0.3750624687656172 | 25000                   | 37525.0000000000000000 |
      | lp2   | 0.3750624687656172 | 50000                   | 37525.0000000000000000 |
      | lp3   | 0.2498750624687656 | 100050                  | 25000.0000000000000000 |

    # -------------------------------------------------------------------------------------------------------------------

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 30     | 1001  | 1                | TYPE_LIMIT | TIF_GTC |

    # Trigger entry into next market period
    And the network moves ahead "1" blocks:
    And time is updated to "2019-11-30T00:15:05Z"

    # Confirm equity-like-shares are unchanged by the network moving forwards (as virtual-stakes scaled by same factor, r)
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation | virtual stake          |
      | lp1   | 0.3750624687656172 | 25000                   | 50041.6666666666689275 |
      | lp2   | 0.3750624687656172 | 50000                   | 50041.6666666666689275 |
      | lp3   | 0.2498750624687656 | 100050                  | 33338.8851876526775000 |

    # -------------------------------------------------------------------------------------------------------------------
    # -------------------------------------------------------------------------------------------------------------------

    # 3rd period: Negative growth (all LP virtual-stake values greater than their physical-stake values)
    # -------------------------------------------------------------------------------------------------------------------

    Given the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 10     | 1001  | 1                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "1" blocks:
    And time is updated to "2019-11-30T00:20:06Z"

    # Confirm equity-like-shares are unchanged by network moving forwards (as new market period not entered)
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation | virtual stake          |
      | lp1   | 0.3750624687656172 | 25000                   | 43787.5000000000035533 |
      | lp2   | 0.3750624687656172 | 50000                   | 43787.5000000000035533 |
      | lp3   | 0.2498750624687656 | 100050                  | 29172.2185209860116944 |

    # -------------------------------------------------------------------------------------------------------------------
    # -------------------------------------------------------------------------------------------------------------------

    # 4th period: Negative growth (not all LP virtual-stake values greater than their physical-stake values)
    # -------------------------------------------------------------------------------------------------------------------

    Given time is updated to "2019-11-30T00:25:07Z"

    # Confirm equity-like-shares are unchanged by network moving forwards (as new market period not entered)
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation | virtual stake          |
      | lp1   | 0.3685041026719966 | 25000                   | 35030.0000000000028426 |
      | lp2   | 0.3685041026719966 | 50000                   | 35030.0000000000028426 |
      | lp3   | 0.2629917946560067 | 100050                  | 25000.0000000000000000 |

    # -------------------------------------------------------------------------------------------------------------------
    # -------------------------------------------------------------------------------------------------------------------

    # 5th period: Negative growth (not all LP virtual-stake values greater than their physical-stake values)
    # -------------------------------------------------------------------------------------------------------------------

    Given time is updated to "2019-11-30T00:30:08Z"

    # Confirm equity-like-shares are unchanged by the network moving forwards (as virtual-stakes scaled by same factor, r)
    When the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation | virtual stake          |
      | lp1   | 0.3500899460323806 | 25000                   | 29191.6666666666678679 |
      | lp2   | 0.3500899460323806 | 50000                   | 29191.6666666666678679 |
      | lp3   | 0.2998201079352388 | 100050                  | 25000.0000000000000000 |

    # -------------------------------------------------------------------------------------------------------------------
    # -------------------------------------------------------------------------------------------------------------------

    # 6th period: Negative growth (not all LP virtual-stake values greater than their physical-stake values)
    # -------------------------------------------------------------------------------------------------------------------

    Given time is updated to "2019-11-30T00:35:09Z"

    # Confirm equity-like-shares are unchanged by the network moving forwards (as virtual-stakes scaled by same factor, r)
    When the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation | virtual stake          |
      | lp1   | 0.3334285170378831 | 25000                   | 25021.4285714285712071 |
      | lp2   | 0.3334285170378831 | 50000                   | 25021.4285714285712071 |
      | lp3   | 0.3331429659242338 | 100050                  | 25000.0000000000000000 |

    # -------------------------------------------------------------------------------------------------------------------
    # -------------------------------------------------------------------------------------------------------------------

    # 7th period: Negative growth (all LP virtual-stake equal to their physical-stake values)
    # -------------------------------------------------------------------------------------------------------------------

    Given time is updated to "2019-11-30T00:40:10Z"

    # Confirm equity-like-shares are unchanged by the network moving forwards (as virtual-stakes scaled by same factor, r)
    When the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation | virtual stake          |
      | lp1   | 0.3333333333333333 | 25000                   | 25000.0000000000000000 |
      | lp2   | 0.3333333333333333 | 50000                   | 25000.0000000000000000 |
      | lp3   | 0.3333333333333333 | 100050                  | 25000.0000000000000000 |

# -------------------------------------------------------------------------------------------------------------------
# -------------------------------------------------------------------------------------------------------------------
