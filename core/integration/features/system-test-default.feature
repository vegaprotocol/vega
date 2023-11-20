Feature: Basic feature-file matching the system-test setup like for like

  Background:
    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 18             |
      | USD | 0              |
    And the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | lqm-params         | 0.0              | 10s         | 5              |  
    
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model             | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | decimal places | position decimal places | sla params      |
      | ETH/DEC19 | ETH        | ETH   | lqm-params           | default-st-risk-model  | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.1                    | 0                         | 5              | 5                       | default-futures |
      | ETH/DEC20 | ETH        | ETH   | lqm-params           | closeout-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.1                    | 0                         | 5              | 5                       | default-futures |
      | ETH/DEC21 | ETH        | USD   | lqm-params           | closeout-st-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.1                    | 0                         | 0              | 0                       | default-futures |
      | ETH/DEC23 | ETH        | ETH   | lqm-params           | default-st-risk-model  | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.01                   | 0                         | 5              | 5                       | default-st      |
    And the following network parameters are set:
      | name                                          | value |
      | network.markPriceUpdateMaximumFrequency       | 0s    |
      | market.auction.minimumDuration                | 1     |
      | market.fee.factors.infrastructureFee          | 0.001 |
      | market.fee.factors.makerFee                   | 0.004 |
      | market.value.windowLength                     | 60s   |
      | market.liquidity.bondPenaltyParameter         | 0.1   |
      | validators.epoch.length                       | 5s    |
      #| limits.markets.maxPeggedOrders                | 2     |
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
      | limits.markets.maxPeggedOrders                | 2     |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 3905000000000000  | 0.3 | submission |
      | lp1 | lpprov | ETH/DEC19 | 3905000000000000  | 0.3 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | side | volume | peak size | minimum visible size | pegged reference | offset |
      | lpprov | ETH/DEC19 | buy  | 20     | 2         | 1                    | BID              | 1      |
      | lpprov | ETH/DEC19 | sell | 130    | 2         | 1                    | ASK              | 1      |
  
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



  @SystemTestBase
  Scenario: 002 Funding insurance pool balance by closing a trader out - note this scenario is a template. It does not actually close out the trader, it's just the first steps from the system test. With this scenario, we can check margin requirements before and after MTM settlement
    Given the parties deposit on asset's general account the following amount:
      | party           | asset | amount                     |
      | party1          | ETH   | 10000000000000000000000000 |
      | party2          | ETH   | 10000000000000000000000000 |
      | designatedloser | ETH   | 18000000000000000000000    |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 3905000000000000  | 0.3 | submission |
      | lp1 | lpprov | ETH/DEC20 | 3905000000000000  | 0.3 | submission |
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
      | 976        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 1190402800000 | 3905000000000000 | 5             |

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




@SystemTestBase @NoPerp
  Scenario: 003 Funding insurance pool 
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | lpprov           | USD   | 10000000000  |
      | aux1             | USD   | 1000000      |
      | aux2             | USD   | 1000000      |
      | sellSideProvider | USD   | 200000000000 |
      | buySideProvider  | USD   | 200000000000 |
      | designatedloser  | USD   | 33000        |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC21 | 390500            | 0.3 | submission |
      | lp1 | lpprov | ETH/DEC21 | 390500            | 0.3 | submission |
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC21 | buy  | 400    | 1     | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC21 | sell | 300    | 2000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux1  | ETH/DEC21 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC |
    When the opening auction period ends for market "ETH/DEC21"
    Then the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            | auction trigger             | target stake  | supplied stake   | open interest |
      | 150        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 365           | 390500 | 1             |

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

  @AuctionDelay
  Scenario: Opening auction end is delayed for no apparent reason
    Given the parties deposit on asset's general account the following amount:
      | party | asset | amount                      |
      | aux1  | ETH   | 100000000000000000000000000 |
      | aux2  | ETH   | 100000000000000000000000000 |
    # block 425 -> creates proposal, 433 passes vote, 434 proposal is enacted, start opening auction
    # Pretend market ETH/DEC23 passed the vote here
    And the network moves ahead "12" blocks

    # Block 445, LP is submitted (433 + 12)
    When the parties submit the following liquidity provision:
      | id   | party  | market id | commitment amount        | fee | lp type    |
      | lp_1 | lpprov | ETH/DEC23 | 390500000000000000000000 | 0.3 | submission |
    And the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC23" should be:
      | mark price | trading mode                 | auction trigger         |
      | 0          | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING |

    # LP orders are submitted at block 450 -> 445 + 5
    When the network moves ahead "4" blocks
    Then the parties place the following orders:
      | party  | market id | side | volume      | price      | resulting trades | type       | tif     | reference     |
      | lpprov | ETH/DEC23 | buy  | 39050000000 | 100000     | 0                | TYPE_LIMIT | TIF_GTC | lp-buy-order  |
      | lpprov | ETH/DEC23 | sell | 1           | 1000000000 | 0                | TYPE_LIMIT | TIF_GTC | lp-sell-order |
    And the network moves ahead "7" blocks
    Then the market data for the market "ETH/DEC23" should be:
      | mark price | trading mode                 | auction trigger         |
      | 0          | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING |

    # First party submits a buy order at block 457
    When the parties place the following orders:
      | party   | market id | side | volume    | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC23 | buy  | 100100000 | 500000 | 0                | TYPE_LIMIT | TIF_GTC | buy-457   |
    Then the network moves ahead "1" blocks
    And the market data for the market "ETH/DEC23" should be:
      | mark price | trading mode                 | auction trigger         |
      | 0          | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING |

    # More orders are submitted in block 458
    When the parties place the following orders:
      | party   | market id | side | volume      | price  | resulting trades | type       | tif     | reference  |
      | trader2 | ETH/DEC23 | sell | 95100000    | 500000 | 0                | TYPE_LIMIT | TIF_GTC | sell-458-1 |
      | trader1 | ETH/DEC23 | buy  | 90000000    | 500000 | 0                | TYPE_LIMIT | TIF_GTC | buy-458-1  |
      | trader2 | ETH/DEC23 | sell | 120000000   | 500000 | 0                | TYPE_LIMIT | TIF_GTC | sell-458-2 |
      | trader1 | ETH/DEC23 | buy  | 10000000    | 100000 | 0                | TYPE_LIMIT | TIF_GTC | buy-458-2  |
      | trader2 | ETH/DEC23 | sell | 10000000000 | 100000 | 0                | TYPE_LIMIT | TIF_GTC | sell-458-3 |
    # For some reason, even though we have orders that can uncross, we remain in auction
    Then the network moves ahead "1" blocks
    And the market data for the market "ETH/DEC23" should be:
      | mark price | trading mode            | auction trigger             | target stake            | supplied stake           | open interest |
      | 100000     | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 27645000000000000000000 | 390500000000000000000000 | 10000000000   |

