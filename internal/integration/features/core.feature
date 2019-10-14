Feature: Test trading-core flow

    Background:
        ## mark price will be set on instrument, given + data table
        ## With these values, we get risk factors:
        ## short=0.11000000665311127, long=0.10036253585651489
        Given the market:
            | name      | markprice | lamd | tau         | mu | r | sigma     | release factor | initial factor | search factor |
            | ETH/DEC19 | 100       | 0.01 | 0.000114077 | 0  | 0 | 3.6907199 | 1.4            | 1.2            | 1.1           |
        And the system accounts:
            | type       | asset | balance |
            | settlement |  ETH  | 0       |
            | insurance  |  ETH  | 0       |
        And traders have the following state:
            | trader  | position | margin | general | asset | markprice |
            | trader1 | 0        | 0      | 100000  | ETH   | 100       |
            | trader2 | 0        | 0      | 100000  | ETH   | 100       |
            | trader3 | 0        | 0      | 100000  | ETH   | 100       |

    Scenario: trader places unmatched order and creates a position. The margin balance is created
        Given the following orders:
            | trader  | type | volume | price | resulting trades |
            | trader1 | sell | 1      | 1010  | 0                |
        Then I expect the trader to have a margin liability:
            | trader  | position | buy | sell | margin | general |
            | trader 1| 0        | 0   | 1    | 132    | 99868   |
        And "trader2" has not been added to the market

    Scenario: two traders place orders at different prices
        Given the following orders:
            | trader  | type | volume | price | resulting trades |
            | trader1 | sell | 1      | 1010  | 0                |
            | trader2 | buy  | 1      | 1030  | 0                |
        Then I expect the trader to have a margin liability:
            | trader  | position | buy | sell | margin | general |
            | trader 1| 0        | 0   | 1    | 132    | 99868   |
            | trader 2| 0        | 1   | 0    | 120    | 99880   |
        And "trader3" has not been added to the market

    Scenario: Three traders place orders, resulting in two trade
        Given the following orders:
            | trader  | type | volume | price | resulting trades |
            | trader1 | sell | 1      | 980   | 0                |
            | trader1 | sell | 1      | 1020  | 0                |
        Then I expect the trader to have a margin liability:
            | trader  | position | buy | sell | margin | general |
            | trader 1| 0        | 0   | 2    | 269    | 99741   |
        When I place the following orders:
            | trader  | type | volume | price | resulting trades |
            | trader2 | buy  | 1      | 980   | 1                |
            | trader3 | buy  | 1      | 1020  | 1                |
        Then I expect the trader to have a margin liability:
            | trader  | position | buy | sell | margin | general |
            | trader 1| -2       | 0   | 0    | 269    | 99741   |
            | trader 2| 1        | 0   | 0    | 135    | 99865   |
            | trader 3| 1        | 0   | 0    | 135    | 99865   |
