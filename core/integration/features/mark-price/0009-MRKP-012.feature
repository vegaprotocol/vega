Feature: Test setting of mark price
  Background:
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 4s    |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 0.00             | 24h         | 1e-9           |
    And the simple risk model named "simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 100         | -100          | 0.2                    |

    # this is just an example of setting up oracles
    And the composite price oracles from "0xCAFECAFE1":
      | name    | price property   | price type   | price decimals |
      | oracle1 | price1.USD.value | TYPE_INTEGER | 0              |
      | oracle2 | price2.USD.value | TYPE_INTEGER | 0              |
      | oracle3 | price3.USD.value | TYPE_INTEGER | 0              |
      | oracle4 | price4.USD.value | TYPE_INTEGER | 0              |
      | oracle5 | price4.USD.value | TYPE_INTEGER | 0              |

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model        | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | price type | decay weight | decay power | cash amount | source weights  | source staleness tolerance                                | oracle1 | oracle2 | oracle3 | oracle4 | oracle5 |
      | ETH/FEB23 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.25                   | 0                         | default-futures | weight     | 1            | 1           | 0           | 1,0,0,0,0,0,0,0 | 3h0m0s,2s,24h0m0s,24h0m0s,24h0m0s,24h0m0s,24h0m0s,1h25m0s | oracle1 | oracle2 | oracle3 | oracle4 | oracle5 |

  @SLABug
  Scenario: 001 check mark price using weight average
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | party            | USD   | 48050        |
    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 14900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 1      | 15000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 3      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party            | ETH/FEB23 | sell | 3      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 2      | 15920  | 0                | TYPE_LIMIT | TIF_GTC | sell-2    |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 15940  | 0                | TYPE_LIMIT | TIF_GTC | sell-3    |
      | sellSideProvider | ETH/FEB23 | sell | 3      | 15960  | 0                | TYPE_LIMIT | TIF_GTC | sell-4    |
      | sellSideProvider | ETH/FEB23 | sell | 5      | 15990  | 0                | TYPE_LIMIT | TIF_GTC | sell-5    |
      | sellSideProvider | ETH/FEB23 | sell | 2      | 16000  | 0                | TYPE_LIMIT | TIF_GTC | sell-7    |
      | sellSideProvider | ETH/FEB23 | sell | 4      | 16020  | 0                | TYPE_LIMIT | TIF_GTC | sell-8    |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC |           |

    # AC 0009-MRKP-012
    When the network moves ahead "2" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"

    And the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | buySideProvider | ETH/FEB23 | buy  | 2      | 15920 | 1                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "1" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"

    And the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | buySideProvider | ETH/FEB23 | buy  | 1      | 15940 | 1                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "1" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"

    When the network moves ahead "2" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"

    # example of pushing oracle prices 
    Then the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value | time offset |
      | price1.USD.value | 16000 | -1s         |
      | price2.USD.value | 15900 | -1s         |
      | price3.USD.value | 15940 | -1s         |
      | price4.USD.value | 16001 | -1s         |
      | price5.USD.value | 16002 | -1s         |

    When the network moves ahead "1" blocks
    Then the mark price should be "15940" for the market "ETH/FEB23"

    When the markets are updated:
      | id        | decay weight | decay power | linear slippage factor | quadratic slippage factor |
      | ETH/FEB23 | 0            | 1           | 0.25                   | 0                         |

    Then the mark price should be "15940" for the market "ETH/FEB23"

    And the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | buySideProvider | ETH/FEB23 | buy  | 3      | 15960 | 1                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "1" blocks
    Then the mark price should be "15940" for the market "ETH/FEB23"

    And the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | buySideProvider | ETH/FEB23 | buy  | 5      | 15990 | 1                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "2" blocks
    Then the mark price should be "15940" for the market "ETH/FEB23"

    When the network moves ahead "1" blocks
    Then the mark price should be "15990" for the market "ETH/FEB23"


