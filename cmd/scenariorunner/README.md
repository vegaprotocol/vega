# `scenario-runner-cli`

This document provides the overview of the `scenario-runner-cli` tool. It starts with a simple how-to guide showing how the tool can be used. The following sections list the currently available instructions and their syntax. It finishes with a discussion of the current limitations of the tool and the planned modifications.

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

* The step above should result in creation of the `exampleResult.json` file. More information of the structure of the instruction set and the result file can be found in the ???? section below.

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
