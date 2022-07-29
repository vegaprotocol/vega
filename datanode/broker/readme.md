# Event bus

The broker is the entry point for core events. Core engines only ever use a single method of the broker interface. Subscribers don't interact with the broker directly, but are called by the broker (which pushes events to them).

## For the core

Where engines previously depended on any number of buffers to push data to that plugins, API's, and stores needed to handle, they will now push events onto the bus via the broker. In the `events` package, you'll find a number of events (trade event, order event, market data, etc...). These events all have a constructor for convenience. Most events can even be created through the generic constructor. Say, for example, we receive a new order, or the state of an order changed:

```go
func (e *Engine) Foo(ctx, order *types.Order) {
    // some code, order changes:
    e.broker.Send(events.NewOrderEvent(ctx, *order))
    // or, using the generic constructor
    if evt, err := events.New(ctx, *order); err == nil {
        e.broker.Send(evt)
    }
}
```

The buffers needed to be flushed at the end of a block. Now, the end of a block is considered an event (`TimeUpdate`). If some events/data needs to be batched, and only processed at the end of the block, the subscriber should subscribe to the time event, and use that as a trigger/signal to do what is expected.

Some events (e.g. position related events) are created a ton of times. Sending them individually creates a metric ton of routines sending out all the data, which is bad for performance. To get around this issue, the broker includes a `SendBatch(events []Events)` function. The events in the slice can be of different types, but we derive the subscriber types from the first event in the slice, and assume that subscribers who listen for this event can either safely ignore any other events in the slice, or can handle all of them (currently, all subscribers can handle unknown events - they simply ignore them).
It might be a good idea to have some placeholder event added for things like `MarketEvent` (which basically is a catch-all event type for things we want to log).

## For non-core (subscribers etc...)

The core spits out events, without caring where the data ends up. The subscribers need to be registered with the broker to receive the data and process it. There are 2 main categories of subscribers: required (think of it as ack) and non-required subscribers. If a subscriber is registered as required, the broker will make sure that the subscriber receives all events, in the correct order. The non-required subscribers have a channel onto which the broker pushes events, unless the channel buffer is full. The subscriber is not required, so rather than blocking the broker, the broker is free to skip these subscribers and simply carry on. As a result the latter type of subscriber is not guaranteed to receive every single event, but unlike their required counterparts, they are unable to have a meaningful impact on the performance of the broker.

Subscribers implement a fairly simple interface:

```go
type Subscriber interface {
    Ack() bool
	Push(val ...events.Event)
	Skip() <-chan struct{}
	Closed() <-chan struct{}
	C() chan<- events.Event
	Types() []events.Type
	SetID(id int)
	ID() int
}
```

* `Ack()`: Indicates whether or not this subscriber "Ack's" the events it receives. In this case: the event has to be passed to the `Push` function.
* `Types`: The broker uses this call to determine what events this subscriber wants to receive.
* `Push`: A required subscriber will receive all its events from the broker through a normal function call. This ensures the event is indeed received. This function accepts one or more events. If `SendBatch()` was called on the broker, ack'ing subscribers will receive the entire batch of events in a single call.
* `C`: This is used for non-required subscribers. This returns a write channel where the prober attempts to push an event onto. If this fails (because the buffer is full), the event is dropped for that subscriber
* `Closed`: A subscriber can be halted (if it's no longer needed). This function will return a closed channel indicating that this subscriber is redundant, and should be removed
* `Skip`: If a subscriber only periodically needs to get data, we kan keep it registered, but put it in a _"paused"_ state. A paused subscriber will return a closed channel for as long as we're not interested in receiving data.
* `ID` and `SetID`: Subscribers have a broker ID (int). This is unique for all subscribers, and allows us to remove a specific subscriber manually, should we need to. The broker will also call `SetID` when the subscriber is registered.
