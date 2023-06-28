Feature: Simple example of successor markets

  Background:
    Given time is updated to "2019-11-30T00:00:00Z"
    And the following assets are registered:
      | id  | decimal places |
      | ETH | 18             |
      | USD | 0              |

    # Create some oracles
    ## oracle for parent
    And the oracle spec for settlement data filtering data from "0xCAFECAFE1" named "ethDec19Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |
    And the oracle spec for trading termination filtering data from "0xCAFECAFE1" named "ethDec19Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |
    And the settlement data decimals for the oracle named "ethDec19Oracle" is given in "5" decimal places
    ## oracle for a successor
    And the oracle spec for settlement data filtering data from "0xCAFECAFE" named "ethDec20Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |
    And the oracle spec for trading termination filtering data from "0xCAFECAFE" named "ethDec20Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |
    And the settlement data decimals for the oracle named "ethDec20Oracle" is given in "5" decimal places

    And the following network parameters are set:
      | name                                          | value |
      | network.markPriceUpdateMaximumFrequency       | 0s    |
      | market.liquidity.targetstake.triggering.ratio | 0.01  |
      | market.stake.target.timeWindow                | 10s   |
      | market.stake.target.scalingFactor             | 5     |
      | market.auction.minimumDuration                | 1     |
      | market.fee.factors.infrastructureFee          | 0.001 |
      | market.fee.factors.makerFee                   | 0.004 |
      | market.value.windowLength                     | 60s   |
      | market.liquidity.bondPenaltyParameter         | 0.1   |
      | market.liquidityProvision.shapes.maxSize      | 10    |
      | validators.epoch.length                       | 5s    |
      | market.liquidity.stakeToCcyVolume             | 0.2   |
      | market.liquidity.successorLaunchWindowLength | 1h |
    And the average block duration is "1"


