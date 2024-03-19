Feature: Spot market

  Background:
    Given time is updated to "2024-01-01T00:00:00Z"

    Given the following network parameters are set:
      | name                                                | value |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
      | market.value.windowLength                           | 1h    |
    
    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 2              |
      | BTC | 2              |

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.01      | 0.03               |
    Given the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 60      | 0.999       | 1                 |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | decimal places | position decimal places | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 100              | fees-config-1 | price-monitoring-1 | 2              | 2                       | default-basic |

    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 10000  |
      | party2 | ETH   | 10000  |
      | party4 | BTC   | 1000   |
      | party5 | BTC   | 1000   |
    And the average block duration is "1"

  Scenario: A market with default trading mode "continuous trading" will start with an opening auction. The opening auction will
            run from the close of voting on the market proposal (assumed to pass successfully) until: the enactment time assuming
            there are orders crossing on the book, there is no need for the supplied liquidity to exceed a threshold to exit an auction: (0026-AUCT-029)

    # Place some orders that cross
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | buy1      |
      | party5 | BTC/ETH   | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GFA | sell1     |

    # Check nothing is crossed before the end of the auction
    When the network moves ahead "50" blocks
    Then the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "BTC/ETH"

    # Now we have gone past the enactment time, we should come out of auction
    When the network moves ahead "52" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "BTC/ETH"


  Scenario: No crossed orders, no leaving the auction

    # Place some orders that don't cross so we can't leave auction
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | BTC/ETH   | buy  | 1      | 999   | 0                | TYPE_LIMIT | TIF_GFA | buy1      |
      | party5 | BTC/ETH   | sell | 1      | 1001  | 0                | TYPE_LIMIT | TIF_GFA | sell1     |

    # After the enactment time has expired, the auction can't leave
    When the network moves ahead "102" blocks
    Then the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "BTC/ETH"
