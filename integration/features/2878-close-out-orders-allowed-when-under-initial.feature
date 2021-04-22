Feature: Trader below initial margin, but above maintenance can submit an order to close their own position

  Background:
    Given time is updated to "2020-10-16T00:00:00Z"
    And the markets:
      | id      | quote name | asset | maturity date        | auction duration | risk model                  | margin calculator         | fees         | price monitoring | oracle config          |
      | ETH/DEC20 | ETH        | ETH   | 2020-12-31T23:59:59Z | 1                | default-simple-risk-model-3 | default-margin-calculator | default-none | default-none     | default-eth-for-future |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Trader under initial margin closes out their own position
    Given the traders deposit on asset's general account the following amount:
      | trader    | asset | amount         |
      | trader1   | ETH   | 10000000000000 |
      | trader2   | ETH   | 10000000000000 |
      | trader3   | ETH   | 1220           |
      | trader4   | ETH   | 10000000000000 |
      | trader5   | ETH   | 10000000000000 |
      | auxiliary | ETH   | 100000000000   |
      | aux2      | ETH   | 100000000000   |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader     | market id | side | volume | price    | resulting trades | type        | tif     | 
      | auxiliary  | ETH/DEC20 | buy  | 1      | 1        | 0                | TYPE_LIMIT  | TIF_GTC | 
      | auxiliary  | ETH/DEC20 | sell | 1      | 200      | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux2       | ETH/DEC20 | buy  | 1      | 100      | 0                | TYPE_LIMIT  | TIF_GTC | 
      | auxiliary  | ETH/DEC20 | sell | 1      | 100      | 0                | TYPE_LIMIT  | TIF_GTC | 
    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "100" for the market "ETH/DEC20"

    # T0 + 1min - this causes the price for comparison of the bounds to be 567
    Then time is updated to "2020-10-16T00:01:00Z"

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif      | reference |
      | trader3 | ETH/DEC20 | sell | 10     | 100   | 0                | TYPE_LIMIT | TIF_GTC  | ref-1     |

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader5 | ETH/DEC20 | buy  | 10     | 100   | 1                | TYPE_LIMIT | TIF_FOK | ref-1     |
      | trader4 | ETH/DEC20 | buy  | 10     | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |
      | trader4 | ETH/DEC20 | sell | 10     | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |

    Then the traders should have the following account balances:
      | trader  | asset | market id | margin | general       |
      | trader4 | ETH   | ETH/DEC20 | 133    | 9999999999867 |
      | trader5 | ETH   | ETH/DEC20 | 1320   | 9999999998680 |
      | trader3 | ETH   | ETH/DEC20 | 1220   | 0             |
    And the traders should have the following margin levels:
      | trader  | market id | maintenance | search | initial | release |
      | trader3 | ETH/DEC20 | 1100        | 1210   | 1320    | 1540    |

    ## Now trader 3, though below initial margin places a buy order to close their position out
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader5 | ETH/DEC20 | sell | 20     | 115   | 0                | TYPE_LIMIT | TIF_GTC | ref-6     |
      | trader4 | ETH/DEC20 | buy  | 15     | 115   | 1                | TYPE_LIMIT | TIF_GTC | ref-7     |
      | trader3 | ETH/DEC20 | buy  | 10     | 115   | 1                | TYPE_LIMIT | TIF_GTC | ref-8     |
    ## The trades have happened, trader 3 bought 5 -> margin requirements go down
    Then the mark price should be "115" for the market "ETH/DEC20"
    And the traders should have the following margin levels:
      | trader  | market id | maintenance | search | initial | release |
      | trader3 | ETH/DEC20 | 83          | 91     | 99      | 116     |
    ## Balances of the trader accounts reflect the change, total adds up to 1070 -> trader3 lost money
    ## as expected, but was able to close their position
    Then the traders should have the following account balances:
      | trader  | asset | market id | margin | general |
      | trader3 | ETH   | ETH/DEC20 | 99     | 971     |