# All parties have 1,000,000.000,000,000,000,000,000
    # Add as many parties as needed here
    And the parties deposit on asset's general account the following amount:
      | party   | asset | amount                     |
      | lpprov  | ETH   | 10000000000000000000000000 |
      | trader1 | ETH   | 10000000000000000000000000 |
      | trader2 | ETH   | 10000000000000000000000000 |
      | trader3 | ETH   | 10000000000000000000000000 |
      | trader4 | ETH   | 10000000000000000000000000 |
      | trader5 | ETH   | 10000000000000000000000000 |


  @SuccessorMarketSimple
  Scenario: 001 Enact a successor market once the parent market is settled
    Given the markets:
      | id        | quote name | asset | risk model            | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | decimal places | position decimal places | parent market id | insurance pool fraction | successor auction |
      | ETH/DEC19 | ETH        | ETH   | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | ethDec19Oracle         | 0.1                    | 0                         | 5              | 5                       |                  |                         |                   |
      | ETH/DEC20 | ETH        | ETH   | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.1                    | 0                         | 5              | 5                       | ETH/DEC19        | 1                       | 10                |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 3905000000000000  | 0.3 | buy  | BID              | 2          | 1      | submission |
      | lp1 | lpprov | ETH/DEC19 | 3905000000000000  | 0.3 | sell | ASK              | 13         | 1      | submission |
    And the parties place the following orders:
      | party   | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 5      | 1001   | 0                | TYPE_LIMIT | TIF_GTC | t1-b-1    |
      | trader1 | ETH/DEC19 | buy  | 5      | 900    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-2    |
      | trader1 | ETH/DEC19 | buy  | 1      | 100    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-3    |
      | trader2 | ETH/DEC19 | sell | 5      | 1200   | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |
      | trader2 | ETH/DEC19 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-2    |
      | trader2 | ETH/DEC19 | sell | 5      | 951    | 0                | TYPE_LIMIT | TIF_GTC | t2-s-3    |
    When the opening auction period ends for market "ETH/DEC19"
    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake   | open interest |
      | 976        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 134907600000 | 3905000000000000 | 5             |
    And the parties should have the following account balances:
      | party   | asset | market id | margin       | general                   |
      | trader1 | ETH   | ETH/DEC19 | 113402285504 | 9999999999999886597714496 |
    And the parties should have the following margin levels:
      | party   | market id | maintenance | search       | initial      | release      |
      | trader1 | ETH/DEC19 | 94501904587 | 103952095045 | 113402285504 | 132302666421 |
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 1905000000000000  | 0.1 | buy  | BID              | 2          | 1      | submission |
      | lp1 | lpprov | ETH/DEC20 | 1905000000000000  | 0.1 | sell | ASK              | 13         | 1      | submission |
    Then the oracles broadcast data signed with "0xCAFECAFE1":
      | name               | value |
      | trading.terminated | true  |
      | prices.ETH.value   | 975   |

    When the successor market "ETH/DEC20" is enacted
    Then the parties place the following orders:
      | party   | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | buy  | 5      | 1001   | 0                | TYPE_LIMIT | TIF_GTC | t1-b-1    |
      | trader1 | ETH/DEC20 | buy  | 5      | 900    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-2    |
      | trader1 | ETH/DEC20 | buy  | 1      | 100    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-3    |
      | trader2 | ETH/DEC20 | sell | 5      | 1200   | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |
      | trader2 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-2    |
      | trader2 | ETH/DEC20 | sell | 5      | 951    | 0                | TYPE_LIMIT | TIF_GTC | t2-s-3    |
    When the opening auction period ends for market "ETH/DEC20"
    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake   | open interest |
      | 976        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 134907600000 | 1905000000000000 | 5             |

  @SuccessorMarketSimple
  Scenario: 002 Enacting a successor market rejects any other pending successors
    ## parent market and 2 successors
    Given the markets:
      | id        | quote name | asset | risk model            | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | decimal places | position decimal places | parent market id | insurance pool fraction | successor auction |
      | ETH/DEC19 | ETH        | ETH   | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | ethDec19Oracle         | 0.1                    | 0                         | 5              | 5                       |                  |                         |                   |
      | ETH/DEC20 | ETH        | ETH   | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | ethDec20Oracle         | 0.1                    | 0                         | 5              | 5                       | ETH/DEC19        | 1                       | 10                |
      | ETH/DEC21 | ETH        | ETH   | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.1                    | 0                         | 5              | 5                       | ETH/DEC19        | 1                       | 10                |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 3905000000000000  | 0.3 | buy  | BID              | 2          | 1      | submission |
      | lp1 | lpprov | ETH/DEC19 | 3905000000000000  | 0.3 | sell | ASK              | 13         | 1      | submission |
    And the parties place the following orders:
      | party   | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 5      | 1001   | 0                | TYPE_LIMIT | TIF_GTC | t1-b-1    |
      | trader1 | ETH/DEC19 | buy  | 5      | 900    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-2    |
      | trader1 | ETH/DEC19 | buy  | 1      | 100    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-3    |
      | trader2 | ETH/DEC19 | sell | 5      | 1200   | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |
      | trader2 | ETH/DEC19 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-2    |
      | trader2 | ETH/DEC19 | sell | 5      | 951    | 0                | TYPE_LIMIT | TIF_GTC | t2-s-3    |
    # Both successor markets should be pending
    Then the market state should be "STATE_PENDING" for the market "ETH/DEC20"
    And the market state should be "STATE_PENDING" for the market "ETH/DEC21"
    When the opening auction period ends for market "ETH/DEC19"
    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake   | open interest |
      | 976        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 134907600000 | 3905000000000000 | 5             |
    And the parties should have the following account balances:
      | party   | asset | market id | margin       | general                   |
      | trader1 | ETH   | ETH/DEC19 | 113402285504 | 9999999999999886597714496 |
    And the parties should have the following margin levels:
      | party   | market id | maintenance | search       | initial      | release      |
      | trader1 | ETH/DEC19 | 94501904587 | 103952095045 | 113402285504 | 132302666421 |
    # LP submissions are being made on both pending markets
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lpprov | ETH/DEC20 | 1905000000000000  | 0.1 | buy  | BID              | 2          | 1      | submission |
      | lp2 | lpprov | ETH/DEC20 | 1905000000000000  | 0.1 | sell | ASK              | 13         | 1      | submission |
      | lp3 | lpprov | ETH/DEC21 | 1905000000000000  | 0.1 | buy  | BID              | 2          | 1      | submission |
      | lp3 | lpprov | ETH/DEC21 | 1905000000000000  | 0.1 | sell | ASK              | 13         | 1      | submission |
    Then the oracles broadcast data signed with "0xCAFECAFE1":
      | name               | value |
      | trading.terminated | true  |
    And the parties should have the following account balances:
      | party  | asset | market id | margin         | general                   | bond             |
      | lpprov | ETH   | ETH/DEC19 | 53551477859983 | 9999999992231448522140017 | 3905000000000000 |
      | lpprov | ETH   | ETH/DEC20 | 0              | 9999999992231448522140017 | 1905000000000000 |
      | lpprov | ETH   | ETH/DEC21 | 0              | 9999999992231448522140017 | 1905000000000000 |

    When the successor market "ETH/DEC21" is enacted
    Then the network moves ahead "1" blocks
    # The bond for market ETH/DEC20 should be released back to the general balance
    And the parties should have the following account balances:
      | party  | asset | market id | margin         | general                   | bond             |
      | lpprov | ETH   | ETH/DEC19 | 53551477859983 | 9999999992231448522140017 | 3905000000000000 |
      | lpprov | ETH   | ETH/DEC20 | 0              | 9999999992231448522140017 | 1905000000000000 |
      | lpprov | ETH   | ETH/DEC21 | 0              | 9999999992231448522140017 | 1905000000000000 |
    Then the market state should be "STATE_PENDING" for the market "ETH/DEC20"
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC19"
    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name             | value |
      | prices.ETH.value | 975   |
    Then the market state should be "STATE_SETTLED" for the market "ETH/DEC19"

    Then the parties place the following orders:
      | party   | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC21 | buy  | 5      | 1001   | 0                | TYPE_LIMIT | TIF_GTC | t1-b-1    |
      | trader1 | ETH/DEC21 | buy  | 5      | 900    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-2    |
      | trader1 | ETH/DEC21 | buy  | 1      | 100    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-3    |
      | trader2 | ETH/DEC21 | sell | 5      | 1200   | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |
      | trader2 | ETH/DEC21 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-2    |
      | trader2 | ETH/DEC21 | sell | 5      | 951    | 0                | TYPE_LIMIT | TIF_GTC | t2-s-3    |
    When the opening auction period ends for market "ETH/DEC21"
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake   | open interest |
      | 976        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 134907600000 | 1905000000000000 | 5             |

    When the network moves ahead "1" blocks
    # The bond for market ETH/DEC20 should be released back to the general balance
    Then the parties should have the following account balances:
      | party  | asset | market id | margin         | general                   | bond             |
      | lpprov | ETH   | ETH/DEC19 | 0              | 9999999998068871464704366 | 0                |
      | lpprov | ETH   | ETH/DEC20 | 0              | 9999999998068871464704366 | 0                |
      | lpprov | ETH   | ETH/DEC21 | 26128535295634 | 9999999998068871464704366 | 1905000000000000 |

    And the last market state should be "STATE_REJECTED" for the market "ETH/DEC20"
    And the parties should have the following account balances:
      | party  | asset | market id | margin         | general                   | bond             |
      | lpprov | ETH   | ETH/DEC19 | 0              | 9999999998068871464704366 | 0                |
      | lpprov | ETH   | ETH/DEC20 | 0              | 9999999998068871464704366 | 0                |
      | lpprov | ETH   | ETH/DEC21 | 26128535295634 | 9999999998068871464704366 | 1905000000000000 |

  @SuccessorMarketSimple
  Scenario: 003 Enact a successor market while the parent market is still in active state,
    Given the markets:
      | id        | quote name | asset | risk model            | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | decimal places | position decimal places | parent market id | insurance pool fraction | successor auction |
      | ETH/DEC19 | ETH        | ETH   | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | ethDec19Oracle         | 0.1                    | 0                         | 5              | 5                       |                  |                         |                   |
      | ETH/DEC20 | ETH        | ETH   | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.1                    | 0                         | 5              | 5                       | ETH/DEC19        | 1                       | 10                |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 3905000000000000  | 0.3 | buy  | BID              | 2          | 1      | submission |
      | lp1 | lpprov | ETH/DEC19 | 3905000000000000  | 0.3 | sell | ASK              | 13         | 1      | submission |
    And the parties place the following orders:
      | party   | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 5      | 1001   | 0                | TYPE_LIMIT | TIF_GTC | t1-b-1    |
      | trader1 | ETH/DEC19 | buy  | 5      | 900    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-2    |
      | trader1 | ETH/DEC19 | buy  | 1      | 100    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-3    |
      | trader2 | ETH/DEC19 | sell | 5      | 1200   | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |
      | trader2 | ETH/DEC19 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-2    |
      | trader2 | ETH/DEC19 | sell | 5      | 951    | 0                | TYPE_LIMIT | TIF_GTC | t2-s-3    |
    When the opening auction period ends for market "ETH/DEC19"
    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake   | open interest |
      | 976        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 134907600000 | 3905000000000000 | 5             |
    And the parties should have the following account balances:
      | party   | asset | market id | margin       | general                   |
      | trader1 | ETH   | ETH/DEC19 | 113402285504 | 9999999999999886597714496 |
    And the parties should have the following margin levels:
      | party   | market id | maintenance | search       | initial      | release      |
      | trader1 | ETH/DEC19 | 94501904587 | 103952095045 | 113402285504 | 132302666421 |

    # Parent market is still active at this point
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 1905000000000000  | 0.1 | buy  | BID              | 2          | 1      | submission |
      | lp1 | lpprov | ETH/DEC20 | 1905000000000000  | 0.1 | sell | ASK              | 13         | 1      | submission |
    Then the successor market "ETH/DEC20" is enacted
    # fill up the successor market orderbook
    And the parties place the following orders:
      | party   | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | buy  | 5      | 1001   | 0                | TYPE_LIMIT | TIF_GTC | t1-b-1    |
      | trader1 | ETH/DEC20 | buy  | 5      | 900    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-2    |
      | trader1 | ETH/DEC20 | buy  | 1      | 100    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-3    |
      | trader2 | ETH/DEC20 | sell | 5      | 1200   | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |
      | trader2 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-2    |
      | trader2 | ETH/DEC20 | sell | 5      | 951    | 0                | TYPE_LIMIT | TIF_GTC | t2-s-3    |
    # time progresses some more, and leave auctio
    When the opening auction period ends for market "ETH/DEC20"
    # successor market is enacted without issue
    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake   | open interest |
      | 976        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 134907600000 | 1905000000000000 | 5             |
    # Now terminate the parent market
    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name               | value |
      | trading.terminated | true  |
      | prices.ETH.value   | 975   |
    # ensure the parent market is settled, but the successor market is still going
    Then the last market state should be "STATE_SETTLED" for the market "ETH/DEC19"
    And the last market state should be "STATE_ACTIVE" for the market "ETH/DEC20"
    And the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake   | open interest |
      | 976        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 134907600000 | 1905000000000000 | 5             |

  @SuccessorMarketSimple
  Scenario: 004 Enact a successor market while the parent market is still in pending state, 0081-SUCM-009, 0081-SUCM-010, 0081-SUCM-011
    Given the markets:
      | id        | quote name | asset | risk model            | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | decimal places | position decimal places | parent market id | insurance pool fraction | successor auction |
      | ETH/DEC19 | ETH        | ETH   | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | ethDec19Oracle         | 0.1                    | 0                         | 5              | 5                       |                  |                         |                   |
      | ETH/DEC20 | ETH        | ETH   | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.1                    | 0                         | 5              | 5                       | ETH/DEC19        | 1                       | 10                |
      | ETH/DEC21 | ETH        | ETH   | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.1                    | 0                         | 5              | 5                       | ETH/DEC19        | 1                       | 10                |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 3905000000000000  | 0.3 | buy  | BID              | 2          | 1      | submission |
      | lp1 | lpprov | ETH/DEC19 | 3905000000000000  | 0.3 | sell | ASK              | 13         | 1      | submission |
    And the parties place the following orders:
      | party   | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 5      | 1001   | 0                | TYPE_LIMIT | TIF_GTC | t1-b-1    |
      | trader2 | ETH/DEC19 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-2    |
      | trader2 | ETH/DEC19 | sell | 5      | 9510   | 0                | TYPE_LIMIT | TIF_GTC | t2-s-3    |

    # When the opening auction period ends for market "ETH/DEC19"
    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode                 | auction trigger         |
      | 0          | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING |

    # Parent market is still pending at this point
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 1905000000000000  | 0.1 | buy  | BID              | 2          | 1      | submission |
      | lp1 | lpprov | ETH/DEC20 | 1905000000000000  | 0.1 | sell | ASK              | 13         | 1      | submission |
      | lp2 | lpprov | ETH/DEC21 | 1905000000000000  | 0.1 | buy  | BID              | 2          | 1      | submission |
      | lp2 | lpprov | ETH/DEC21 | 1905000000000000  | 0.1 | sell | ASK              | 13         | 1      | submission |

    And the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | buy  | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-1    |
      | trader1 | ETH/DEC20 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | t2-s-2    |
      | trader2 | ETH/DEC20 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | t2-s-2    |
      | trader2 | ETH/DEC20 | sell | 10     | 2000  | 0                | TYPE_LIMIT | TIF_GTC | t2-s-3    |

    When the opening auction period ends for market "ETH/DEC20"
    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake   | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 276450000000 | 1905000000000000 | 10            |
    Then the successor market "ETH/DEC20" is enacted
    And the last market state should be "STATE_REJECTED" for the market "ETH/DEC21"

    #When a successor market is enacted (i.e. leaves the opening auction), all other related successor market proposals, in the state "pending" or "proposed", are automatically rejected. Any LP submissions associated with these proposals are cancelled, and the funds are released
    And the parties should have the following account balances:
      | party  | asset | market id | margin           | general                   | bond             |
      | lpprov | ETH   | ETH/DEC20 | 2673529501825832 | 9999999991516470498174168 | 1905000000000000 |
      | lpprov | ETH   | ETH/DEC21 | 0                | 9999999991516470498174168 | 0                |

  @SuccessorMarketPanic
  Scenario: 005 Enacting a successor market rejects any other pending successors, same as scenario 2, but enact the older pending market
    ## parent market and 2 successors
    Given the markets:
      | id        | quote name | asset | risk model            | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | decimal places | position decimal places | parent market id | insurance pool fraction | successor auction |
      | ETH/DEC19 | ETH        | ETH   | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | ethDec19Oracle         | 0.1                    | 0                         | 5              | 5                       |                  |                         |                   |
      | ETH/DEC20 | ETH        | ETH   | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | ethDec20Oracle         | 0.1                    | 0                         | 5              | 5                       | ETH/DEC19        | 1                       | 10                |
      | ETH/DEC21 | ETH        | ETH   | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.1                    | 0                         | 5              | 5                       | ETH/DEC19        | 1                       | 10                |
      | ETH/DEC22 | ETH        | ETH   | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.1                    | 0                         | 5              | 5                       |                  |                         |                   |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 3905000000000000  | 0.3 | buy  | BID              | 2          | 1      | submission |
      | lp1 | lpprov | ETH/DEC19 | 3905000000000000  | 0.3 | sell | ASK              | 13         | 1      | submission |
    And the parties place the following orders:
      | party   | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | buy  | 5      | 1001   | 0                | TYPE_LIMIT | TIF_GTC | t1-b-1    |
      | trader1 | ETH/DEC19 | buy  | 5      | 900    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-2    |
      | trader1 | ETH/DEC19 | buy  | 1      | 100    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-3    |
      | trader2 | ETH/DEC19 | sell | 5      | 1200   | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |
      | trader2 | ETH/DEC19 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-2    |
      | trader2 | ETH/DEC19 | sell | 5      | 951    | 0                | TYPE_LIMIT | TIF_GTC | t2-s-3    |
    # Both successor markets should be pending
    Then the market state should be "STATE_PENDING" for the market "ETH/DEC20"
    And the market state should be "STATE_PENDING" for the market "ETH/DEC21"
    When the opening auction period ends for market "ETH/DEC19"
    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake   | open interest |
      | 976        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 134907600000 | 3905000000000000 | 5             |
    And the parties should have the following account balances:
      | party   | asset | market id | margin       | general                   |
      | trader1 | ETH   | ETH/DEC19 | 113402285504 | 9999999999999886597714496 |
    And the parties should have the following margin levels:
      | party   | market id | maintenance | search       | initial      | release      |
      | trader1 | ETH/DEC19 | 94501904587 | 103952095045 | 113402285504 | 132302666421 |
    # LP submissions are being made on both pending markets
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lpprov | ETH/DEC20 | 1905000000000000  | 0.1 | buy  | BID              | 2          | 1      | submission |
      | lp2 | lpprov | ETH/DEC20 | 1905000000000000  | 0.1 | sell | ASK              | 13         | 1      | submission |
      | lp3 | lpprov | ETH/DEC21 | 1905000000000000  | 0.1 | buy  | BID              | 2          | 1      | submission |
      | lp3 | lpprov | ETH/DEC21 | 1905000000000000  | 0.1 | sell | ASK              | 13         | 1      | submission |
    Then the oracles broadcast data signed with "0xCAFECAFE1":
      | name               | value |
      | trading.terminated | true  |
    And the parties should have the following account balances:
      | party  | asset | market id | margin         | general                   | bond             |
      | lpprov | ETH   | ETH/DEC19 | 53551477859983 | 9999999992231448522140017 | 3905000000000000 |
      | lpprov | ETH   | ETH/DEC20 | 0              | 9999999992231448522140017 | 1905000000000000 |
      | lpprov | ETH   | ETH/DEC21 | 0              | 9999999992231448522140017 | 1905000000000000 |

    When the successor market "ETH/DEC20" is enacted
    Then the network moves ahead "1" blocks
    # The bond for market ETH/DEC20 should be released back to the general balance
    And the parties should have the following account balances:
      | party  | asset | market id | margin         | general                   | bond             |
      | lpprov | ETH   | ETH/DEC19 | 53551477859983 | 9999999992231448522140017 | 3905000000000000 |
      | lpprov | ETH   | ETH/DEC20 | 0              | 9999999992231448522140017 | 1905000000000000 |
      | lpprov | ETH   | ETH/DEC21 | 0              | 9999999992231448522140017 | 1905000000000000 |
    Then the market state should be "STATE_PENDING" for the market "ETH/DEC20"
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC19"
    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name             | value |
      | prices.ETH.value | 975   |
    Then the market state should be "STATE_SETTLED" for the market "ETH/DEC19"

    Then the parties place the following orders:
      | party   | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | buy  | 5      | 1001   | 0                | TYPE_LIMIT | TIF_GTC | t1-b-1    |
      | trader1 | ETH/DEC20 | buy  | 5      | 900    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-2    |
      | trader1 | ETH/DEC20 | buy  | 1      | 100    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-3    |
      | trader2 | ETH/DEC20 | sell | 5      | 1200   | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |
      | trader2 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-2    |
      | trader2 | ETH/DEC20 | sell | 5      | 951    | 0                | TYPE_LIMIT | TIF_GTC | t2-s-3    |
    When the opening auction period ends for market "ETH/DEC20"
    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake   | open interest |
      | 976        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 134907600000 | 1905000000000000 | 5             |

    When the network moves ahead "1" blocks
    # The bond for market ETH/DEC20 should be released back to the general balance
    Then the parties should have the following account balances:
      | party  | asset | market id | margin         | general                   | bond             |
      | lpprov | ETH   | ETH/DEC19 | 0              | 9999999998068871464704366 | 0                |
      | lpprov | ETH   | ETH/DEC20 | 26128535295634 | 9999999998068871464704366 | 1905000000000000 |
      | lpprov | ETH   | ETH/DEC21 | 0              | 9999999998068871464704366 | 0                |

    And the last market state should be "STATE_REJECTED" for the market "ETH/DEC21"
    And the parties should have the following account balances:
      | party  | asset | market id | margin         | general                   | bond             |
      | lpprov | ETH   | ETH/DEC19 | 0              | 9999999998068871464704366 | 0                |
      | lpprov | ETH   | ETH/DEC20 | 26128535295634 | 9999999998068871464704366 | 1905000000000000 |
      | lpprov | ETH   | ETH/DEC21 | 0              | 9999999998068871464704366 | 0                |


