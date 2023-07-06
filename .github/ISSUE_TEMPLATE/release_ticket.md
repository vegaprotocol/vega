---
name: Release ticket
about: A ticket to capture all the details of the release 
title: '[Release]: Version `X.Y.Z`'
labels: "release"
assignees: '@gordsport'
---

## Release Version `X.Y.Z` of core to Testnet

### Pre-release checklist

- [ ] Key features / fixes described and understood
- [ ] Any breaking changes, deprecations or removals in this release?
- [ ] Smart Contracts updated? (if applicable capture addresses for all networks)
- [ ] Documentation to support feature/api changes?
- [ ] Node operator docs updated?
- [ ] Automated tests pass without blocking incidents? (Front end dApps)
- [ ] Automated tests pass without blocking incidents? (Core QA nightly run AND QA load run)
- [ ] Perfromance tests run
- [ ] Manual tests pass without blocking incidents on Stagnet?
- [ ] Changelog's / GitHub release page updated?
- [ ] Release notes available?
- [ ] Migration / deployment guide?
- [ ] Front end applications versions ready
- [ ] Tag system tests repo
- [ ] Tag vegacapsule repo
- [ ] Tag desktop wallet repo (if required)
- [ ] Release tags captured:

| Deployable: | Core | Front Ends | Desktop Wallet | Capsule |  System Tests |
|:---------:|:--------:|:--------:|:--------:|:--------:|:--------:|
| Versions: | - |  - | - | - | - |


- [ ] Go / No-go (core, QA, community, FE, research)
- [ ] Date / time of deployment decided
- [ ] Community informed of deprecations, removals, 
- [ ] Community given info / timings of deployments
- [ ] Testnet slack channel updated (details pinned / TOPIC updated)
- [ ] Ensure that the [testnet version](https://github.com/vegaprotocol/vega.xyz/blob/main/src/pages/wallet/index.js#L142) download is specified for the [website](https://vega.xyz/wallet)
- [ ] DEPLOY TO FAIRGROUND TESTNET
- [ ] Update [hosted wallet version](https://github.com/vegaprotocol/k8s/blob/main/charts/apps/vegawallet/fairground/VERSION)

#### Post-release checklist

- [ ] Community informed (success or delays)
- [ ] Vega Testnet slack channel informed (success or delays)
- [ ] Engineering monitor network
- [ ] Blameless post mortem (should we face deployment / release version issues)

> NOTE: This list may be edited depending on the context of the release. We may take a calculated risk and not have some of these complete for a deployment (however a good justification is required!).
