# Integration Tests

This is the home of the system integrations tests.

## Running the tests

They can be run from the root of vega with:

```shell
make integrationtest
```

or

```shell
go test ./...
``` 

### Running just the integration tests

The integration tests have been hooked up to run as regular unit tests, so you can run just the integration tests with a simple command:

```shell
go test ./integration/...
```

When running these tests, you'll probably want to get a more verbose output (showing which steps of the tests passed and failed), which can be done by adding 2 flags:

```
go test -v ./integration/... -godog.format=pretty
```

The `-v` flag tells `go test` to run with verbose output (sending logging to stdout). The `-godog.format=pretty` flag (which must be put at the end) instructs godog to print out the scenario's and, in case an assertion fails, show which particular step of a given scenario didn't work.

### Running specific scenario's

To run only certain tests (feature files), you can simply add the paths to a given feature file to the command:

```shell
go test -v ./integration/... -godog.format=pretty $(pwd)/integration/features/my-feature.feature
```

### Race detection and cache

For performance reasons, `go test` will check whether the source of a package has changed, and reuse compiled objects or
even test results in case it determines nothing has changed. Because the integration tests are tucked away in their own
package, and likely won't have changed, changes to _other_ packages might not be compiled, and tests could possibly pass
without changes being applied. To ensure no cached results are used, the `-count` flag can be used:

```shell
go test -v -count=1 ./integration/... -godog.format=pretty
```

Should there be tests that are intermittently failing, this could indicate a data race somewhere in the code. To use the race detector to check for this, you can add the `-race` flag to the command. The full commands then would be:

```shell
# Run all integration tests, verbose mode, ensure recompiled binaries, enable race detection, and use godog pretty formatting
go test -v -count=1 -race ./integration/... -godog.format=pretty

# Same as above, but only run a specific feature file:
go test -v -count=1 -race ./integration/... -godog.format=pretty $(pwd)/integration/feature/my-feature.feature
```

Race detection is a complex thing to do, so it will make running tests significantly slower. The pipeline runs the tests with race detection, so this shouldn't be required to do locally.

### Reproducing/replicating system tests

The system tests run on a higher level. They submit a new market proposal, get said market accepted through governance,
and then start trading. They use a `LogNormal` risk model, and specific fee parameters. David kindly provided the
long/short risk factors for a simple risk model that result in the same margin requirements and same fees being applied
to the trades. To create an integration test that replicates the system test results (transfers, balances, fees, etc...)
, simply start your feature file with the following:

```gherkin
Feature: A feature that reproduces some system test

  Background:
    Given the insurance pool initial balance for the markets is "0":
    And the markets:
      | id        | quote name | asset | mark price | risk model                | margin calculator         | auction duration | maker fee | infrastructure fee | liquidity fee | price monitoring | oracle config      |
      | ETH/DEC20 | ETH        | ETH   | 100        | default-simple-risk-model | default-margin-calculator | 1                | 0.004     | 0.001              | 0.3           | default-none     | default-for-future |
```

## Convention

### Glossary

We should move toward building our ubiquitous language and use domain language and avoid the use of synonyms.

If we talk about `submitting an order`, we avoid using `placing an order.`

### Structuring a feature test

#### File

A feature test's file should be named by the feature / command it's testing, such
has: `maintenance_call_for_margin_account.go`.

The file name should match, or at least be close to, the description of the `Feature` keyword.

To be avoided:

* A prefix with a pull-request or an issue number, such as `4284-cancel-order.go`.
* A vague name, or context, such as `orders.go` or `cancellation.go`

#### Feature

The `Feature` keyword should describe the feature to be tested in proper sentences, with context.

It should match, or at least be close to, the name of the file it lives in.

##### Examples

Let's assume, we have a file called `trader_cancels_orders.go`

###### Good

```gherkin
Feature: Traders can cancel his orders under certain conditions
```

