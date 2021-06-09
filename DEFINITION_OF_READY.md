# Definition of Ready (DoR)

## What is a Definition of Ready (DoR)?

We use a Definition of Ready to clarify what the team means when we declare something ‘ready’.

A 'ready' item should be clear, feasible and testable; the goal of the DoR is to prevent problems before they have a chance to manifest. The Definition of Ready should not be used as a stage gate or to enforce big design up front but should be a set of **guiding principles** to ensure that the team can start work on a feature or work item.

The INVEST criteria gives a useful framework to pick apart work (usually teasing more vertical slices or firming up the current slice):

**Independent** - Are there dependencies? Is anything stopping us from starting work on this?

**Negotiable** - The spec/issue is not an explicit contract, leave space for discussion. Do we need all of it? What options are there for satisfying the stated needs?

**Valuable** - Is this valuable? What is impact of not doing it?

**Estimable** - Could we estimate it if we wanted to? If not, what's stopping us?

**Suitably sized** - Will this fit inside a milestone (~ 3 months) or team sprint (<= 2 weeks)?

**Testable** - Can we test/verify this? How would we test/verify it?


## Vega Core Team Definition of Ready

**Specs**

- A clear overview that describes what this feature **needs** to do and **why** it's valuable.
- Examples of how this should work (that can help define acceptance criteria and/or test scenarios).
- Known dependencies clearly defined.
- Assumptions clearly defined.
- Acceptance criteria (at least considered in that there is enough information these could be written).
- Feature Test Scenarios (at least considered in that there is enough information these could be written).
- Details of which [milestone](https://github.com/vegaprotocol/specs-internal/tree/master/milestones) the spec relates to captured in the milestone index.  
- A question and answer workshop with the spec owner and the team (at least offered).

**Work Items (Issues)**

- An overview that describes what work item **needs** to do and **why** it's valuable.
- Bug re-creation steps (for bugs).
- Acceptance criteria defined.
- Feature Test Scenarios defined (for features).
- Assumptions clearly defined.
- Known dependencies clearly defined.
- Labelled where known and applicable (bug severity, tech debt).
- Discussed within the team.



