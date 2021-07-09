# GraphQL

The query interface is accessible on `http://localhost:3004/`.

## Types

### Dates (time.Time)
* Serialization type: `String`

**Note**
Use `vegatime.Format(vegatime.UnixNano(myDate))` to properly convert it.

### How to's?

### Add new GraphQL type
1. Add the new type in `schema.graphql`
2. Add binding between golang model and GraphQL type in `gqlgen.yml`
3. Generate the GraphQL model and resolver with `make gqlgen`
4. Create a new golang file `my_type_resolver.go`
5. Implement the `MyTypeResolver` interface
    * This interface is located in `generated.go`
6. Add a method `MyType()` on struct `VegaResolverRoot` in `resolvers.go`, 
   as follows :

```golang
func (r *VegaResolverRoot) MyType() MyTypeResolver {
    return (*myTypeResolver)(r)
}
```

### Add a new query?
1. Add a new query in `schema.graphql` in the the `Query` type
2. Generate the GraphQL model and resolver with `make gqlgen`
3. Add a method `MyType()` on struct `myQueryResolver` in `resolvers.go`,
   as follows :

```golang
func (r *myQueryResolver) MyType() types.MyType {
	res, err := r.tradingDataClient.MyType(
		ctx, &protoapi.MyTypeRequest{Id: id},
    )
    if err != nil {
   	    return nil, err
    }
   
   return res.MyType, nil
}
```