By reading this, we get to know who's the main actor, the action and the target. Saying _"Under certain conditions"_ is
vague, but it's enough as that's the purpose of the `Scenario` to be more specific. At least, we know there are
conditions.

###### Bad

```gherkin
Feature: cancel orders
```

This is too vague.

```gherkin
Feature: Should monitor prices
```

This seems completely unrelated to what the file name mentions.

#### Scenario

The `Scenario` keyword should describe the tested case of the command, and, thus, should never be empty.

A file can contain multiple scenarios if they test the same feature. Unrelated tests should live in a dedicated file.

If the feature to be tested is order cancellation, we could have:

##### Examples

###### Good

```gherkin
Feature: Trader can cancel orders under certain condition

  Scenario: Trader can cancel his order if not matched
  ...

  Scenario: Trader cannot cancel another trader's order
  ...
```

We know who is doing what on what.

###### Bad

```gherkin
Scenario: Works
...

Scenario: fail !
...
```

Oh yeah ?

```gherkin
Scenario:
...
```

Okay...


#### Given

`Given` should only be used for prerequisite declaration. Arguably, it's a bit tricky to distinguish a prerequisite from
what's not. For now, as a rule of thumb, we consider market declaration and traders initial deposit to the general
account to be the pre-requisites. Other steps should use the keywords below.

##### Examples

###### Good

```gherkin
Given the market ...
And the traders general account balance ...
```

#### When

`When` should only be used when issuing a command. It shouldn't be used for assertions. The preferred construct of
command steps is:

```
<actor> <action verb> <target>
```

Construction with passive voice is accepted, if it makes more sense than the active voice.

##### Examples

###### Good

```gherkin
When traders submit the following orders
```

We know who does what.

```gherkin
When an oracle data is submitted
```

The passive voice sounds better `The system receives the following oracle data`.

#### Then

`Then` should only be used when asserting a state. It shouldn't be use for commands. The preferred construct
of assertion steps is:

```
<actor> should <state verb> <target>
```

##### Examples

###### Good

```gherkin
Then trader trader-1 should have a balance of 100 ETH
```

We know what we expect from whom.

###### Bad

```gherkin
Then trader trader-1 have a balance of 100 ETH
```

We miss the `should` that emphasize the expectation.

```gherkin
Then the orders should fails
```

This is too vague.

```gherkin
Then the trader places an order
```

It's a command. The keywords should be used to help the reader to distinguish a command from an assertion. Even if the
above sentence makes sense, it breaks the structure `When command > Then assertion`, and we might end up with a list
of `Then`:

```gherkin
Then the trader places an order
Then the trader should have balance ...
Then an oracle data is sent
Then the settlement price should be updated
```

We are no longer be able to sort out the commands from the assertions at first glance.

#### And / But

`And` can be used by any of the previous keywords and should follow the sentence construction of the keyword it is
backing. Use `But` for negative outcomes.

### Step

#### Text

* The first word should start we a lower-case letter.
* Words (and table columns) should be lower-case with space separation, like plain human style. No upper-case location
  to be remembered.
* Acronyms should be lower-case, like the rest, without trailing dot. We want to avoid interrogation such as : `ID` or `Id` or `Id.` ?

###### Good

```gherkin
When the market id should contain the asset "..."
```

All lower-case.

###### Bad

```gherkin
Then The Market Id should appears in U.R.L with QuoteName
```

### Single-line step

#### Error

We should verify the error message on every expected failure using `because` followed by the error message.

###### Good

```gherkin
Then the order "1234" should be rejected because "....."
```

We ensure the error is the expected one, and the context is clear, no need for additional comments.

###### Bad

```gherkin
Then the order "1234" should be rejected
```

It may have not failed for the reason we expected. And, we may be tempted to add a comment to explain the reason of the
failure.

### Table step

#### Error

The column to verify the error should always be named `error`.

#### Date

Prefer `expiration date` over `expires at` or `started at`.
