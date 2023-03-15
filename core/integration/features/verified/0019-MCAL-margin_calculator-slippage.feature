Feature: Test capped maximum slippage values are calculated correctly in range of order-book scenarios and slippage factor values

    Background:

        # Set liquidity parameters to allow "zero" target-stake which is needed to construct the order-book defined in the ACs
        Given the following network parameters are set:
            | name                                          | value |
            | market.stake.target.scalingFactor             | 1e-9  |
            | market.liquidity.targetstake.triggering.ratio | 0     |
            | network.markPriceUpdateMaximumFrequency       | 0s    |

       
        And the simple risk model named "simple-risk-model":
            | long | short | max move up | min move down | probability of trading |
            | 0.1  | 0.1   | 100         | -100          | 0.2                    |

        And the markets:
            | id        | quote name | asset | risk model        | margin calculator         | auction duration | fees         | price monitoring | data source config           | linear slippage factor | quadratic slippage factor |
            | ETH/FEB23 | ETH        | USD   | simple-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future       | 0.25                   | 0.25                      |
            | ETH/MAR23 | ETH        | USD   | simple-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future       | 100                    | 100                       |


    Scenario: Check slippage-factors yield the correct maximum slippage for a specific market state (0019-MCAL-011)(0019-MCAL-012)

        # Create position, order book, and mark price conditions matching the spec
        Given the parties deposit on asset's general account the following amount:
            | party            | asset | amount       |
            | buySideProvider  | USD   | 100000000000 |
            | sellSideProvider | USD   | 100000000000 |
            | party            | USD   | 100000000000 |
        And the parties place the following orders:
            | party           | market id | side | volume | price  | resulting trades | type       | tif     |
            | buySideProvider | ETH/FEB23 | buy  | 10     | 14900  | 0                | TYPE_LIMIT | TIF_GTC |
            | buySideProvider | ETH/FEB23 | buy  | 1      | 15000  | 0                | TYPE_LIMIT | TIF_GTC |
            | buySideProvider | ETH/FEB23 | buy  | 1      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |
            | party           | ETH/FEB23 | sell | 1      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |
            | sellSideProvider| ETH/FEB23 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC |
            | sellSideProvider| ETH/FEB23 | sell | 10     | 100100 | 0                | TYPE_LIMIT | TIF_GTC |
        And the parties place the following orders:
            | party           | market id | side | volume | price  | resulting trades | type       | tif     |
            | buySideProvider | ETH/MAR23 | buy  | 10     | 14900  | 0                | TYPE_LIMIT | TIF_GTC |
            | buySideProvider | ETH/MAR23 | buy  | 1      | 15000  | 0                | TYPE_LIMIT | TIF_GTC |
            | buySideProvider | ETH/MAR23 | buy  | 1      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |
            | party           | ETH/MAR23 | sell | 1      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |
            | sellSideProvider| ETH/MAR23 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC |
            | sellSideProvider| ETH/MAR23 | sell | 10     | 100100 | 0                | TYPE_LIMIT | TIF_GTC |


        # Checks for 0019-MCAL-012
        When the opening auction period ends for market "ETH/FEB23"
        # Check mark-price matches the specification
        Then the mark price should be "15900" for the market "ETH/FEB23"
        # Check order book matches the specification
        And the order book should have the following volumes for market "ETH/FEB23":
            | side | price  | volume |
            | buy  | 14900  | 10     |
            | buy  | 15000  | 1      |
            | sell | 100000 | 1      |
            | sell | 100100 | 10     |
        # Check party margin levels match the specification
        And the parties should have the following margin levels:
            | party | market id | maintenance | search | initial | release |
            | party | ETH/FEB23 | 9540        | 10494  | 11448   | 13356   |


        # Checks for 0019-MCAL-013
        When the opening auction period ends for market "ETH/MAR23"
        # Check mark-price matches the specification
        Then the mark price should be "15900" for the market "ETH/MAR23"
        # Check order book matches the specification
        And the order book should have the following volumes for market "ETH/MAR23":
            | side | price  | volume |
            | buy  | 14900  | 10     |
            | buy  | 15000  | 1      |
            | sell | 100000 | 1      |
            | sell | 100100 | 10     |
        # Check party margin levels match the specification
        And the parties should have the following margin levels:
            | party | market id | maintenance | search | initial | release |
            | party | ETH/MAR23 | 85690       | 94259  | 102828  | 119966  |

    Scenario: Check margin is calculated correctly using capped slippage depending on the volume of orders on the book (0019-MCAL-014)(0019-MCAL-015)(0019-MCAL-016)(0019-MCAL-017)(0019-MCAL-018)

        Given the parties deposit on asset's general account the following amount:
            | party            | asset | amount       |
            | buySideProvider  | USD   | 100000000000 |
            | sellSideProvider | USD   | 100000000000 |
            | longTrader       | USD   | 100000000000 |
            | shortTrader      | USD   | 100000000000 |
            | aux1             | USD   | 100000000000 |
            | aux2             | USD   | 100000000000 |
        And the parties place the following orders:
            | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
            | buySideProvider  | ETH/FEB23 | buy  | 10     | 496   | 0                | TYPE_LIMIT | TIF_GTC | bsp-1     |
            | buySideProvider  | ETH/FEB23 | buy  | 5      | 498   | 0                | TYPE_LIMIT | TIF_GTC | bsp-2     |
            | longTrader       | ETH/FEB23 | buy  | 10     | 500   | 0                | TYPE_LIMIT | TIF_GTC | lt-1      |
            | shortTrader      | ETH/FEB23 | sell | 10     | 500   | 0                | TYPE_LIMIT | TIF_GTC | st-1      |
            | sellSideProvider | ETH/FEB23 | sell | 5      | 502   | 0                | TYPE_LIMIT | TIF_GTC | ssp-2     |
            | sellSideProvider | ETH/FEB23 | sell | 10     | 504   | 0                | TYPE_LIMIT | TIF_GTC | ssp-1     |


        # Case for 0019-MCAL-017 and 0019-MCAL-018:
        #   - For the party 'longTrader'; riskiest long > 0 && abs(riskiest long) < sum of volume of order book bids
        #   - For the party 'shortTrader'; riskiest short < 0 && abs(riskiest long) < sum of volume of order book bids
        When the opening auction period ends for market "ETH/FEB23"
        Then the order book should have the following volumes for market "ETH/FEB23":
            | side | price | volume |
            | buy  | 496   | 10     |
            | buy  | 498   | 5      |
            | sell | 502   | 5      |
            | sell | 504   | 10     |
        Then the mark price should be "500" for the market "ETH/FEB23"
        And the parties should have the following margin levels:
            | party       | market id | maintenance | search | initial | release |
            | longTrader  | ETH/FEB23 | 530         | 583    | 636     | 742     |
            | shortTrader | ETH/FEB23 | 530         | 583    | 636     | 742     |


        # Case for 0019-MCAL-015:
        #   - For the party 'longTrader'; riskiest long > 0 && abs(riskiest long) > sum of volume of order book bids
        When the parties cancel the following orders:
            | party            | reference |
            | buySideProvider  | bsp-1     |
        Then the order book should have the following volumes for market "ETH/FEB23":
            | side | price | volume |
            | buy  | 496   | 0      |
            | buy  | 498   | 5      |
            | sell | 502   | 5      |
            | sell | 504   | 10     |
        # Trigger mark to market, then check margin levels
        When the parties place the following orders:
            | party | market id | side | volume | price | resulting trades | type       | tif     |
            | aux1  | ETH/FEB23 | buy  | 1      | 501   | 0                | TYPE_LIMIT | TIF_GTC |
            | aux2  | ETH/FEB23 | sell | 1      | 501   | 1                | TYPE_LIMIT | TIF_GTC |
            | aux1  | ETH/FEB23 | buy  | 1      | 500   | 0                | TYPE_LIMIT | TIF_GTC |
            | aux2  | ETH/FEB23 | sell | 1      | 500   | 1                | TYPE_LIMIT | TIF_GTC |
        And the network moves ahead "1" blocks
        Then the mark price should be "500" for the market "ETH/FEB23"
        And the parties should have the following margin levels:
            | party       | market id | maintenance | search | initial | release |
            | longTrader  | ETH/FEB23 | 14250       | 15675  | 17100   | 19950   |


        # Case for 0019-MCAL-016:
        #   - For the party 'shortTrader'; riskiest short < 0 && abs(riskiest short) > sum of volume of order book asks
        When the parties cancel the following orders:
            | party            | reference |
            | sellSideProvider | ssp-1     |
        Then the order book should have the following volumes for market "ETH/FEB23":
            | side | price | volume |
            | buy  | 496   | 0      |
            | buy  | 498   | 5      |
            | sell | 502   | 5      |
            | sell | 504   | 0      |
        # Trigger mark to market, then check margin levels
        When the parties place the following orders:
            | party | market id | side | volume | price | resulting trades | type       | tif     |
            | aux1  | ETH/FEB23 | buy  | 1      | 501   | 0                | TYPE_LIMIT | TIF_GTC |
            | aux2  | ETH/FEB23 | sell | 1      | 501   | 1                | TYPE_LIMIT | TIF_GTC |
            | aux1  | ETH/FEB23 | buy  | 1      | 500   | 0                | TYPE_LIMIT | TIF_GTC |
            | aux2  | ETH/FEB23 | sell | 1      | 500   | 1                | TYPE_LIMIT | TIF_GTC |
        And the network moves ahead "1" blocks
        Then the mark price should be "500" for the market "ETH/FEB23"
        And the parties should have the following margin levels:
            | party       | market id | maintenance | search | initial | release |
            | shortTrader | ETH/FEB23 | 14250       | 15675  | 17100   | 19950   |

        
        # Case for 0019-MCAL-014:
        #   - For the party 'longTrader'; riskiest long > 0 && no bids on the book
        #   - For the party 'shortTrader'; riskiest short < 0 && no asks on the book
        When the parties place the following orders:
            | party | market id | side | volume | price | resulting trades | type       | tif     |
            | aux1  | ETH/FEB23 | buy  | 5      | 502   | 1                | TYPE_LIMIT | TIF_GTC |
            | aux2  | ETH/FEB23 | sell | 5      | 498   | 1                | TYPE_LIMIT | TIF_GTC |
        And the network moves ahead "1" blocks:
        Then the order book should have the following volumes for market "ETH/FEB23":
            | side | price | volume |
            | buy  | 496   | 0      |
            | buy  | 498   | 0      |
            | sell | 502   | 0      |
            | sell | 504   | 0      |
        Then the mark price should be "498" for the market "ETH/FEB23"
        And the parties should have the following margin levels:
            | party       | market id | maintenance | search | initial | release |
            | longTrader  | ETH/FEB23 | 14193       | 15612  | 17031   | 19870   |
            | shortTrader | ETH/FEB23 | 14193       | 15612  | 17031   | 19870   |
