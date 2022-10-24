# Conventions Followed in `datanode/api` Code Implementation

## Errors

For all types of errors that are handled in the datanode/api scope, please use errors part of the package.
Example:

```
return nil, apiError(codes.InvalidArgument, ErrMalformedRequest)
```

The purpose is to have a common way of handling error reporting outside of the core component and keep them clean and simple.
Some gradual refactoring work on places where the above is not yet applied is welcomed, though as a friendly and respectful reminder - please consider not breaking the current state.

### 
