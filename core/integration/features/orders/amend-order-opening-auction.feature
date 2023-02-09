Feature: Amend orders

  Background:
    Given the average block duration is "1"
    And the markets:
      | id        | quote name | asset | risk model                  | margin calculator                  | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | BTC        | BTC   | default-simple-risk-model-2 | default-overkill-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  @AmendOA
  Scenario: Amend an order during opening auction, we should leave the auction the next time update
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party | asset | amount    |
      | myboi | BTC   | 10000000  |
      | aux   | BTC   | 100000000 |
      | aux2  | BTC   | 100000000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | aux   | ETH/DEC19 | buy  | 100    | 9900  | 0                | TYPE_LIMIT | TIF_GTC | aux-b-9   |
      | aux   | ETH/DEC19 | sell | 100    | 10010 | 0                | TYPE_LIMIT | TIF_GTC | aux2-s-10 |
      | aux2  | ETH/DEC19 | buy  | 1      | 10000 | 0                | TYPE_LIMIT | TIF_GTC | aux2-b-k  |
      | aux   | ETH/DEC19 | sell | 1      | 10001 | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1   |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC19"
    #And the mark price should be "10000" for the market "ETH/DEC19"

    # Amend order, we remain in auction
    When the parties amend the following orders:
      | party | reference | price | size delta | tif     |
      | aux   | aux-s-1   | 10000 | 0          | TIF_GTC |
    Then the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC19"

    # next block
    When the network moves ahead "1" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the mark price should be "10000" for the market "ETH/DEC19"
