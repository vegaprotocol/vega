Feature: Test liquidity provider reward distribution; Should also cover liquidity-fee-setting and equity-like-share calc and total stake.
  # to look into and test: If an equity-like share is small and LP rewards are distributed immediately, then how do we round? (does a small share get rounded up or down, do they all add up?)
  #Check what happens with time and distribution period (both in genesis and mid-market)

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
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 0     |
      | market.liquidity.providers.fee.distributionTimeStep | 10m   |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/MAR22 | USD        | USD   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 0.2                    | 0                         |

    # block duration of 2 seconds
    And the average block duration is "2"

  @VirtStake
  Scenario: 001 1 LP joining at start, checking liquidity rewards over 3 periods, 1 period with no trades (0042-LIQF-001)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | USD   | 1000000000 |
      | party1 | USD   | 100000000  |
      | party2 | USD   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 10000             | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | lp1   | ETH/MAR22 | 10000             | 0.001 | buy  | MID              | 2          | 1      | amendment  |
      | lp1 | lp1   | ETH/MAR22 | 10000             | 0.001 | sell | ASK              | 1          | 2      | amendment  |
      | lp1 | lp1   | ETH/MAR22 | 10000             | 0.001 | sell | MID              | 2          | 1      | amendment  |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/MAR22"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 10   | party2 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 500       | 1500      | 1000         | 10000          | 10            |

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 1                 | 10000                   |

    And the liquidity fee factor should be "0.001" for the market "ETH/MAR22"

    # No fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"


    When the network moves ahead "1" blocks

    And the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/MAR22 | buy  | 20     | 1000  | 2                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party2 | 951   | 8    | lp1    |
      | party2 | 1000  | 12   | party1 |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general   | bond  |
      | lp1    | USD   | ETH/MAR22 | 2400   | 999987212 | 10000 |
      | party1 | USD   | ETH/MAR22 | 1200   | 99998805  |       |
      | party2 | USD   | ETH/MAR22 | 1932   | 99998411  |       |

    And the accumulated liquidity fees should be "20" for the market "ETH/MAR22"


    # Trigger distribution of liquidity fees
    When the network moves ahead "301" blocks

    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 20     | USD   |

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"


    # Move to new block
    When the network moves ahead "301" blocks

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | buy  | 40     | 1100  | 1                | TYPE_LIMIT | TIF_GTC | party1-buy  |
      | party2 | ETH/MAR22 | sell | 40     | 1100  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell |

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 951   | 8    | lp1    |

    And the accumulated liquidity fees should be "8" for the market "ETH/MAR22"

    # Trigger distribution of liquidity fees
    When the network moves ahead "301" blocks

    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 8      | USD   |

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

  @VirtStake
  Scenario: 002 2 LPs joining at start, equal commitments (0042-LIQF-002)

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | USD   | 1000000000 |
      | lp2    | USD   | 1000000000 |
      | party1 | USD   | 100000000  |
      | party2 | USD   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 5000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | lp1   | ETH/MAR22 | 5000              | 0.001 | buy  | MID              | 2          | 1      | amendment  |
      | lp1 | lp1   | ETH/MAR22 | 5000              | 0.001 | sell | ASK              | 1          | 2      | amendment  |
      | lp1 | lp1   | ETH/MAR22 | 5000              | 0.001 | sell | MID              | 2          | 1      | amendment  |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 5000              | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/MAR22 | 5000              | 0.002 | buy  | MID              | 2          | 1      | amendment  |
      | lp2 | lp2   | ETH/MAR22 | 5000              | 0.002 | sell | ASK              | 1          | 2      | amendment  |
      | lp2 | lp2   | ETH/MAR22 | 5000              | 0.002 | sell | MID              | 2          | 1      | amendment  |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 90     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 90     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/MAR22"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 90   | party2 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 500       | 1500      | 9000         | 10000          | 90            |

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.5               | 5000                    |
      | lp2   | 0.5               | 10000                   |

    And the liquidity fee factor should be "0.002" for the market "ETH/MAR22"

    # No fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"


    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/MAR22 | buy  | 20     | 1000  | 3                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party2 | 951   | 4    | lp1    |
      | party2 | 951   | 4    | lp2    |
      | party2 | 1000  | 12   | party1 |

    And the accumulated liquidity fees should be "40" for the market "ETH/MAR22"

    # Trigger fee distribution
    When the network moves ahead "301" blocks

    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 20     | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 20     | USD   |

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"


    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | buy  | 40     | 1100  | 2                | TYPE_LIMIT | TIF_GTC | party1-buy  |
      | party2 | ETH/MAR22 | sell | 40     | 1100  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell |

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 951   | 4    | lp1    |
      | party1 | 951   | 4    | lp2    |

    And the accumulated liquidity fees should be "16" for the market "ETH/MAR22"


    # Trigger fee distribution
    When the network moves ahead "301" blocks

    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 8      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 8      | USD   |
    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

  @FeeRound
  Scenario: 003 2 LPs joining at start, unequal commitments. Checking calculation of equity-like-shares and liquidity-fee-distribution in a shrinking market. (0042-LIQF-008 0042-LIQF-011)

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
      | party  | asset | amount |
      | lp1    | USD   | 100000 |
      | lp2    | USD   | 100000 |
      | party1 | USD   | 100000 |
      | party2 | USD   | 100000 |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | buy  | MID              | 3          | 1      | amendment  |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | sell | ASK              | 1          | 2      | amendment  |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | sell | MID              | 3          | 1      | amendment  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 6000              | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/MAR22 | 6000              | 0.002 | buy  | MID              | 3          | 1      | amendment  |
      | lp2 | lp2   | ETH/MAR22 | 6000              | 0.002 | sell | ASK              | 1          | 2      | amendment  |
      | lp2 | lp2   | ETH/MAR22 | 6000              | 0.002 | sell | MID              | 3          | 1      | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 50     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 50     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |


    # 0th period (bootstrap period): no LP changes, no trades
    Then the opening auction period ends for market "ETH/MAR22"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 50   | party2 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 500       | 1500      | 5000         | 10000          | 50            |

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.4               | 4000                    |
      | lp2   | 0.6               | 10000                   |

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"


    # 1st period: 1 LPs decrease commitment, some trades occur:
    When the network moves ahead "2" blocks:

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | buy  | MID              | 3          | 1      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | sell | MID              | 3          | 1      | amendment |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 2      | 1001  | 1                | TYPE_LIMIT | TIF_GTC |

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1001  | 2    | lp2    |

    # liquidity_fee = ceil(volume * price * liquidity_fee_factor) =  ceil(1001 * 2 * 0.002) = ceil(4.004) = 5

    And the following transfers should happen:
      | from   | to     | from account         | to account                  | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY | ETH/MAR22 | 5      | USD   |

    And the accumulated liquidity fees should be "5" for the market "ETH/MAR22"

    And the market data for the market "ETH/MAR22" should be:
      | mark price | last traded price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | 1001              | TRADING_MODE_CONTINUOUS | 1       | 500       | 1500      | 5200         | 9000           | 52            |

    # Trigger next liquidity fee distribution without triggering next period
    When the network moves ahead "1" blocks:

    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.3333333333333333 | 4000                    |
      | lp2   | 0.6666666666666667 | 10000                   |

    And the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 1      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 3      | USD   |

    And the accumulated liquidity fees should be "1" for the market "ETH/MAR22"


    # 2nd period: 1 LPs increase commitment, some trades occur
    When the network moves ahead "1" blocks:

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | buy  | MID              | 3          | 1      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | sell | MID              | 3          | 1      | amendment |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 2      | 1001  | 1                | TYPE_LIMIT | TIF_GTC |

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1001  | 2    | lp2    |

    # liquidity_fee = ceil(volume * price * liquidity_fee_factor) =  ceil(1001 * 2 * 0.002) = ceil(4.004) = 5

    And the following transfers should happen:
      | from   | to     | from account         | to account                  | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY | ETH/MAR22 | 5      | USD   |

    # 5 + 1 remainder
    And the accumulated liquidity fees should be "6" for the market "ETH/MAR22"

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001       | TRADING_MODE_CONTINUOUS | 1       | 502       | 1500      | 5405         | 10000          | 54            |

    # Trigger next liquidity fee distribution without triggering next period
    When the network moves ahead "1" blocks:

    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.4               | 5500                    |
      | lp2   | 0.6               | 10000                   |

    And the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 2      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 3      | USD   |

    And the accumulated liquidity fees should be "1" for the market "ETH/MAR22"


    # 3rd period: 2 LPs decrease commitment, some trades occur
    When the network moves ahead "1" blocks:

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | buy  | MID              | 3          | 1      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | sell | MID              | 3          | 1      | amendment |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp2 | lp2   | ETH/MAR22 | 5000              | 0.002 | buy  | BID              | 1          | 2      | amendment |
      | lp2 | lp2   | ETH/MAR22 | 5000              | 0.002 | buy  | MID              | 3          | 1      | amendment |
      | lp2 | lp2   | ETH/MAR22 | 5000              | 0.002 | sell | ASK              | 1          | 2      | amendment |
      | lp2 | lp2   | ETH/MAR22 | 5000              | 0.002 | sell | MID              | 3          | 1      | amendment |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 3      | 1001  | 1                | TYPE_LIMIT | TIF_GTC |

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1001  | 3    | lp1    |

    # liquidity_fee = ceil(volume * price * liquidity_fee_factor) =  ceil(1001 * 3 * 0.002) = ceil(6.006) = 7

    And the following transfers should happen:
      | from   | to     | from account         | to account                  | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY | ETH/MAR22 | 7      | USD   |

    And the accumulated liquidity fees should be "8" for the market "ETH/MAR22"

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001       | TRADING_MODE_CONTINUOUS | 1       | 502       | 1500      | 5705         | 8000           | 57            |

    # Trigger next liquidity fee distribution without triggering next period
    When the network moves ahead "1" blocks:

    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.375             | 5500                    |
      | lp2   | 0.625             | 10000                   |

    And the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 2      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 5      | USD   |

    And the accumulated liquidity fees should be "1" for the market "ETH/MAR22"


    # 4nd period: 2 LPs increase commitment, some trades occur
    When the network moves ahead "2" blocks:

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | buy  | MID              | 3          | 1      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | sell | MID              | 3          | 1      | amendment |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp2 | lp2   | ETH/MAR22 | 6000              | 0.002 | buy  | BID              | 1          | 2      | amendment |
      | lp2 | lp2   | ETH/MAR22 | 6000              | 0.002 | buy  | MID              | 3          | 1      | amendment |
      | lp2 | lp2   | ETH/MAR22 | 6000              | 0.002 | sell | ASK              | 1          | 2      | amendment |
      | lp2 | lp2   | ETH/MAR22 | 6000              | 0.002 | sell | MID              | 3          | 1      | amendment |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 4      | 1001  | 2                | TYPE_LIMIT | TIF_GTC |

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1001  | 3    | lp1    |
      | party1 | 1001  | 1    | lp2    |

    # liquidity_fee = ceil(volume * price * liquidity_fee_factor) =  ceil(1001 * 3 * 0.002) + ceil(1001 * 1 * 0.002) = ceil(6.006) + ceil(2.002)= 10

    And the following transfers should happen:
      | from   | to     | from account         | to account                  | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY | ETH/MAR22 | 10     | USD   |

    And the accumulated liquidity fees should be "11" for the market "ETH/MAR22"

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001       | TRADING_MODE_CONTINUOUS | 1       | 502       | 1500      | 6106         | 10000          | 61            |

    # Trigger next liquidity fee distribution without triggering next period
    When the network moves ahead "1" blocks:

    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.4               | 6375                    |
      | lp2   | 0.6               | 10000                   |

    And the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 4      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 6      | USD   |

    And the accumulated liquidity fees should be "1" for the market "ETH/MAR22"


    # 5th period: 1 LPs decrease commitment 1 LPs increase commitment, some trades occur
    When the network moves ahead "1" blocks:

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | buy  | MID              | 3          | 1      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | sell | MID              | 3          | 1      | amendment |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp2 | lp2   | ETH/MAR22 | 7000              | 0.002 | buy  | BID              | 1          | 2      | amendment |
      | lp2 | lp2   | ETH/MAR22 | 7000              | 0.002 | buy  | MID              | 3          | 1      | amendment |
      | lp2 | lp2   | ETH/MAR22 | 7000              | 0.002 | sell | ASK              | 1          | 2      | amendment |
      | lp2 | lp2   | ETH/MAR22 | 7000              | 0.002 | sell | MID              | 3          | 1      | amendment |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 5      | 1001  | 2                | TYPE_LIMIT | TIF_GTC |

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1001  | 3    | lp1    |
      | party1 | 1001  | 2    | lp2    |

    # liquidity_fee = ceil(volume * price * liquidity_fee_factor) =  ceil(1001 * 3 * 0.002) + ceil(1001 * 2 * 0.002) = ceil(6.006) + ceil(4.004)= 12

    And the following transfers should happen:
      | from   | to     | from account         | to account                  | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_GENERAL | ACCOUNT_TYPE_FEES_LIQUIDITY | ETH/MAR22 | 12     | USD   |

    And the accumulated liquidity fees should be "13" for the market "ETH/MAR22"

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001       | TRADING_MODE_CONTINUOUS | 1       | 502       | 1500      | 6606         | 10000          | 66            |

    # Trigger next liquidity fee distribution without triggering next period
    When the network moves ahead "1" blocks:

    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.3               | 6375                    |
      | lp2   | 0.7               | 10000                   |

    And the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 3      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 9      | USD   |

    And the accumulated liquidity fees should be "1" for the market "ETH/MAR22"

  @FeeRound
  Scenario: 004 2 LPs joining at start, 1 LP forcibly closed out (0042-LIQF-008)

    Given the average block duration is "601"

    When the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | lp1    | USD   | 10000    |
      | lp2    | USD   | 10000000 |
      | party1 | USD   | 10000000 |
      | party2 | USD   | 10000000 |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 1000              | 0.001 | buy  | BID              | 1          | 51     | submission |
      | lp1 | lp1   | ETH/MAR22 | 1000              | 0.001 | sell | ASK              | 1          | 51     | amendment  |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 9000              | 0.002 | buy  | BID              | 1          | 51     | submission |
      | lp2 | lp2   | ETH/MAR22 | 9000              | 0.002 | sell | ASK              | 1          | 51     | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | pa1-b1    |
      | party1 | ETH/MAR22 | buy  | 15     | 950   | 0                | TYPE_LIMIT | TIF_GTC | pa1-b2    |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | pa2-s1    |
      | lp1    | ETH/MAR22 | sell | 15     | 950   | 0                | TYPE_LIMIT | TIF_GTC | lp1-s1    |

    Then the opening auction period ends for market "ETH/MAR22"

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | bond |
      | lp1   | USD   | ETH/MAR22 | 5244   | 3756    | 1000 |


    # 1st set of trades: market moves against lp1s position, margin-insufficient, margin topped up from general and bond
    When the network moves ahead "1" blocks:

    And the parties amend the following orders:
      | party  | reference | price | size delta | tif     |
      | party1 | pa1-b1    | 1050  | 0          | TIF_GTC |
      | party2 | pa2-s1    | 1250  | 0          | TIF_GTC |

    And the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 30     | 1150  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 30     | 1150  | 1                | TYPE_LIMIT | TIF_GTC |

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1150  | 30   | party2 |

    # liquidity_fee = ceil(volume * price * liquidity_fee_factor) =  ceil(1150 * 30 * 0.002) = ceil(69) = 69

    And the accumulated liquidity fees should be "69" for the market "ETH/MAR22"

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | bond |
      | lp1   | USD   | ETH/MAR22 | 6348   | 0       | 652 |

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.1               | 1000                    |
      | lp2   | 0.9               | 10000                   |

    # Trigger liquidity distribution
    When the network moves ahead "1" blocks:

    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 6      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 62     | USD   |

    And the accumulated liquidity fees should be "1" for the market "ETH/MAR22"


    # 2nd set of trades: market moves against LP1s position, margin-insufficient, margin topped up from general and bond
    When the network moves ahead "1" blocks:

    When the parties amend the following orders:
      | party  | reference | price | size delta | tif     |
      | party1 | pa1-b1    | 1200  | 0          | TIF_GTC |
      | party2 | pa2-s1    | 1400  | 0          | TIF_GTC |

    And the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 30     | 1300  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 30     | 1300  | 1                | TYPE_LIMIT | TIF_GTC |

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1300  | 30   | party2 |

    # liquidity_fee = ceil(volume * price * liquidity_fee_factor) =  ceil(1300 * 30 * 0.002) = ceil(78) = 78

    And the accumulated liquidity fees should be "79" for the market "ETH/MAR22"

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | bond |
      | lp1   | USD   | ETH/MAR22 | 4756   | 0       | 0    |

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp2   | 1                 | 10000                   |

    # Trigger liquidity distribution
    When the network moves ahead "1" blocks:

    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 79     | USD   |

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"


    
