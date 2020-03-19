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

Signal structure:
- signal type (TODO: __would enum suffice or do we need more details here?__);
- singal sequence number;
- block time (TODO: alternatively just block number).

### Event

Event is what happens in in response to a signal.

Events is represented as data / notification that is sent on to the bus. Any state changes to the core data (trader positions, mark price, collateral, ...) will produce an event. Some state changes will produce multiple events.

Every event has to have:

- topic (channel) - category of the event (e.g. positions; orders; etc)
- a signal - reason why event was emitted;
- emitter of the signal, or its source;
- data payload - abstract data associated with that event (TODO: __needs clarification__)
- expiration time - how long is the event going to be considered valid;
- processed flag - some indication whether event has been consumed already or not (TODO: __do we want to know what consumer precessed this event? Or generally if it was processed?__)

### Consumer

Event consumer connecting to the event bus and processing its data. Consumers are expected to precess events by topic.

## Assumptions



## Scope

## Out of scope / not doing

## TODO

- [ ] Full data on every signal or summary in general stream and details in specialised streams?
- [ ] Error handling. What should happen to the node if event bus cannot process a new signal?
- [ ] Logging for event bus. Shall we have a dedicated logger like we do for engines and services?
