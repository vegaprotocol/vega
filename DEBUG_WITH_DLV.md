# Debugging with `dlv`

## VSCode

There are two options:
1) You can start entire `V E G A` and `tendermint` and fire instructions at your node and debug the results.
2) You can attach the `dlv` debugger to the integration tests.

For both you'll need to
- Read and follow instructions in `GETTING_STARTED.md` first.
- Install `dlv` if you've not already done so. In `VSCode` you can do this by launching the "Command Palette" and running `Go: Install/Update Tools`, select `dlv`, press OK.


### Entire V E G A with tendermint

Once you have successfully installed `V E G A` with `tendermint` these are the steps to let you debug.

- Build the debug version (with optimisations disabled) of all the binaries with:

    ```bash
    DEBUGVEGA=yes make build
    ```
    If you're on a Mac `make build` will fail. This is because Mac OS isn't updating `bash` beyond version 3.x (something to do with GPL). The easiest solution is to install a recent `bash` e.g. with `https://brew.sh/` and then either run the above in the new `bash` instance or add your new bash to `/etc/shells` and then do `chsh` to change your default shell.

- Use "Command Palette" to run `Debug: Open launch.json`. If you didn't already have a `launch.json` file, this will create one with the below default configuration which can be used to debug the current package. Enter the following into `launch.json` (which will be by default created inside `vega/.vscode/`):

    ```json
    {
        "version": "0.2.0",
        "configurations": [
            {
                "name": "Debug V E G A",
                "type": "go",
                "request": "attach",
                "mode": "remote",
                "remotePath": "/path/to/vega", // trading-core
                "port": 2345,
                "host": "127.0.0.1",
                "showLog":true,
                "trace":"log"
            }
        ]
    }
    ```
    Edit the `"remotePath"` appropriately to reflect where your trading-core source code lives.

- Now open ideally two terminal windows. In the first one launch `tendermint` as you normally would. To hard-reset the chain and start from scratch use

    ```bash
    tendermint unsafe_reset_all && tendermint init && tendermint node 2>./tendermint.stderr.out 1>./tendermint.stdout.out
    ```
    Now launch the `dlv` debugger with `V E G A` by running

    ```bash
    dlv exec /path/to/vega/cmd/vega/vega-dbg --headless --listen=:2345 --log --api-version=2 -- node
    ```
    again replacing the path to match where your git copy of trading core lives.  If all went well you'll see something like:

    ```
    API server listening at: [::]:2345
    2019-10-13T20:37:41+01:00 info layer=debugger launching process with args: [/path/to/vega/cmd/vega/vega-dbg node]
    debugserver-@(#)PROGRAM:LLDB  PROJECT:lldb-1100.0.28..1
     for x86_64.
    Got a connection, launched process /path/to/vega/cmd/vega/vega (pid = 35671).
    ```
- Finally in `VSCode` open the Debug panel and run the `Debug V E G A` configuration created in the 2nd step above. At this point `V E G A` should be running.
- Test that `V E G A` is running as expected by e.g. visiting `http://localhost:3003/statistics` or trying something in the GraphQL playground at `http://localhost:3004/`. If all is well you should be able to create users, place orders etc. as normal. More to the point breakpoints, call stack and variables should be usable as normal in `VSCode`.



### Debugging integration tests

- Build a debug version of the `godog` test harness: from the root of the `V E G A` core repository

    ```bash
    DEBUGVEGA=yes go test -c ./integration/...
    ```

- Use "Command Palette" to run `Debug: Open launch.json`. If you didn't already have a `launch.json` file, this will create one with the below default configuration which can be used to debug the current package. Enter the following into `launch.json` (which will be by default created inside `vega/.vscode/`):

    ```json
    {
        "version": "0.2.0",
        "configurations": [
            {
                "name": "Debug Test",
                "type": "go",
                "request": "attach",
                "mode": "remote",
                "port": 2345,
                "host": "127.0.0.1",
                "showLog":true,
                "trace":"log"
            }
        ]
    }
    ```

- Stick a breakpoint somewhere in the code that you *know* will be triggered by your integration test (this clearly has to be in a `.go` file, not `.feature`).


