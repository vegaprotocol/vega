# Ethereum event forwarder

This package contains the specific implementation of the event forwarder for Ethereum blockchain.

It reads the logs of Vega's bridge contracts, filters on specific events, translates them to ChainEvent transactions and use the Event Forwarder engine to forward it to the network.
