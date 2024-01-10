Feature: Profit and loss for network a running liquidation strategy

  Background:

    # Configure the network
    Given the average block duration is "1"
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
    And the following assets are registered:
      | id      | decimal places | quantum |
      | USD.0.1 | 0              | 1       |

    # Configure the markets
    Given the liquidation strategies:
      | name              | disposal step | disposal fraction | full disposal size | max fraction consumed |
      | liquidation-strat | 3600          | 1                 | 0                  | 1                     |
    And the markets:
      | id        | quote name | asset    | risk model                    | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | liquidation strategy | sla params    |
      | ETH/MAR22 | ETH        | USD.0.10 | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.001                  | 0                         | liquidation-strat    | default-basic |


  @NoPerp @NetPNL
  Scenario: Network long then liquidates short positions (0003-MTMK-015)(0012-POSR-016)

    # Setup the market
    Given the initial insurance pool balance is "10000" for all the markets
    And the parties deposit on asset's general account the following amount:
      | party | asset    | amount       |
      | lp1   | USD.0.10 | 100000000000 |
      | aux1  | USD.0.10 | 10000000000  |
      | aux2  | USD.0.10 | 10000000000  |
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
      | party       | asset    | amount |
      | atRiskParty | USD.0.10 | 100    |
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
      | party       | asset    | market id | margin | general |
      | atRiskParty | USD.0.10 | ETH/MAR22 | 16     | 84      |

    # Market moves against atRiskParty whom is liquidated
    Given the parties amend the following orders:
      | party | reference | price | size delta | tif     |
      | lp1   | best-bid  | 99    | 0          | TIF_GTC |
      | lp1   | best-ask  | 101   | 0          | TIF_GTC |
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/MAR22 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/MAR22 | sell | 1      | 100   | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the mark price should be "100" for the market "ETH/MAR22"
    And the parties should have the following profit and loss:
      | party       | volume | unrealised pnl | realised pnl |
      | atRiskParty | 0      | 0              | -100         |
      | network     | 1      | 0              | 0            |
    And the insurance pool balance should be "10000" for the market "ETH/MAR22"

    # atRiskPary opens a short position
    Given the parties deposit on asset's general account the following amount:
      | party       | asset    | amount |
      | atRiskParty | USD.0.10 | 20     |
    And the parties place the following orders:
      | party       | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1        | ETH/MAR22 | buy  | 2      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | atRiskParty | ETH/MAR22 | sell | 2      | 100   | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the parties should have the following profit and loss:
      | party       | volume | unrealised pnl | realised pnl |
      | atRiskParty | -2     | 0              | -100         |
    And the parties should have the following margin levels:
      | party       | market id | maintenance | search | initial | release |
      | atRiskParty | ETH/MAR22 | 16          | 17     | 19      | 22      |
    And the parties should have the following account balances:
      | party       | asset    | market id | margin | general |
      | atRiskParty | USD.0.10 | ETH/MAR22 | 18     | 2       |

    # Market moves against atRiskParty whom is liquidated
    Given the parties amend the following orders:
      | party | reference | price | size delta | tif     |
      | lp1   | best-ask  | 121   | 0          | TIF_GTC |
      | lp1   | best-bid  | 119   | 0          | TIF_GTC |
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/MAR22 | buy  | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/MAR22 | sell | 1      | 120   | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the mark price should be "120" for the market "ETH/MAR22"
    And the parties should have the following profit and loss:
      | party       | volume | unrealised pnl | realised pnl |
      | atRiskParty | 0      | 0              | -140         |
      | network     | -1     | 0              | 20           |
    And the insurance pool balance should be "10000" for the market "ETH/MAR22"

    # Market moves in favour of the network
    Given the parties amend the following orders:
      | party | reference | price | size delta | tif     |
      | lp1   | best-bid  | 59    | 0          | TIF_GTC |
      | lp1   | best-ask  | 61    | 0          | TIF_GTC |
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/MAR22 | buy  | 1      | 60    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/MAR22 | sell | 1      | 60    | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the mark price should be "60" for the market "ETH/MAR22"
    And the parties should have the following profit and loss:
      | party       | volume | unrealised pnl | realised pnl |
      | atRiskParty | 0      | 0              | -140         |
      | network     | -1     | 60             | 20           |
    And the insurance pool balance should be "10060" for the market "ETH/MAR22"


  @NoPerp @NetPNL
  Scenario: Network long then liquidates further long positions (0012-POSR-017)

    # Setup the market
    Given the initial insurance pool balance is "10000" for all the markets
    And the parties deposit on asset's general account the following amount:
      | party | asset    | amount       |
      | lp1   | USD.0.10 | 100000000000 |
      | aux1  | USD.0.10 | 10000000000  |
      | aux2  | USD.0.10 | 10000000000  |
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
      | party       | asset    | amount |
      | atRiskParty | USD.0.10 | 100    |
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
      | party       | asset    | market id | margin | general |
      | atRiskParty | USD.0.10 | ETH/MAR22 | 16     | 84      |

    # Market moves against atRiskParty whom is liquidated
    Given the parties amend the following orders:
      | party | reference | price | size delta | tif     |
      | lp1   | best-bid  | 99    | 0          | TIF_GTC |
      | lp1   | best-ask  | 101   | 0          | TIF_GTC |
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/MAR22 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/MAR22 | sell | 1      | 100   | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the mark price should be "100" for the market "ETH/MAR22"
    And the parties should have the following profit and loss:
      | party       | volume | unrealised pnl | realised pnl |
      | atRiskParty | 0      | 0              | -100         |
      | network     | 1      | 0              | 0            |
    And the insurance pool balance should be "10000" for the market "ETH/MAR22"

    # atRiskPary opens a long position
    Given the parties deposit on asset's general account the following amount:
      | party       | asset    | amount |
      | atRiskParty | USD.0.10 | 10     |
    And the parties place the following orders:
      | party       | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1        | ETH/MAR22 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | atRiskParty | ETH/MAR22 | buy  | 1      | 100   | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the parties should have the following profit and loss:
      | party       | volume | unrealised pnl | realised pnl |
      | atRiskParty | 1      | 0              | -100         |
    And the parties should have the following margin levels:
      | party       | market id | maintenance | search | initial | release |
      | atRiskParty | ETH/MAR22 | 8           | 8      | 9       | 11      |
    And the parties should have the following account balances:
      | party       | asset    | market id | margin | general |
      | atRiskParty | USD.0.10 | ETH/MAR22 | 8      | 2       |

    # Market moves against atRiskParty whom is liquidated
    Given the parties amend the following orders:
      | party | reference | price | size delta | tif     |
      | lp1   | best-bid  | 89    | 0          | TIF_GTC |
      | lp1   | best-ask  | 91    | 0          | TIF_GTC |
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/MAR22 | buy  | 1      | 90    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/MAR22 | sell | 1      | 90    | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the mark price should be "90" for the market "ETH/MAR22"
    And the parties should have the following profit and loss:
      | party       | volume | unrealised pnl | realised pnl |
      | atRiskParty | 0      | 0              | -110         |
      | network     | 2      | -10            | 0            |
    And the insurance pool balance should be "9990" for the market "ETH/MAR22"

    # Market moves against the network
    Given the parties amend the following orders:
      | party | reference | price | size delta | tif     |
      | lp1   | best-bid  | 59    | 0          | TIF_GTC |
      | lp1   | best-ask  | 61    | 0          | TIF_GTC |
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/MAR22 | buy  | 1      | 60    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/MAR22 | sell | 1      | 60    | 1                | TYPE_LIMIT | TIF_GTC |
    When the network moves ahead "1" blocks
    Then the mark price should be "60" for the market "ETH/MAR22"
    And the parties should have the following profit and loss:
      | party       | volume | unrealised pnl | realised pnl |
      | atRiskParty | 0      | 0              | -110         |
      | network     | 2      | -70            | 0            |
    And the insurance pool balance should be "9930" for the market "ETH/MAR22"