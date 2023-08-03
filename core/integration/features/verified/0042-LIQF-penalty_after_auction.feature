Feature: Very simple SLA test showing LP meeting their commitment but LP fees being penalised

  Background:

    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 2     |
      | validators.epoch.length                 | 5s    |
    And the markets:
      | id        | quote name | asset | risk model                    | margin calculator         | auction duration | fees         | price monitoring | linear slippage factor | quadratic slippage factor | sla params      | data source config     |
      | ETH/DEC19 | USD        | USD   | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | 1e6                    | 1e6                       | default-futures | default-eth-for-future |

    Given time is updated to "2021-08-26T00:00:00Z"
    And the average block duration is "1"

  Scenario: LP meets obligation but is penalised the first epoch after the opening auction

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | aux1   | USD   | 100000000 |
      | aux2   | USD   | 100000000 |
      | lpprov | USD   | 100000000 |

    # Commit liquidity and place orders which easily meet obligation
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 1000              | 0.001 | submission |
      | lp1 | lpprov | ETH/DEC19 | 1000              | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov | ETH/DEC19 | 10000     | 1                    | buy  | BID              | 10000  | 1      |
      | lpprov | ETH/DEC19 | 10000     | 1                    | sell | ASK              | 10000  | 1      |

    # Exit the opening auction
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC19 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2  | ETH/DEC19 | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | aux1  | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | aux2  | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
    And the opening auction period ends for market "ETH/DEC19"
    Then the mark price should be "1000" for the market "ETH/DEC19"

    # Generate some trades and fees which are immediately trasfered to the LPs liquidity fee acount
    Given the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux1  | ETH/DEC19 | buy  | 100    | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux2  | ETH/DEC19 | sell | 100    | 1000  | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
    When the network moves ahead "1" blocks
    Then the following transfers should happen:
      | from   | to     | from account                | to account                     | market id | amount | asset |
      | aux2   | market | ACCOUNT_TYPE_GENERAL        | ACCOUNT_TYPE_FEES_LIQUIDITY    | ETH/DEC19 | 100    | USD   |
      | market | lpprov | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC19 | 100    | USD   |

    # Move to the next epoch and observe transfers
    When the network moves ahead "5" blocks
    # Note a penalty has been applied to the fees but no penalty has been applied to the bond
    # i.e. feePenalty=1, bondPenalty=0. How can this be....
    Then the following transfers should happen:
      | from   | to     | from account                   | to account             | market id | amount | asset |
      | lpprov | market | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ACCOUNT_TYPE_INSURANCE | ETH/DEC19 | 100    | USD   |
      | lpprov | market | ACCOUNT_TYPE_BOND              | ACCOUNT_TYPE_INSURANCE | ETH/DEC19 | 0      | USD   |