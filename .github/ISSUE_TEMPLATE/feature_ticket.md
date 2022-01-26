---
name: Feature ticket
about: A full description of a new feature, or part of a feature, that we wish to develop
title: ''
labels: "feature"
assignees: ''
---

# Feature Overview

**In order to** (context - overcome a problem or meet a requirement)
**We will** (what - carry out this piece of work / action)
**So that** (why - we create these outcomes)

## Specs
- [Link](xyz) to spec or section within a spec

# Tasks
A checklist of the tasks that are needed to develop the feature and meet the acceptance criteria and feature test scenarios. Ideally, tasks would reflect the issues/pull requests likely to be created when developing the feature. 
- [ ]
- [ ]

# Acceptance Criteria
A list of criteria (aim for 3!) that have to be met for this feature to be accepted as 'done' by the product owner.

- It is possible to

Acceptance Criteria and Feature Test Scenarios can, in some cases, be closely related. If acceptance criteria become fully covered by feature test scenarios they can be removed leaving on acceptance criteria that can't be directly or fully proven with tests.

# Test Scenarios
Detailed scenarios (1-3!) that can be executed as feature tests to verify that the feature has been implemented as expected. We use the follow format:

GIVEN (setup/context) 
WHEN (action) 
THEN (assertion) For example...
See [here](https://github.com/vegaprotocol/vega/tree/develop/integration/) for more format information and examples.

# Additional Details (optional)
Any additional information that provides context or gives information that will help us develop the feature. 

# Examples (optional)
Code snippets from the spec for reference

# Definition of Done
>ℹ️ Not every issue will need every item checked, however, every item on this list should be properly considered and actioned to meet the [DoD](https://github.com/vegaprotocol/vega/blob/develop/DEFINITION_OF_DONE.md).

**Before Merging**
- [ ] Create relevant for [system-test](https://github.com/vegaprotocol/system-tests/issues) tickets with feature labels
- [ ] Code refactored to meet SOLID and other code design principles
- [ ] Code is compilation error, warning, and hint free
- [ ] Carry out a basic happy path end-to-end check of the new code
- [ ] All APIs are documented so auto-generated documentation is created
- [ ] All acceptance criteria confirmed to be met, or, reasons why not discussed with the engineering leadership team
- [ ] All Unit, Integration and BVT tests are passing
- [ ] Implementation is peer reviewed (coding standards, meeting acceptance criteria, code/design quality)
- [ ] Create [front end](https://github.com/vegaprotocol/token-frontend/issues) or [console](https://github.com/vegaprotocol/console/issues) tickets with feature labels (should be done when starting the work if dependencies known i.e. API changes)

**After Merging**
- [ ] Move development ticket to `Done` if there is **NO** requirement for new system-tests
- [ ] Resolve any issues with broken system-tests
- [ ] Create [documentation](https://github.com/vegaprotocol/documentation/issues) tickets with feature labels if functionality has changed, or is a new feature
