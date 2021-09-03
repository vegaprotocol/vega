# Vega core architecture

Data node is a stand alone product that is built on the top of Vega core product.
It consumes stream of events from core Vega via socket using [Broker](./broker.md) then aggregates the events and save them to storage.

## Component relationships

The following diagram shows how the various components of this implementation interact with each other at a high level.

![Vega core protocol architecture](diagrams/design-architecture-191003001.svg "Vega core protocol architecture")

## Modelling the domain

Some subdirectories contain Golang packages which represent a discrete domain or concept from the whitepaper.