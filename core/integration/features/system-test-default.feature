Feature: Basic feature-file matching the system-test setup like for like

  Background:
    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 18             |
      | USD | 0              |

    And the markets:
      | id        | quote name | asset | risk model             | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | decimal places | position decimal places |
      | ETH/DEC19 | ETH        | ETH   | default-st-risk-model  | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.1                    | 0                         | 5              | 5                       |
      | ETH/DEC20 | ETH        | ETH   | closeout-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.1                    | 0                         | 5              | 5                       |
      | ETH/DEC21 | ETH        | USD   | closeout-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.1                    | 0                         | 0              | 0                       |
    And the following network parameters are set:
      | name                                          | value |
      | network.markPriceUpdateMaximumFrequency       | 0s    |
      | market.liquidity.targetstake.triggering.ratio | 0     |
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


  @SystemTestBase
  Scenario: 001 Create a new market and leave opening auction in the same way the system tests do
    # the amount ought to be 390,500.000,000,000,000,000,000
    Given the following network parameters are set:
      | name                                          | value |
      | market.liquidity.targetstake.triggering.ratio | 0.01  |
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
      | 976        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 269815200000 | 3905000000000000 | 5             |
    And the parties should have the following account balances:
      | party   | asset | market id | margin       | general                   |
      | trader1 | ETH   | ETH/DEC19 | 113402285504 | 9999999999999886597714496 |
    And the parties should have the following margin levels:
      | party   | market id | maintenance | search       | initial      | release      |
      | trader1 | ETH/DEC19 | 94501904587 | 103952095045 | 113402285504 | 132302666421 |
    And debug orders
    And debug detailed orderbook volumes for market "ETH/DEC19"

  @SystemTestBase
  Scenario: 002 Funding insurance pool balance by closing a trader out - note this scenario is a template. It does not actually close out the trader, it's just the first steps from the system test. With this scenario, we can check margin requirements before and after MTM settlement
    Given the parties deposit on asset's general account the following amount:
      | party           | asset | amount                     |
      | party1          | ETH   | 10000000000000000000000000 |
      | party2          | ETH   | 10000000000000000000000000 |
      | designatedloser | ETH   | 18000000000000000000000    |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 3905000000000000  | 0.3 | buy  | BID              | 2          | 1      | submission |
      | lp1 | lpprov | ETH/DEC20 | 3905000000000000  | 0.3 | sell | ASK              | 13         | 1      | submission |
    And the parties place the following orders:
      | party   | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | buy  | 5      | 1001   | 0                | TYPE_LIMIT | TIF_GTC | t1-b-1    |
      | trader1 | ETH/DEC20 | buy  | 5      | 900    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-2    |
      | trader1 | ETH/DEC20 | buy  | 1      | 100    | 0                | TYPE_LIMIT | TIF_GTC | t1-b-3    |
      | trader2 | ETH/DEC20 | sell | 5      | 1200   | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |
      | trader2 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-2    |
      | trader2 | ETH/DEC20 | sell | 5      | 951    | 0                | TYPE_LIMIT | TIF_GTC | t2-s-3    |
    When the opening auction period ends for market "ETH/DEC20"
    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | auction trigger             | target stake  | supplied stake   | open interest |
      | 976        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 2380805600000 | 3905000000000000 | 5             |

    # Now place orders to cause designatedloser party to be distressed
    When the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1          | ETH/DEC20 | sell | 1      | 960   | 0                | TYPE_LIMIT | TIF_GTC | p1-s-1    |
      | party2          | ETH/DEC20 | buy  | 1      | 950   | 0                | TYPE_LIMIT | TIF_GTC | p1-b-1    |
      | designatedloser | ETH/DEC20 | buy  | 450    | 960   | 1                | TYPE_LIMIT | TIF_GTC | dl-b-1    |
    Then the parties should have the following account balances:
      | party           | asset | market id | margin         | general                 |
      | designatedloser | ETH   | ETH/DEC20 | 17753938373119 | 17999999982216781626881 |
    And the parties should have the following margin levels:
      | party           | market id | maintenance    | search         | initial        | release        |
      | designatedloser | ETH/DEC20 | 14794948644266 | 16274443508692 | 17753938373119 | 20712928101972 |
    When the network moves ahead "1" blocks
    Then the parties should have the following account balances:
      | party           | asset | market id | margin         | general                 |
      | designatedloser | ETH   | ETH/DEC20 | 17753938373119 | 17999999982216781626881 |
    And the parties should have the following margin levels:
      | party           | market id | maintenance    | search         | initial        | release        |
      | designatedloser | ETH/DEC20 | 14552408502557 | 16007649352812 | 17462890203068 | 20373371903579 |
    Then debug detailed orderbook volumes for market "ETH/DEC20"
    And debug orders
    And debug detailed orderbook volumes for market "ETH/DEC20"

@SystemTestBase
  Scenario: 003 Funding insurance pool 
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | lpprov           | USD   | 10000000000  |
      | aux1             | USD   | 1000000      |
      | aux2             | USD   | 1000000      |
      | sellSideProvider | USD   | 200000000000 |
      | buySideProvider  | USD   | 200000000000 |
      | designatedloser  | USD   | 33000         |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC21 | 390500  | 0.3 | buy  | BID              | 2          | 100      | submission |
      | lp1 | lpprov | ETH/DEC21 | 390500  | 0.3 | sell | ASK              | 13         | 100      | submission |
        Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1   | ETH/DEC21 | buy  | 400    | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1   | ETH/DEC21 | sell | 300    | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1   | ETH/DEC21 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2   | ETH/DEC21 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
    When the opening auction period ends for market "ETH/DEC21"
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | auction trigger             | target stake  | supplied stake   | open interest |
      | 150        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 731 | 390500 | 1             |

    # Now place orders to cause designatedloser party to be distressed
    When the parties place the following orders with ticks:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference       |
      | sellSideProvider | ETH/DEC21 | sell | 290    | 150   | 0                | TYPE_LIMIT | TIF_GTC | sell-provider-1 |
      | buySideProvider  | ETH/DEC21 | buy  | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-1  |

    When the parties place the following orders with ticks:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | designatedloser  | ETH/DEC21 | buy  | 290    | 150   | 1                | TYPE_LIMIT | TIF_GTC | ref-loser-1     |

    Then the parties should have the following margin levels:
      | party            | market id | maintenance | search | initial | release |
      | designatedloser | ETH/DEC21 | 19004       | 20904  | 22804   | 26605   |
       Then the parties should have the following account balances:
      | party            | asset | market id | margin | general |
      | designatedloser | USD   | ETH/DEC21 | 19732  | 0       |

    Then the parties cancel the following orders:
      | party           | reference      |
      | buySideProvider | buy-provider-1 |
    When the parties place the following orders with ticks:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | buySideProvider | ETH/DEC21 | buy  | 290    | 120   | 0                | TYPE_LIMIT | TIF_GTC | buy-provider-2 |

    When the parties place the following orders with ticks:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/DEC21 | sell | 1      | 140   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | buySideProvider  | ETH/DEC21 | buy  | 1      | 140   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
    And the network moves ahead "1" blocks
    And the insurance pool balance should be "417" for the market "ETH/DEC21"


   
