Feature: Test liquidity provider reward distribution

# Spec file: ../spec/0042-setting-fees-and-rewarding-lps.md

  Background:
    Given the simple risk model named "simple-risk-model-1": 
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 500         | 500           | 0.1                    |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the price monitoring updated every "1" seconds named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |
    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | oracle config          |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future |

    And the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 1h    |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 0     |
      | market.liquidity.providers.fee.distributionTimeStep | 10m   |

  Given the average block duration is "2"

  Scenario: 1 LP joining at start, checking liquidity rewards over 3 periods, 1 period with no trades
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1     | ETH   | 1000000000 |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type |
      | lp1 | lp1   | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | lp1   | ETH/DEC21 | 10000             | 0.001 | buy  | MID              | 2          | 1      | amendment |
      | lp1 | lp1   | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | lp1   | ETH/DEC21 | 10000             | 0.001 | sell | MID              | 2          | 1      | amendment |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"

    And the following trades should be executed:
      | buyer   | price | size | seller  |
      | party1 | 1000  | 10   | party2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the mark price should be "1000" for the market "ETH/DEC21"
    And the open interest should be "10" for the market "ETH/DEC21"
    And the target stake should be "1000" for the market "ETH/DEC21"
    And the supplied stake should be "10000" for the market "ETH/DEC21"

    And the liquidity provider fee shares for the market "ETH/DEC21" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 1                 | 10000                   |

    Then the network moves ahead "1" blocks

    And the price monitoring bounds for the market "ETH/DEC21" should be:
      | min bound | max bound |
      | 500       | 1500      | 

    And the liquidity fee factor should "0.001" for the market "ETH/DEC21"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | party1 | ETH/DEC21 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/DEC21 | buy  | 20     | 1000  | 2                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the accumulated liquidity fees should be "20" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 20     | ETH   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    Then time is updated to "2019-11-30T00:20:05Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | party1 | ETH/DEC21 | buy  | 40     | 1100  | 1                | TYPE_LIMIT | TIF_GTC | party1-buy  |
      | party2 | ETH/DEC21 | sell | 40     | 1100  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell |

    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"

    # here we get only a trade for a volume of 15 as it's what was on the LP
    # order, then the 25 remaining from party1 are cancelled for self trade
    And the following trades should be executed:
      | buyer   | price | size | seller |
      | party1 | 951   | 15   | lp1    |

    # this is slightly different than expected, as the trades happen against the LP,
    # which is probably not what you expected initially
    And the accumulated liquidity fees should be "15" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:30:05Z"

    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 15     | ETH   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"


  Scenario: 2 LPs joining at start, equal commitments

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1     | ETH   | 1000000000 |
      | lp2     | ETH   | 1000000000 |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type |
      | lp1 | lp1   | ETH/DEC21 | 5000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | lp1   | ETH/DEC21 | 5000              | 0.001 | buy  | MID              | 2          | 1      | amendment |
      | lp1 | lp1   | ETH/DEC21 | 5000              | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | lp1   | ETH/DEC21 | 5000              | 0.001 | sell | MID              | 2          | 1      | amendment |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type |
      | lp2 | lp2   | ETH/DEC21 | 5000              | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/DEC21 | 5000              | 0.002 | buy  | MID              | 2          | 1      | amendment |
      | lp2 | lp2   | ETH/DEC21 | 5000              | 0.002 | sell | ASK              | 1          | 2      | amendment |
      | lp2 | lp2   | ETH/DEC21 | 5000              | 0.002 | sell | MID              | 2          | 1      | amendment |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 90     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 90     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"

    And the following trades should be executed:
      | buyer   | price | size | seller  |
      | party1 | 1000  | 90   | party2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the mark price should be "1000" for the market "ETH/DEC21"
    And the open interest should be "90" for the market "ETH/DEC21"
    And the target stake should be "9000" for the market "ETH/DEC21"
    And the supplied stake should be "10000" for the market "ETH/DEC21"

    And the liquidity provider fee shares for the market "ETH/DEC21" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.5               | 10000                   |
      | lp2   | 0.5               | 10000                   |

    And the price monitoring bounds for the market "ETH/DEC21" should be:
      | min bound | max bound |
      | 500       | 1500      |

    And the liquidity fee factor should "0.002" for the market "ETH/DEC21"

    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | party1 | ETH/DEC21 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/DEC21 | buy  | 20     | 1000  | 3                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the following trades should be executed:
      | buyer   | price | size | seller  |
      | party2 | 951   | 8    | lp1     |
      | party2 | 951   | 8    | lp2     |
      | party2 | 1000  | 4    | party1 |

    And the accumulated liquidity fees should be "40" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 20     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 20     | ETH   |


    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | party1 | ETH/DEC21 | buy  | 40     | 1100  | 2                | TYPE_LIMIT | TIF_GTC | party1-buy  |
      | party2 | ETH/DEC21 | sell | 40     | 1100  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell |

    And the following trades should be executed:
      | buyer   | price | size | seller |
      | party1 | 951   | 8    | lp1    |
      | party1 | 951   | 8    | lp2    |

    And the accumulated liquidity fees should be "32" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:20:08Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 16     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 16     | ETH   |

  Scenario: 2 LPs joining at start, unequal commitments

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1     | ETH   | 1000000000 |
      | lp2     | ETH   | 1000000000 |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | buy  | MID              | 2          | 1      | amendment |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | sell | MID              | 2          | 1      | amendment |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | buy  | MID              | 2          | 1      | amendment |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | sell | ASK              | 1          | 2      | amendment |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | sell | MID              | 2          | 1      | amendment |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"

    And the following trades should be executed:
      | buyer   | price | size | seller  |
      | party1 | 1000  | 60   | party2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the mark price should be "1000" for the market "ETH/DEC21"
    And the open interest should be "60" for the market "ETH/DEC21"
    And the target stake should be "6000" for the market "ETH/DEC21"
    And the supplied stake should be "10000" for the market "ETH/DEC21"

    And the liquidity provider fee shares for the market "ETH/DEC21" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.8               | 10000                   |
      | lp2   | 0.2               | 10000                   |

    And the price monitoring bounds for the market "ETH/DEC21" should be:
      | min bound | max bound |
      | 500       | 1500      |

    And the liquidity fee factor should "0.001" for the market "ETH/DEC21"

    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | party1 | ETH/DEC21 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/DEC21 | buy  | 20     | 1000  | 3                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"

    And the following trades should be executed:
      | buyer  | price | size | seller  |
      | party2 | 951   | 12   | lp1     |
      | party2 | 951   | 3    | lp2     |
      | party2 | 1000  | 5    | party1 |

    And the accumulated liquidity fees should be "20" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 16     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 4      | ETH   |


    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | party1 | ETH/DEC21 | buy  | 40     | 1000  | 2                | TYPE_LIMIT | TIF_GTC | party1-buy  |
      | party2 | ETH/DEC21 | sell | 40     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell |

    And the following trades should be executed:
      | buyer   | price | size | seller |
      | party1 | 951   | 12   | lp1    |
      | party1 | 951   | 3    | lp2    |

    And the accumulated liquidity fees should be "15" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:20:06Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 12     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 3      | ETH   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

  Scenario: 2 LPs joining at start, unequal commitments, 1 LP joining later

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1     | ETH   | 1000000000 |
      | lp2     | ETH   | 1000000000 |
      | lp3     | ETH   | 1000000000 |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | buy  | MID              | 2          | 1      | amendment |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | sell | MID              | 2          | 1      | amendment |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | buy  | MID              | 2          | 1      | amendment |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | sell | ASK              | 1          | 2      | amendment |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | sell | MID              | 2          | 1      | amendment |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"

    And the following trades should be executed:
      | buyer   | price | size | seller  |
      | party1 | 1000  | 60   | party2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the mark price should be "1000" for the market "ETH/DEC21"
    And the open interest should be "60" for the market "ETH/DEC21"
    And the target stake should be "6000" for the market "ETH/DEC21"
    And the supplied stake should be "10000" for the market "ETH/DEC21"

    And the liquidity provider fee shares for the market "ETH/DEC21" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.8               | 10000                   |
      | lp2   | 0.2               | 10000                   |

    And the price monitoring bounds for the market "ETH/DEC21" should be:
      | min bound | max bound |
      | 500       | 1500      |

    And the liquidity fee factor should "0.001" for the market "ETH/DEC21"

    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"


    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | party1 | ETH/DEC21 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/DEC21 | buy  | 20     | 1000  | 3                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party2 | 951   | 12   | lp1    |
      | party2 | 951   | 3    | lp2    |
      | party2 | 1000  | 5    | party1 |

    And the accumulated liquidity fees should be "20" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 16     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 4      | ETH   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type |
      | lp3 | lp3   | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | 2      | submission|
      | lp3 | lp3   | ETH/DEC21 | 10000             | 0.001 | buy  | MID              | 2          | 1      | amendment |
      | lp3 | lp3   | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp3 | lp3   | ETH/DEC21 | 10000             | 0.001 | sell | MID              | 2          | 1      | amendment |

    And the liquidity provider fee shares for the market "ETH/DEC21" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.7362029616262406 | 10000                   |
      | lp2   | 0.1840507404065602 | 10000                   |
      | lp3   | 0.0797462979671992 | 115397.670549084859473  |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 40     | 1000  | 3                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 40     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    And the following trades should be executed:
      | buyer   | price | size | seller |
      | party1 | 951   | 12   | lp1    |
      | party1 | 951   | 3    | lp2    |
      | party1 | 951   | 15   | lp3    |

    And the accumulated liquidity fees should be "30" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:20:06Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 22     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 5      | ETH   |
      | market | lp3 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 3      | ETH   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"
