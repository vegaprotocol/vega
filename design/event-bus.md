# Even bus

Node event stream - a general event sink, able to track all data and state changes.

## Definitions

### Signal / trigger

Action or a side-effect that triggered an event on the node.
Examples:

- The mark price changes (for whatever reason)
- Traders with open positions get market to market
- Some traders may end up in a distressed state as a result
- Their pending orders get closed
- Traders who are still distressed get closed out (position resolution - distressed traders close each other out)
- The network trades with non-distressed traders
- Any balance on the insurance account for the market might get updated (balance of distressed traders moves to insurance pool, loss socialization taps into insurance pool)

#### Signal structure

- signal type (TODO: __would enum suffice or do we need more details here?__);
- singal sequence number;
- block time (TODO: alternatively just block number).

### Event

Event is what happens in in response to a signal.

Events is represented as data / notification that is sent on to the bus. Any state changes to the core data (trader positions, mark price, collateral, ...) will produce an event. Some state changes will produce multiple events.

#### Every structure

- topic (channel) - category of the event (e.g. positions; orders; etc)
- a signal - reason why event was emitted;
- emitter of the signal, or its source;
- data payload - abstract data associated with that event (TODO: __needs clarification__)
- expiration time - how long is the event going to be considered valid;
- processed flag - some indication whether event has been consumed already or not (TODO: __do we want to know what consumer precessed this event? Or generally if it was processed?__)

### Consumer

Event consumer connecting to the event bus and processing its data. Consumers are expected to precess events by topic.

## Assumptions

### Events and the buffers/plugins

From the examples above, we can identify various events that essentially duplicate, and therefore can replace the way we're currently interacting with the buffers and plugins:

- Trader balances get updated -> Accounts buffer
- Positions get updated -> position plugin
- Each trade is an event -> trades buffer
- Orders are events -> orders buffer
- Ledger movements as events -> buffer

For this reason, it makes little to no sense to have both the buffers and the event bus in place. Instead the core just pushes the events onto the bus, and buffers subscribe to the events that contain the data they're aggregating. The same applies to the plugins.

Currently, we only have one plugin (positions). To feed data into the positions plugin, we have a positions buffer. The positions plugin subscribes to this buffer, and receives a channel which gets populated with the data once the buffer is flushed. The plugin takes this data, calculates the P&L etc... This is, for the most part, going to remain unchanged with the introduction of the event bus. Instead of receiving the data from a buffer, however, the plugins will subscribe to the event bus directly.

#### The issue of flushing

Buffers are flushed by the execution engine at the end of each block, or transaction. The event bus won't have this same `Flush` mechanic. Instead, we will be pushing an event indicating the start/end of a new block, and the end of a transaction. These events can be used as key points by stores to commit a transaction, or by plugins to process the state they've been aggregating.

### Domain models

The core currently uses the types defined in the protofile directly. This restricts us in terms of what data an event can represent. A trade event should, naturally, contain the trade object itself, but over time, we might want to have the realised/unrealised P&L values as part of the trade event available. This requires us to update the core to use domain models that are not directly bound to the current types we're using. There will be type embedding, so events can be type-cast to various event interfaces and multiplexed, of course.
Something worth considering is to develop a way to generate some of the boilerplate code that this approach will inevitably bring with it, although this is not a priority by any means.

## TODO

- [ ] Full data on every signal or summary in general stream and details in specialised streams?
- [ ] Error handling. What should happen to the node if event bus cannot process a new signal?
- [ ] Logging for event bus. Shall we have a dedicated logger like we do for engines and services?
