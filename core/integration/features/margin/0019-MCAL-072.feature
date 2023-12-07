Feature: Test closeout under isolated margin mode when party has bond account
  Background:
    # Set liquidity parameters to allow "zero" target-stake which is needed to construct the order-book defined in the ACs
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 0.00             | 24h         | 1e-9           |
    And the simple risk model named "simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 100         | -100          | 0.2                    |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model        | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/FEB23 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.25                   | 0                         | default-futures |

  Scenario: 001 closeout when party has open position and bond account (0019-MCAL-072)
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | party1           | USD   | 173500       |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | party1 | ETH/FEB23 | 1000              | 0.1 | submission |

    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 14900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 6      | 15800  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1           | ETH/FEB23 | sell | 10     | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1           | ETH/FEB23 | sell | 8      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 200100 | 0                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "2" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"
    And the order book should have the following volumes for market "ETH/FEB23":
      | side | price  | volume |
      | buy  | 14900  | 10     |
      | buy  | 15800  | 6      |
      | sell | 15900  | 8      |
      | sell | 200000 | 1      |
      | sell | 200100 | 10     |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/FEB23 | 68370       | 75207  | 82044   | 95718   |
    #margin = min(3*(100000-15900), 15900*(0.25))+0.1*15900=5565

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 82044  | 90456   | 1000 |

    #switch to isolated margin
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor |
      | party1 | ETH/FEB23 | isolated margin | 0.6           |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 55650       | 0      | 66780   | 0       | isolated margin | 0.6           | 76320 |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin |
      | party1 | USD   | ETH/FEB23 | 95400  | 780     | 76320        |

    #trigger more MTM with party has both short position and short orders
    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 17000 | 1                | TYPE_LIMIT | TIF_GTC |
      | sellSideProvider | ETH/FEB23 | sell | 2      | 17000 | 1                | TYPE_LIMIT | TIF_GTC |

    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            |
      | 15900      | TRADING_MODE_CONTINUOUS |
    And the network moves ahead "1" blocks

    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            |
      | 17000      | TRADING_MODE_CONTINUOUS |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin | bond |
      | party1 | USD   | ETH/FEB23 | 151920 | 780     | 0            | 1000 |

    #MTM for party1: 18*(17000-15900)=19800
    And the following transfers should happen:
      | from   | to              | from account            | to account              | market id | amount | asset |
      | party1 | market          | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/FEB23 | 19800  | USD   |
      | market | buySideProvider | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/FEB23 | 19800  | USD   |

    #increase margin factor
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error                                                                      |
      | party1 | ETH/FEB23 | isolated margin | 0.9           | insufficient balance in general account to cover for required order margin |

    #trigger more MTM with party has both short position and short orders
    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     |
      | buySideProvider  | ETH/FEB23 | buy  | 1      | 20000 | 0                | TYPE_LIMIT | TIF_GTC |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 20000 | 1                | TYPE_LIMIT | TIF_GTC |
    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            |
      | 17000      | TRADING_MODE_CONTINUOUS |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin | bond |
      | party1 | USD   | ETH/FEB23 | 151920 | 780     | 0            | 1000 |

    And the network moves ahead "1" blocks

    # what happens here is that the party gets closed out, their general account balance is untouched
    # but their bond balance is taken and then topped up from the general account
    # it's a bit weird but that's how it works
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin | bond |
      | party1 | USD   | ETH/FEB23 | 0      | 0       | 0            | 780  |

    And the following transfers should happen:
      | from   | to     | from account        | to account              | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_MARGIN | ACCOUNT_TYPE_SETTLEMENT | ETH/FEB23 | 54000  | USD   |
      | party1 | party1 | ACCOUNT_TYPE_BOND   | ACCOUNT_TYPE_MARGIN     | ETH/FEB23 | 1000   | USD   |
      | party1 | market | ACCOUNT_TYPE_MARGIN | ACCOUNT_TYPE_INSURANCE  | ETH/FEB23 | 98920  | USD   |


  Scenario: 002 closeout when party has open position, order, and bond account(0019-MCAL-073)
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | party1           | USD   | 216500       |
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | party1 | ETH/FEB23 | 1000              | 0.1 | submission |

    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     | reference  |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 14900  | 0                | TYPE_LIMIT | TIF_GTC |            |
      | buySideProvider  | ETH/FEB23 | buy  | 6      | 15800  | 0                | TYPE_LIMIT | TIF_GTC |            |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 15900  | 0                | TYPE_LIMIT | TIF_GTC |            |
      | party1           | ETH/FEB23 | sell | 10     | 15900  | 0                | TYPE_LIMIT | TIF_GTC |            |
      | party1           | ETH/FEB23 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC | party-sell |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 200000 | 0                | TYPE_LIMIT | TIF_GTC |            |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 200100 | 0                | TYPE_LIMIT | TIF_GTC |            |

    When the network moves ahead "2" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"
    And the order book should have the following volumes for market "ETH/FEB23":
      | side | price  | volume |
      | buy  | 14900  | 10     |
      | buy  | 15800  | 6      |
      | sell | 200000 | 11     |
    #   | sell | 200100 | 10     |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | initial |
      | party1 | ETH/FEB23 | 57240       | 68688   |
    #slippage_per_unit: (19500+200000*9)/10-15900)=166050
    #margin: 10*min(166050, 15900*0.25)+0.1*15900*11=57240

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 68688  | 146812  | 1000 |

    #switch to isolated margin
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor |
      | party1 | ETH/FEB23 | isolated margin | 0.6           |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order  |
      | party1 | ETH/FEB23 | 55650       | 0      | 66780   | 0       | isolated margin | 0.6           | 120000 |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin | bond |
      | party1 | USD   | ETH/FEB23 | 95400  | 100     | 120000       | 1000 |

    #trigger more MTM (18000-15900)*10= 21000 with party has both short position and short orders, when party is distressed, order will remain active
    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     |
      | buySideProvider  | ETH/FEB23 | buy  | 1      | 18000 | 0                | TYPE_LIMIT | TIF_GTC |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 18000 | 1                | TYPE_LIMIT | TIF_GTC |

    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            |
      | 15900      | TRADING_MODE_CONTINUOUS |
    And the network moves ahead "1" blocks

    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            |
      | 18000      | TRADING_MODE_CONTINUOUS |

    #party1's open position is distressed
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin | bond |
      | party1 | USD   | ETH/FEB23 | 74400  | 100     | 120000       | 1000 |

    # #MTM for party1: 10*(25440-18000)=74400
    #trigger more MTM with party has both short position and short orders and empty the margin account
    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     |
      | buySideProvider  | ETH/FEB23 | buy  | 1      | 25440 | 0                | TYPE_LIMIT | TIF_GTC |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 25440 | 1                | TYPE_LIMIT | TIF_GTC |

    And the network moves ahead "1" blocks

    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            |
      | 25440      | TRADING_MODE_CONTINUOUS |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin | bond |
      | party1 | USD   | ETH/FEB23 | 0      | 0       | 120000       | 100  |

    # #MTM for party1: 10*(25540-25440)=1000
    #trigger more MTM with party has both short position and the margin account is alreay empty
    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     |
      | buySideProvider  | ETH/FEB23 | buy  | 1      | 25540 | 0                | TYPE_LIMIT | TIF_GTC |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 25540 | 1                | TYPE_LIMIT | TIF_GTC |

    And the network moves ahead "1" blocks

    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            |
      | 25540      | TRADING_MODE_CONTINUOUS |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin | bond |
      | party1 | USD   | ETH/FEB23 | 0      | 100     | 120000       | 0    |

    And the insurance pool balance should be "0" for the market "ETH/FEB23"


