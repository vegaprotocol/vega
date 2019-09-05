Feature: Test trading-core flow

    Background:
        Given the market ETH/DEC19
        And the system accounts:
            | type       | asset | balance |
            | settlement |  ETH  | 0       |
            | insurance  |  ETH  | 0       |
        And traders have the following state:
            | trader  | position | margin | general | asset | markprice |
            | trader1 | 10       | 1000   | 9000    | ETH   | 100       |
            | trader2 | -5       | 2000   | 10000   | ETH   | 100       |
            | trader3 | -5       | 100    | 0       | ETH   | 100       |

    Scenario: A simple trade between active traders, short wins
        Given the following orders:
            | trader  | type | volume | price |
            | trader1 | sell | 1      | 50    |
            | trader2 | buy  | 1      | 50    |
        When I check the updated balances and positions
        Then I expect to see:
            | trader  | position | margin | general | asset | markprice |
            | trader1 | 9        | 550    | 9000    | ETH   | 50        |
            | trader2 | -4       | 2200   | 10000   | ETH   | 50        |
            | trader3 | -5       | 350    | 0       | ETH   | 50        |
