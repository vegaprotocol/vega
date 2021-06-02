Feature: Test liquidity provider reward distribution

# Spec file: ../spec/0042-setting-fees-and-rewarding-lps.md

  Background:
    Given the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 500         | -500          | 0.1                    |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the price monitoring updated every "1" seconds named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |
    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | oracle config          | maturity date        |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring     | default-eth-for-future | 2019-12-31T23:59:59Z |

    And the following network parameters are set:
      | name                                                | value   |
      | market.value.windowLength                           | 1h      |
      | market.stake.target.timeWindow                      | 24h     |
      | market.stake.target.scalingFactor                   | 1       |
      | market.liquidity.targetstake.triggering.ratio       | 0       |
      | market.liquidity.providers.fee.distributionTimeStep | 10m     |


  Scenario: 1 LP joining at start, checking liquidity rewards over 3 periods, 1 period with no trades
    # setup accounts
    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount     |
      | lp1     | ETH   | 1000000000 |
      | trader1 | ETH   | 100000000  |
      | trader2 | ETH   | 100000000  |

    And the traders submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
      | lp1 | lp1   | ETH/DEC21 | 10000             | 0.001 | buy        | BID             | 1                | -2           |
      | lp1 | lp1   | ETH/DEC21 | 10000             | 0.001 | buy        | MID             | 2                | -1           |
      | lp1 | lp1   | ETH/DEC21 | 10000             | 0.001 | sell       | ASK             | 1                | 2            |
      | lp1 | lp1   | ETH/DEC21 | 10000             | 0.001 | sell       | MID             | 2                | 1            |

    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"

    And the following trades should be executed:
      | buyer   | price | size | seller  |
      | trader1 | 1000  | 10   | trader2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the mark price should be "1000" for the market "ETH/DEC21"
    And the open interest should be "10" for the market "ETH/DEC21"
    And the target stake should be "1000" for the market "ETH/DEC21"
    And the supplied stake should be "10000" for the market "ETH/DEC21"

    And the liquidity provider fee shares for the market "ETH/DEC21" should be:
      | party | equity like share | average entry valuation |
      | lp1   |                 1 |                   10000 |

    And the price monitoring bounds for the market "ETH/DEC21" should be:
      | min bound | max bound |
      |       500 |     1500  |

    And the liquidity fee factor should "0.001" for the market "ETH/DEC21"

    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | buy  | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the accumulated liquidity fees should be "20" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    Then the following transfers should happen:
      | from    | to  | from account                | to account           | market id | amount  | asset |
      | market  | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_MARGIN  | ETH/DEC21 | 20      | ETH   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    Then time is updated to "2019-11-30T00:20:05Z"

    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 40     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 40     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    Then debug orders

    # here we get only a trade for a volume of 15 as it's what was on the LP
    # order, then the 25 remaining from trader1 are cancelled for self trade
    And the following trades should be executed:
      | buyer   | price | size | seller  |
      | trader1 | 951   | 15   | lp1     |

    # this is slightly different than expected, as the trades happen against the LP,
    # which is probably not what you expected initially
    And the accumulated liquidity fees should be "15" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:30:05Z"

    Then the following transfers should happen:
      | from    | to  | from account                | to account           | market id | amount  | asset |
      | market  | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_MARGIN  | ETH/DEC21 | 15      | ETH   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"


  Scenario: 2 LPs joining at start, equal commitments

    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount     |
      | lp1     | ETH   | 1000000000 |
      | lp2     | ETH   | 1000000000 |
      | trader1 | ETH   | 100000000  |
      | trader2 | ETH   | 100000000  |

    And the traders submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
      | lp1 | lp1   | ETH/DEC21 | 5000              | 0.001 | buy        | BID             | 1                | -2           |
      | lp1 | lp1   | ETH/DEC21 | 5000              | 0.001 | buy        | MID             | 2                | -1           |
      | lp1 | lp1   | ETH/DEC21 | 5000              | 0.001 | sell       | ASK             | 1                | 2            |
      | lp1 | lp1   | ETH/DEC21 | 5000              | 0.001 | sell       | MID             | 2                | 1            |
    And the traders submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
      | lp2 | lp2   | ETH/DEC21 | 5000              | 0.002 | buy        | BID             | 1                | -2           |
      | lp2 | lp2   | ETH/DEC21 | 5000              | 0.002 | buy        | MID             | 2                | -1           |
      | lp2 | lp2   | ETH/DEC21 | 5000              | 0.002 | sell       | ASK             | 1                | 2            |
      | lp2 | lp2   | ETH/DEC21 | 5000              | 0.002 | sell       | MID             | 2                | 1            |

    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC21 | buy  | 90     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 90     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"

    And the following trades should be executed:
      | buyer   | price | size | seller  |
      | trader1 | 1000  | 90   | trader2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the mark price should be "1000" for the market "ETH/DEC21"
    And the open interest should be "90" for the market "ETH/DEC21"
    And the target stake should be "9000" for the market "ETH/DEC21"
    And the supplied stake should be "10000" for the market "ETH/DEC21"

    And the liquidity provider fee shares for the market "ETH/DEC21" should be:
      | party | equity like share  | average entry valuation |
      | lp1   |                0.5 |                   10000 |
      | lp2   |                0.5 |                   10000 |

    And the price monitoring bounds for the market "ETH/DEC21" should be:
      | min bound | max bound |
      |       500 |     1500  |

    And the liquidity fee factor should "0.002" for the market "ETH/DEC21"

    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | buy  | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    Then debug trades
    And the following trades should be executed:
      | buyer   | price | size | seller  |
      | trader2 | 951   | 8    | lp1     |
      | trader2 | 951   | 8    | lp2     |
      | trader2 | 1000  | 4    | trader1 |

    And the accumulated liquidity fees should be "40" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from    | to  | from account                | to account           | market id | amount  | asset |
      | market  | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_MARGIN  | ETH/DEC21 | 20      | ETH   |
      | market  | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_MARGIN  | ETH/DEC21 | 20      | ETH   |


    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 40     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 40     | 1100  | 1                | TYPE_LIMIT | TIF_GTC |

    And the following trades should be executed:
      | buyer   | price | size | seller  |
      | trader1 | 951   | 8    | lp1     |
      | trader1 | 951   | 8    | lp2     |

    And the accumulated liquidity fees should be "32" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:20:08Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from    | to  | from account                | to account           | market id | amount  | asset |
      | market  | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_MARGIN  | ETH/DEC21 | 16      | ETH   |
      | market  | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_MARGIN  | ETH/DEC21 | 16      | ETH   |

  Scenario: 2 LPs joining at start, unequal commitments

    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount     |
      | lp1     | ETH   | 1000000000 |
      | lp2     | ETH   | 1000000000 |
      | trader1 | ETH   | 100000000  |
      | trader2 | ETH   | 100000000  |

    And the traders submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | buy        | BID             | 1                | -2           |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | buy        | MID             | 2                | -1           |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | sell       | ASK             | 1                | 2            |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | sell       | MID             | 2                | 1            |
    And the traders submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | buy        | BID             | 1                | -2           |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | buy        | MID             | 2                | -1           |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | sell       | ASK             | 1                | 2            |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | sell       | MID             | 2                | 1            |

    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC21 | buy  | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"

    And the following trades should be executed:
      | buyer   | price | size | seller  |
      | trader1 | 1000  | 60   | trader2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the mark price should be "1000" for the market "ETH/DEC21"
    And the open interest should be "60" for the market "ETH/DEC21"
    And the target stake should be "6000" for the market "ETH/DEC21"
    And the supplied stake should be "10000" for the market "ETH/DEC21"

    And the liquidity provider fee shares for the market "ETH/DEC21" should be:
      | party | equity like share  | average entry valuation |
      | lp1   |                0.8 |                   10000 |
      | lp2   |                0.2 |                   10000 |

    And the price monitoring bounds for the market "ETH/DEC21" should be:
      | min bound | max bound |
      |       500 |     1500  |

    And the liquidity fee factor should "0.001" for the market "ETH/DEC21"

    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | buy  | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"

    And the following trades should be executed:
      | buyer   | price | size | seller  |
      | trader2 | 951   | 3    | lp2     |
      | trader2 | 951   | 12   | lp1     |
      | trader2 | 1000  | 5    | trader1 |

    And the accumulated liquidity fees should be "20" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from    | to  | from account                | to account           | market id | amount  | asset |
      | market  | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_MARGIN  | ETH/DEC21 | 16      | ETH   |
      | market  | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_MARGIN  | ETH/DEC21 | 4       | ETH   |


    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 40     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 40     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    And the following trades should be executed:
      | buyer   | price | size | seller  |
      | trader1 | 951   | 12   | lp1     |
      | trader1 | 951   | 3    | lp2     |

    And the accumulated liquidity fees should be "15" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:20:06Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from    | to  | from account                | to account           | market id | amount  | asset |
      | market  | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_MARGIN  | ETH/DEC21 | 12      | ETH   |
      | market  | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_MARGIN  | ETH/DEC21 | 3       | ETH   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

  Scenario: 2 LPs joining at start, unequal commitments, 1 LP joining later

    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount     |
      | lp1     | ETH   | 1000000000 |
      | lp2     | ETH   | 1000000000 |
      | lp3     | ETH   | 1000000000 |
      | trader1 | ETH   | 100000000  |
      | trader2 | ETH   | 100000000  |

    And the traders submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | buy        | BID             | 1                | -2           |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | buy        | MID             | 2                | -1           |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | sell       | ASK             | 1                | 2            |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | sell       | MID             | 2                | 1            |
    And the traders submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | buy        | BID             | 1                | -2           |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | buy        | MID             | 2                | -1           |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | sell       | ASK             | 1                | 2            |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | sell       | MID             | 2                | 1            |

    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | trader1 | ETH/DEC21 | buy  | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"

    And the following trades should be executed:
      | buyer   | price | size | seller  |
      | trader1 | 1000  | 60   | trader2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the mark price should be "1000" for the market "ETH/DEC21"
    And the open interest should be "60" for the market "ETH/DEC21"
    And the target stake should be "6000" for the market "ETH/DEC21"
    And the supplied stake should be "10000" for the market "ETH/DEC21"

    And the liquidity provider fee shares for the market "ETH/DEC21" should be:
      | party | equity like share  | average entry valuation |
      | lp1   |                0.8 |                   10000 |
      | lp2   |                0.2 |                   10000 |

    And the price monitoring bounds for the market "ETH/DEC21" should be:
      | min bound | max bound |
      |       500 |     1500  |

    And the liquidity fee factor should "0.001" for the market "ETH/DEC21"

    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"


    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | buy  | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    And the following trades should be executed:
      | buyer   | price | size | seller  |
      | trader2 | 951   | 3    | lp2     |
      | trader2 | 951   | 12   | lp1     |
      | trader2 | 1000  | 5    | trader1 |

    And the accumulated liquidity fees should be "20" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from    | to  | from account                | to account           | market id | amount  | asset |
      | market  | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_MARGIN  | ETH/DEC21 | 16      | ETH   |
      | market  | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_MARGIN  | ETH/DEC21 | 4       | ETH   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    And the traders submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
      | lp3 | lp3   | ETH/DEC21 | 10000              | 0.001 | buy        | BID             | 1                | -2           |
      | lp3 | lp3   | ETH/DEC21 | 10000              | 0.001 | buy        | MID             | 2                | -1           |
      | lp3 | lp3   | ETH/DEC21 | 10000              | 0.001 | sell       | ASK             | 1                | 2            |
      | lp3 | lp3   | ETH/DEC21 | 10000              | 0.001 | sell       | MID             | 2                | 1            |

    And the liquidity provider fee shares for the market "ETH/DEC21" should be:
      | party | equity like share  | average entry valuation |
      | lp1   |             0.7362 |                   10000 |
      | lp2   |             0.1840 |                   10000 |
      | lp3   |             0.0797 |                  115397 |

    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     |
      | trader1 | ETH/DEC21 | buy  | 40     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC21 | sell | 40     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    And the following trades should be executed:
      | buyer   | price | size | seller  |
      | trader1 | 951   | 12   | lp1     |
      | trader1 | 951   | 3    | lp2     |
      | trader1 | 951   | 15   | lp3     |

    And the accumulated liquidity fees should be "30" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:20:06Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from    | to  | from account                | to account           | market id | amount  | asset |
      | market  | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_MARGIN  | ETH/DEC21 | 22      | ETH   |
      | market  | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_MARGIN  | ETH/DEC21 | 5       | ETH   |
      | market  | lp3 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_MARGIN  | ETH/DEC21 | 3       | ETH   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"
