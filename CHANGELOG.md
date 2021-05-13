# Changelog

## 0.36.0
*2021-05-13*

### Improvements
- [#3408](https://github.com/vegaprotocol/vega/pull/3408) - Add more information on token proportion/weight on proposal votes APIs
- [#3360](https://github.com/vegaprotocol/vega/pull/3360) - :fire: REST: Move deposits endpoint to `/parties/[partyId]/deposits`
- [#3431](https://github.com/vegaprotocol/vega/pull/3431) - Improve caching of values when exiting auctions
- [#3459](https://github.com/vegaprotocol/vega/pull/3459) - Add extra validation for Order, Vote, Withdrawal and LP transactions
- [#3433](https://github.com/vegaprotocol/vega/pull/3433) - Reject non-persistent orders that fall outside price monitoring bounds
- [#3443](https://github.com/vegaprotocol/vega/pull/3443) - Party is no longer required when submitting an order amendment
- [#3446](https://github.com/vegaprotocol/vega/pull/3443) - Party is no longer required when submitting an order cancellation
- [#3449](https://github.com/vegaprotocol/vega/pull/3449) - Party is no longer required when submitting an withdrawal request

### Fixes
- [#3451](https://github.com/vegaprotocol/vega/pull/3451) - Remove float usage in liquidity engine
- [#3447](https://github.com/vegaprotocol/vega/pull/3447) - Clean up order submission code
- [#3436](https://github.com/vegaprotocol/vega/pull/3436) - Break up internal proposal definitions
- [#3452](https://github.com/vegaprotocol/vega/pull/3452) - Tidy up LP implementation internally
- [#3458](https://github.com/vegaprotocol/vega/pull/3458) - Fix spelling errors in GraphQL docs
- [#3434](https://github.com/vegaprotocol/vega/pull/3434) - Improve test coverage around Liquidity Provisions on auction close
- [#3411](https://github.com/vegaprotocol/vega/pull/3411) - Fix settlement tests
- [#3418](https://github.com/vegaprotocol/vega/pull/3418) - Rename External Resource Checker to Witness
- [#3419](https://github.com/vegaprotocol/vega/pull/3419) - Fix blank IDs on oracle specs in genesis markets
- [#3412](https://github.com/vegaprotocol/vega/pull/3412) - Refactor internal Vote Submission type to be separate from Vote type
- [#3421](https://github.com/vegaprotocol/vega/pull/3421) - Improve test coverage around order uncrossing
- [#3425](https://github.com/vegaprotocol/vega/pull/3425) - Remove debug steps from feature tests
- [#3430](https://github.com/vegaprotocol/vega/pull/3430) - Remove `LiquidityPoolBalance` from configuration
- [#3468](https://github.com/vegaprotocol/vega/pull/3468) - Increase rate limit that was causing mempools to fill up unnecessarily
- [#3438](https://github.com/vegaprotocol/vega/pull/3438) - Split protobuf definitions
- [#3450](https://github.com/vegaprotocol/vega/pull/3450) - Do not emit amendments from liquidity engine

## 0.35.0
*2021-04-21*

### Improvements
- [#3341](https://github.com/vegaprotocol/vega/pull/3341) - Add logging for transactions rejected for having no accounts
- [#3339](https://github.com/vegaprotocol/vega/pull/3339) - Reimplement amending LPs not to be cancel and replace
- [#3371](https://github.com/vegaprotocol/vega/pull/3371) - Optimise calculation of cumulative price levels
- [#3339](https://github.com/vegaprotocol/vega/pull/3339) - Reuse LP orders IDs when they are re-created
- [#3385](https://github.com/vegaprotocol/vega/pull/3385) - Track the time spent in auction via Prometheus metrics
- [#3376](https://github.com/vegaprotocol/vega/pull/3376) - Implement a simple benchmarking framework for the core trading
- [#3371](https://github.com/vegaprotocol/vega/pull/3371) - Optimize indicative price and volume calculation

### Fixes
- [#3356](https://github.com/vegaprotocol/vega/pull/3356) - Auctions are extended if exiting auction would leave either side of the book empty
- [#3348](https://github.com/vegaprotocol/vega/pull/3348) - Correctly set time when liquidity engine is created
- [#3321](https://github.com/vegaprotocol/vega/pull/3321) - Fix bond account use on LP submission
- [#3369](https://github.com/vegaprotocol/vega/pull/3369) - Reimplement amending LPs not to be cancel and replace
- [#3358](https://github.com/vegaprotocol/vega/pull/3358) - Improve event bus stability
- [#3363](https://github.com/vegaprotocol/vega/pull/3363) - Fix behaviour when leaving auctions
- [#3321](https://github.com/vegaprotocol/vega/pull/3321) - Do not slash bond accounts on LP submission
- [#3350](https://github.com/vegaprotocol/vega/pull/3350) - Fix equity like share in the market data
- [#3363](https://github.com/vegaprotocol/vega/pull/3363) - Ensure leaving an auction cannot trigger another auction / auction leave
- [#3369](https://github.com/vegaprotocol/vega/pull/3369) - Fix LP order deployments
- [#3366](https://github.com/vegaprotocol/vega/pull/3366) - Set the fee paid in uncrossing auction trades
- [#3364](https://github.com/vegaprotocol/vega/pull/3364) - Improve / fix positions tracking
- [#3358](https://github.com/vegaprotocol/vega/pull/3358) - Fix event bus by deep cloning all messages
- [#3374](https://github.com/vegaprotocol/vega/pull/3374) - Check trades in integration tests

## 0.34.1

*2021-04-08*

### Fixes
- [#3324](https://github.com/vegaprotocol/vega/pull/3324) - CI: Fix multi-architecture build

## 0.34.0

*2021-04-07*

### Improvements

- [#3302](https://github.com/vegaprotocol/vega/pull/3302) - Add reference to LP in orders created by LP
- [#3183](https://github.com/vegaprotocol/vega/pull/3183) - All orders from LP - including rejected orders - are now sent through the event bus
- [#3248](https://github.com/vegaprotocol/vega/pull/3248) - Store and propagate bond penalty
- [#3266](https://github.com/vegaprotocol/vega/pull/3266) - Add network parameters to control auction duration & extension
- [#3264](https://github.com/vegaprotocol/vega/pull/3264) - Add Liquidity Provision ID to orders created by LP commitments
- [#3126](https://github.com/vegaprotocol/vega/pull/3126) - Add transfer for bond slashing
- [#3281](https://github.com/vegaprotocol/vega/pull/3281) - Update scripts to go 1.16.2
- [#3280](https://github.com/vegaprotocol/vega/pull/3280) - Update to go 1.16.2
- [#3235](https://github.com/vegaprotocol/vega/pull/3235) - Extend unit test coverage for products
- [#3219](https://github.com/vegaprotocol/vega/pull/3219) - Remove `liquidityFee` network parameter
- [#3217](https://github.com/vegaprotocol/vega/pull/3217) - Add an event bus event when a market closes
- [#3214](https://github.com/vegaprotocol/vega/pull/3214) - Add arbitrary data signing wallet endpoint
- [#3316](https://github.com/vegaprotocol/vega/pull/3316) - Add tests for traders closing their own position
- [#3270](https://github.com/vegaprotocol/vega/pull/3270) - _Feature test refactor_: Add Liquidity Provision feature tests
- [#3289](https://github.com/vegaprotocol/vega/pull/3289) - _Feature test refactor_: Remove unused steps
- [#3275](https://github.com/vegaprotocol/vega/pull/3275) - _Feature test refactor_: Refactor order cancellation steps
- [#3230](https://github.com/vegaprotocol/vega/pull/3230) - _Feature test refactor_: Refactor trader amends step
- [#3226](https://github.com/vegaprotocol/vega/pull/3226) - _Feature test refactor_: Refactor features with invalid order specs
- [#3200](https://github.com/vegaprotocol/vega/pull/3200) - _Feature test refactor_: Add step to end opening auction
- [#3201](https://github.com/vegaprotocol/vega/pull/3201) - _Feature test refactor_: Add step to amend order by reference
- [#3204](https://github.com/vegaprotocol/vega/pull/3204) - _Feature test refactor_: Add step to place pegged orders
- [#3207](https://github.com/vegaprotocol/vega/pull/3207) - _Feature test refactor_: Add step to create Liquidity Provision
- [#3212](https://github.com/vegaprotocol/vega/pull/3212) - _Feature test refactor_: Remove unused settlement price step
- [#3203](https://github.com/vegaprotocol/vega/pull/3203) - _Feature test refactor_: Rework Submit Order step
- [#3251](https://github.com/vegaprotocol/vega/pull/3251) - _Feature test refactor_:  Split market declaration
- [#3314](https://github.com/vegaprotocol/vega/pull/3314) - _Feature test refactor_:  Apply naming convention to assertions
- [#3295](https://github.com/vegaprotocol/vega/pull/3295) - Refactor governance engine tests
- [#3298](https://github.com/vegaprotocol/vega/pull/3298) - Add order book caching
- [#3307](https://github.com/vegaprotocol/vega/pull/3307) - Use `UpdateNetworkParams` to validate network parameter updates
- [#3308](https://github.com/vegaprotocol/vega/pull/3308) - Add probability of trading

### Fixes
- [#3249](https://github.com/vegaprotocol/vega/pull/3249) - GraphQL: `LiquidityProvision` is no longer missing from the `EventBus` union
- [#3253](https://github.com/vegaprotocol/vega/pull/3253) - Verify all properties on oracle specs
- [#3224](https://github.com/vegaprotocol/vega/pull/3224) - Check for wash trades when FOK orders uncross
- [#3257](https://github.com/vegaprotocol/vega/pull/3257) - Order Status is now only `Active` when it is submitted to the book
- [#3285](https://github.com/vegaprotocol/vega/pull/3285) - LP provisions are now properly stopped when a market is rejected
- [#3290](https://github.com/vegaprotocol/vega/pull/3290) - Update Market Value Proxy at the end of each block
- [#3267](https://github.com/vegaprotocol/vega/pull/3267) - Ensure Liquidity Auctions are not left if it would result in an empty book
- [#3286](https://github.com/vegaprotocol/vega/pull/3286) - Reduce some log levels
- [#3263](https://github.com/vegaprotocol/vega/pull/3263) - Fix incorrect context object in Liquidity Provisions
- [#3283](https://github.com/vegaprotocol/vega/pull/3283) - Remove debug code
- [#3198](https://github.com/vegaprotocol/vega/pull/3198) - chore: Add spell checking to build pipeline
- [#3303](https://github.com/vegaprotocol/vega/pull/3303) - Reduce market depth updates when nothing changes
- [#3310](https://github.com/vegaprotocol/vega/pull/3310) - Fees are no longer paid to inactive LPs
- [#3305](https://github.com/vegaprotocol/vega/pull/3305) - Fix validation of governance proposal terms
- [#3311](https://github.com/vegaprotocol/vega/pull/3311) - `targetStake` is now an unsigned integer
- [#3313](https://github.com/vegaprotocol/vega/pull/3313) - Fix invalid account wrapping

## 0.33.0

*2021-02-16*

As per the previous release notes, this release brings a lot of fixes, most of which aren't exciting new features but improve either the code quality or the developer experience. This release is pretty hefty, as the last few updates have been patch releases. It represents a lot of heavy testing and bug fixing on Liquidity Commitment orders. Alongside that, the feature test suite (we use [godog](https://github.com/cucumber/godog)) has seen some serious attention so that we can specify more complex scenarios easily.

### Improvements
- [#3094](https://github.com/vegaprotocol/vega/pull/3094) - :fire: GraphQL: Use `ID` scalar for IDs, ensure capitalisation is correct (`marketID` -> `marketId`)
- [#3093](https://github.com/vegaprotocol/vega/pull/3093) - :fire: GraphQL: Add LP Commitment field to market proposal
- [#3061](https://github.com/vegaprotocol/vega/pull/3061) - GraphQL: Add market proposal to markets created via governance
- [#3060](https://github.com/vegaprotocol/vega/pull/3060) - Add maximum LP shape size limit network parameter
- [#3089](https://github.com/vegaprotocol/vega/pull/3089) - Add `OracleSpec` to market
- [#3148](https://github.com/vegaprotocol/vega/pull/3148) - Add GraphQL endpoints for oracle spec
- [#3179](https://github.com/vegaprotocol/vega/pull/3179) - Add metrics logging for LPs
- [#3127](https://github.com/vegaprotocol/vega/pull/3127) - Add validation for Oracle Specs on market proposals
- [#3129](https://github.com/vegaprotocol/vega/pull/3129) - Update transfers to use `uint256`
- [#3091](https://github.com/vegaprotocol/vega/pull/3091) - Refactor: Standardise how `InAuction` is detected in the core
- [#3133](https://github.com/vegaprotocol/vega/pull/3133) - Remove `log.error` when TX rate limit is hit
- [#3140](https://github.com/vegaprotocol/vega/pull/3140) - Remove `log.error` when cancel all orders fails
- [#3072](https://github.com/vegaprotocol/vega/pull/3072) - Re-enable disabled static analysis
- [#3068](https://github.com/vegaprotocol/vega/pull/3068) - Add `dlv` to docker container
- [#3067](https://github.com/vegaprotocol/vega/pull/3067) - Add more LP unit tests
- [#3066](https://github.com/vegaprotocol/vega/pull/3066) - Remove `devnet` specific wallet initialisation
- [#3041](https://github.com/vegaprotocol/vega/pull/3041) - Remove obsolete `InitialMarkPrice` network parameter
- [#3035](https://github.com/vegaprotocol/vega/pull/3035) - Documentation fixed for infrastructure fee field
- [#3034](https://github.com/vegaprotocol/vega/pull/3034) - Add `buf` to get tools script
- [#3032](https://github.com/vegaprotocol/vega/pull/3032) - Move documentation generation to [`vegaprotocol/api`](https://github.com/vegaprotocol/api) repository
- [#3030](https://github.com/vegaprotocol/vega/pull/3030) - Add more debug logging in execution engine
- [#3114](https://github.com/vegaprotocol/vega/pull/3114) - _Feature test refactor_: Standardise market definitions
- [#3122](https://github.com/vegaprotocol/vega/pull/3122) - _Feature test refactor_: Remove unused trading modes
- [#3124](https://github.com/vegaprotocol/vega/pull/3124) - _Feature test refactor_: Move submit order step to separate package
- [#3141](https://github.com/vegaprotocol/vega/pull/3141) - _Feature test refactor_: Move oracle data step to separate package
- [#3142](https://github.com/vegaprotocol/vega/pull/3142) - _Feature test refactor_: Move market steps to separate package
- [#3143](https://github.com/vegaprotocol/vega/pull/3143) - _Feature test refactor_: Move confirmed trades step to separate package
- [#3144](https://github.com/vegaprotocol/vega/pull/3144) - _Feature test refactor_: Move cancelled trades step to separate package
- [#3145](https://github.com/vegaprotocol/vega/pull/3145) - _Feature test refactor_: Move traders step to separate package
- [#3146](https://github.com/vegaprotocol/vega/pull/3146) - _Feature test refactor_: Create new step to verify margin accounts for a market
- [#3153](https://github.com/vegaprotocol/vega/pull/3153) - _Feature test refactor_: Create step to verify one account of each type per asset
- [#3152](https://github.com/vegaprotocol/vega/pull/3152) - _Feature test refactor_: Create step to deposit collateral
- [#3151](https://github.com/vegaprotocol/vega/pull/3151) - _Feature test refactor_: Create step to withdraw collateral
- [#3149](https://github.com/vegaprotocol/vega/pull/3149) - _Feature test refactor_: Merge deposit & verification steps
- [#3154](https://github.com/vegaprotocol/vega/pull/3154) - _Feature test refactor_: Create step to verify settlement balance for market
- [#3156](https://github.com/vegaprotocol/vega/pull/3156) - _Feature test refactor_: Rewrite margin levels step
- [#3178](https://github.com/vegaprotocol/vega/pull/3178) - _Feature test refactor_: Unify error handling steps
- [#3157](https://github.com/vegaprotocol/vega/pull/3157) - _Feature test refactor_: Various small fixes
- [#3101](https://github.com/vegaprotocol/vega/pull/3101) - _Feature test refactor_: Remove outdated feature tests
- [#3092](https://github.com/vegaprotocol/vega/pull/3092) - _Feature test refactor_: Add steps to test handling of LPs during auction
- [#3071](https://github.com/vegaprotocol/vega/pull/3071) - _Feature test refactor_: Fix typo

### Fixes
- [#3018](https://github.com/vegaprotocol/vega/pull/3018) - Fix crash caused by distressed traders with LPs
- [#3029](https://github.com/vegaprotocol/vega/pull/3029) - API: LP orders were missing their reference data
- [#3031](https://github.com/vegaprotocol/vega/pull/3031) - Parties with cancelled LPs no longer receive fees
- [#3033](https://github.com/vegaprotocol/vega/pull/3033) - Improve handling of genesis block errors
- [#3036](https://github.com/vegaprotocol/vega/pull/3036) - Equity share is now correct when submitting initial order
- [#3048](https://github.com/vegaprotocol/vega/pull/3048) - LP submission now checks margin engine is started
- [#3070](https://github.com/vegaprotocol/vega/pull/3070) - Rewrite amending LPs
- [#3053](https://github.com/vegaprotocol/vega/pull/3053) - Rewrite cancel all order implementation
- [#3050](https://github.com/vegaprotocol/vega/pull/3050) - GraphQL: Order in `LiquidityOrder` is now nullable
- [#3056](https://github.com/vegaprotocol/vega/pull/3056) - Move `vegastream` to a separate repository
- [#3057](https://github.com/vegaprotocol/vega/pull/3057) - Ignore error if Tendermint stats is temporarily unavailable
- [#3058](https://github.com/vegaprotocol/vega/pull/3058) - Fix governance to use total supply rather than total deposited into network
- [#3062](https://github.com/vegaprotocol/vega/pull/3070) - Opening Auction no longer set to null on a market when auction completes
- [#3051](https://github.com/vegaprotocol/vega/pull/3051) - Rewrite LP refresh mechanism
- [#3080](https://github.com/vegaprotocol/vega/pull/3080) - Auctions now leave auction when `maximumDuration` is exceeded
- [#3075](https://github.com/vegaprotocol/vega/pull/3075) - Bond account is now correctly cleared when LPs are cancelled
- [#3074](https://github.com/vegaprotocol/vega/pull/3074) - Switch error reporting mechanism to stream error
- [#3069](https://github.com/vegaprotocol/vega/pull/3069) - Switch more error reporting mechanisms to stream error
- [#3081](https://github.com/vegaprotocol/vega/pull/3081) - Fix fee check for LP orders
- [#3087](https://github.com/vegaprotocol/vega/pull/3087) - GraphQL schema grammar & spelling fixes
- [#3185](https://github.com/vegaprotocol/vega/pull/3185) - LP orders are now accessed deterministically
- [#3131](https://github.com/vegaprotocol/vega/pull/3131) - GRPC api now shuts down gracefully
- [#3110](https://github.com/vegaprotocol/vega/pull/3110) - LP Bond is now returned if a market is rejected
- [#3115](https://github.com/vegaprotocol/vega/pull/3115) - Parties with closed out LPs can now submit new LPs
- [#3123](https://github.com/vegaprotocol/vega/pull/3123) - New market proposals with invalid Oracle definitions no longer crash core
- [#3131](https://github.com/vegaprotocol/vega/pull/3131) - GRPC api now shuts down gracefully
- [#3137](https://github.com/vegaprotocol/vega/pull/3137) - Pegged orders that fail to reprice correctly are now properly removed from the Market Depth API
- [#3168](https://github.com/vegaprotocol/vega/pull/3168) - Fix `intoProto` for `OracleSpecBinding`
- [#3106](https://github.com/vegaprotocol/vega/pull/3106) - Target Stake is now used as the Market Value Proxy during opening auction
- [#3103](https://github.com/vegaprotocol/vega/pull/3103) - Ensure all filled and partially filled orders are remove from the Market Depth API
- [#3095](https://github.com/vegaprotocol/vega/pull/3095) - GraphQL: Fix missing data in proposal subscription
- [#3085](https://github.com/vegaprotocol/vega/pull/3085) - Minor tidy-up of errors reported by `goland`

## 0.32.0

*2021-02-23*

More fixes, primarily related to liquidity provisioning (still disabled in this release) and asset withdrawals, which will soon be enabled in the UI.

Two minor breaking changes in the GraphQL API are included - one fixing a typo, the other changing the content of date fields on the withdrawal object - they're now date formatted.

### Improvements
- [#3004](https://github.com/vegaprotocol/vega/pull/3004) - Incorporate `buf.yaml` tidy up submitted by `bufdev` on api-clients repo
- [#3002](https://github.com/vegaprotocol/vega/pull/3002) -🔥GraphQL: Withdrawal fields `expiry`, `createdAt` & `updatedAt` are now `RFC3339Nano` date formatted
- [#3000](https://github.com/vegaprotocol/vega/pull/3002) -🔥GraphQL: Fix typo in `prepareVote` mutation - `propopsalId` is now `proposalId`
- [#2957](https://github.com/vegaprotocol/vega/pull/2957) - REST: Add missing prepare endpoints (`PrepareProposal`, `PrepareVote`, `PrepareLiquiditySubmission`)

### Fixes
- [#3011](https://github.com/vegaprotocol/vega/pull/3011) - Liquidity fees are distributed in to margin accounts, not general accounts
- [#2991](https://github.com/vegaprotocol/vega/pull/2991) - Liquidity Provisions are now rejected if there is not enough collateral
- [#2990](https://github.com/vegaprotocol/vega/pull/2990) - Fix a lock caused by GraphQL subscribers unsubscribing from certain endpoints
- [#2996](https://github.com/vegaprotocol/vega/pull/2986) - Liquidity Provisions are now parked when repricing fails
- [#2951](https://github.com/vegaprotocol/vega/pull/2951) - Store reference prices when parking pegs for auction
- [#2982](https://github.com/vegaprotocol/vega/pull/2982) - Fix withdrawal data availability before it is verified
- [#2981](https://github.com/vegaprotocol/vega/pull/2981) - Fix sending multisig bundle for withdrawals before threshold is reached
- [#2964](https://github.com/vegaprotocol/vega/pull/2964) - Extend auctions if uncrossing price is unreasonable
- [#2961](https://github.com/vegaprotocol/vega/pull/2961) - GraphQL: Fix incorrect market in bond account resolver
- [#2958](https://github.com/vegaprotocol/vega/pull/2958) - Create `third_party` folder to avoid excluding vendor protobuf files in build
- [#3009](https://github.com/vegaprotocol/vega/pull/3009) - Remove LP commitments when a trader is closed out
- [#3012](https://github.com/vegaprotocol/vega/pull/3012) - Remove LP commitments when a trader reduces their commitment to 0

## 0.31.0

*2021-02-09*

Many of the fixes below relate to Liquidity Commitments, which are still disabled in testnet, and Data Sourcing, which is also not enabled. Data Sourcing (a.k.a Oracles) is one of the last remaining pieces we need to complete settlement at instrument expiry, and Liquidity Commitment will be enabled when the functionality has been stabilised.

This release does improve protocol documentation, with all missing fields filled in and the explanations for Pegged Orders expanded. Two crashers have been fixed, although the first is already live as hotfix on testnet, and the other is in functionality that is not yet enabled.

This release also makes some major API changes:

- `api.TradingClient` -> `api.v1.TradingServiceClient`
- `api.TradingDataClient` -> `api.v1.TradingDataServiceClient`
- Fields have changed from camel-case to snake-cased (e.g. `someFieldName` is now `some_field_name`)
- All API calls now have request and response messages whose names match the API call name (e.g. `GetSomething` now has a request called `GetSomethingRequest` and a response called `GetSomethingResponse`)
- See [#2879](https://github.com/vegaprotocol/vega/pull/2879) for details

### Improvements
- [#2879](https://github.com/vegaprotocol/vega/pull/2879) - 🔥Update all the protobuf files with Buf recommendations
- [#2847](https://github.com/vegaprotocol/vega/pull/2847) - Improve proto documentation, in particular for pegged orders
- [#2905](https://github.com/vegaprotocol/vega/pull/2905) - Update `vega verify` command to verify genesis block files
- [#2851](https://github.com/vegaprotocol/vega/pull/2851) - Enable distribution of liquidity fees to liquidity providers
- [#2871](https://github.com/vegaprotocol/vega/pull/2871) - Add `submitOracleData` command
- [#2887](https://github.com/vegaprotocol/vega/pull/2887) - Add Open Oracle data processing & data normalisation
- [#2915](https://github.com/vegaprotocol/vega/pull/2915) - Add Liquidity Commitments to API responses

### Fixes
- [#2913](https://github.com/vegaprotocol/vega/pull/2913) - Fix market lifecycle events not being published through event bus API
- [#2906](https://github.com/vegaprotocol/vega/pull/2906) - Add new process for calculating margins for orders during auction
- [#2887](https://github.com/vegaprotocol/vega/pull/2887) - Liquidity Commitment fix-a-thon
- [#2879](https://github.com/vegaprotocol/vega/pull/2879) - Apply `Buf` lint recommendations
- [#2872](https://github.com/vegaprotocol/vega/pull/2872) - Improve field names in fee distribution package
- [#2867](https://github.com/vegaprotocol/vega/pull/2867) - Fix GraphQL bug: deposits `creditedAt` incorrectly showed `createdAt` time, not credit time
- [#2858](https://github.com/vegaprotocol/vega/pull/2858) - Fix crasher caused by parking pegged orders for auction
- [#2852](https://github.com/vegaprotocol/vega/pull/2852) - Remove unused binaries from CI builds
- [#2850](https://github.com/vegaprotocol/vega/pull/2850) - Fix bug that caused fees to be charged for pegged orders
- [#2893](https://github.com/vegaprotocol/vega/pull/2893) - Remove unused dependency in repricing
- [#2929](https://github.com/vegaprotocol/vega/pull/2929) - Refactor GraphQL resolver for withdrawals
- [#2939](https://github.com/vegaprotocol/vega/pull/2939) - Fix crasher caused by incorrectly loading Fee account for transfers

## 0.30.0

*2021-01-19*

This release enables (or more accurately, re-enables previously disabled) pegged orders, meaning they're finally here :tada:

The Ethereum bridge also received some work - in particular the number of confirmations we wait for on Ethereum is now controlled by a governance parameter. Being a governance parameter, that means that the value can be changed by a governance vote. Slightly related: You can now fetch active governance proposals via REST.

:one: We also switch to [Buf](https://buf.build/) for our protobuf workflow. This was one of the pre-requisites for opening up our api clients build process, and making the protobuf files open source. More on that soon!

:two: This fixes an issue on testnet where votes were not registered when voting on open governance proposals. The required number of Ropsten `VOTE` tokens was being calculated incorrectly on testnet, leading to all votes quietly being ignored. In 0.30.0, voting works as expected again.

### New
- [#2732](https://github.com/vegaprotocol/vega/pull/2732) Add REST endpoint to fetch all proposals (`/governance/proposals`)
- [#2735](https://github.com/vegaprotocol/vega/pull/2735) Add `FeeSplitter` to correctly split fee portion of an aggressive order
- [#2745](https://github.com/vegaprotocol/vega/pull/2745) Add transfer bus events for withdrawals and deposits
- [#2754](https://github.com/vegaprotocol/vega/pull/2754) Add New Market bus event
- [#2778](https://github.com/vegaprotocol/vega/pull/2778) Switch to [Buf](https://buf.build/) :one:
- [#2785](https://github.com/vegaprotocol/vega/pull/2785) Add configurable required confirmations for bridge transactions
- [#2791](https://github.com/vegaprotocol/vega/pull/2791) Add Supplied State to market data
- [#2793](https://github.com/vegaprotocol/vega/pull/2793) 🔥Rename `marketState` to `marketTradingMode`, add new `marketState` enum (`ACTIVE`, `SUSPENDED` or `PENDING`)
- [#2833](https://github.com/vegaprotocol/vega/pull/2833) Add fees estimate for pegged orders
- [#2838](https://github.com/vegaprotocol/vega/pull/2838) Add bond and fee transfers

### Improvements
- [#2835](https://github.com/vegaprotocol/vega/pull/2835) Fix voting for proposals :two:
- [#2830](https://github.com/vegaprotocol/vega/pull/2830) Refactor pegged order repricing
- [#2827](https://github.com/vegaprotocol/vega/pull/2827) Refactor expiring orders lists
- [#2821](https://github.com/vegaprotocol/vega/pull/2821) Handle liquidity commitments on market proposals
- [#2816](https://github.com/vegaprotocol/vega/pull/2816) Add changing liquidity commitment when not enough stake
- [#2805](https://github.com/vegaprotocol/vega/pull/2805) Fix read nodes lagging if they receive votes but not a bridge event
- [#2804](https://github.com/vegaprotocol/vega/pull/2804) Fix various minor bridge confirmation bugs
- [#2800](https://github.com/vegaprotocol/vega/pull/2800) Fix removing pegged orders that are rejected when unparked
- [#2799](https://github.com/vegaprotocol/vega/pull/2799) Fix crasher when proposing an update to network parameters
- [#2797](https://github.com/vegaprotocol/vega/pull/2797) Update target stake to include mark price
- [#2783](https://github.com/vegaprotocol/vega/pull/2783) Fix price monitoring integration tests
- [#2780](https://github.com/vegaprotocol/vega/pull/2780) Add more unit tests for pegged order price amends
- [#2774](https://github.com/vegaprotocol/vega/pull/2774) Fix cancelling all orders
- [#2768](https://github.com/vegaprotocol/vega/pull/2768) Fix GraphQL: Allow `marketId` to be null when it is invalid
- [#2767](https://github.com/vegaprotocol/vega/pull/2767) Fix parked pegged orders to have a price of 0 explicitly
- [#2766](https://github.com/vegaprotocol/vega/pull/2766) Disallow GFN to GTC/GTT amends
- [#2765](https://github.com/vegaprotocol/vega/pull/2765) Fix New Market bus event being sent more than once
- [#2763](https://github.com/vegaprotocol/vega/pull/2763) Add rounding to pegged order mid prices that land on non integer values
- [#2795](https://github.com/vegaprotocol/vega/pull/2795) Fix typos in GraphQL schema documentation
- [#2762](https://github.com/vegaprotocol/vega/pull/2762) Fix more typos in GraphQL schema documentation
- [#2758](https://github.com/vegaprotocol/vega/pull/2758) Fix error handling when amending some pegged order types
- [#2757](https://github.com/vegaprotocol/vega/pull/2757) Remove order from pegged list when it becomes inactive
- [#2756](https://github.com/vegaprotocol/vega/pull/2756) Add more panics to the core
- [#2750](https://github.com/vegaprotocol/vega/pull/2750) Remove expiring orders when amending to non GTT
- [#2671](https://github.com/vegaprotocol/vega/pull/2671) Add extra integration tests for uncrossing at auction end
- [#2746](https://github.com/vegaprotocol/vega/pull/2746) Fix potential divide by 0 in position calculation
- [#2743](https://github.com/vegaprotocol/vega/pull/2743) Add check for pegged orders impacted by order expiry
- [#2737](https://github.com/vegaprotocol/vega/pull/2737) Remove the ability to amend a pegged order's price
- [#2724](https://github.com/vegaprotocol/vega/pull/2724) Add price monitoring tests for order amendment
- [#2723](https://github.com/vegaprotocol/vega/pull/2723) Fix fee monitoring values during auction
- [#2721](https://github.com/vegaprotocol/vega/pull/2721) Fix incorrect reference when amending pegged orders
- [#2717](https://github.com/vegaprotocol/vega/pull/2717) Fix incorrect error codes for IOC and FOK orders during auction
- [#2715](https://github.com/vegaprotocol/vega/pull/2715) Update price monitoring to use reference price when syncing with risk model
- [#2711](https://github.com/vegaprotocol/vega/pull/2711) Refactor governance event subscription

## 0.29.0

*2020-12-07*

Note that you'll see a lot of changes related to **Pegged Orders** and **Liquidity Commitments**. These are still in testing, so these two types cannot currently be used in _Testnet_.

### New
- [#2534](https://github.com/vegaprotocol/vega/pull/2534) Implements amends for pegged orders
- [#2493](https://github.com/vegaprotocol/vega/pull/2493) Calculate market target stake
- [#2649](https://github.com/vegaprotocol/vega/pull/2649) Add REST governance endpoints
- [#2429](https://github.com/vegaprotocol/vega/pull/2429) Replace inappropriate wording in the codebase
- [#2617](https://github.com/vegaprotocol/vega/pull/2617) Implements proposal to update network parameters
- [#2622](https://github.com/vegaprotocol/vega/pull/2622) Integrate the liquidity engine into the market
- [#2683](https://github.com/vegaprotocol/vega/pull/2683) Use the Ethereum block log index to de-duplicate Ethereum transactions
- [#2674](https://github.com/vegaprotocol/vega/pull/2674) Update ERC20 token and bridges ABIs / codegen
- [#2690](https://github.com/vegaprotocol/vega/pull/2690) Add instruction to debug integration tests with DLV
- [#2680](https://github.com/vegaprotocol/vega/pull/2680) Add price monitoring bounds to the market data API

### Improvements
- [#2589](https://github.com/vegaprotocol/vega/pull/2589) Fix cancellation of pegged orders
- [#2659](https://github.com/vegaprotocol/vega/pull/2659) Fix panic in execution engine when GFN order are submit at auction start
- [#2661](https://github.com/vegaprotocol/vega/pull/2661) Handle missing error conversion in GraphQL API
- [#2621](https://github.com/vegaprotocol/vega/pull/2621) Fix pegged order creating duplicated order events
- [#2666](https://github.com/vegaprotocol/vega/pull/2666) Prevent the node to DDOS the Ethereum node when lots of deposits happen
- [#2653](https://github.com/vegaprotocol/vega/pull/2653) Fix indicative price and volume calculation
- [#2649](https://github.com/vegaprotocol/vega/pull/2649) Fix a typo in market price monitoring parameters API
- [#2650](https://github.com/vegaprotocol/vega/pull/2650) Change governance minimum proposer balance to be a minimum amount of token instead of a factor of the total supply
- [#2675](https://github.com/vegaprotocol/vega/pull/2675) Fix an GraphQL enum conversion
- [#2691](https://github.com/vegaprotocol/vega/pull/2691) Fix spelling in a network parameter
- [#2696](https://github.com/vegaprotocol/vega/pull/2696) Fix panic when uncrossing auction
- [#2984](https://github.com/vegaprotocol/vega/pull/2698) Fix price monitoring by feeding it the uncrossing price at end of opening auction
- [#2705](https://github.com/vegaprotocol/vega/pull/2705) Fix a bug related to order being sorted by creating time in the matching engine price levels

## 0.28.0

*2020-11-25*

Vega release logs contain a 🔥 emoji to denote breaking API changes. 🔥🔥 is a new combination denoting something that may significantly change your experience - from this release forward, transactions from keys that have no collateral on the network will *always* be rejected. As there are no transactions that don't either require collateral themselves, or an action to have been taken that already required collateral, we are now rejecting these as soon as possible.

We've also added support for synchronously submitting transactions. This can make error states easier to catch. Along with this you can now subscribe to error events in the event bus.

Also: Note that you'll see a lot of changes related to **Pegged Orders** and **Liquidity Commitments**. These are still in testing, so these two types cannot currently be used in _Testnet_.

### New
- [#2634](https://github.com/vegaprotocol/vega/pull/2634) Avoid caching transactions before they are rate/balance limited
- [#2626](https://github.com/vegaprotocol/vega/pull/2626) Add a transaction submit type to GraphQL
- [#2624](https://github.com/vegaprotocol/vega/pull/2624) Add mutexes to assets maps
- [#2593](https://github.com/vegaprotocol/vega/pull/2503) 🔥🔥 Reject transactions
- [#2453](https://github.com/vegaprotocol/vega/pull/2453) 🔥 Remove `baseName` field from markets
- [#2536](https://github.com/vegaprotocol/vega/pull/2536) Add Liquidity Measurement engine
- [#2539](https://github.com/vegaprotocol/vega/pull/2539) Add Liquidity Provisioning Commitment handling to markets
- [#2540](https://github.com/vegaprotocol/vega/pull/2540) Add support for amending pegged orders
- [#2549](https://github.com/vegaprotocol/vega/pull/2549) Add calculation for liquidity order sizes
- [#2553](https://github.com/vegaprotocol/vega/pull/2553) Allow pegged orders to have a price of 0
- [#2555](https://github.com/vegaprotocol/vega/pull/2555) Update Event stream votes to contain proposal ID
- [#2556](https://github.com/vegaprotocol/vega/pull/2556) Update Event stream to contain error events
- [#2560](https://github.com/vegaprotocol/vega/pull/2560) Add Pegged Order details to GraphQL
- [#2607](https://github.com/vegaprotocol/vega/pull/2807) Add support for parking orders during auction

### Improvements
- [#2634](https://github.com/vegaprotocol/vega/pull/2634) Avoid caching transactions before they are rate/balance limited
- [#2626](https://github.com/vegaprotocol/vega/pull/2626) Add a transaction submit type to GraphQL
- [#2624](https://github.com/vegaprotocol/vega/pull/2624) Add mutexes to assets maps
- [#2623](https://github.com/vegaprotocol/vega/pull/2623) Fix concurrent map access in assets
- [#2608](https://github.com/vegaprotocol/vega/pull/2608) Add sync/async equivalents for `submitTX`
- [#2618](https://github.com/vegaprotocol/vega/pull/2618) Disable storing API-related data on validator nodes
- [#2615](https://github.com/vegaprotocol/vega/pull/2618) Expand static checks
- [#2613](https://github.com/vegaprotocol/vega/pull/2613) Remove unused internal `cancelOrderById` function
- [#2530](https://github.com/vegaprotocol/vega/pull/2530) Governance asset for the network is now set in the genesis block
- [#2533](https://github.com/vegaprotocol/vega/pull/2533) More efficiently close channels in subscriptions
- [#2554](https://github.com/vegaprotocol/vega/pull/2554) Fix mid-price to 0 when best bid and average are unavailable and pegged order price is 0
- [#2565](https://github.com/vegaprotocol/vega/pull/2565) Cancelled pegged orders now have the correct status
- [#2568](https://github.com/vegaprotocol/vega/pull/2568) Prevent pegged orders from being repriced
- [#2570](https://github.com/vegaprotocol/vega/pull/2570) Expose probability of trading
- [#2576](https://github.com/vegaprotocol/vega/pull/2576) Use static best bid/ask price for pegged order repricing
- [#2581](https://github.com/vegaprotocol/vega/pull/2581) Fix order of messages when cancelling a pegged order
- [#2586](https://github.com/vegaprotocol/vega/pull/2586) Fix blank `txHash` in deposit API types
- [#2591](https://github.com/vegaprotocol/vega/pull/2591) Pegged orders are now cancelled when all orders are cancelled
- [#2609](https://github.com/vegaprotocol/vega/pull/2609) Improve expiry of pegged orders
- [#2610](https://github.com/vegaprotocol/vega/pull/2609) Improve removal of liquidity commitment orders when manual orders satisfy liquidity provisioning commitments

## 0.27.0

*2020-10-30*

This release contains a fix (read: large reduction in memory use) around auction modes with particularly large order books that caused slow block times when handling orders placed during an opening auction. It also contains a lot of internal work related to the liquidity provision mechanics.

### New
- [#2498](https://github.com/vegaprotocol/vega/pull/2498) Automatically create a bond account for liquidity providers
- [#2596](https://github.com/vegaprotocol/vega/pull/2496) Create liquidity measurement API
- [#2490](https://github.com/vegaprotocol/vega/pull/2490) GraphQL: Add Withdrawal and Deposit events to event bus
- [#2476](https://github.com/vegaprotocol/vega/pull/2476) 🔥`MarketData` now uses RFC339 formatted times, not seconds
- [#2473](https://github.com/vegaprotocol/vega/pull/2473) Add network parameters related to target stake calculation
- [#2506](https://github.com/vegaprotocol/vega/pull/2506) Network parameters can now contain JSON configuration

### Improvements
- [#2521](https://github.com/vegaprotocol/vega/pull/2521) Optimise memory usage when building cumulative price levels
- [#2520](https://github.com/vegaprotocol/vega/pull/2520) Fix indicative price calculation
- [#2517](https://github.com/vegaprotocol/vega/pull/2517) Improve command line for rate limiting in faucet & wallet
- [#2510](https://github.com/vegaprotocol/vega/pull/2510) Remove reference to external risk model
- [#2509](https://github.com/vegaprotocol/vega/pull/2509) Fix panic when loading an invalid genesis configuration
- [#2502](https://github.com/vegaprotocol/vega/pull/2502) Fix pointer when using amend in place
- [#2487](https://github.com/vegaprotocol/vega/pull/2487) Remove context from struct that didn't need it
- [#2485](https://github.com/vegaprotocol/vega/pull/2485) Refactor event bus event transmission
- [#2481](https://github.com/vegaprotocol/vega/pull/2481) Add `LiquidityProvisionSubmission` transaction
- [#2480](https://github.com/vegaprotocol/vega/pull/2480) Remove unused code
- [#2479](https://github.com/vegaprotocol/vega/pull/2479) Improve validation of external resources
- [#1936](https://github.com/vegaprotocol/vega/pull/1936) Upgrade to Tendermint 0.33.8

## 0.26.1

*2020-10-23*

Fixes a number of issues discovered during the testing of 0.26.0.

### Improvements
- [#2463](https://github.com/vegaprotocol/vega/pull/2463) Further reliability fixes for the event bus
- [#2469](https://github.com/vegaprotocol/vega/pull/2469) Fix incorrect error returned when a trader places an order in an asset that they have no account for (was `InvalidPartyID`, now `InsufficientAssetBalance`)
- [#2458](https://github.com/vegaprotocol/vega/pull/2458) REST: Fix a crasher when a market is proposed without specifying auction times

## 0.26.0

*2020-10-20*

The events API added in 0.25.0 had some reliability issues when a large volume of events were being emitted. This release addresses that in two ways:
 - The gRPC event stream now takes a parameter that sets a batch size. A client will receive the events when the batch limit is hit.
 - GraphQL is now limited to one event type per subscription, and we also removed the ALL event type as an option. This was due to the GraphQL gateway layer taking too long to process the full event stream, leading to sporadic disconnections.

These two fixes combined make both the gRPC and GraphQL streams much more reliable under reasonably heavy load. Let us know if you see any other issues. The release also adds some performance improvements to the way the core processes Tendermint events, some documentation improvements, and some additional debug tools.

### New
- [#2319](https://github.com/vegaprotocol/vega/pull/2319) Add fee estimate API endpoints to remaining APIs
- [#2321](https://github.com/vegaprotocol/vega/pull/2321) 🔥 Change `estimateFee` to `estimateOrder` in GraphQL
- [#2327](https://github.com/vegaprotocol/vega/pull/2327) 🔥 GraphQL: Event bus API - remove ALL type & limit subscription to one event type
- [#2343](https://github.com/vegaprotocol/vega/pull/2343) 🔥 Add batching support to stream subscribers

### Improvements
- [#2229](https://github.com/vegaprotocol/vega/pull/2229) Add Price Monitoring module
- [#2246](https://github.com/vegaprotocol/vega/pull/2246) Add new market depth subscription methods
- [#2298](https://github.com/vegaprotocol/vega/pull/2298) Improve error messages for Good For Auction/Good For Normal rejections
- [#2301](https://github.com/vegaprotocol/vega/pull/2301) Add validation for GFA/GFN orders
- [#2307](https://github.com/vegaprotocol/vega/pull/2307) Implement app state hash
- [#2312](https://github.com/vegaprotocol/vega/pull/2312) Add validation for market proposal risk parameters
- [#2313](https://github.com/vegaprotocol/vega/pull/2313) Add transaction replay protection
- [#2314](https://github.com/vegaprotocol/vega/pull/2314) GraphQL: Improve response when market does not exist
- [#2315](https://github.com/vegaprotocol/vega/pull/2315) GraphQL: Improve response when party does not exist
- [#2316](https://github.com/vegaprotocol/vega/pull/2316) Documentation: Improve documentation for fee estimate endpoint
- [#2318](https://github.com/vegaprotocol/vega/pull/2318) Documentation: Improve documentation for governance data endpoints
- [#2324](https://github.com/vegaprotocol/vega/pull/2324) Cache transactions already seen by `checkTX`
- [#2328](https://github.com/vegaprotocol/vega/pull/2328) Add test covering context cancellation mid data-sending
- [#2331](https://github.com/vegaprotocol/vega/pull/2331) Internal refactor of network parameter storage
- [#2334](https://github.com/vegaprotocol/vega/pull/2334) Rewrite `vegastream` to use the event bus
- [#2333](https://github.com/vegaprotocol/vega/pull/2333) Fix context for events, add block hash and event id
- [#2335](https://github.com/vegaprotocol/vega/pull/2335) Add ABCI event recorder
- [#2341](https://github.com/vegaprotocol/vega/pull/2341) Ensure event slices cannot be empty
- [#2345](https://github.com/vegaprotocol/vega/pull/2345) Handle filled orders in the market depth service before new orders are added
- [#2346](https://github.com/vegaprotocol/vega/pull/2346) CI: Add missing environment variables
- [#2348](https://github.com/vegaprotocol/vega/pull/2348) Use cached transactions in `checkTX`
- [#2349](https://github.com/vegaprotocol/vega/pull/2349) Optimise accounts map accesses
- [#2351](https://github.com/vegaprotocol/vega/pull/2351) Fix sequence ID related to market `OnChainTimeUpdate`
- [#2355](https://github.com/vegaprotocol/vega/pull/2355) Update coding style doc with info on log levels
- [#2358](https://github.com/vegaprotocol/vega/pull/2358) Add documentation and comments for `events.proto`
- [#2359](https://github.com/vegaprotocol/vega/pull/2359) Fix out of bounds index crash
- [#2364](https://github.com/vegaprotocol/vega/pull/2364) Add mutex to protect map access
- [#2366](https://github.com/vegaprotocol/vega/pull/2366) Auctions: Reject IOC/FOK orders
- [#2368](https://github.com/vegaprotocol/vega/pull/2370) Tidy up genesis market instantiation
- [#2369](https://github.com/vegaprotocol/vega/pull/2369) Optimise event bus to reduce CPU usage
- [#2370](https://github.com/vegaprotocol/vega/pull/2370) Event stream: Send batches instead of single events
- [#2376](https://github.com/vegaprotocol/vega/pull/2376) GraphQL: Remove verbose logging
- [#2377](https://github.com/vegaprotocol/vega/pull/2377) Update tendermint stats less frequently for Vega stats API endpoint
- [#2381](https://github.com/vegaprotocol/vega/pull/2381) Event stream: Reduce CPU load, depending on batch size
- [#2382](https://github.com/vegaprotocol/vega/pull/2382) GraphQL: Make event stream batch size mandatory
- [#2401](https://github.com/vegaprotocol/vega/pull/2401) Event stream: Fix CPU spinning after stream close
- [#2404](https://github.com/vegaprotocol/vega/pull/2404) Auctions: Add fix for crash during auction exit
- [#2419](https://github.com/vegaprotocol/vega/pull/2419) Make the price level wash trade check configurable
- [#2432](https://github.com/vegaprotocol/vega/pull/2432) Use `EmitDefaults` on `jsonpb.Marshaler`
- [#2431](https://github.com/vegaprotocol/vega/pull/2431) GraphQL: Add price monitoring
- [#2433](https://github.com/vegaprotocol/vega/pull/2433) Validate amend orders with GFN and GFA
- [#2436](https://github.com/vegaprotocol/vega/pull/2436) Return a permission denied error for a non-allowlisted public key
- [#2437](https://github.com/vegaprotocol/vega/pull/2437) Undo accidental code removal
- [#2438](https://github.com/vegaprotocol/vega/pull/2438) GraphQL: Fix a resolver error when markets are in auction mode
- [#2441](https://github.com/vegaprotocol/vega/pull/2441) GraphQL: Remove unnecessary validations
- [#2442](https://github.com/vegaprotocol/vega/pull/2442) GraphQL: Update library; improve error responses
- [#2447](https://github.com/vegaprotocol/vega/pull/2447) REST: Fix HTTP verb for network parameters query
- [#2443](https://github.com/vegaprotocol/vega/pull/2443) Auctions: Add check for opening auction duration during market creation

## 0.25.1

*2020-10-14*

This release backports two fixes from the forthcoming 0.26.0 release.

### Improvements
- [#2354](https://github.com/vegaprotocol/vega/pull/2354) Update `OrderEvent` to copy by value
- [#2379](https://github.com/vegaprotocol/vega/pull/2379) Add missing `/governance/prepare/vote` REST endpoint

## 0.25.0

*2020-09-24*

This release adds the event bus API, allowing for much greater introspection in to the operation of a node. We've also re-enabled the order amends API, as well as a long list of fixes.

### New
- [#2281](https://github.com/vegaprotocol/vega/pull/2281) Enable opening auctions
- [#2205](https://github.com/vegaprotocol/vega/pull/2205) Add GraphQL event stream API
- [#2219](https://github.com/vegaprotocol/vega/pull/2219) Add deposits API
- [#2222](https://github.com/vegaprotocol/vega/pull/2222) Initial asset list is now loaded from genesis configuration, not external configuration
- [#2238](https://github.com/vegaprotocol/vega/pull/2238) Re-enable order amend API
- [#2249](https://github.com/vegaprotocol/vega/pull/2249) Re-enable TX rate limit by party ID
- [#2240](https://github.com/vegaprotocol/vega/pull/2240) Add time to position responses

### Improvements
- [#2211](https://github.com/vegaprotocol/vega/pull/2211) 🔥 GraphQL: Field case change `proposalId` -> `proposalID`
- [#2218](https://github.com/vegaprotocol/vega/pull/2218) 🔥 GraphQL: Withdrawals now return a `Party`, not a party ID
- [#2202](https://github.com/vegaprotocol/vega/pull/2202) Fix time validation for proposals when all times are the same
- [#2206](https://github.com/vegaprotocol/vega/pull/2206) Reduce log noise from statistics endpoint
- [#2207](https://github.com/vegaprotocol/vega/pull/2207) Automatically reload node configuration
- [#2209](https://github.com/vegaprotocol/vega/pull/2209) GraphQL: fix proposal rejection enum
- [#2210](https://github.com/vegaprotocol/vega/pull/2210) Refactor order service to not require blockchain client
- [#2213](https://github.com/vegaprotocol/vega/pull/2213) Improve error clarity for invalid proposals
- [#2216](https://github.com/vegaprotocol/vega/pulls/2216) Ensure all GRPC endpoints use real time, not Vega time
- [#2231](https://github.com/vegaprotocol/vega/pull/2231) Refactor processor to no longer require collateral
- [#2232](https://github.com/vegaprotocol/vega/pull/2232) Clean up logs that dumped raw bytes
- [#2233](https://github.com/vegaprotocol/vega/pull/2233) Remove generate method from execution engine
- [#2234](https://github.com/vegaprotocol/vega/pull/2234) Remove `authEnabled` setting
- [#2236](https://github.com/vegaprotocol/vega/pull/2236) Simply order amendment logging
- [#2237](https://github.com/vegaprotocol/vega/pull/2237) Clarify fees attribution in transfers
- [#2239](https://github.com/vegaprotocol/vega/pull/2239) Ensure margin is released immediately, not on next mark to market
- [#2241](https://github.com/vegaprotocol/vega/pull/2241) Load log level in processor app
- [#2245](https://github.com/vegaprotocol/vega/pull/2245) Fix a concurrent map access in positions API
- [#2247](https://github.com/vegaprotocol/vega/pull/2247) Improve logging on a TX with an invalid signature
- [#2252](https://github.com/vegaprotocol/vega/pull/2252) Fix incorrect order count in Market Depth API
- [#2254](https://github.com/vegaprotocol/vega/pull/2254) Fix concurrent map access in Market Depth API
- [#2269](https://github.com/vegaprotocol/vega/pull/2269) GraphQL: Fix party filtering for event bus API
- [#2266](https://github.com/vegaprotocol/vega/pull/2266) Refactor transaction codec
- [#2275](https://github.com/vegaprotocol/vega/pull/2275) Prevent opening auctions from closing early
- [#2262](https://github.com/vegaprotocol/vega/pull/2262) Clear potential position properly when an order is cancelled for self trading
- [#2286](https://github.com/vegaprotocol/vega/pull/2286) Add sequence ID to event bus events
- [#2288](https://github.com/vegaprotocol/vega/pull/2288) Fix auction events not appearing in GraphQL event bus
- [#2294](https://github.com/vegaprotocol/vega/pull/2294) Fixing incorrect order iteration in auctions
- [#2285](https://github.com/vegaprotocol/vega/pull/2285) Check auction times
- [#2283](https://github.com/vegaprotocol/vega/pull/2283) Better handling of 0 `expiresAt`

## 0.24.0

*2020-09-04*

One new API endpoint allows cancelling multiple orders simultaneously, either all orders by market, a single order in a specific market, or just all open orders.

Other than that it's mainly bugfixes, many of which fix subtly incorrect API output.

### New

- [#2107](https://github.com/vegaprotocol/vega/pull/2107) Support for cancelling multiple orders at once
- [#2186](https://github.com/vegaprotocol/vega/pull/2186) Add per-party rate-limit of 50 requests over 3 blocks

### Improvements

- [#2177](https://github.com/vegaprotocol/vega/pull/2177) GraphQL: Add Governance proposal metadata
- [#2098](https://github.com/vegaprotocol/vega/pull/2098) Fix crashed in event bus
- [#2041](https://github.com/vegaprotocol/vega/pull/2041) Fix a rounding error in the output of Positions API
- [#1934](https://github.com/vegaprotocol/vega/pull/1934) Improve API documentation
- [#2110](https://github.com/vegaprotocol/vega/pull/2110) Send Infrastructure fees to the correct account
- [#2117](https://github.com/vegaprotocol/vega/pull/2117) Prevent creation of withdrawal requests for more than the available balance
- [#2136](https://github.com/vegaprotocol/vega/pull/2136) gRPC: Fetch all accounts for a market did not return all accounts
- [#2151](https://github.com/vegaprotocol/vega/pull/2151) Prevent wasteful event bus subscriptions
- [#2167](https://github.com/vegaprotocol/vega/pull/2167) Ensure events in the event bus maintain their order
- [#2178](https://github.com/vegaprotocol/vega/pull/2178) Fix API returning incorrectly formatted orders when a party has no collateral

## 0.23.1

*2020-08-27*

This release backports a fix from the forthcoming 0.24.0 release that fixes a GraphQL issue with the new `Asset` type. When fetching the Assets from the top level, all the details came through. When fetching them as a nested property, only the ID was filled in. This is now fixed.

### Improvements

- [#2140](https://github.com/vegaprotocol/vega/pull/2140) GraphQL fix for fetching assets as nested properties

## 0.23.0

*2020-08-10*

This release contains a lot of groundwork for Fees and Auction mode.

**Fees** are incurred on every trade on Vega. Those fees are divided between up to three recipient types, but traders will only see one collective fee charged. The fees reward liquidity providers, infrastructure providers and market makers.

* The liquidity portion of the fee is paid to market makers for providing liquidity, and is transferred to the market-maker fee pool for the market.
* The infrastructure portion of the fee, which is paid to validators as a reward for running the infrastructure of the network, is transferred to the infrastructure fee pool for that asset. It is then periodically distributed to the validators.
* The maker portion of the fee is transferred to the non-aggressive, or passive party in the trade (the maker, as opposed to the taker).

**Auction mode** is not enabled in this release, but the work is nearly complete for Opening Auctions on new markets.

💥 Please note, **this release disables order amends**. The team uncovered an issue in the Market Depth API output that is caused by order amends, so rather than give incorrect output, we've temporarily disabled the amendment of orders. They will return when the Market Depth API is fixed. For now, *amends will return an error*.

### New

- 💥 [#2092](https://github.com/vegaprotocol/vega/pull/2092) Disable order amends
- [#2027](https://github.com/vegaprotocol/vega/pull/2027) Add built in asset faucet endpoint
- [#2075](https://github.com/vegaprotocol/vega/pull/2075), [#2086](https://github.com/vegaprotocol/vega/pull/2086), [#2083](https://github.com/vegaprotocol/vega/pull/2083), [#2078](https://github.com/vegaprotocol/vega/pull/2078) Add time & size limits to faucet requests
- [#2068](https://github.com/vegaprotocol/vega/pull/2068) Add REST endpoint to fetch governance proposals by Party
- [#2058](https://github.com/vegaprotocol/vega/pull/2058) Add REST endpoints for fees
- [#2047](https://github.com/vegaprotocol/vega/pull/2047) Add `prepareWithdraw` endpoint

### Improvements

- [#2061](https://github.com/vegaprotocol/vega/pull/2061) Fix Network orders being left as active
- [#2034](https://github.com/vegaprotocol/vega/pull/2034) Send `KeepAlive` messages on GraphQL subscriptions
- [#2031](https://github.com/vegaprotocol/vega/pull/2031) Add proto fields required for auctions
- [#2025](https://github.com/vegaprotocol/vega/pull/2025) Add auction mode (currently never triggered)
- [#2013](https://github.com/vegaprotocol/vega/pull/2013) Add Opening Auctions support to market framework
- [#2010](https://github.com/vegaprotocol/vega/pull/2010) Add documentation for Order Errors to proto source files
- [#2003](https://github.com/vegaprotocol/vega/pull/2003) Add fees support
- [#2004](https://github.com/vegaprotocol/vega/pull/2004) Remove @deprecated field from GraphQL input types (as it’s invalid)
- [#2000](https://github.com/vegaprotocol/vega/pull/2000) Fix `rejectionReason` for trades stopped for self trading
- [#1990](https://github.com/vegaprotocol/vega/pull/1990) Remove specified `tickSize` from market
- [#2066](https://github.com/vegaprotocol/vega/pull/2066) Fix validation of proposal timestamps to ensure that datestamps specify events in the correct order
- [#2043](https://github.com/vegaprotocol/vega/pull/2043) Track Event Queue events to avoid processing events from other chains twice
## 0.22.0

### Bugfixes
- [#2096](https://github.com/vegaprotocol/vega/pull/2096) Fix concurrent map access in event forward

*2020-07-20*

This release primarily focuses on setting up Vega nodes to deal correctly with events sourced from other chains, working towards bridging assets from Ethereum. This includes responding to asset events from Ethereum, and support for validator nodes notarising asset movements and proposals.

It also contains a lot of bug fixes and improvements, primarily around an internal refactor to using an event bus to communicate between packages. Also included are some corrections for order statuses that were incorrectly being reported or left outdated on the APIs.

### New

- [#1825](https://github.com/vegaprotocol/vega/pull/1825) Add new Notary package for tracking multisig decisions for governance
- [#1837](https://github.com/vegaprotocol/vega/pull/1837) Add support for two-step governance processes such as asset proposals
- [#1856](https://github.com/vegaprotocol/vega/pull/1856) Implement handling of external chain events from the Event Queue
- [#1927](https://github.com/vegaprotocol/vega/pull/1927) Support ERC20 deposits
- [#1987](https://github.com/vegaprotocol/vega/pull/1987) Add `OpenInterest` field to markets
- [#1949](https://github.com/vegaprotocol/vega/pull/1949) Add `RejectionReason` field to rejected governance proposals

### Improvements
- 💥 [#1988](https://github.com/vegaprotocol/vega/pull/1988) REST: Update orders endpoints to use POST, not PUT or DELETE
- 💥 [#1957](https://github.com/vegaprotocol/vega/pull/1957) GraphQL: Some endpoints returned a nullable array of Strings. Now they return an array of nullable strings
- 💥 [#1928](https://github.com/vegaprotocol/vega/pull/1928) GraphQL & GRPC: Remove broken `open` parameter from Orders endpoints. It returned ambiguous results
- 💥 [#1858](https://github.com/vegaprotocol/vega/pull/1858) Fix outdated order details for orders amended by cancel-and-replace
- 💥 [#1849](https://github.com/vegaprotocol/vega/pull/1849) Fix incorrect status on partially filled trades that would have matched with another order by the same user. Was `stopped`, now `rejected`
- 💥 [#1883](https://github.com/vegaprotocol/vega/pull/1883) REST & GraphQL: Market name is now based on the instrument name rather than being set separately
- [#1699](https://github.com/vegaprotocol/vega/pull/1699) Migrate Margin package to event bus
- [#1853](https://github.com/vegaprotocol/vega/pull/1853) Migrate Market package to event bus
- [#1844](https://github.com/vegaprotocol/vega/pull/1844) Migrate Governance package to event
- [#1877](https://github.com/vegaprotocol/vega/pull/1877) Migrate Position package to event
- [#1838](https://github.com/vegaprotocol/vega/pull/1838) GraphQL: Orders now include their `version` and `updatedAt`, which are useful when dealing with amended orders
- [#1841](https://github.com/vegaprotocol/vega/pull/1841) Fix: `expiresAt` on orders was validated at submission time, this has been moved to post-chain validation
- [#1849](https://github.com/vegaprotocol/vega/pull/1849) Improve Order documentation for `Status` and `TimeInForce`
- [#1861](https://github.com/vegaprotocol/vega/pull/1861) Remove single mutex in event bus
- [#1866](https://github.com/vegaprotocol/vega/pull/1866) Add mutexes for event bus access
- [#1889](https://github.com/vegaprotocol/vega/pull/1889) Improve event broker performance
- [#1891](https://github.com/vegaprotocol/vega/pull/1891) Fix context for event subscribers
- [#1889](https://github.com/vegaprotocol/vega/pull/1889) Address event bus performance issues
- [#1892](https://github.com/vegaprotocol/vega/pull/1892) Improve handling for new chain connection proposal
- [#1903](https://github.com/vegaprotocol/vega/pull/1903) Fix regressions in Candles API introduced by event bus
- [#1940](https://github.com/vegaprotocol/vega/pull/1940) Add new asset proposals to GraphQL API
- [#1943](https://github.com/vegaprotocol/vega/pull/1943) Validate list of allowed assets

## 0.21.0

*2020-06-18*

A follow-on from 0.20.1, this release includes a fix for the GraphQL API returning inconsistent values for the `side` field on orders, leading to Vega Console failing to submit orders. As a bonus there is another GraphQL improvement, and two fixes that return more correct values for filled network orders and expired orders.

### Improvements

- 💥 [#1820](https://github.com/vegaprotocol/vega/pull/1820) GraphQL: Non existent parties no longer return a GraphQL error
- 💥 [#1784](https://github.com/vegaprotocol/vega/pull/1784) GraphQL: Update schema and fix enum mappings from Proto
- 💥 [#1761](https://github.com/vegaprotocol/vega/pull/1761) Governance: Improve processing of Proposals
- [#1822](https://github.com/vegaprotocol/vega/pull/1822) Remove duplicate updates to `createdAt`
- [#1818](https://github.com/vegaprotocol/vega/pull/1818) Trades: Replace buffer with events
- [#1812](https://github.com/vegaprotocol/vega/pull/1812) Governance: Improve logging
- [#1810](https://github.com/vegaprotocol/vega/pull/1810) Execution: Set order status for fully filled network orders to be `FILLED`
- [#1803](https://github.com/vegaprotocol/vega/pull/1803) Matching: Set `updatedAt` when orders expire
- [#1780](https://github.com/vegaprotocol/vega/pull/1780) APIs: Reject `NETWORK` orders
- [#1792](https://github.com/vegaprotocol/vega/pull/1792) Update Golang to 1.14 and tendermint to 0.33.5

## 0.20.1

*2020-06-18*

This release fixes one small bug that was causing many closed streams, which was a problem for API clients.

## Improvements

- [#1813](https://github.com/vegaprotocol/vega/pull/1813) Set `PartyEvent` type to party event

## 0.20.0

*2020-06-15*

This release contains a lot of fixes to APIs, and a minor new addition to the statistics endpoint. Potentially breaking changes are now labelled with 💥. If you have implemented a client that fetches candles, places orders or amends orders, please check below.

### Features
- [#1730](https://github.com/vegaprotocol/vega/pull/1730) `ChainID` added to statistics endpoint
- 💥 [#1734](https://github.com/vegaprotocol/vega/pull/1734) Start adding `TraceID` to core events

### Improvements
- 💥 [#1721](https://github.com/vegaprotocol/vega/pull/1721) Improve API responses for `GetProposalById`
- 💥 [#1724](https://github.com/vegaprotocol/vega/pull/1724) New Order: Type no longer defaults to LIMIT orders
- 💥 [#1728](https://github.com/vegaprotocol/vega/pull/1728) `PrepareAmend` no longer accepts expiry time
- 💥 [#1760](https://github.com/vegaprotocol/vega/pull/1760) Add proto enum zero value "unspecified" to Side
- 💥 [#1764](https://github.com/vegaprotocol/vega/pull/1764) Candles: Interval no longer defaults to 1 minute
- 💥 [#1773](https://github.com/vegaprotocol/vega/pull/1773) Add proto enum zero value "unspecified" to `Order.Status`
- 💥 [#1776](https://github.com/vegaprotocol/vega/pull/1776) Add prefixes to enums, add proto zero value "unspecified" to `Trade.Type`
- 💥 [#1781](https://github.com/vegaprotocol/vega/pull/1781) Add prefix and UNSPECIFIED to `ChainStatus`, `AccountType`, `TransferType`
- [#1714](https://github.com/vegaprotocol/vega/pull/1714) Extend governance error handling
- [#1726](https://github.com/vegaprotocol/vega/pull/1726) Mark Price was not always correctly updated on a partial fill
- [#1734](https://github.com/vegaprotocol/vega/pull/1734) Feature/1577 hash context propagation
- [#1741](https://github.com/vegaprotocol/vega/pull/1741) Fix incorrect timestamps for proposals retrieved by GraphQL
- [#1743](https://github.com/vegaprotocol/vega/pull/1743) Orders amended to be GTT now return GTT in the response
- [#1745](https://github.com/vegaprotocol/vega/pull/1745) Votes blob is now base64 encoded
- [#1747](https://github.com/vegaprotocol/vega/pull/1747) Markets created from proposals now have the same ID as the proposal that created them
- [#1750](https://github.com/vegaprotocol/vega/pull/1750) Added datetime to governance votes
- [#1751](https://github.com/vegaprotocol/vega/pull/1751) Fix a bug in governance vote counting
- [#1752](https://github.com/vegaprotocol/vega/pull/1752) Fix incorrect validation on new orders
- [#1757](https://github.com/vegaprotocol/vega/pull/1757) Fix incorrect party ID validation on new orders
- [#1758](https://github.com/vegaprotocol/vega/pull/1758) Fix issue where markets created via governance were not tradable
- [#1763](https://github.com/vegaprotocol/vega/pull/1763) Expiration settlement date for market changed to 30/10/2020-22:59:59
- [#1777](https://github.com/vegaprotocol/vega/pull/1777) Create `README.md`
- [#1764](https://github.com/vegaprotocol/vega/pull/1764) Add proto enum zero value "unspecified" to Interval
- [#1767](https://github.com/vegaprotocol/vega/pull/1767) Feature/1692 order event
- [#1787](https://github.com/vegaprotocol/vega/pull/1787) Feature/1697 account event
- [#1788](https://github.com/vegaprotocol/vega/pull/1788) Check for unspecified Vote value
- [#1794](https://github.com/vegaprotocol/vega/pull/1794) Feature/1696 party event

## 0.19.0

*2020-05-26*

This release fixes a handful of bugs, primarily around order amends and new market governance proposals.

### Features

- [#1658](https://github.com/vegaprotocol/vega/pull/1658) Add timestamps to proposal API responses
- [#1656](https://github.com/vegaprotocol/vega/pull/1656) Add margin checks to amends
- [#1679](https://github.com/vegaprotocol/vega/pull/1679) Add topology package to map Validator nodes to Vega keypairs

### Improvements
- [#1718](https://github.com/vegaprotocol/vega/pull/1718) Fix a case where a party can cancel another party's orders
- [#1662](https://github.com/vegaprotocol/vega/pull/1662) Start moving to event-based architecture internally
- [#1684](https://github.com/vegaprotocol/vega/pull/1684) Fix order expiry handling when `expiresAt` is amended
- [#1686](https://github.com/vegaprotocol/vega/pull/1686) Fix participation stake to have a maximum of 100%
- [#1607](https://github.com/vegaprotocol/vega/pull/1607) Update `gqlgen` dependency to 0.11.3
- [#1711](https://github.com/vegaprotocol/vega/pull/1711) Remove ID from market proposal input
- [#1712](https://github.com/vegaprotocol/vega/pull/1712) `prepareProposal` no longer returns an ID on market proposals
- [#1707](https://github.com/vegaprotocol/vega/pull/1707) Allow overriding default governance parameters via `ldflags`.
- [#1715](https://github.com/vegaprotocol/vega/pull/1715) Compile testing binary with short-lived governance periods

## 0.18.1

*2020-05-13*

### Improvements
- [#1649](https://github.com/vegaprotocol/vega/pull/1649)
    Fix github artefact upload CI configuration

## 0.18.0

*2020-05-12*

From this release forward, compiled binaries for multiple platforms will be attached to the release on GitHub.

### Features

- [#1636](https://github.com/vegaprotocol/vega/pull/1636)
    Add a default GraphQL query complexity limit of 5. Currently configured to 17 on testnet to support Console.
- [#1656](https://github.com/vegaprotocol/vega/pull/1656)
    Add GraphQL queries for governance proposals
- [#1596](https://github.com/vegaprotocol/vega/pull/1596)
    Add builds for multiple architectures to GitHub releases

### Improvements
- [#1630](https://github.com/vegaprotocol/vega/pull/1630)
    Fix amends triggering multiple updates to the same order
- [#1564](https://github.com/vegaprotocol/vega/pull/1564)
    Hex encode keys

## 0.17.0

*2020-04-21*

### Features

- [#1458](https://github.com/vegaprotocol/vega/issues/1458) Add root GraphQL Orders query.
- [#1457](https://github.com/vegaprotocol/vega/issues/1457) Add GraphQL query to list all known parties.
- [#1455](https://github.com/vegaprotocol/vega/issues/1455) Remove party list from stats endpoint.
- [#1448](https://github.com/vegaprotocol/vega/issues/1448) Add `updatedAt` field to orders.

### Improvements

- [#1102](https://github.com/vegaprotocol/vega/issues/1102) Return full Market details in nested GraphQL queries.
- [#1466](https://github.com/vegaprotocol/vega/issues/1466) Flush orders before trades. This fixes a rare scenario where a trade can be available through the API, but not the order that triggered it.
- [#1491](https://github.com/vegaprotocol/vega/issues/1491) Fix `OrdersByMarket` and `OrdersByParty` 'Open' parameter.
- [#1472](https://github.com/vegaprotocol/vega/issues/1472) Fix Orders by the same party matching.

### Upcoming changes

This release contains the initial partial implementation of Governance. This will be finished and documented in 0.18.0.

## 0.16.2

*2020-04-16*

### Improvements

- [#1545](https://github.com/vegaprotocol/vega/pull/1545) Improve error handling in `Prepare*Order` requests

## 0.16.1

*2020-04-15*

### Improvements

- [!651](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/651) Prevent bad ED25519 key length causing node panic.

## 0.16.0

*2020-03-02*

### Features

- The new authentication service is in place. The existing authentication service is now deprecated and will be removed in the next release.

### Improvements

- [!609](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/609) Show trades resulting from Orders created by the network (for example close outs) in the API.
- [!604](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/604) Add `lastMarketPrice` settlement.
- [!614](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/614) Fix casing of Order parameter `timeInForce`.
- [!615](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/615) Add new order statuses, `Rejected` and `PartiallyFilled`.
- [!622](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/622) GraphQL: Change Buyer and Seller properties on Trades from string to Party.
- [!599](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/599) Pin Market IDs to fixed values.
- [!603](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/603), [!611](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/611) Remove `NotifyTraderAccount` from API documentation.
- [!624](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/624) Add protobuf validators to API requests.
- [!595](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/595), [!621](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/621), [!623](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/623) Fix a flaky integration test.
- [!601](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/601) Improve matching engine coverage.
- [!612](https://gitlab.com/vega-protocol/trading-core/-/merge_requests/612) Improve collateral engine test coverage.
