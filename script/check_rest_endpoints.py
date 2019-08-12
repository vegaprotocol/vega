#!/usr/bin/python3

import argparse
import json
import sys
import yaml


def parse_args():
    parser = argparse.ArgumentParser(
        description="Check gRPC REST bindings YAML against Swagger JSON")

    parser.add_argument(
        "--bindings", type=str,
        help="path to gRPC REST bindings YAML file")

    parser.add_argument(
        "--swagger", type=str,
        help="path to Swagger JSON file")

    return parser.parse_args()


def main():
    args = parse_args()

    try:
        # Try the newer loader. See https://msg.pyyaml.org/load
        bindings = yaml.load(open(args.bindings), Loader=yaml.FullLoader)
    except AttributeError:
        # Fall back on the old (exploitable) loader
        bindings = yaml.load(open(args.bindings))
    bindings_paths = sorted([
        (method, rule[method])
        for method in ["delete", "get", "post", "put"]
        for rule in bindings["http"]["rules"]
        if method in rule
    ])

    swagger = json.load(open(args.swagger))
    swagger_paths = sorted([
        (method, path)
        for path in swagger["paths"]
        for method in swagger["paths"][path]
    ])

    code = 0
    missing = set(swagger_paths) - set(bindings_paths)
    if len(missing) > 0:
        print("In Swagger but not Bindings: ", len(missing), missing)
        code += 1

    missing = set(bindings_paths) - set(swagger_paths)
    if len(missing) > 0:
        print("In Bindings but not Swagger: ", len(missing), missing)
        code += 1

    return code


if __name__ == "__main__":
    sys.exit(main())
