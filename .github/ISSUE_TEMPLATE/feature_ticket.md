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

# Acceptance Criteria
A list of criteria (aim for 3!) that have to be met for this feature to be accepted as 'done' by the product team. Acceptance criteria should be simple, single sentence statements written from the perspective of the work already having been done. Each statement should be able to be objectively determined to be true or false. For example:

- It is possible to
- It is possible to
- Vega does
- Vega does

# Feature Test Scenarios
Links to scenarios (at least 1!) that can be executed as feature tests to verify that the feature has been implemented as expected. We use the follow format:

GIVEN (setup/context) 

WHEN (action) 

THEN (assertion) 

For example...

    Feature: Account Holder withdraws cash

    Scenario: Account has sufficient funds
   
    GIVEN the account balance is $100
    
      AND the card is valid
    
      AND the machine contains enough money  
    
    WHEN the Account Holder requests $20
    
    THEN the ATM should dispense $20
     
      AND the account balance should be $80
     
      AND the card should be returned
     
See https://github.com/vegaprotocol/vega/tree/develop/integration/ for more format information and examples.

_TBC whether these should be separate files  - will come back to this_

# Impacted Engines
A list of the engines that we believe will be impacted by the development of this feature.

- [ ] banking
- [ ] collateral
- [ ] execution
- [ ] fee
- [ ] governance
- [ ] liquidity
- [ ] oracles
- [ ] positions
- [ ] risk
- [ ] settlement
- [ ] monitor

# API Calls 
A list of the API calls that are needed for this feature, written in an implementation-agnostic format:

- "Get a list of widgets, categorised by X", rather than "GET /widgets?category=X"

# Dependencies
Links to the tickets that reflect all our known dependencies, along with a categorisation of the type of dependency - mandatory/hard OR discretionary/soft - and whether it is internal or external to the team.

# Additional Details (optional)
Any additional information that provides context or gives information that will help us develop the feature. 

Feature spec:

Release overview document:

Release board:

# Examples (optional)
Code snippets from the spec for reference
