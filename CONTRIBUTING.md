# Contributing

Thank you for your interest in contributing to the Vega Protocol. There are always bugs to find, bugs to fix, improvements to documentation, and more work to be done.

### Code of conduct

Please read the [Code of Conduct](./CODE_OF_CONDUCT.md); it is expected all project participants to adhere to it.

### Open development

The project development follows the Vega Protocol [engineering roadmap](https://github.com/vegaprotocol/roadmap). All engineering work happens directly in GitHub. Bugs committed to be resolved are triaged, assigned to a developer and are in the current, or next, iteration. This can be seen on the [team project board](https://github.com/orgs/vegaprotocol/projects/106/views/4). 

Other bugs are fair game, however, if you are not familiar with the project get up-to-speed with the: 

- [White papers](https://vega.xyz/papers/)
- [Protocol specifications](https://github.com/vegaprotocol/specs)
- [Documentation](https://docs.vega.xyz/)

Also ask in [Discord](https://discord.com/invite/3hQyGgZ) or [Discussions](https://github.com/vegaprotocol/feedback) before working on a particular bug.

Once up-to-speed with the protocol, fork the repositories (as per the [git workflow](./CONTRIBUTING.md#git-workflow)) and get set up using the [getting started](./GETTING_STARTED.md) information.

### Contributing to the project

Create an issue for the feature, bugfix or enhancement and assign to yourself. Start a discussion, pull in relevant people and build a technical design.

If possible, issues should be linked back to a feature in the protocol specs repo.

#### Git workflow

- Fork the repositories required
- Perform a `git clone` to get a copy on your local machine
- When ready publish a local commit to your own public repository
- When the change has been sufficiently self-tested file a pull request with the main repository
- Ask for review on Discord, discuss suggestions and get the branch merged.

This flow allows the project team to know that an update is ready to be integrated. **Pairing and early review is highly encouraged.**

> Policy: If work is being started on a new engine, have a short workshop so that questions (on technical implementation) can be asked. The output of the workshop is a README, to go in the engine subdirectory, containing the questions and answers that came up during the workshop.

#### Adding tests

The aim for overall test coverage to go *up*. 

All software development work should be tested locally and at a bare minimum unit tests and basic end-to-end tests written, more information can be found in the team [definition of done](./DEFINITION_OF_DONE.md).

> Policy: Write tests that cover edge/corner cases, and/or document assumptions, so that if an assumption is later invalidated, a test fails.

#### Before merge

Before code changes are accepted the project team will conduct a full code review.

> Policy: For contributors outside of the project team before any pull requests can be accepted you will need to sign a Contributor Licence Agreement (CLA) or something similar.