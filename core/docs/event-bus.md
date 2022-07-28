# Event bus
The event bus is an internal system that can be used to expose data that is created in the core - i.e. anything that is not sourced from the chain). It is being introduced to provide more insight in to the working of the core than we have currently, and will enable us to build richer non-core software such as external storage, auditing and analysis tools that cannot be built without access to the data computed by the core.

## Event
An event is an action or a side-effect that triggered by trading-core in response to state change on the node. All events will have a root cause - for example an incoming order that triggers a trade - which will be encoded in the event in the form of a trace field, which will contain a hash of the transaction that triggered the actions leading to an event being emitted.

An event is represented as data / notification that is sent on to the bus. Any state changes to the core data (for example trader positions, mark price, collateral, ...) will produce an event. Some state changes will produce multiple events.

Expected errors encountered during workflow (e.g. rejected invalid order) are considered valid events.

### Data Structure
The following struct is the generic container for all events:

```go
struct Event {
	ID string			// Sequenced a output of the chain
	Ts time.Time			// Current time on the node
	e interface{}		// The actual event
	Trace {
		hash string		// A hash of the initial transaction that triggered this event
		seq int			// use to order the events triggered by the above hash
	}
}
```

## Consumer
A consumer is any engine, plug-in or other piece of code that publishes or subscribes on the event bus. Consumers are expected to receive all events or no events.

### Consuming events
The consumer will receive all events published on the event bus. A consumer can filter based on the type of the `Event.e`:

```go
func eventListener(rawEvent Event) {
    switch evt := rawEvent.(type) {
      case *NewOrderEvent:
           // Process new order event
       case *AmendOrderEvent:
           // Process amend order event
       default:
           // Ignore
    }
}
```

Topics were initially discussed, but have been left out of this initial implementation, but can be added at a later date if they turn out to be required.

## Acceptance Criteria
- An event consumer can filter out of the stream the events it needs
- Metadata in events provides enough information that an audit log component could be built that, given a hash of a transaction, could list all events caused by that transaction.
- Metadata in events provides enough information that by using the trace fields, the initial action that caused it can be determined

## Out of scope
- __[Logging events]__ Logging for the event bus is to be implemented similarly to other core services and engines. Event bus logs are not expected to dump all processed events, although a separate consumer could be built for that.
- __[Error handling in isolation]__ The event bus is expected to be tightly coupled with the emitters in core. Invalid event types are to be ignored. Consumer errors (e.g. inability to consume events) are not part of event bus error handling path.
- __[Buffering]__ Event bus is not expected to buffer events, limit their lifetime or guarantee delivery.
- __[API]__ There will be no externally facing API to interact with event bus directly. This can be implementing as a separate consumer.
- __[External event bus]__ The event bus is internal to the core. It replaces the buffers we have currently, rather than restructuring the way internal components communicate. A new event consumer could be written to pass events out to an external event queue, but this is out of scope.