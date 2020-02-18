#Integration Tests

This is the home of the system integrations tests. They can be run from the root of vega with:

```shell
make integrationtest
```

or  

```shell
go test ./...
``` 

To run specific tests you can go to the vega/integration folder and run:

```shell
cd integration
godog features/<test case>.feature
```

If `godog` is not installed on your system you can get it with:

```shell
go get github.com/cucumber/godog/cmd/godog
```

Or you can run it without installing:

```shell
cd integration
GOPATH= go run -v github.com/cucumber/godog/cmd/godog ./features/
```
