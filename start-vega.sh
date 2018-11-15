#!/bin/bash
PATH=/usr/local/bin:/usr/bin:/bin
mkdir -p ./log/vega
chown -R vega:vega ./log/vega
exec ./current/vega -remove_expired_gtt=true -log_price_levels=false 2>&1 | multilog t s10485760 n100 ./log/vega 
