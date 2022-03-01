---
name: Bug report
about: Create a report to help us improve
title: ''
labels: "bug, Low, Medium, Critical, Crasher"
assignees: ''

---

# Problem encountered
A clear and concise description of what the bug is. Adjust the bug labels to identify assumed severity.

# Observed behaviour
A clear and concise description of how the system is behaving.

# Expected behaviour
A clear and concise description of what you expected to happen.

# System response
Describe what the system response was, include the output from the command, automation, or else.

# Steps to reproduce

## Manual
Steps to reproduce the behaviour manually:
1. Go to '...'
2. Click on '....'
3. Scroll down to '....'
4. See error

## Automation
Link to automation and explanation on how to run it to reproduce the problem/bug

# Evidence

## Logs
If applicable, add logs and/or screenshots to help explain your problem.

## Additional context
Add any other context about the problem here including; system version numbers, components affected.

# Definition of Done
>ℹ️ Not every issue will need every item checked, however, every item on this list should be properly considered and actioned to meet the [DoD](https://github.com/vegaprotocol/vega/blob/develop/DEFINITION_OF_DONE.md).

**Before Merging**
- [ ] Code refactored to meet SOLID and other code design principles
- [ ] Code is compilation error, warning, and hint free
- [ ] Carry out a basic happy path end-to-end check of the new code
- [ ] All APIs are documented so auto-generated documentation is created
- [ ] All bug recreation steps can be followed without presenting the original error/bug
- [ ] All Unit, Integration and BVT tests are passing
- [ ] Implementation is peer reviewed (coding standards, meeting acceptance criteria, code/design quality)
- [ ] Create [front end](https://github.com/vegaprotocol/token-frontend/issues) or [console](https://github.com/vegaprotocol/console/issues) tickets with feature labels (should be done when starting the work if dependencies known i.e. API changes)

**After Merging**
- [ ] Move development ticket to `Done` if there is **NO** requirement for new [system-tests](https://github.com/vegaprotocol/system-tests/issues)
- [ ] Resolve any issues with broken system-tests
- [ ] Create [documentation](https://github.com/vegaprotocol/documentation/issues) tickets with feature labels if functionality has changed, or is a new feature