- Launch the feature test you care about (in the example below it's `2668-price-monitoring.feature`)
    ```bash
    dlv exec ./integration.test  --headless --listen=:2345 --log --api-version=2    -- -godog.format=pretty --  $(pwd)/integration/features/2668-price-monitoring.feature
    ```
    The `godog` test harness is now running inside `dlv` and it has launched the integration test you chose.

- Finally in `VSCode` open the Debug panel and run the `Debug Test`.

----

## Vim

Prerequisites: Vim version >= 8.0 and the [vim-go](https://github.com/fatih/vim-go) plug-in.

### Build a debug binary for delve

A debug binary can be built using the `go test -c` command, which works in the same way `go build` would. Some compiler optimisations, however, can mess up the output when trying to print certain variables (e.g. slices being passed appearing empty, or containing seemingly garbage values).
To disable any and all compiler optimisations we might encounter, simply add the `gcflags="ALL=-N -l"` option.

To build the vega binary for debugging, then:

```bash
go test -c -gcflags="ALL=-N -l" ./cmd/vega
```

To build a binary to step through integration tests, simply run

```bash
go test -c -gcflags="ALL=-N -l" ./core/integration
```

The output will be a binary called either `vega.test` or `integration.test`.

### Start delve in headless mode

As-is, you can use the compiled binary to step through the code in dlv. Setting breakpoints using things like `b core/execution/market.go:2107` is quite tedious, though, and only shows you a few lines of code around the breakpoint. Having the ability to set breakpoints and check out variables in our editor is what we're after. To do that, we need to start dlv as a debug server, so we can connect to it from inside Vim:

```bash
dlv exec --headless --api-version=2 --listen 127.0.0.1:9876 ./integration.test -- ./core/integration/features
```

The address and port to listen to is arbitrary, but in this particular example, we'll use 9876. In this example we're starting a debug session on the integration test binary. To debug `vega.test`, just replace `integration.test` with `vega.test` (obviously).

### Passing arguments to the test binary

Passing in additional arguments/parameters is as easy as just appending `--` to the command above and specifying the desired arguments and flags. To run integration tests with a specific tag, for example, the full command looks like this:

```bash
dlv exec --headless --api-version=2 --listen 127.0.0.1:9876 ./integration.test -- --godog.tags=LPWrong -- ./core/integration/features
```

### Connecting to delve from Vim

Now that dlv is running, we can open the code we want to step through in Vim, and connect to our debugger:

```bash
vim core/execution/market.go
```

Once in our editor, just enter the  command `:GoDebugConnect 127.0.0.1:9876`

Debugger buffers will load as you've configured (default is the call stack top left, call stack, arguments, and registers bottom left, runtime and routine info bottom buffer). Open any file, jump to any line where you want to set a breakpoint, and run the `:GoDebugBreakpoint` (or `:GoDebugBr`) command.

To start executing the code, run `:GoDebugContinue`, and the test binary will run until a breakpoint is encountered. From that point on, you can use standard debugger commands (like `:GoDebugStep`, `:GoDebugStepOut`, and of course `:GoDebugContinue`). A full list of commands, what they do can be found by running `:h :GoDebug`.
The `vim-go` plug-in documentation also contains a list of default bindings (e.g. F9 for toggling a breakpoint, F5 for `:GoDebugContinue`), and detailed instructions on how to create your own bindings, how to customise your setup (`:h go-debug-settings`).

To stop debugging, run `:GoDebugStop`. This will close the debug buffers, and the dlv process will return.

### Inspecting and setting variables

Seeing what value a variable is set to is arguably the most common thing to do when debugging. Simply run `:GoDebugPrint foo` to see a full print-out of what a variable actually holds. Seeing as we're dealing with maps and arrays of objects quite a lot. It's important to note that dlv (and by extension vim-go debugging) does not restrict access to unexported fields, so things like `:GoDebugPrint order.Price.u[0]` work fine. `:GoDebugPrint` evaluates its argument as an expression, so things like `:GoDebugPrint foo == 10` is valid.

Setting a variable to a different value is equally possible, but just like delve itself, this is limited to types like `bool`, `float`, `int`, and `uint` variants, and pointers. This can be useful when debugging functions that have some boolean argument (e.g. `force` is set to `false`, change to true by running `:GoDebugSet force true`).
