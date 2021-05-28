# Definition of Done (DoD)

## What is a Definition of Done (DoD)?

We use a Definition of Done to clarify what exactly the team means when we declare something ‘done’.

In more concrete terms, it lists the criteria we’ll meet before you claim a feature, enhancement or bug fix can be shipped (released).

The difference between a Definition of Done and Acceptance Criteria...

- The Definition of Done applies to all your work. Acceptance Criteria only ever apply to a single item on your backlog
- The Definition of Done clarifies what you do as a team. Acceptance Criteria clarify what the product does — its functionality
- The Definition of Done can contain criteria for “non-functional” aspects and cross-cutting concerns. Acceptance Criteria talk of functional aspects of a single item of work. You’ll only rarely find non-functional aspects in them. And when you do, they just list the exceptions to the general non-functional requirements, making these more strict or lenient for that work item

## What are the benefits of creating and using a good Definition of Done?

**Transparency of responsibilities** - You know what you need to do and understand what you do not need to do. For example, you know it’s your job to implement correct behaviour, but creating training videos is not. You’ll become more predictable in what you deliver.

**Realistic Sprint Commitment** - With an itemized list of exactly what you need to do to get to done, you’re better able to assess how much you can realistically take on in a Sprint. You’ll become more reliable.

**Reduced risk of rework** - Checklists work. Aviation wouldn’t be as safe without them. When you reduce the work left undone when you declare something done, it means less (risk of) rework later. You’ll become more efficient.

**Higher quality, less effort everywhere** - As you mature in your development practices, you’ll raise the bar on the quality you deliver and reduce other staff’s workload. For example, fewer escaped bugs mean lower demands on support staff. You’ll deliver better quality and help your company become more efficient.

**Predictability and Sustainable Pace** - Fewer escaped defects means you can create value with the whole team instead of having to divert some of you to resolve bugs. You’ll become more productive and predictable, and get to work at a constant sustainable pace.

All this helps raise the confidence a team feels in delivering valuable software and grow the trust other teams and stakeholders will place in a team.


## Vega Core Team Definition of Done

**The Very Basic Basics**

- Implementation peer reviewed or pair-programmed
- All Acceptance Criteria confirmed to be met by Product Owner
- Unit tests written, running and passing
- No known defects
- Code integrated (merged)
- User Guide (customer documentation) updated or input provided to others?

**Code Quality**

- Compilation error, warning, and hint free
- Coding standards followed
- Static analysis metrics at or above target
- Dependencies minimized
- Code refactored to meet SOLID and other code design principles

**Ready for release**

- Integrated build error, warning, and hint free?
- BVT tests passing?
- Package creation error, warning, and hint free?
- Deployed to the test environment

**Functional Regression Tests**

- Every item in Acceptance Criteria covered by at least one acceptance test?
- Unit tests passing?
- Integration tests passing
- Functional or system tests passing?
- Acceptance (feature) tests passing?
- Tests executed on all supported platforms?

**For all automated tests:**

- Coverage on or above the norm? (For example, 85%.)
- All new units / integrations / functionality covered by tests?
