# Release processes


This document outlines the steps required in order to create a core protocol release.

Please be aware of the [version numbering pattern](#version-numbering-pattern) that must be used.


## Major/minor release process

1. The default branch is: `develop`.
1. Create a `release/vX.Y.Z` branch off the head of **`develop`**.
1. Update all readme files, changelog, and set version strings as required:
    - remove `Unreleased` from the changelog for the version to be released
    - ensure that the readme is up-to-date for the version being released
    - update the version number in `version/version.go`
    - update the version number in `protos/sources/vega/api/v1/core.proto`
    - update the version number in `protos/sources/vega/api/v1/corestate.proto`
    - update the version number in `protos/sources/datanode/api/v2/trading_data.proto`
    - update the version number in `protos/sources/blockexplorer/api/v1/blockexplorer.proto`
    - run `make proto`
    - run `git commit -asm "chore: release version vX.Y.Z`
1. Create a pull request to merge `release/vX.Y.Z` into `master` and have it reviewed and merged
1. Once the pull request has been merged, create a tag on the `master` branch
1. The CI will see the tag and create all the release artifacts
1. Follow the [common release process](./#common-release-process) steps


## Patch release process

1. Get the patch fix pull requests merged into `develop`
1. Create a `release/vX.Y.Z` branch off **`master`** or previous release branch
1. Cherry pick the fixes into the `release/vX.Y.Z` branch
    - use the merge commit hash of a PR for the cherry picks
    - run `git cherry-pick -m 1 <merge commit hash>`
1. Update all readme files, changelog, and set version strings as required:
    - ensure the changelog is correct for the patch version to be released
    - ensure that the readme is up-to-date for the patch version being released
    - update the version number in `version/version.go`
    - update the version number in `protos/sources/vega/api/v1/core.proto`
    - update the version number in `protos/sources/vega/api/v1/corestate.proto`
    - update the version number in `protos/sources/datanode/api/v2/trading_data.proto`
    - update the version number in `protos/sources/blockexplorer/api/v1/blockexplorer.proto`
    - run `make proto`
    - run `git commit -asm "chore: release version vX.Y.Z`
1. Create a tag on the patch `release/vX.Y.Z` branch
1. The CI will see the tag and create all the release artifacts
1. Follow the [common release process](./#common-release-process) steps


## Common release process

Once the above steps have been taken for the required type of release, the following steps for all releases need to be taken:

1. Cut and paste the following instructions
   ```bash
   git fetch --all
   git checkout master
   git pull --rebase origin master

   # Create a message for the tag.
   # NOTE: Do not use markdown headings with '#'. Lines beginning with '#' are
   #       ignored by git.
   cat >/tmp/tagcommitmsg.txt <<-EOF
   Release vX.Y.Z <-- insert v$NEWVER

   *20YY-MM-DD*

   Security vulnerabilities: <-- no hashes here
   - #123 Fix a vulnerability

   Breaking changes: <-- no hashes here
   - #124 Rename a thing

   Deprecation: <-- no hashes here
   - #125 Deprecate a thing

   Improvements: <-- no hashes here
   - #126 Add a thing

   Fixes: <-- no hashes here
   - #126 Fix a bug
   EOF
   git tag "v$NEWVER" -F /tmp/tagcommitmsg.txt "$(git log -n1 | awk '/^commit / {print $2}')"
   git show "v$NEWVER" # Check the tag message
   git push --tags origin master
   # Wait for the pipeline for the tag to finish, to reduce resource contention.
   git checkout develop
   git pull --rebase origin develop
   git merge master
   git push origin develop
   ```
1. The GitHub release notes can be auto-generated when creating/editing the release in the GitHub UI using the `Generate release notes` button:
    - the [`release.yml`](https://github.com/vegaprotocol/vega/blob/develop/.github/release.yml) details the headings and associated labels used to generate the release notes
    - check that all pull requests in the release have the correct labels paying special attention to those related to the labels `vulnerability`, `breaking-change` and `deprecation`
    - when all pull requests have been checked run the `Generate release notes` action
1. Notify devops that the release version needs to be deployed onto the `stagnet1` environment for verification
1. Notify the `@release` group on Slack in the `#engineering` channel


## Version numbering pattern

To ensure no confusion between testnet (pre-release) versions and the versions deemed ready for the validators (latest), the following pattern will be used:

- Mainnet ready "latest" versions: `X.Y.Z`
- Staging / Testnet release candidate versions: `X.Y.Z-preview.n`

Where `n` increments with each pre-release candidate version, and therefore:

- `0.71.6-preview.2` is greater than `0.71.6-preview.1`
- `0.71.6` is greater than `0.71.6-preview.2`
- `0.72.0` is greater than `0.71.6`
