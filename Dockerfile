FROM alpine:latest

# VEGA working folder
WORKDIR /bin

# Copy the VEGA binary
COPY vega /bin/

# Run the VEGA blockchain
ENTRYPOINT ./vega --chain