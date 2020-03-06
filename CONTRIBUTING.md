# Contributing

Create an Issue for the feature, bugfix or enhancement. Start a discussion, pull
in relevant people, build a technical design.

If possible, Issues should be linked back to a feature in the Product repo.

Make a branch off `develop`, perhaps create a WIP MR to go with it. Work on the
Issue, push code to `origin` every now and again, continue discussion either on
the Issue or in the WIP MR.

> Policy: If work is being started on a new engine, have a short workshop so
> that questions (on technical implementation) can be asked. The output of the
> workshop is a README, to go in the engine subdirectory, containing the
> questions and answers that came up during the workshop.

Pairing and early review is encouraged.

Add tests! Aim for overall test coverage to go *up*. At a bare minimum, the
main/longest code path of each function should be tested.

> Policy: Write tests that cover edge/corner cases, and/or document assumptions,
> so that if an assumption is later invalidated, a test fails.

When ready, create a merge request (or remove the WIP from the existing one), ask
for review on Slack, discuss suggestions, get the branch merged.
