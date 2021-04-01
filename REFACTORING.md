# Refactoring

This document is an attempt to get the team on the same page about the refactoring process we want to adopt.

## Why?
* Defined a goal to reach.
  * Make the code easier to deal with.
  * Ease the addition (or update) of features.

* Remove technical limitation
* Maintainability:  Clean code / SOLID philosophy
* Make it easier to understand
* Pony lang philosophy

## How?
* Defined how we can reach that goal.
  * Which techniques?
    * More domain-oriented and less "algorithmic". `order.IsUncrossed()` instead of `order.offet < -1`
    * avoid graph style code, more like tree structure.
    * SOLID
    * clean code
    * avoid coupling: Better interfaces, or event driven packages.
* Defined critical code path for maximum impact.
  * Identify the hottest path of code change :
    * compare results with cyclomatic complexity (gocyclo).
    * rate of changes of the code and files 
    * number of import inside a file -> reveal a code smell ?
* Defined medium to share the knowledge.
  * Create workshops on specific problems to find a solution, and be on the same page.
  * Save our decisions into a file.
  * Use external articles on the matter. file with "do this" and "don't do this".

## When?

Be discussed before being addressed.

Verify the test coverage (coverage is not everything) before refactor.

Beware of refactoring scope to not go too far, and cause conflict.

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

## Planing

1. find a tool to highlight hot code path
2. Bootstrap files for coding style, convention and stuff.
3. Find the biggest offenders and start from there to build our guideline.
4. golangci : linters to get more insight on smells.
