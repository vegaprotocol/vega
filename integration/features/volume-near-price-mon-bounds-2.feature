# Test volume and margin when LP volume is pushed inside price monitoring bounds
# and the price monitoring bounds happen to be best bid/ask
Feature: Test margin for lp near price monitoring boundaries
  Background:
    Given the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 1h    |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 0     |
      | market.liquidity.providers.fee.distributionTimeStep | 10m   |
    
    And the average block duration is "1"

  Scenario: second scenario for volume at near price monitoring bounds with log-normal

    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau     | mu | r   | sigma  |
      | 0.000001      | 0.00273 | 0  | 0   |  1.2   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee | liquidity fee |
      | 0.004     | 0.001              | 0.003           |
    And the price monitoring updated every "1" seconds named "price-monitoring-2":
      | horizon | probability  | auction extension |
      | 43200   | 0.982        | 300                 |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | oracle config          | maturity date        |
      | ETH2/MAR22 | ETH2       | ETH2  | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-2 | default-eth-for-future | 2022-03-31T23:59:59Z |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name              | value  |
      | prices.ETH2.value | 100000   |
    And the traders deposit on asset's general account the following amount:
      | trader  | asset | amount     |
      | lp1     | ETH2   | 10000000000   |
      | trader1 | ETH2   |  1000000000   |
      | trader2 | ETH2   |  1000000000   |
    
    Given the traders submit the following liquidity provision:
      | id          | party   | market id | commitment amount | fee   | order side | order reference | order proportion | order offset |
      | commitment1 | lp1     | ETH2/MAR22 | 3000000         | 0.001 | buy        | BID             | 500              | -100         |
      | commitment1 | lp1     | ETH2/MAR22 | 3000000         | 0.001 | sell       | ASK             | 500              | 100          |
 
    And the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | trader1 | ETH2/MAR22 | buy  | 1      | 89942   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | trader1 | ETH2/MAR22 | buy  | 10     | 100000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | trader2 | ETH2/MAR22 | sell | 1      | 110965  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | trader2 | ETH2/MAR22 | sell | 10     | 100000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |
      
    When the opening auction period ends for market "ETH2/MAR22"
    Then the auction ends with a traded volume of "10" at a price of "100000"

    And the traders should have the following profit and loss:
      | trader           | volume | unrealised pnl | realised pnl |
      | trader1          |  10    | 0              | 0            |
      | trader2          | -10    | 0              | 0            |

    And the market data for the market "ETH2/MAR22" should be:
       | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest   |
       | 100000     | TRADING_MODE_CONTINUOUS | 43200   | 89942     | 110965    | 361194       | 3000000        | 10            |
    
    And the order book should have the following volumes for market "ETH2/MAR22":
      | side | price    | volume |
      | sell | 110965     | 56   |
      | buy  | 89943      | 0    |
      | buy  | 89942      | 68   |


    # # at this point what's left on the book (static) is the buy @ 89942 and sell @ 110965
    # # so the best bid/ask coincides with the price monitoring bounds.
    # # Since the lp1 offset is +/- 100 the lp1 volume "should" go to 89842 and 111065
    # # but because the price monitoring bounds are 89942 and 110965 the volume gets pushed to these
    # # i.e. it's placed at 89942 / 110965. 
    # # As these are the best bid / best ask the probability of trading used is 1/2.  

    And the traders should have the following margin levels:
      | trader    | market id  | maintenance | search   | initial  | release  |
      | lp1       | ETH2/MAR22 | 1986563     | 2185219  | 2383875  | 2781188  |

    And the traders should have the following account balances:
      | trader    | asset  | market id | margin      | general      | bond    |
      | lp1       | ETH2   | ETH2/MAR22 | 2383875    | 9994616125   | 3000000 |

    Then the traders place the following orders:
       | trader  | market id | side | volume | price    | resulting trades | type       | tif     | reference  |
       | trader1 | ETH2/MAR22 | buy  | 1      | 89942   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-3  |
    
    And the order book should have the following volumes for market "ETH2/MAR22":
      | side | price    | volume |
      | sell | 110965     | 56   |
      | buy  | 89943      | 0    |
      | buy  | 89942      | 69   |


    And the traders should have the following margin levels:
      | trader    | market id  | maintenance | search   | initial  | release  |
      | lp1       | ETH2/MAR22 | 1986563     | 2185219  | 2383875  | 2781188  |

    # # now we place an order which makes the best bid 89943. 
    Then the traders place the following orders:
        | trader  | market id  | side  | volume | price   | resulting trades   | type       | tif     | reference  |
        | trader1 | ETH2/MAR22 | buy   | 1      | 89943   | 0                 | TYPE_LIMIT | TIF_GTC | buy-ref-4  |
    
    And the market data for the market "ETH2/MAR22" should be:
       | mark price   | trading mode            | horizon | min bound   | max bound   | target stake   | supplied stake | open interest  |
       | 100000       | TRADING_MODE_CONTINUOUS | 43200   | 89942       | 110965      | 361194         | 3000000        | 10             |
    

    # # the lp1 one volume on this side should go to 89843 but because price monitoring bound is still 89942 it gets pushed to 89942.
    # # but 89942 is no longer the best bid, so the risk model is used to get prob of trading. This now given by the log-normal model
    # # Hence a bit volume is required to meet commitment and thus the margin requirement moves but not much.

    And the order book should have the following volumes for market "ETH2/MAR22":
      | side | price    | volume |
      | sell | 110965   | 56     |
      | buy  | 89943    | 1      |
      | buy  | 89942    | 136    |


    And the traders should have the following margin levels:
      | trader    | market id  | maintenance | search   | initial  | release |
      | lp1       | ETH2/MAR22 | 3592950     | 3952245  | 4311540  | 5030130 |


    
