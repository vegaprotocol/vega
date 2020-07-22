Feature: Test trading-core flow with simple risk model

    Background:
        Given the market:
            | name      | markprice | risk model | lamd/long | tau/short | mu | r | sigma | release factor | initial factor | search factor | openAuction | trading mode |
            | ETH/DEC19 | 1000      | simple     | 0.11       | 0.1      | 0  | 0 | 0     | 1.4            | 1.2            | 1.1           | 0           | continuous   |
        And the system accounts:
            | type       | asset | balance |
            | settlement | ETH   | 0       |
            | insurance  | ETH   | 0       |
        And traders have the following state:
            | trader  | position | margin | general | asset | markprice |
            | trader1 | 0        | 0      | 100000  | ETH   | 1000      |
            | trader2 | 0        | 0      | 100000  | ETH   | 1000      |
            | trader3 | 0        | 0      | 100000  | ETH   | 1000      |

    Scenario: trader places unmatched order and creates a position. The margin balance is created
        Given the following orders:
            | trader  | type | volume | price | resulting trades |
            | trader1 | sell | 1      | 1010  | 0                |
        Then I expect the trader to have a margin liability:
            | trader  | position | buy | sell | margin | general |
            | trader1 | 0        | 0   | 1    | 132    | 99868   |
        And "trader2" has not been added to the market
        And the mark price is "1000"

    Scenario: two traders place orders at different prices, lower buy price (no trade is created)
        Given the following orders:
            | trader  | type | volume | price | resulting trades |
            | trader1 | sell | 1      | 1010  | 0                |
            | trader2 | buy  | 1      | 1005  | 0                |
        Then I expect the trader to have a margin liability:
            | trader  | position | buy | sell | margin | general |
            | trader1 | 0        | 0   | 1    | 132    | 99868   |
            | trader2 | 0        | 1   | 0    | 120    | 99880   |
        And "trader3" has not been added to the market
        And the mark price is "1000"

    Scenario: Three traders place orders, resulting in two trade
        Given the following orders:
            | trader  | type | volume | price | resulting trades |
            | trader1 | sell | 1      | 980   | 0                |
            | trader1 | sell | 1      | 1020  | 0                |
        Then I expect the trader to have a margin liability:
            | trader  | position | buy | sell | margin | general |
            | trader1 | 0        | 0   | 2    | 264    | 99736   |
        When I place the following orders:
            | trader  | type | volume | price | resulting trades |
            | trader2 | buy  | 1      | 980   | 1                |
            | trader3 | buy  | 1      | 1020  | 1                |
        Then I expect the trader to have a margin liability:
            | trader  | position | buy | sell | margin | general |
            | trader1 |       -2 |   0 |    0 |    267 |   99693 |
            | trader2 |        1 |   0 |    0 |    122 |   99918 |
            | trader3 |        1 |   0 |    0 |    117 |   99883 |
        And the mark price is "1020"

    Scenario: two traders place orders at different prices, with a higher buy price (creates trade)
        Given the following orders:
            | trader  | type | volume | price | resulting trades |
            | trader1 | sell | 1      | 1010  | 0                |
            | trader2 | buy  | 1      | 1012  | 1                |
        Then I expect the trader to have a margin liability:
            | trader  | position | buy | sell | margin | general |
            | trader1 | -1       | 0   | 0    | 132    | 99868   |
            | trader2 | 1        | 0   | 0    | 120    | 99880   |
        And "trader3" has not been added to the market
        And the mark price is "1010"
