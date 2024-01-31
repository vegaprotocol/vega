Feature: replicate settlement bug on mainnet 31 Jan 2024

  Background:

    Given time is updated to "2020-10-16T00:00:00Z"

    # Configure the network
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
    And the following assets are registered:
      | id  | decimal places |
      | ETH | 0              |

    And the perpetual oracles from "0xCAFECAFE1":
      | name        | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals |
      | perp-oracle | ETH   | perp.ETH.value      | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | 0.9                   | 0.1           | 0                 | 0                 | ETH        | 0                   |
    Given the liquidation strategies:
      | name                | disposal step | disposal fraction | full disposal size | max fraction consumed |
      | liquidation-strat-1 | 3600          | 0.5               | 0                  | 1                     |
    And the markets:
      | id        | quote name | asset | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config | linear slippage factor | quadratic slippage factor | liquidation strategy | sla params    | market type |
      | ETH/MAR22 | ETH        | ETH   | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | perp-oracle        | 0.001                  | 0                         | liquidation-strat-1  | default-basic | perp        |

  @Perpetual
  Scenario:
    # start with solvent but not rich parties.
    # make sure some volume is on the book
    # suspend market
    # keep sending external price different to last trade price causing funding payments.
    # some parties will become distressed due to funding payments but can’t be closed out in an auction
    # terminate market and see what happens.

    Given the initial insurance pool balance is "10000" for all the markets
    And the parties deposit on asset's general account the following amount:
      | party | asset | amount       |
      | lp1   | ETH   | 100000000000 |
      | aux1  | ETH   | 10000000000  |
      | aux2  | ETH   | 10000000000  |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 500000            | 0   | submission |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | lp1   | ETH/MAR22 | buy  | 1000   | 199   | 0                | TYPE_LIMIT | TIF_GTC | best-bid  |
      | lp1   | ETH/MAR22 | sell | 1000   | 201   | 0                | TYPE_LIMIT | TIF_GTC | best-ask  |
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/MAR22 | buy  | 1      | 200   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/MAR22 | sell | 1      | 200   | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/MAR22"
    Then the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            |
      | 200        | TRADING_MODE_CONTINUOUS |

    # atRiskPary opens a long position
    Given the parties deposit on asset's general account the following amount:
      | party       | asset | amount |
      | atRiskParty | ETH   | 100    |
    And the parties place the following orders:
      | party       | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1        | ETH/MAR22 | sell | 1      | 200   | 0                | TYPE_LIMIT | TIF_GTC |
      | atRiskParty | ETH/MAR22 | buy  | 1      | 200   | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the parties should have the following profit and loss:
      | party       | volume | unrealised pnl | realised pnl |
      | atRiskParty | 1      | 0              | 0            |
    And the parties should have the following margin levels:
      | party       | market id | maintenance | search | initial | release |
      | atRiskParty | ETH/MAR22 | 15          | 16     | 18      | 21      |
    And the parties should have the following account balances:
      | party       | asset | market id | margin | general |
      | atRiskParty | ETH   | ETH/MAR22 | 16     | 84      |

    When the market states are updated through governance:
      | market id | state                            | settlement price |
      | ETH/MAR22 | MARKET_STATE_UPDATE_TYPE_SUSPEND | 198              |
    Then the market state should be "STATE_SUSPENDED_VIA_GOVERNANCE" for the market "ETH/MAR22"

    And the parties should have the following account balances:
      | party       | asset | market id | margin | general |
      | atRiskParty | ETH   | ETH/MAR22 | 16     | 84      |

    # Start a new funding period
    Given time is updated to "2020-10-16T00:05:00Z"
    And the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value      | time offset |
      | perp.funding.cue | 1602806700 | 0s          |
      | perp.ETH.value   | 220        | 1s          |

    # Negative funding payment, shorts pay longs, gains paid into insurance pool
    Given time is updated to "2020-10-16T00:10:00Z"
    And the product data for the market "ETH/MAR22" should be:
      | internal twap | external twap | funding payment | funding rate |
      | 0             | 0             | 0               | 0            |




