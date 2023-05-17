# API docs style guide 

When writing docs annotations for the protos, refer to the following guidance for good practices and specific style choices. While some of the guidance below is specific to protos, the general good practice tips are applicable to all docs.

## Good practice tips

- Look at how other doc-strings are written and formatted as a guide
- Check your spelling for typos
- Make sure you didn’t just copy and paste from a different string
- Imagine trying to interact with the API as an end user, and write with that in mind

## Title structure

Text should match the rpc endpoints and have the titles like:
- “List Deposits” instead of “Deposits list”
- “Get Deposit” instead of “Deposit”

When adding a new API, if the title seems a bit odd, reconsider the rpc endpoint name. “Get governances” for example, is not a real word and isn’t a helpful title.

## Naming terminology

- **Get**: Used when API is for requesting a single data point. Get order; Get latest trade.
- **List**: Used when API is for requesting multiple data points. List orders; List trades; List network parameters. 
- **Estimate**: Used when API is requesting approximate information based on other pieces of data. Estimate margin; Estimate fee.
- **Export**: Used when API will provide a CSV file output of the data, or potentially other output formats.
- **Observe**: Used for WebSocket only, to denote streaming/subscription. Observe trades; Observe candle data. 

If your new API doesn't fit into any of these categories then this is a prompt to start a discussion about the best name for it.

## What not to use, and what to use instead

- I
- You
- We
- Id, id, or identifier
  - *USE*: ID
- ID of X
  - *USE*: X ID. Example: Market ID, *not* ID of the market
- data-node, or datanode
  - *USE*: data node
- Pubkey 
  - *USE*: party ID, optionally can describe that party ID is the same as public key

## Wording

- Use the imperative verb. Example: “Get a list”, *not* “Gets a list”
- Use standardised information for pagination connection. 
  - Example: “Page of positions data and corresponding page information.”, *not* “List of 0 or more positions.”
- The description should not be the title repeated. 
  - Example: “Get deposit”, should say more than just “Get deposit.” as a summary.

## Things to avoid

- Statements in parentheses 
  - Example: ”Get the current network limits, such as if bootstrapping is finished, and if proposals are enabled, etc.”, *not* “Get the current network limits (is bootstrapping finished, are proposals enabled etc..)”
- Beginning descriptions with A/an/the
  - Example: “Name of the node operator”, *not* “The name of the node operator”

## Capitalisation

- Capitalise Vega. If referring to the token, use VEGA
- REST, GraphQL, gRPC, Tendermint (or CometBFT), ERC-20, etc.
- CSV
- ERC-20. In some cases it’s already coded as ERC20 so that is an exception, but it should be capitalised, at a minimum
- Unless it’s a specific project name, transport, or registered name, it does not need to be capitalised
- *Do not capitalise* protocol upgrade, trade, data node, validator, node operator, core, etc.. 

## Add helpful guidance for users

- If a returned value is a string that represents an integer, ensure that it describes if it is a signed or unsigned integer. Note: If the field is described as a number but the data type is a string, it is so that there is no loss of precision, or risk of integer overflow for large numbers.
- If it’s a string that represents a decimalised number, describe how someone can determine what the decimal place is.

## Formatting protos

Use correct formatting when adding new API endpoints to the protos: 

- Required fields  
- Tags - Choose from existing tags before adding a new tag. Tags are used to group similar APIs together in the REST documentation. If your new API doesn't fit into any of the existing tags, then this is a prompt to start a discussion about the best category for it.
- Titles & descriptions 
- Use a full stop for a description with no title. 

### How to format a proto API

```
service ColourService {
  // Get colour
  //
  // Get the first colour that this API offers
  rpc GetColour(GetColourRequest) returns (GetColourResponse) {}
  // List colours
  //
  // List all the colours available
  rpc ListColours(ListColoursRequest) returns (ListColoursResponse) {}
}

// Request for more information about colours.
//
// The response with details for each colour.
message ListColoursRequest {
   // Determines if colours that use blue should be shown. 
   bool with_blue = 1;
   
   // There are lots of colours, so you can paginate the results.
   optional Pagination pagination = 2;
}
```

### Generic format for title and description

``` 
#### example

// This is a title, which needs to have a blank line below it
//
// This is a description
// You can add more descriptions on further lines 
string my_field = 1;

#### next example

// This is a title, and has no full stop at the end

#### next example

// This is a description, and it needs needs a full stop at the end.
string my_field =1;
```

### Required and optional fields
All fields in an API request should either be required and marked with “[(google.api.field.behaviour = REQUIRED)]”, or optional and marked explicitly as “optional” against the type in the protos. 

When describing a field in a doc-string, it does not need to explicitly say whether it is optional or required. It is implied by the annotations. 

Example: *Do not* start a doc-string with “Optionally….”, or end one with “is required.”

*If a field is optional* describe what happens if it is not set, but only if it is not obvious. 

Example: An optional field that filters the returned list. It can be made clear in the description (words like “restrict” and “filter”) that not providing a value will return them all. On the other hand, an optional field that takes an epoch-seq will need to state somewhere that not setting it will return the data for latest epoch.

*Do not* use fields that do not use “optional” but where the default value means “not set/ignored”, for example:

```
message NewAPIRequest {

    // A required field. Being set to “” is a VALIDATION ERROR
    string market_id = 1; [(google.api.field.behaviour = REQUIRED)]

    // An optional field. 
    optional string party_id = 2;

    // DO NOT DO THIS
    // Optional asset_id, if empty will not filter on asset_id
    string asset_id = 3;

}

```
