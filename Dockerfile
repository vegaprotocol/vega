FROM alpine:latest

# VEGA working folder
WORKDIR /app

# Copy the VEGA binary
COPY vega /app/

# Run the VEGA blockchain
ENTRYPOINT ./vega --chain