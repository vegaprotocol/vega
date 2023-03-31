Feature: Test mark to market settlement with insurance pool

  Background:
    Given the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e0                    | 0                         |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: If settlement amount > party’s margin account balance + party’s general account balance for the asset, the full balance of the party’s margin account is transferred to the market’s temporary settlement account, the full balance of the party’s general account for the assets are transferred to the market’s temporary settlement account, the minimum insurance pool account balance for the market & asset, and the remainder, i.e. the difference between the total amount transferred from the trader’s margin + general accounts and the settlement amount, is transferred from the insurance pool account for the market to the temporary settlement account for the market (0003-MTMK-003)
    Given the initial insurance pool balance is "10000" for all the markets
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 5122   |
      | party2 | ETH   | 10000  |
      | party3 | ETH   | 10000  |
      | aux    | ETH   | 10000  |
      | aux2   | ETH   | 10000  |
      | lpprov | ETH   | 100000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000             | 0.1 | sell | ASK              | 50         | 10     | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux   | ETH/DEC19 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux   | ETH/DEC19 | sell | 1      | 6001  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | aux2  | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | aux   | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1100         | 0              | 1             |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | lpprov | ETH   | ETH/DEC19 | 0      | 100000  |
     
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC19 | 1320   | 3802    |
      | party2 | ETH   | ETH/DEC19 | 132    | 9868    |

    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC19 | buy  | 1      | 6000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party2 | ETH   | ETH/DEC19 | 265    | 9735    |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC19 | sell | 1      | 5000  | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | ETH   | ETH/DEC19 | 0      | 0       |
      | party2 | ETH   | ETH/DEC19 | 15000  | 0       |
      | party3 | ETH   | ETH/DEC19 | 7920   | 2080    |

    And the cumulated balance for all accounts should be worth "155122"
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
    And the insurance pool balance should be "10121" for the market "ETH/DEC19"
