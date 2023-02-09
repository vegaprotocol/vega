Feature: Test LP orders invalid shapes

  Background:
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator         | auction duration | fees         | price monitoring | data source config     | position decimal places | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | ETH   | default-simple-risk-model-3 | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 2                       | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |

  Scenario: create liquidity provisions with shape > 5 on buy side 0078-NWLI-005, 0078-NWLI-006, 0078-NWLI-008
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount    |
      | party1           | ETH   | 100000000 |
      | sellSideProvider | ETH   | 100000000 |
      | buySideProvider  | ETH   | 100000000 |
      | auxiliary        | ETH   | 100000000 |
      | aux2             | ETH   | 100000000 |
      | lpprov           | ETH   | 100000000 |
      | lpprov2          | ETH   | 100000000 |
      | lpprov3          | ETH   | 100000000 |

    When the parties place the following orders:
      | party     | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | auxiliary | ETH/DEC19 | buy  | 100    | 80    | 0                | TYPE_LIMIT | TIF_GTC | oa-b-1    |
      | auxiliary | ETH/DEC19 | sell | 100    | 120   | 0                | TYPE_LIMIT | TIF_GTC | oa-s-1    |
      | aux2      | ETH/DEC19 | buy  | 100    | 100   | 0                | TYPE_LIMIT | TIF_GTC | oa-b-2    |
      | auxiliary | ETH/DEC19 | sell | 100    | 100   | 0                | TYPE_LIMIT | TIF_GTC | oa-s-2    |

    # default max shape is 5 so expect a failure to submit a shape 6 lp commitment
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    | error                              |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 100    | submission |                                    |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 200    | amendment  |                                    |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 300    | amendment  |                                    |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 400    | amendment  |                                    |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 500    | amendment  |                                    |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 600    | amendment  | SIDE_BUY shape size exceed max (5) |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 100    | submission |                                    |
    And the supplied stake should be "0" for the market "ETH/DEC19"

    # now submit full 5 shape
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    | error |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 100    | submission |       |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 200    | amendment  |       |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 300    | amendment  |       |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 400    | amendment  |       |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 500    | amendment  |       |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 100    | submission |       |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 200    | amendment  |       |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 300    | amendment  |       |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 400    | amendment  |       |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 500    | amendment  |       |
    And the supplied stake should be "900000" for the market "ETH/DEC19"

    # increase the shape limit to 10 and submit a shape 10 commitment
    Given the following network parameters are set:
      | name                                     | value |
      | market.liquidityProvision.shapes.maxSize | 10    |

    And the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    | error |
      | lp2 | lpprov2 | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 100    | submission |       |
      | lp2 | lpprov2 | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 200    | amendment  |       |
      | lp2 | lpprov2 | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 300    | amendment  |       |
      | lp2 | lpprov2 | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 400    | amendment  |       |
      | lp2 | lpprov2 | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 500    | amendment  |       |
      | lp2 | lpprov2 | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 600    | amendment  |       |
      | lp2 | lpprov2 | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 700    | amendment  |       |
      | lp2 | lpprov2 | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 800    | amendment  |       |
      | lp2 | lpprov2 | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 900    | amendment  |       |
      | lp2 | lpprov2 | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 1000   | amendment  |       |
      | lp2 | lpprov2 | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 100    | submission |       |
    And the supplied stake should be "1800000" for the market "ETH/DEC19"

    # we change the max shape to 4, nothing existing is affected but new provisions won't be accepted with more than 4
    Given the following network parameters are set:
      | name                                     | value |
      | market.liquidityProvision.shapes.maxSize | 4     |

    Then the liquidity provisions should have the following states:
      | id  | party   | market    | commitment amount | status         | buy shape | sell shape |
      | lp2 | lpprov2 | ETH/DEC19 | 900000            | STATUS_PENDING | 10        | 1          |

    # now that max shape is 4 expect an error trying to submit shape > 4
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    | error                              |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 100    | submission |                                    |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 200    | amendment  |                                    |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 300    | amendment  |                                    |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 400    | amendment  |                                    |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 500    | amendment  | SIDE_BUY shape size exceed max (4) |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 100    | submission |                                    |
    And the supplied stake should be "1800000" for the market "ETH/DEC19"

  Scenario: create liquidity provisions with shape > 5 on sell side 0078-NWLI-005, 0078-NWLI-007, 0078-NWLI-008
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount    |
      | party1           | ETH   | 100000000 |
      | sellSideProvider | ETH   | 100000000 |
      | buySideProvider  | ETH   | 100000000 |
      | auxiliary        | ETH   | 100000000 |
      | aux2             | ETH   | 100000000 |
      | lpprov           | ETH   | 100000000 |
      | lpprov2          | ETH   | 100000000 |
      | lpprov3          | ETH   | 100000000 |


    When the parties place the following orders:
      | party     | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | auxiliary | ETH/DEC19 | buy  | 100    | 80    | 0                | TYPE_LIMIT | TIF_GTC | oa-b-1    |
      | auxiliary | ETH/DEC19 | sell | 100    | 120   | 0                | TYPE_LIMIT | TIF_GTC | oa-s-1    |
      | aux2      | ETH/DEC19 | buy  | 100    | 100   | 0                | TYPE_LIMIT | TIF_GTC | oa-b-2    |
      | auxiliary | ETH/DEC19 | sell | 100    | 100   | 0                | TYPE_LIMIT | TIF_GTC | oa-s-2    |

    # default max shape is 5 so expect a failure to submit a shape 6 lp commitment
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    | error                               |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 100    | submission |                                     |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 100    | submission |                                     |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 200    | amendment  |                                     |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 300    | amendment  |                                     |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 400    | amendment  |                                     |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 500    | amendment  |                                     |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 600    | amendment  | SIDE_SELL shape size exceed max (5) |
    And the supplied stake should be "0" for the market "ETH/DEC19"
    # now submit full 5 shape
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    | error |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 100    | submission |       |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 200    | amendment  |       |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 300    | amendment  |       |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 400    | amendment  |       |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 500    | amendment  |       |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 100    | submission |       |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 200    | amendment  |       |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 300    | amendment  |       |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 400    | amendment  |       |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 500    | amendment  |       |
    And the supplied stake should be "900000" for the market "ETH/DEC19"

    # increase the shape limit to 10 and submit a shape 10 commitment
    Given the following network parameters are set:
      | name                                     | value |
      | market.liquidityProvision.shapes.maxSize | 10    |

    And the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    | error |
      | lp2 | lpprov2 | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 100    | submission |       |
      | lp2 | lpprov2 | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 100    | submission |       |
      | lp2 | lpprov2 | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 200    | amendment  |       |
      | lp2 | lpprov2 | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 300    | amendment  |       |
      | lp2 | lpprov2 | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 400    | amendment  |       |
      | lp2 | lpprov2 | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 500    | amendment  |       |
      | lp2 | lpprov2 | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 600    | amendment  |       |
      | lp2 | lpprov2 | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 700    | amendment  |       |
      | lp2 | lpprov2 | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 800    | amendment  |       |
      | lp2 | lpprov2 | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 900    | amendment  |       |
      | lp2 | lpprov2 | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 1000   | amendment  |       |
    And the supplied stake should be "1800000" for the market "ETH/DEC19"
    # we change the max shape to 4, nothing existing is affected but new provisions won't be accepted with more than 4
    Given the following network parameters are set:
      | name                                     | value |
      | market.liquidityProvision.shapes.maxSize | 4     |

    Then the liquidity provisions should have the following states:
      | id  | party   | market    | commitment amount | status         | buy shape | sell shape |
      | lp2 | lpprov2 | ETH/DEC19 | 900000            | STATUS_PENDING | 1         | 10         |

    # now that max shape is 4 expect an error trying to submit shape > 4
    And the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    | error                               |
      | lp3 | lpprov3 | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 50         | 100    | submission |                                     |
      | lp3 | lpprov3 | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 100    | submission |                                     |
      | lp3 | lpprov3 | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 200    | amendment  |                                     |
      | lp3 | lpprov3 | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 300    | amendment  |                                     |
      | lp3 | lpprov3 | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 400    | amendment  |                                     |
      | lp3 | lpprov3 | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 50         | 500    | amendment  | SIDE_SELL shape size exceed max (4) |
    And the supplied stake should be "1800000" for the market "ETH/DEC19"

  Scenario: liquidity provision submission with all proportions set to 0 results in an error
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | lpprov | ETH   | 100000000 |
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    | error                               |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | buy  | BID              | 0          | 100    | submission |                                     |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 0          | 100    | amendment  |                                     |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 0          | 200    | amendment  |                                     |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 0          | 300    | amendment  |                                     |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 0          | 400    | amendment  |                                     |
      | lp1 | lpprov | ETH/DEC19 | 900000            | 0.1 | sell | ASK              | 0          | 500    | amendment  | order in shape without a proportion |
    Then the supplied stake should be "0" for the market "ETH/DEC19"