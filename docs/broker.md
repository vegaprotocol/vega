# Event bus

This document provides some information regarding the design and use of the event bus as it currently stands

## Broker

The broker package defines a generic event broker. This is the compontent to which events are passed to be passed on to the subscribers/consumers. There is just a single `Send()` function. Any registered subscriber can receive the event that was sent through this function. Whether or not they will receive the event depends on the subscriber configuration.

Subscribers are registered (and can be removed) using the `Subscribe` and `Unsubscribe` methods:

* `Subscribe(s Subscriber, required bool) int`: This method takes a Subscriber (interface defined in the package), a bool flag indicating whether or not this subscriber _requires_ all events. The method returns the subscriber ID (an int)
* `Unsubscribe(key int)`: this method removes a subscriber by ID. The ID can be reused from that point on. If the subscriber has already been removed, this call is a noop.

The broker will check the its context every time it tries to send an event to a subscriber.

## Subscribers

Subscribers implement an interface with 4 methods:

* `Push(interface{})`: This method is called on _required_ subscribers. The call is treated as a blocking call, pushing a single event to the subscriber.
* `C() chan<- interface{}`: Similar to the `Push` method, only the broker _attempts_ to push the event onto the subscriber channel, but if the channel is not being read from, or its buffer is full, the event is skipped instead.
* `Skip() <-chan struct{}`: This method returns a channel that the broker checks to see if the subscriber is in a suspended state. As long as this function returns a closed channel (or the broker can read from this channel), the subscriber won't receive any new events, but the subscriber remains registered.
* `Closed() <-chan struct{}`: This works in the same way as `Skip()`, but if the subscriber is marked as closed, the subscriber is automatically deregistered from the broker.

