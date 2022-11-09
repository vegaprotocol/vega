Feature: Replicate unexpected margin issues.

  Background:
    Given the following assets are registered:
      | id  | decimal places |
      | DAI | 5              |
    And the log normal risk model named "dai-lognormal-risk":
      | risk aversion | tau         | mu | r | sigma |
      | 0.00001       | 0.000114077 | 0  | 0 | 0.41  |
    And the markets:
      | id        | quote name | asset | risk model         | margin calculator         | auction duration | fees         | price monitoring | data source config          | decimal places |
      | DAI/DEC22 | DAI        | DAI   | dai-lognormal-risk | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 5              |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | market.stake.target.scalingFactor       | .1    |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  @NoLP
  Scenario: Attempt to recreate margin drain for LP
    Given the parties deposit on asset's general account the following amount:
      | party           | asset | amount       |
      | party1          | DAI   | 110000000000 |
      | party2          | DAI   | 110000000000 |
      | party3          | DAI   | 110000000000 |

    When the network moves ahead "3" blocks
    Then the market data for the market "DAI/DEC22" should be:
      | trading mode                 | supplied stake |
      | TRADING_MODE_OPENING_AUCTION | 0              |

    ## Place some orders, move time forwards a bit more
    When the parties place the following orders:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     | reference |
      | party2 | DAI/DEC22 | buy  | 1      | 3500000000 | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party2 | DAI/DEC22 | buy  | 1      | 3500000010 | 0                | TYPE_LIMIT | TIF_GTC | party2-2  |
      | party3 | DAI/DEC22 | sell | 1      | 3500000000 | 0                | TYPE_LIMIT | TIF_GTC | party3-1  |
      | party3 | DAI/DEC22 | sell | 1      | 3500000020 | 0                | TYPE_LIMIT | TIF_GTC | party3-2  |
    And the network moves ahead "3" blocks
    Then the market data for the market "DAI/DEC22" should be:
      | trading mode                 | supplied stake | target stake |
      | TRADING_MODE_OPENING_AUCTION | 0              | 6926500      |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee  | side | pegged reference | proportion | offset | reference | lp type    |
      | lp1 | party1 | DAI/DEC22 | 20000000000       | 0.01 | buy  | BID              | 10         | 10     | lp-1      | submission |
      | lp1 | party1 | DAI/DEC22 | 20000000000       | 0.01 | buy  | BID              | 10         | 20     | lp-1      | submission |
      | lp1 | party1 | DAI/DEC22 | 20000000000       | 0.01 | sell | ASK              | 10         | 10     | lp-1      | submission |
      | lp1 | party1 | DAI/DEC22 | 20000000000       | 0.01 | sell | ASK              | 10         | 20     | lp-1      | submission |
      # Balances (DAI): 110000000000

    ## Let's start trading, set a mark price ~3.5bln

    And the network moves ahead "3" blocks
    Then the following trades should be executed:
      | buyer  | price      | size | seller |
      | party2 | 3500000005 | 1    | party3 |
    And the mark price should be "3500000005" for the market "DAI/DEC22"
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search    | initial   | release    |
      | party1 | DAI/DEC22 | 831472306   | 914619536 | 997766767 | 1164061228 |
      | party2 | DAI/DEC22 | 136014608   | 149616068 | 163217529 | 190420451  |
      | party3 | DAI/DEC22 | 138578733   | 152436606 | 166294479 | 194010226  |
    And the order book should have the following volumes for market "DAI/DEC22":
      | side | price      | volume |
      | sell | 3500000020 | 1      |
      | sell | 3500000030 | 6      |
      | sell | 3500000040 | 6      |
      | buy  | 3499999980 | 6      |
      | buy  | 3499999990 | 6      |
      | buy  | 3500000000 | 1      |

    And clear transfer response events
    
    When the parties place the following orders:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     | reference |
      | party3 | DAI/DEC22 | sell | 1      | 3500000020 | 0                | TYPE_LIMIT | TIF_GTC | party3-3  |
      | party2 | DAI/DEC22 | buy  | 1      | 3500000020 | 1                | TYPE_LIMIT | TIF_GTC | party2-3  |
    Then the mark price should be "3500000020" for the market "DAI/DEC22"

    ## Always keep track of what's going on
    And clear transfer response events

    When the parties place the following orders:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     | reference |
      | party2 | DAI/DEC22 | buy  | 1      | 3500000015 | 0                | TYPE_LIMIT | TIF_GTC | p2-1      |
      | party3 | DAI/DEC22 | buy  | 1      | 3500000000 | 0                | TYPE_LIMIT | TIF_GTC | p3-1      |
      | party1 | DAI/DEC22 | buy  | 1      | 3499999960 | 0                | TYPE_LIMIT | TIF_GTC | p1-1      |
      | party2 | DAI/DEC22 | sell | 1      | 3500000010 | 0                | TYPE_LIMIT | TIF_GTC | p2-2      |
      | party3 | DAI/DEC22 | sell | 1      | 3500000040 | 0                | TYPE_LIMIT | TIF_GTC | p3-2      |
      | party1 | DAI/DEC22 | sell | 1      | 3500000015 | 1                | TYPE_LIMIT | TIF_GTC | p1-2      |
    Then the mark price should be "3500000015" for the market "DAI/DEC22"
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search    | initial    | release    |
      | party1 | DAI/DEC22 | 900761672   | 990837839 | 1080914006 | 1261066340 |

    And the liquidity provisions should have the following states:
      | id  | party  | market    | commitment amount | status        |
      | lp1 | party1 | DAI/DEC22 | 20000000000       | STATUS_ACTIVE |

    When the parties place the following orders:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     |
      | party1 | DAI/DEC22 | buy  | 1      | 3500000020 | 1                | TYPE_LIMIT | TIF_GTC |
    Then the mark price should be "3500000020" for the market "DAI/DEC22"
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search    | initial    | release    |
      | party1 | DAI/DEC22 | 884094922   | 972504414 | 1060913906 | 1237732890 |
    And the liquidity provisions should have the following states:
      | id  | party  | market    | commitment amount | status        |
      | lp1 | party1 | DAI/DEC22 | 20000000000       | STATUS_ACTIVE |
