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
      | market.liquidityV2.bondPenaltyParameter       | 0.1   |
      | validators.epoch.length                       | 5s    |
      | market.liquidityV2.stakeToCcyVolume           | 0.2   |
      | market.liquidity.successorLaunchWindowLength | 8s |
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


  @SuccessorMarketExpires
  Scenario: 0081-SUCM-007 Enact a successor market once the parent market is settled, and the succession window has expired
    Given the markets:
      | id        | quote name | asset | risk model            | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | decimal places | position decimal places | parent market id | insurance pool fraction | successor auction | sla params      |
      | ETH/DEC19 | ETH        | ETH   | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | ethDec19Oracle         | 0.1                    | 0                         | 5              | 5                       |                  |                         |                   | default-futures |
      | ETH/DEC20 | ETH        | ETH   | default-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.1                    | 0                         | 5              | 5                       | ETH/DEC19        | 1                       | 10                | default-futures |
    Given the initial insurance pool balance is "1000" for all the markets
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 3905000000000000  | 0.3 | submission |
      | lp1 | lpprov | ETH/DEC19 | 3905000000000000  | 0.3 | submission |
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
    And the liquidity provider fee shares for the market "ETH/DEC19" should be:
      | party  | equity like share | average entry valuation |
      | lpprov | 1                 | 3905000000000000        |
    # provide liquidity for successor market
    When the oracles broadcast data signed with "0xCAFECAFE1":
      | name               | value |
      | trading.terminated | true  |
      | prices.ETH.value | 975 |
    Then the market state should be "STATE_SETTLED" for the market "ETH/DEC19"

    # enactment timestamp
    When the successor market "ETH/DEC20" is enacted
    Then the market data for the market "ETH/DEC20" should be:
      | trading mode                 |
      | TRADING_MODE_OPENING_AUCTION |
    And the last market state should be "STATE_SETTLED" for the market "ETH/DEC19"
    And the last market state should be "STATE_PENDING" for the market "ETH/DEC20"
    And the insurance pool balance should be "1000" for the market "ETH/DEC19"
    And the insurance pool balance should be "1000" for the market "ETH/DEC20"
    And the global insurance pool balance should be "0" for the asset "ETH"

    #now ensure the succession time window has elapsed
    When the network moves ahead "8" blocks
    And the last market state should be "STATE_SETTLED" for the market "ETH/DEC19"
    And the last market state should be "STATE_PENDING" for the market "ETH/DEC20"
    And the insurance pool balance should be "1000" for the market "ETH/DEC19"
    And the insurance pool balance should be "1000" for the market "ETH/DEC20"
    And the global insurance pool balance should be "0" for the asset "ETH"

    When the network moves ahead "5" blocks
    And the last market state should be "STATE_SETTLED" for the market "ETH/DEC19"
    And the last market state should be "STATE_PENDING" for the market "ETH/DEC20"
    And the insurance pool balance should be "0" for the market "ETH/DEC19"
    And the insurance pool balance should be "1500" for the market "ETH/DEC20"
    And the global insurance pool balance should be "500" for the asset "ETH"

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 1905000000000000  | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC20 | 1905000000000000  | 0.1 | submission |

    # successor market should still be in opening auction, no insurance pool balance is transferred
    When the parties place the following orders:
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
      | 976 | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 134907600000 | 1905000000000000 | 5 |
    # Average entry valuation, though the stake is less/different is not carried over
    When the network moves ahead "2" blocks
    And the liquidity provider fee shares for the market "ETH/DEC20" should be:
      | party  | equity like share | average entry valuation |
      | lpprov | 1 | 1905000000000000 |

    And the last market state should be "STATE_SETTLED" for the market "ETH/DEC19"
    And the last market state should be "STATE_ACTIVE" for the market "ETH/DEC20"
    And the insurance pool balance should be "0" for the market "ETH/DEC19"
    And the insurance pool balance should be "1500" for the market "ETH/DEC20"
    And the global insurance pool balance should be "500" for the asset "ETH"

