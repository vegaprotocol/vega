# scenario-runner-cli

This document provides the documentation for the `scenario-runner-cli` tool. It starts with a simple how-to guide showing how the tool can be used. The subsequent section lists the currently available instructions and their syntax. It finishes with a discussion of the current limitations of the tool and the planned modifications.

---
**NOTE**

When creating your own scenarios please add them to the `/cmd/scenariorunner/scenarios` folder - please feel free to create any subfolders you see fit. This should make it easier to collaborate on scenarios and investigate any possible bugs. It will also make allow us introduce breaking changes to the tool as that way we can amend any scenarios that might get broken due to such a change. While we aim to assure forward compatibility, there might still be a need to introduce such changes at the early stages of the project.

---

## HOW-TO: Submit an instruction set

---
**NOTE**

If you haven't cloned `trading-core` locally yet you can do it by following the steps outlined in the [Getting Started](../../GETTING_STARTED.md) guide.

Throughout this document we will assume that your local folder containing the `trading-core` repository is named `vega`.

---

An instruction set contains a collection of instructions that can be submitted to the `scenario-runner-cli`. You can find an example of it in `vega/cmd/scenariorunner/scenarios/examples/exampleInstructions.json`.
The steps described below show how to submit an instruction set to the `scenario-runner-cli`.

### Setup

