#Integration Tests

This is the home of the system integrations tests. They can be run from the root of vega with:

```make integrationtest```

To run specific tests you can go to the vega/integration folder and run:

```godog features/<test case>.feature```

If `godog` is not installed on your system you can get it with:

```go get github.com/DATA-DOG/godog/cmd/godog```
