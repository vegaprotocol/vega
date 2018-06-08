FROM alpine:latest

# VEGA working folder
WORKDIR /vega

# Copy the VEGA binary
COPY vega /vega/

# Run the VEGA blockchain
ENTRYPOINT ./vega --chain