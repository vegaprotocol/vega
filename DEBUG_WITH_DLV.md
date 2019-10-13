# Debugging with dlv

This document is a guide describing how to debug with `dlv` and `VSCode`. 

You will need to read and follow instructions in `GETTING_STARTED.md` first. 

Once you have sucessfully installed `V E G A` with `tendermint` these are the steps to let you debug. 
- Install `dlv` if you've not already done so. In `VSCode` you can do this by launching the "Command Pallete" and running `Go: Install/Update Tools`, select `dlv`, press Ok.
- Use "Command Pallete" to run `Debug: Open launch.json`. If you didn't already have a launch.json file, this will create one with the below default configuration which can be used to debug the current package. Enter the following into `launch.json` (which will be by default created inside `trading-core/.vscode/`): 
```
{
	"version": "0.2.0",
	"configurations": [
		{
			"name": "Debug V E G A",
			"type": "go",
			"request": "attach",
			"mode": "remote",
			"remotePath": "/Users/davidsiska/gits/code.vegaprotocol.io/trading-core",
			"port": 2345,
			"host": "127.0.0.1",
			"showLog":true,
			"trace":"log"
		}		
	]
}
```
Edit the `"remotePath"` appropriately to reflect where your trading-core source code lives.
- Open the `Makefile` within `trading-core`, find the `build:` target and edit the `go build` command to include the flags `-gcflags="all=-N -l"` to disable optimisation. The full `build:` target may as a result look like:
```
build: ## install the binaries in cmd/{progname}/
	@echo "Version: ${VERSION} (${VERSION_HASH})"
	@for app in $(APPS) ; do \
		env CGO_ENABLED=0 go build -v -gcflags="all=-N -l" -ldflags "-X main.Version=${VERSION} -X main.VersionHash=${VERSION_HASH}" -o "./cmd/$$app/$$app" "./cmd/$$app" || exit 1 ; \
	done
```
Run `make all` to rebuild with the flags above. 
- Now open ideally two terminal windows. In the first one launch `tendermint` as you normally would. To hard-reset the chain and start from scratch use `tendermint unsafe_reset_all && tendermint init && tendermint node  2> ./tendermint.stderr.out 1> ./tendermint.stdout.out`. Now launch the `dlv` debugger with V E G A by running `dlv exec /Users/davidsiska/gits/code.vegaprotocol.io/trading-core/cmd/vega/vega  --headless --listen=:2345 --log --api-version=2  -- node` again replacing the path to match where your git copy of trading core lives. 
If all went well you'll see something like: 
```
API server listening at: [::]:2345
2019-10-13T20:37:41+01:00 info layer=debugger launching process with args: [/Users/davidsiska/gits/code.vegaprotocol.io/trading-core/cmd/vega/vega node]
debugserver-@(#)PROGRAM:LLDB  PROJECT:lldb-1100.0.28..1
 for x86_64.
Got a connection, launched process /Users/davidsiska/gits/code.vegaprotocol.io/trading-core/cmd/vega/vega (pid = 35671).
```
- Finally in `VSCode` open the Debug panel and rund the `Debug V E G A` configuration created in the 2nd step above. At this point `V E G A` should be running. 
- Test that `V E G A` is running as expected by e.g. visiting `http://localhost:3003/statistics` or trying something in the GraphQL playground at `http://localhost:3004/`. If all is well you should be able to create users, place orders etc. as normal. More to the point breakpoints, call stack and variables should be usable as normal in `VSCode`. 
 