# Debugging with `dlv`

This document is a guide describing how to debug with `dlv` and `VSCode`.

You will need to read and follow instructions in `GETTING_STARTED.md` first.

Once you have successfully installed `V E G A` with `tendermint` these are the steps to let you debug.

- Install `dlv` if you've not already done so. In `VSCode` you can do this by launching the "Command Palette" and running `Go: Install/Update Tools`, select `dlv`, press OK.
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
- Build the debug version (with optimisations disabled) of all the binaries with:

    ```bash
    DEBUGVEGA=yes make build
    ```
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
