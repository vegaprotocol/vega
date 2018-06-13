#!/bin/bash

echo "Downloading Swagger Spec"
curl -o swagger.yaml https://gitlab.com/vega-protocol/trading-api-docs/raw/master/swagger.yaml?private_token=$GITLAB_PRIVATE_TOKEN -s

echo "Generating models"
swagger generate model -t api --skip-validation -q

grep operationId: swagger.yaml | while read -r line
do
  OPERATION="$(echo $line | cut -c 14-)"
  echo "Generating operation $OPERATION"
  swagger generate operation -n $OPERATION -t api/endpoints/ -q -s rest --skip-validation
done

rm -f swagger.yaml

echo "Done"