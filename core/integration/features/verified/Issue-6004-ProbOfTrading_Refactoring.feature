Feature: test probability of trading used in LP vol when best bid/ask is changing 

  Background:

    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 10000   | 0.99        | 300               |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config          |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | default-none | default-eth-for-future |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party0 | USD   | 5000000000   |
      | party1 | USD   | 10000000000 |
      | party2 | USD   | 10000000000 |
      | party3 | USD   | 10000000000 |

    Given the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.bondPenaltyParameter         | 0.2   |
      | market.liquidity.targetstake.triggering.ratio | 0.1   |
      | market.liquidity.stakeToCcySiskas             | 1.0   |
      | network.markPriceUpdateMaximumFrequency       | 0s    |

   And the average block duration is "1"

   Scenario: 001, LP price at 0, check what's happening with LP volume; 0038-OLIQ-002

   Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/MAR22 | 50000             | 0.001 | sell | ASK              | 500        | 1      | submission |
      | lp1 | party0 | ETH/MAR22 | 50000             | 0.001 | buy  | BID              | 500        | 1      | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/MAR22 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/MAR22 | buy  | 1      | 890   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party2 | ETH/MAR22 | sell | 3      | 900   | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/MAR22 | sell | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

    When the opening auction period ends for market "ETH/MAR22"
    Then the auction ends with a traded volume of "1" at a price of "900"

    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | sell | 990   | 1      |
      | sell | 901   | 112    |
      | sell | 900   | 2      |
      | buy  | 890   | 1      |
      | buy  | 889   | 113    |
      | buy  | 1     | 1      |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | party3 | ETH/MAR22 | sell | 114    | 889   | 2                | TYPE_LIMIT | TIF_GTC | party3-buy-1 |

    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | sell | 990   | 1      |
      | sell | 901   | 112    |
      | sell | 900   | 2      |
      | buy  | 890   | 0      |
      | buy  | 889   | 0      |
      | buy  | 1     | 100001 | #50000/(0.5*1)=100000
      | buy  | 0     | 0      |

    Then the market data for the market "ETH/MAR22" should be:
      | trading mode            | supplied stake | target stake |
      | TRADING_MODE_CONTINUOUS | 50000          | 363639       |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | party1 | ETH/MAR22 | buy  | 3      | 900   | 1                | TYPE_LIMIT | TIF_GTC | party1-buy-1 |

    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | sell | 991   | 102    |
      | sell | 990   | 1      |
      | buy  | 900   | 1      |
      | buy  | 899   | 112    | 
      | buy  | 1     | 1      |
  
    Then the market data for the market "ETH/MAR22" should be:
      | trading mode            | supplied stake | target stake |
      | TRADING_MODE_CONTINUOUS | 50000          | 374541       |

  Scenario: 002, market starts with a low best bid price 1 (ProbTrading is large), and then best bid goes to 899; test of the new ProbTrading is reasonable, and LP is not distressed; 0038-OLIQ-002 

    Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/MAR22 | 50000             | 0.001 | sell | ASK              | 500        | 1      | submission |
      | lp1 | party0 | ETH/MAR22 | 50000             | 0.001 | buy  | BID              | 500        | 1      | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/MAR22 | buy  | 1      | 1     | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party2 | ETH/MAR22 | sell | 3      | 900   | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/MAR22 | sell | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

    When the opening auction period ends for market "ETH/MAR22"
    Then the auction ends with a traded volume of "1" at a price of "900"

    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | sell | 990   | 1      |
      | sell | 901   | 112    |
      | sell | 900   | 2      |
      | buy  | 1     | 100001 |

    And the insurance pool balance should be "0" for the market "ETH/MAR22"

    Then the parties should have the following account balances:
      | party  | asset | market id | margin    | general    | bond  |
      | party0 | USD   | ETH/MAR22 | 86478646  | 4913471354 | 50000 |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | party3 | ETH/MAR22 | buy  | 20     | 899   | 0                | TYPE_LIMIT | TIF_GTC | party3-buy-1 |

    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | sell | 990   | 1      |
      | sell | 901   | 112    |
      | sell | 900   | 2      |
      | buy  | 899   | 20     |
      | buy  | 898   | 112    |
      | buy  | 1     | 1      | 

    Then the market data for the market "ETH/MAR22" should be:
      | trading mode            | supplied stake | target stake |
      | TRADING_MODE_CONTINUOUS | 50000          | 3201         |

    And the insurance pool balance should be "0" for the market "ETH/MAR22"

    Then the parties should have the following account balances:
      | party  | asset | market id | margin    | general    | bond |
      | party0 | USD   | ETH/MAR22 | 430243    | 4999519757 | 50000|

