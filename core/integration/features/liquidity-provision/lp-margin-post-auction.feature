Feature: Assure LP margin is correct

  Background:

    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau  | mu | r | sigma |
      | 0.000001      | 0.01 | 0  | 0 | 1.0   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 3600    | 0.99        | 300               |
    And the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1.5   |
      | market.liquidity.bondPenaltyParameter         | 0.2   |
      | market.liquidity.targetstake.triggering.ratio | 0.24  |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e6                    | 1e6                       |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party0 | USD   | 500000000 |
      | party1 | USD   | 100000000 |
      | party2 | USD   | 100000000 |
      | party3 | USD   | 100000000 |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: Assure LP margin is released when opening auction concludes with a price lower than indicative uncrossing price at the time of LP submission

    Given the average block duration is "1"
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/MAR22 | 50000             | 0.001 | sell | ASK              | 500        | 17     | submission |
      | lp1 | party0 | ETH/MAR22 | 50000             | 0.001 | buy  | BID              | 500        | 17     | amendment  |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/MAR22 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party1 | ETH/MAR22 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-3  |
      | party1 | ETH/MAR22 | buy  | 100    | 50000 | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-4  |
      | party2 | ETH/MAR22 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |
      | party2 | ETH/MAR22 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party3 | ETH/MAR22 | sell | 100    | 50000 | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-4 |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode                 | indicative price | indicative volume |
      | 0          | TRADING_MODE_OPENING_AUCTION | 50000            | 100               |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | party0 | ETH/MAR22 | 55000             | 0.001 | sell | ASK              | 500        | 17     | amendment |
      | lp1 | party0 | ETH/MAR22 | 55000             | 0.001 | buy  | BID              | 500        | 17     | amendment |
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party0 | ETH/MAR22 | 63256       | 69581  | 75907   | 88558   |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general   | bond  |
      | party0 | USD   | ETH/MAR22 | 75907  | 499869093 | 55000 |

    When the parties cancel the following orders:
      | party  | reference  |
      | party3 | sell-ref-4 |
      | party1 | buy-ref-4  |
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode                 | indicative price | indicative volume |
      | 0          | TRADING_MODE_OPENING_AUCTION | 1000             | 10                |

    When the opening auction period ends for market "ETH/MAR22"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 973       | 1027      | 9484         | 55000          | 10            |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party0 | ETH/MAR22 | 34147       | 37561  | 40976   | 47805   |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general   | bond  |
      | party0 | USD   | ETH/MAR22 | 40976  | 499904024 | 55000 |