* Make sure you have `trading-core` clonned locally (see guide mentioned at the beginning of this section if that's not the case).
* Open the command line tool and navigate to the directory where you have cloned `trading-core`.
* Call

    ```bash
    git checkout develop
    git pull
    ```

    to assure you're on the `develop` branch and you have the latest version of code.
* Call

    ```bash
    make install
    ```

    to assure that the latest version of the `scenario-runner-cli` gets installed.
* Verify that it has installed successfully by calling:

    ```bash
    scenariorunner --help
    ```

### Empty instruction set

The instruction set doesn't need to contin any instructions for the tool to be able to process it. While this feature is of limited practical use, it is useful to go through an example involving it to make it easier to see how both the instruction set and the optional result file that can be output by the `scenario-runner-cli` are structured.

* Navigate to the `scneariorunner` subfolder (this step is optional, it's only effect is to shorten the paths used in calls to the `scenario-runner-cli`)

    ```b
    cd cmd/scenariorunner
    ```

* Submit the instruction set by calling

    ```bash
    scenariorunner submit scenarios/examples/empty.json --config configs/noMarkets.json --result example1Result.json
    ```

* The step above should result in creation of the `exampleResult.json` file. More information of the structure of the instruction set and the result file can be found in the [instructions](#instructions) section below.

Let's finish this subsection with the discussion of flags used in a call to the `submit` command specified above - note that a list of available subcommands can be obtained by calling:

```bash
scenariorunner submit --help`.
```

We used:

* `--config` to point the tool to a config file that we wanted to use when processing the instructions - more details in ???? section. On this occassion we have used a config file that doesn't specify any markets to keep the result file light so that its' structure can be easily examined.
* `--result` to provide a directory for the result file to be saved in.

### Generating a trade

For a more involved example please execute

```bash
scenariorunner submit scenarios/examples/tradeGeneration.json --config configs/standard.json --result example2Result.json
```

This should generate the `example2Result.json` file.

### Multiple instruction sets

Multiple instruction sets can be submitted at once. The state of the execution engine doesn't reset between sets, hence scenarios can be broken down into multiple instruction sets. We ilustrate it with a simple example:

```bash
scenariorunner submit scenarios/examples/empty.json scenarios/examples/tradeGeneration.json --config configs/standard.json --result example3Result.json
```

The above command should create 2 result files: "example3Result_1of2.json" & "example3Result_2of2.json", which contain results for the "empty.json" & "tradeGeneration.json" instruction sets specified in the above call.

## Inputs and outputs

Currently the tool supports only JSON for its' inputs, outputs and the config file. The section discusses the structure and syntax of those files.

### Instruction set

Instruction set holds a list of instructions that the `scenario-runner-cli` tool processes sequentially and an optional description field.

```json

{
  "description": "An example instruction set, instructions get listed between the two square brackets - '[]' - and get separated by commas",
  "instructions": []
}

```

It gets submitted to the tool by invoking the "submit" command and providing a path to it - see the how-to guide in the preceding section for more details.

### Instructions

This subsection discusses the available `scenario-runner-cli` instructions and their syntax.

Each instruction holds the following fields:

* description - holds an optional description for an instruction. It doesn't affect the outcome of an instruction in any way, it's just an optional field that can be used for commenting or referance as it gets printed in the results.
* request - specifies the instruction type to be executed
* message - holds the parameters needed for the specified request

Below is an example of an instruction with all of those fields populated:

```json
{
    "description": "Set the time to 8am, 2 January 2019",
    "request": "SET_TIME",
    "message": {
        "@type": "core.SetTimeRequest",
        "Time": "2019-01-02T08:00:00Z"
    }
}

```

The reminder of this subsection lists all the available requests along with their message types and fields.

* SET_TIME

Sets the time that the execution engine relies on. Please note this can also be specified in a config file and doesn't need to be called explicitly as an instruction.

```json
{
    "request": "SET_TIME",
    "message": {
        "@type": "core.SetTimeRequest",
        "Time": "2019-01-02T08:00:00Z"
    }
}

```

* ADVANCE_TIME

Advances the time within execution engine by a specified amount. Please note config file has options to automatically advance time after each instruction.

```json

{
    "request": "ADVANCE_TIME",
    "message": {
    "@type": "core.AdvanceTimeRequest",
    "TimeDelta": "0.1s"
    }
}

```

* NOTIFY_TRADER_ACCOUNT

Creates an account for a specified trader and adds funds to it.

```json

{
    "request": "NOTIFY_TRADER_ACCOUNT",
    "message": {
    "@type": "api.NotifyTraderAccountRequest",
    "notif": {
        "traderID": "trader1",
        "amount": 1000
        }
    }
}

```

* SUBMIT_ORDER

Submits an order to the execution enginge. Market IDs are specified in the config file. The expiry time needs to be specified as the Unix time, see the [time package](https://golang.org/pkg/time/#Time.UnixNano) documentation for details.

```json

{
    "request": "SUBMIT_ORDER",
    "message": {
    "@type": "api.SubmitOrderRequest",
    "submission": {
        "marketID": "JXGQYDVQAP5DJUAQBCB4PACVJPFJR4XI",
        "partyID": "trader1",
        "price": 100,
        "size": 3,
        "side": "Sell",
        "TimeInForce": "GTC",
        "expiresAt": 1924991999000000000
        }
    }
}

```

* CANCEL_ORDER

Cancels an order - please note it's not currently possible to submit a successful order cancellation request.

```json
{
    "request": "CANCEL_ORDER",
    "message": {
    "@type": "api.CancelOrderRequest",
    "cancellation": {
        "orderID": "order1",
        "marketID": "JXGQYDVQAP5DJUAQBCB4PACVJPFJR4XI",
        "partyID": "trader1"
        }
    }
}

```

* AMEND_ORDER

Amends an order - please note it's not currently possible to submit a successful order amendment request.

```json

"request": "AMEND_ORDER",
"message": {
    "@type": "api.AmendOrderRequest",
    "amendment": {
        "orderID": "order1",
        "marketID": "JXGQYDVQAP5DJUAQBCB4PACVJPFJR4XI",
        "partyID": "trader1",
        "price": 100,
        "size": 3,
        "side": "Buy",
        "expiresAt": 1924991999000000000
        }
    }
}

```

* WITHDRAW

Withdraws a specified amount from party's account.

```json

{
    "request": "WITHDRAW",
    "message": {
    "@type": "api.WithdrawRequest",
    "withdraw": {
        "partyID": "trader1",
        "amount": 10,
        "asset": "BTC"
        }
    }
}

```

* MARKET_SUMMARY

Provides a summary for the specified market.

```json

{
    "request": "MARKET_SUMMARY",
    "message": {
    "@type": "core.MarketSummaryRequest",
    "marketID": "JXGQYDVQAP5DJUAQBCB4PACVJPFJR4XI"
    }
}

```

* SUMMARY

Provides a summary for all markets and parties

```json

{
    "request": "SUMMARY",
    "message": {
    "@type": "core.SummaryRequest"
    }
}

```

### Results

When invoking the `submit` command with the `--result` flag along with the output file name an output file gets generated per each instruction set specified. It contain the following elements:

* metadata

Spefies the number of instructions that were processed and omitted, number of trades generated and the time it took to process a given instruction set.

* Results

Displays an array of results. Each result displays the instruction that was submitted, an error (if it occured) and a request received (unless an error occured) - please note some instructions result in empty resposes, it is an expected behavior.

* Initial state

Displays the summary of all markets and parties prior to processing any instructions from a current set.

* Final state

Displays the summary of all markets and parties after all instructions from a current set have been processed.

* Config

Displays the configuration used during the execution of a given instruction set.

* Version

Displays information on the version used for the processing of the specified instruction set.

## Current limitations and planned modifications
