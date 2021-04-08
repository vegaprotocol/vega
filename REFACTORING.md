# Refactoring

This document is an attempt to get the team on the same page about the refactoring process we want to adopt.

## Why?
* Defined a goal to reach.
  * Make the code easier to deal with.
  * Ease the addition (or update) of features.

## How?
* Defined how we can reach that goal.
  * Which techniques? More domain-oriented and less "algorithmic".
* Defined critical code path for maximum impact.
  * Identify the hottest path of code change.
* Defined medium to share the knowledge.
  * Create workshops on specific problems to find a solution, and be on the same page.
  * Save our decisions into a file.

## When?
### Opportunistic refactoring
It's done along the way.

#### Preparatory refactoring
Refactor just before we add a feature to a code base, to ease its addition.

**Questions:**

* What kind of code makes feature harder to add?

#### Comprehension refactoring
Refactor to make the code more understandable.

**Questions:**

* What's a code that is easier to understand?

#### Clean-up refactoring
Basically, the boy scout's rule.

> Always leave the camp site cleaner than when you found it.

**Questions:**

* How to avoid overlapping with someone else?
* Into the same PR as the one of the feature or a different one?
