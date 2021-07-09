# Event bus

## Add a new event
1. Create a protobuf message to describe your event in the `proto` folder.
2. Register your event in `BusEventType` enum and `BusEvent.event` message in
   `proto/events.proto`.
3. Generate the code with `make proto`.
4. In `events/bus.go`, create a constant to identify the event and map it to the
   protobuf enum type `BusEventType` in variable `protoMap` and `toProto`. Give
   it a name in `eventStrings`.
5. In the `events` folder, create a file `my_event.go` where the Golang
   definition of the new event will live:

```golang
type MyEvent struct {
	*Base
	o proto.MyEvent
}
```

6. Implement the `StreamEvent` interface on it.
7. Add the support for this new event into the `Service` responsible for it.
