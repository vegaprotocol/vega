Feature: Test trading-core flow

    Background:
        ## mark price will be set on instrument, given + data table
        ## Get tradable instrument from J.Latang PR
        Given the market ETH/DEC19
        And the system accounts:
            | type       | asset | balance |
            | settlement |  ETH  | 0       |
            | insurance  |  ETH  | 0       |
        And traders have the following state:
            | trader  | position | margin | general | asset | markprice |
            | trader1 | 0        | 0      | 10000   | ETH   | 100       |
            | trader2 | 0        | 0      | 10000   | ETH   | 100       |
            | trader3 | 0        | 0      | 10000   | ETH   | 100       |

    Scenario: 1. A simple trade between active traders, short wins
        Given the following order:
            | trader  | type | volume | price | resulting trades |
            | trader1 | sell | 1      | 50    | 0                |
        Then I expect the trader to have a margin liability:
            | trader  | position | buy | sell | margin | general |
            | trader 1| 0        | 0   | 1    | 1000   | 9000    |
        And "trader2" has not been added to the market

    Scenario: 2. A simple trade between active traders, MTM == zero
        Given the following orders:
            | trader  | type | volume | price | resulting trades |
            | trader1 | sell | 1      | 98    | 0                |
            | trader2 | buy  | 1      | 98    | 1                |
        When I check the updated balances and positions
        Then I expect to see:
            | trader  | position | margin | general | asset | markprice |
            | trader1 | -1       | 1000   | 9000    | ETH   | 98        |
            | trader2 | 1        | 999    | 9001    | ETH   | 98        |
            | trader3 | 0        | 0      | 10000   | ETH   | 98        |


    Scenario: 3. A simple trade between active traders, 3 traders, MTM cashflow
        Given the following orders:
            | trader  | type | volume | price | resulting trades |
            | trader1 | sell | 1      | 98    | 0                |
            | trader1 | sell | 1      | 102   | 0                |
            | trader2 | buy  | 1      | 98    | 1                |
            | trader3 | buy  | 1      | 102   | 1                |
        When I check the updated balances and positions
        Then I expect to see:
            | trader  | position | margin | general | asset | markprice |
            | trader1 | -2       | 1000   | 9000    | ETH   | 102       |
            | trader2 | 1        | 999    | 9001    | ETH   | 102       |
            | trader3 | 1        | 999    | 9001    | ETH   | 102       |
