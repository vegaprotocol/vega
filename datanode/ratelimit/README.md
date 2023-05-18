# Rate Limiting

To prevent abuse of the APIs provided by datanode, we provide optional tooling to limit the rate
of API requests.

Currently the rate limiting is simply applied on a per-remote-ip address basis, though we may implement
more sophisticated methods in the future.

The rate limiting mechanism is based on a [token bucket](https://en.wikipedia.org/wiki/Token_bucket) algorithm.

The idea is that:

- Each IP address which connects to datanode is assigned a `bucket` of `tokens`
- That bucket has a maximum capacity and is initially full of `tokens`
- Each API request costs one token, which is removed from the bucket when the call is made
- Datanode adds some number of tokens each seconds (the `rate` of the limiter) to the bucket _up to it's maximum capacity_

On average over time, this enforces the average rate of API requests to not exceed `rate` requests/second. However it allows temporary periods of more intensive use; the maximum rate being to use the entire capacity of the bucket within one second. For this reason the capacity of the bucket is called the `burst`.

There is an [IETF RFC](https://datatracker.ietf.org/doc/html/draft-ietf-httpapi-ratelimit-headers) for clients to get information about the rate limiting status. We implement this by sending the following headers in each API response (success or failure)

- `RateLimit-Limit` The maximum request limit within the time window (1s).
- `RateLimit-Reset` The rate-limiter time window duration in seconds (always 1s).
- `RateLimit-Remaining` The remaining tokens.

Upon rejection, the following HTTP response headers are available to users:

- `X-Rate-Limit-Limit` The maximum request limit.
- `X-Rate-Limit-Duration` The rate-limiter duration.
- `X-Rate-Limit-Request-Forwarded-For` The rejected request X-Forwarded-For.
- `X-Rate-Limit-Request-Remote-Addr` The rejected request RemoteAddr.

If a client continues to make requests despite having no tokens available, the response will be
 - HTTP `429` `StatusTooManyRequests` for HTTP APIs
 - gRPC `14` Unavailable for the gRPC API.

**Each unsuccessful response will deduct a token from a separate bucket** with the same refill rate and capacity as the before.

**Exhausting the supply of tokens in this second bucket will result in the client's IP address being banned for a period of time.**

If the user is banned the reponse will be 
 - HTTP 403 Forbidden for HTTP APIs
 - gRPC 14 Unavailable for the gRPC API.

## GraphQL, gRPC, and REST

You can currently configure `GraphQL` and `GRPC` rate limiting separately. `REST` inherits and shares the limits of the GRPC API.

There is a mechanism in places so that the `GRPC` `API` calls generated inside a `GraphQL` call are not rate limited a second time.

## Configuration

For example in datanode's `config.toml` the GRPC API rate limiting is configured
```
[API]
  [API.RateLimit]
    Enabled = true   # Set to false to disable rate limiting
    Rate = 10.0      # Refill rate of token bucket per second i.e. limit of average request rate
    Burst = 50       # Size of token bucket; maximum number of requests in short time window
    TTL = "1h0m0s"   # Time after which inactive token buckets are reset
    BanFor = "10m0s" # If IP continues to make requests after passing rate limit threshold,
                     # ban for this duration. Setting to 0 seconds prevents banning.
```

That configuration will apply to gRPC and REST. GraphQL is configured separately in
```
[Gateway]
  [Gateway.RateLimits]
    Enabled = true
    Rate = 10.0
    Burst = 50
    TTL = "1h0m0s"
    BanFor = "10m0s"
```

## WebSocket streams

WebSocket connections use a different rate limiting mechanism. They are rate limited by a maximum allowed number of subscriptions per IP address. The default maximum is set to 250 connections. This can be changed in the following section of the data node configuration:

```
[API]
  Level = "Info"
  ...
  MaxSubscriptionPerClient = 250

```
