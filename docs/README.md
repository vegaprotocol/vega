# Vega core architecture

Our core protocol implementation aims to be a reflection on the protocol design outlined in the whitepaper. It is currently written in Golang.

## Component relationships

The following diagram shows how the various components of this implementation interact with each other at a high level.

![Vega core protocol architecture](diagrams/design-architecture-191003001.svg "Vega core protocol architecture")

## Modelling the domain

Some subdirectories contain Golang packages which represent a discrete domain or concept from the whitepaper.

### Design documentation

In order to document the design, each package should have a single markdown file in the /design directory

1. [Matching package](../matching/README.md)
2. [Position package](../positions/README.md)

#
