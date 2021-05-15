---
name: Feature ticket
about: A full description of a new feature, or part of a feature, that we wish to develop
title: ''
labels: "feature"
assignees: ''
---

_Ensure the ticket title clearly communicates what this feature ticket makes possible_

_Add the **milestone** and **project** of the Major Release this feature is part of i.e. Oregon Trail_

_Add a **label** for the feature that this ticket is part of i.e. Data Sourcing_

# Feature Overview
A simple overview that describes what this feature (or sub feature) needs to do and why it's valuable. 
This should be brief, understandable by anyone in the business and use the format: 

As a (who - the type of user/actor the feature serves)

I want (what - the action to be performed / made possible)

So that (why - the goal/result/value it achieves to the user/actor or the business)

# Tasks
A checklist of the tasks that are needed to develop the feature and meet the accceptance criteria and feature test scenarios. Ideally, tasks would reflect the pull requests likely to be created when developing the feature. 

- [ ]
- [ ]
- [ ]

# Product Owner
The name of the person in the Product team that is responsible for this feature. This will be the go-to person for the engineer working on this ticket for any questions / clarifications, to get feedback on work in progress and who will ultimately accept the feature ticket as 'done'.

_**Assign** the named product owner to the ticket_

# Acceptance Criteria
A list of criteria (aim for 3!) that have to be met for this feature to be accepted as 'done' by the product owner. Acceptance criteria should be simple, single sentence, statements written from the perspective of the work already having been done. Each statement should be able to be objectively determined to be true or false. For example:

- It is possible to
- It is possible to
- Vega does
- Vega does

Acceptance Criteria and Feature Test Scenarios can, in some cases, be closely related. If acceptance criteria become fully covered by feature test scenarios they can be removed leaving on acceptance criteria that can't be directly or fully proven with tests.

# Test Scenarios
Detailed scenarios (1-3!) that can be executed as feature tests to verify that the feature has been implemented as expected. We use the follow format:

GIVEN (setup/context) 

WHEN (action) 

THEN (assertion) 

For example...

```gherkin
    Feature: Account Holder withdraws cash
    Scenario: Account has sufficient funds
    Given the account balance is $100
      And the card is valid
      And the machine contains enough money  
    When the Account Holder requests $20
    Then the ATM should dispense $20
      And the account balance should be $80
      And the card should be returned
```     

See https://github.com/vegaprotocol/vega/tree/develop/integration/ for more format information and examples.

_TBC whether these should be separate files  - will come back to this_

# Impacted Systems / Engines
A list of the engines that we believe will be impacted by the development of this feature. _Delete as appropriate_

- Core - banking
- Core - collateral
- Core - execution
- Core - fee
- Core - governance
- Core - liquidity
- Core - oracles
- Core - positions
- Core - risk
- Core - settlement
- Core - monitor
- API
- Wallet
- Liquidity bot
- Trader bot

# API Calls 
A list of the API calls that are needed for this feature, written in an implementation-agnostic format i.e. "Get a list of widgets, categorised by X", rather than "GET /widgets?category=X":

- Get a list of widgets, categorised by X
- Get x
- Get y

# Dependencies
Links to the tickets that represent work that this feature ticket is dependent on to be able to start / finish development. This could be another feature ticket, an important refactoring task, some infrastructure work etc. For each dependency please add a categorisation of the type of dependency (hard or soft) and whether it is internal or external to the team.

- #Link to ticket + Dependency ticket name | hard/soft | internal/external 
- #Link to ticket + Dependency ticket name | hard/soft | internal/external
- #Link to ticket + Dependency ticket name | hard/soft | internal/external 

_Add a **label(s)** to represent the types of dependency this feature ticket has (soft_internal_dependency, soft_external_dependency, hard_internal_dependency, hard_external_dependency)_

# Additional Details (optional)
Any additional information that provides context or gives information that will help us develop the feature. 

Feature spec: 

# Examples (optional)
Code snippets from the spec for reference
