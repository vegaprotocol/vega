---
name: Technical task
about: A description of the technical task to be carried out
title: ''
labels: ''
assignees: ''
---

# Task Overview

**In order to** (context - overcome a problem or meet a requirement)
**We will** (what - carry out this piece of work / action)
**So that** (why - we create these outcomes)

## Specs
- [Link](xyz) to spec or milestone document info for the feature

# Acceptance Criteria
How do we know when this technical task is complete:
- It is possible to...
- Vega is able to...

# Test Scenarios
Detailed scenarios (1-3!) that can be executed as feature tests to verify that the feature has been implemented as expected.

GIVEN (setup/context) 
WHEN (action) 
THEN (assertion) 
See [here](https://github.com/vegaprotocol/vega/tree/develop/integration/) for more format information and examples.

# Dependencies
Links to any tickets that have a dependant relationship witht his task.

# Additional Details (optional)
Any additional information including known dependencies, impacted components.

# Examples (optional)
Code snippets, API calls that could be used on dependant tasks.

# Definition of Done
>ℹ️ Not every issue will need every item checked, however, every item on this list should be properly considered and actioned to meet the [DoD](https://github.com/vegaprotocol/vega/blob/develop/DEFINITION_OF_DONE.md).

**Before Merging**
- [ ] Create relevant for [system-test](https://github.com/vegaprotocol/system-tests/issues) tickets with feature labels
- [ ] Code refactored to meet SOLID and other code design principles
- [ ] Code is compilation error, warning, and hint free
- [ ] Carry out a basic happy path end-to-end check of the new code
- [ ] All acceptance criteria confirmed to be met, or, reasons why not discussed with the engineering leadership team
- [ ] All APIs are documented so auto-generated documentation is created
- [ ] All Unit, Integration and BVT tests are passing
- [ ] Implementation is peer reviewed (coding standards, meeting acceptance criteria, code/design quality)
- [ ] Create [front end](https://github.com/vegaprotocol/token-frontend/issues) or [console](https://github.com/vegaprotocol/console/issues) tickets with feature labels (should be done when starting the work if dependencies known i.e. API changes)

**After Merging**
- [ ] Move development ticket to `Done` if there is **NO** requirement for new system-tests
- [ ] Resolve any issues with broken system-tests
- [ ] Create [documentation](https://github.com/vegaprotocol/documentation/issues) tickets with feature labels if functionality has changed, or is a new feature
