Feature: Test mark to market settlement with insurance pool

  Background:
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | oracle config          |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: If settlement amount > trader’s margin account balance + trader’s general account balance for the asset, the full balance of the trader’s margin account is transferred to the market’s temporary settlement account, the full balance of the trader’s general account for the assets are transferred to the market’s temporary settlement account, the minimum insurance pool account balance for the market & asset, and the remainder, i.e. the difference between the total amount transferred from the trader’s margin + general accounts and the settlement amount, is transferred from the insurance pool account for the market to the temporary settlement account for the market
    Given the initial insurance pool balance is "10000" for the markets:
    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount |
      | trader1 | ETH   | 5122   |
      | trader2 | ETH   | 10000  |
      | trader3 | ETH   | 10000  |
      | aux     | ETH   | 10000  |
      | aux2    | ETH   | 10000  |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the traders place the following orders:
      | trader | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux    | ETH/DEC19 | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | aux    | ETH/DEC19 | sell | 1      | 6001  | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | aux2   | ETH/DEC19 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | aux    | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC19 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC19 | buy  | 1      | 1000  | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
    Then the traders should have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 | 5122   | 0       |
      | trader2 | ETH   | ETH/DEC19 | 132    | 9868    |

    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader2 | ETH/DEC19 | buy  | 1      | 6000  | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the traders should have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader2 | ETH   | ETH/DEC19 | 265    | 9735    |

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC19 | sell | 1      | 5000  | 1                | TYPE_LIMIT | TIF_GTC | ref-1     |
    Then the traders should have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader1 | ETH   | ETH/DEC19 | 0      | 0       |
      | trader2 | ETH   | ETH/DEC19 | 13586  | 1414    |
      | trader3 | ETH   | ETH/DEC19 | 721    | 9279    |

    And the cumulated balance for all accounts should be worth "55122"
    And the settlement account should have a balance of "0" for the market "ETH/DEC19"
    And the insurance pool balance should be "10122" for the market "ETH/DEC19"
