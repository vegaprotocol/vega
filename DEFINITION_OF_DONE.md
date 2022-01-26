# Definition of Done (DoD)

## What is a Definition of Done (DoD)?

We use a Definition of Done to clarify what exactly the team means when we declare something ‘done’.

In more concrete terms, it lists the criteria we’ll meet before you claim a feature, enhancement or bug fix can be shipped (released).

The difference between a Definition of Done and Acceptance Criteria...

- The Definition of Done applies to all your work. Acceptance Criteria only ever apply to a single item on your backlog
- The Definition of Done clarifies what you do as a team. Acceptance Criteria clarify what the protocol does — its functionality
- The Definition of Done can contain criteria for “non-functional” aspects and cross-cutting concerns.

## Vega Core Team Definition of Done

**Before merging**
- Create relevant for [system-test](https://github.com/vegaprotocol/system-tests/issues) tickets with feature labels (should be done when starting the work)
- Code refactored to meet SOLID and other code design principles
- Code is compilation error, warning, and hint free
- Carry out a basic happy path end-to-end check of the new code
- All acceptance criteria confirmed to be met, or, reasons why not discussed with the engineering leadership team
- All APIs are documented so auto-generated documentation is created
- All Unit, Integration and BVT tests are passing
- Implementation is peer reviewed (coding standards, meeting acceptance criteria, code/design quality)
- Create [front end](https://github.com/vegaprotocol/token-frontend/issues) or [console](https://github.com/vegaprotocol/console/issues) tickets with feature labels (should be done when starting the work if dependencies known i.e. API changes)

> ℹ️ In most cases the person that raised the PR should be the one to squash branch history to string of passing, sensible commits, and merge the PR.

**After merging**
- Move development ticket to `Done` if there is **NO** requirement for new system-tests
- Resolve any issues with broken system-tests
- Create [documentation](https://github.com/vegaprotocol/documentation/issues) tickets with feature labels if functionality has changed, or is a new feature

**Before Testnet**
- Acceptance (feature) tests passing (end to end / black-box testing)
- Full set of functional regression tests passing
- Incentives (where applicable) have been planned with the community team

**Before Mainnet**
- Incentives (where applicable) have been run successfully in testnet

> ℹ️ In most cases the person that raised the PR should be the one to squash branch history to string of passing, sensible commits, and merge the PR.

