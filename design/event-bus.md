# Event bus

Node event stream - a general event sink, capable of tracking all data and state changes.

## Definitions

### Event

Action or a side-effect that triggered by trading-core in response to state change on the node.

Events is represented as data / notification that is sent on to the bus. Any state changes to the core data (trader positions, mark price, collateral, ...) will produce an event. Some state changes will produce multiple events.

Workflow errors (e.g. rejected invalid order) are considered valid events.

#### Examples

- The mark price changes (for whatever reason)
- Traders with open positions get market to market
- Some traders may end up in a distressed state as a result
- Their pending orders get closed
- Traders who are still distressed get closed out (position resolution - distressed traders close each other out)
- The network trades with non-distressed traders
- Any balance on the insurance account for the market might get updated (balance of distressed traders moves to insurance pool, loss socialization taps into insurance pool)

#### Every structure

- topic (channel) - category of the event (e.g. positions; orders; etc)
- trigger - reason why event was emitted;
- trigger type;
- emitter of the signal, or its source;
- data payload - abstract data associated with that event (full copy of the data generated in response to the event; events are expected to be encapsulated)
- sequence number;
- emitted block time.

### Consumer

Event consumer (aka plug-in) connecting to the event bus and processing its data. Consumers are expected to precess events by topic.

Stores are to be populated by the consumers moving/copying payloads from the event bus.

Event bus will the way to send data from the core engines to underlying stores. Engines and services handling data in real-time will connect directly to the event bus (acting as consumers). Engines and services handling aggregated data will read data off the stores, not event bus (since event bus has no means of buffering the data).

## Assumptions

### Events and the buffers/plug-ins

We can identify various events that essentially duplicate, and therefore will replace the way we're currently interacting with the buffers and plug-ins:

- Trader balances get updated -> Accounts buffer
- Positions get updated -> position plug-in
- Each trade is an event -> trades buffer
- Orders are events -> orders buffer
- Ledger movements as events -> buffer

For this reason, it makes little to no sense to have both the buffers and the event bus in place. Instead the core just pushes the events onto the bus, and buffers subscribe to the events that contain the data they're aggregating. The same applies to the plug-ins.

Currently, we only have one plug-in (positions). To feed data into the positions plug-in, we have a positions buffer. The positions plug-in subscribes to this buffer, and receives a channel which gets populated with the data once the buffer is flushed. The plug-in takes this data, calculates the P&L etc... This is, for the most part, going to remain unchanged with the introduction of the event bus. Instead of receiving the data from a buffer, however, the plug-ins will subscribe to the event bus directly.

#### The issue of flushing

Buffers are flushed by the execution engine at the end of each block, or transaction. The event bus won't have this same `Flush` mechanic. Instead, we will be pushing an event indicating the start/end of a new block, and the end of a transaction. These events can be used as key points by stores to commit a transaction, or by plug-ins to process the state they've been aggregating.

### Domain models

The core currently uses the types defined in the protofile directly. This restricts us in terms of what data an event can represent. A trade event should, naturally, contain the trade object itself, but over time, we might want to have the realised/unrealised P&L values as part of the trade event available. This requires us to update the core to use domain models that are not directly bound to the current types we're using. There will be type embedding, so events can be type-cast to various event interfaces and multiplexed, of course.
Something worth considering is to develop a way to generate some of the boilerplate code that this approach will inevitably bring with it, although this is not a priority by any means.

<<<<<<< HEAD

## TODO


- [ ] Full data on every signal or summary in general stream and details in specialised streams?
- [ ] Error handling. What should happen to the node if event bus cannot process a new signal?
- [ ] Logging for event bus. Shall we have a dedicated logger like we do for engines and services?

=======
## Out of scope

- __[Logging events]__ Logging for event bus is to be implemented similarly to other core services and engines. Event bus logs are not expected to dump all processed events (a separate consumer might be built for that outside of this feature).
- __[Error handling in isolation]__ Event bus is expected to be tightly coupled with the emitters in core. Client errors are considered logic errors and are all expected to be detected during testing. Invalid event types are to be ignored. Consumer errors (e.g. inability to consume events) are not part of event bus error handling path.
- __[Buffering]__ Event bus is not expected to buffer events and limit their lifetime.
- __[API]__ There will be no API to interact with event bus directly.
>>>>>>> [skip ci] Correct proposal
