Feature: Verify the order size is correctly cumulated.

  Background:
    Given the log normal risk model named "my-log-normal-risk-model":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.001         | 0.00000190128526884174 | 0  | 0.016 | 2.5   |
    And the markets:
      | id        | quote name | asset | risk model               | margin calculator         | auction duration | fees         | price monitoring | data source config     | position decimal places | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | ETH   | my-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0                       | 1e6                    | 1e6                       |
      | ETH/DEC20 | ETH        | ETH   | my-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 3                       | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
    And the parties deposit on asset's general account the following amount:
      | party      | asset | amount       |
      | party1     | ETH   | 10000000     |
      | party2     | ETH   | 10000000     |
      | party-lp-1 | ETH   | 100000000000 |
      | party3     | ETH   | 1000000000   |
      | lpprov     | ETH   | 1000000000   |

  Scenario: Order from liquidity provision and from normal order submission are correctly cumulated in order book's total size (MID pegs).
    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 1      | 11999980 | 0                | TYPE_LIMIT | TIF_GTC | party1-1  |
      | party2 | ETH/DEC19 | sell | 1      | 12000020 | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party1 | ETH/DEC19 | buy  | 1      | 12000000 | 0                | TYPE_LIMIT | TIF_GFA | party1-2  |
      | party2 | ETH/DEC19 | sell | 1      | 12000000 | 0                | TYPE_LIMIT | TIF_GFA | party2-2  |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "12000000" for the market "ETH/DEC19"

    When the parties submit the following liquidity provision:
      | id  | party      | market id | commitment amount | fee | side | pegged reference | proportion | offset | reference | lp type    |
      | lp2 | party-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | buy  | MID              | 1          | 10     | lp-1-ref  | submission |
      | lp2 | party-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | sell | MID              | 1          | 10     | lp-1-ref  | submission |
    Then the liquidity provisions should have the following states:
      | id  | party      | market    | commitment amount | status        |
      | lp2 | party-lp-1 | ETH/DEC19 | 1000000000        | STATUS_ACTIVE |
    And the market data for the market "ETH/DEC19" should be:
      | mark price | best static bid price | static mid price | best static offer price | trading mode            |
      | 12000000   | 11999980              | 12000000         | 12000020                | TRADING_MODE_CONTINUOUS |
    And the orders should have the following states:
      | party      | market id | side | volume | price    | status        | reference |
      | party-lp-1 | ETH/DEC19 | buy  | 84     | 11999990 | STATUS_ACTIVE | lp2       |
      | party-lp-1 | ETH/DEC19 | sell | 84     | 12000010 | STATUS_ACTIVE | lp2       |
    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price    | volume |
      | sell | 12000120 | 8      |
      | sell | 12000020 | 1      |
      | sell | 12000010 | 84     |
      | buy  | 11999990 | 84     |
      | buy  | 11999980 | 1      |
      | buy  | 11999880 | 8      |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | buy  | 100    | 11999990 | 0                | TYPE_LIMIT | TIF_GTC | party3-1  |
      | party3 | ETH/DEC19 | sell | 100    | 12000010 | 0                | TYPE_LIMIT | TIF_GTC | party3-2  |
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price    | volume |
      | sell | 12000110 | 8      |
      | sell | 12000020 | 1      |
      | sell | 12000010 | 184    |
      | buy  | 11999990 | 184    |
      | buy  | 11999980 | 1      |
      | buy  | 11999890 | 8      |

  Scenario: Order from liquidity provision and from normal order submission are correctly cumulated in order book's total size (BID/ASK pegs).
    # Trigger an auction to set the mark price
    When the parties place the following orders:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 1      | 11999999 | 0                | TYPE_LIMIT | TIF_GTC | party1-1  |
      | party2 | ETH/DEC19 | sell | 1      | 12000001 | 0                | TYPE_LIMIT | TIF_GTC | party2-1  |
      | party1 | ETH/DEC19 | buy  | 1      | 12000000 | 0                | TYPE_LIMIT | TIF_GFA | party1-2  |
      | party2 | ETH/DEC19 | sell | 1      | 12000000 | 0                | TYPE_LIMIT | TIF_GFA | party2-2  |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |
    Then the opening auction period ends for market "ETH/DEC19"
    And the mark price should be "12000000" for the market "ETH/DEC19"

    When the parties submit the following liquidity provision:
      | id  | party      | market id | commitment amount | fee | side | pegged reference | proportion | offset | reference | lp type    |
      | lp2 | party-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | buy  | BID              | 1          | 9      | lp-1-ref  | submission |
      | lp2 | party-lp-1 | ETH/DEC19 | 1000000000        | 0.1 | sell | ASK              | 1          | 9      | lp-1-ref  | submission |
    Then the liquidity provisions should have the following states:
      | id  | party      | market    | commitment amount | status        |
      | lp2 | party-lp-1 | ETH/DEC19 | 1000000000        | STATUS_ACTIVE |
    And the market data for the market "ETH/DEC19" should be:
      | mark price | best static bid price | static mid price | best static offer price | trading mode            |
      | 12000000   | 11999999              | 12000000         | 12000001                | TRADING_MODE_CONTINUOUS |
    And the orders should have the following states:
      | party      | market id | side | volume | price    | status        | reference |
      | party-lp-1 | ETH/DEC19 | buy  | 84     | 11999990 | STATUS_ACTIVE | lp2       |
      | party-lp-1 | ETH/DEC19 | sell | 84     | 12000010 | STATUS_ACTIVE | lp2       |
    And the order book should have the following volumes for market "ETH/DEC19":
      | side | price    | volume |
      | sell | 12000101 | 8      |
      | sell | 12000001 | 1      |
      | sell | 12000010 | 84     |
      | buy  | 11999990 | 84     |
      | buy  | 11999999 | 1      |
      | buy  | 11999899 | 8      |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | buy  | 100    | 11999990 | 0                | TYPE_LIMIT | TIF_GTC | party3-1  |
      | party3 | ETH/DEC19 | sell | 100    | 12000010 | 0                | TYPE_LIMIT | TIF_GTC | party3-2  |
    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price    | volume |
      | sell | 12000101 | 8      |
      | sell | 12000001 | 1      |
      | sell | 12000010 | 184    |
      | buy  | 11999990 | 184    |
      | buy  | 11999999 | 1      |
      | buy  | 11999899 | 8      |

  Scenario: Tripling commitment amount triples deployed volumes
    # Trigger an auction to set the mark price
    Given the following network parameters are set:
      | name                              | value |
      | market.liquidity.stakeToCcyVolume | 1     |

    When the parties place the following orders:
      | party  | market id | side | volume | price    | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | buy  | 100    | 11999999 | 0                | TYPE_LIMIT | TIF_GTC | bestBid   |
      | party2 | ETH/DEC20 | sell | 100    | 12000001 | 0                | TYPE_LIMIT | TIF_GTC | bestAsk   |
      | party1 | ETH/DEC20 | buy  | 100    | 12000000 | 0                | TYPE_LIMIT | TIF_GFA |           |
      | party2 | ETH/DEC20 | sell | 100    | 12000000 | 0                | TYPE_LIMIT | TIF_GFA |           |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |
    Then the opening auction period ends for market "ETH/DEC20"
    And the liquidity provisions should have the following states:
      | id  | party  | market    | commitment amount | status        |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | STATUS_ACTIVE |
    And the orders should have the following states:
      | party  | market id | side | volume | price    | status        | reference |
      | lpprov | ETH/DEC20 | buy  | 7501   | 11999899 | STATUS_ACTIVE | lp1       |
      | lpprov | ETH/DEC20 | sell | 7500   | 12000101 | STATUS_ACTIVE | lp1       |

    Then clear all events
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type   |
      | lp1 | lpprov | ETH/DEC20 | 270000000         | 0.1 | buy  | BID              | 50         | 100    | amendment |
      | lp1 | lpprov | ETH/DEC20 | 270000000         | 0.1 | sell | ASK              | 50         | 100    | amendment |
    Then the liquidity provisions should have the following states:
      | id  | party  | market    | commitment amount | status        |
      | lp1 | lpprov | ETH/DEC20 | 270000000         | STATUS_ACTIVE |


    Then the orders should have the following states:
      | party  | market id | side | volume | price    | status        | reference |
      # 2251/751 = 2.9997333689 (not exact due to rounding up to smallest position precision)
      | lpprov | ETH/DEC20 | buy  | 22501  | 11999899 | STATUS_ACTIVE | lp1       |
      # 2250/750 = 3
      | lpprov | ETH/DEC20 | sell | 22500  | 12000101 | STATUS_ACTIVE | lp1       |
