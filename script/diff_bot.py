#!/usr/bin/python3

# This script is to be run on commit to trading-core develop branch.
#
# The script:
# 1. diffs the schema in trading-core against the one in client.
# 2. if there are no differences, the process stops, otherwise ...
# 3. create a branch in client with the schema.graphql from trading-core
# 4. create a merge request in client for that branch to develop

import argparse
import binascii
import difflib
import json
import requests
import urllib.parse
from collections import namedtuple
from typing import Any, Dict

MAX_DIFF_LEN = 3500

API_URL = "https://gitlab.com/api/v4"

# Doc: https://docs.gitlab.com/ee/api/
#        repository_files.html#get-file-from-repository
GET_FILE = "/projects/{id}/repository/files/{file_path}"

GitFile = namedtuple("GitFile", "project_id branch file_path")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="GitLab file diff-checker and merge request creator")

    # GitLab API

    parser.add_argument(
        "--token", type=str, required=True,
        help="File1: GitLab API token")

    # Files to diff

    parser.add_argument(
        "--project1", type=str, required=True,
        help="File1: GitLab numeric project ID and optional colon-separated "
        "project name")
    parser.add_argument(
        "--project2", type=str, required=True,
        help="File2: GitLab numeric project ID and optional colon-separated "
        "project name")

    parser.add_argument(
        "--branch1", type=str, required=True,
        help="File1: GitLab branch name")
    parser.add_argument(
        "--branch2", type=str, required=True,
        help="File2: GitLab branch name")

    parser.add_argument(
        "--file1", type=str, required=True,
        help="File1: GitLab file name")
    parser.add_argument(
        "--file2", type=str, required=True,
        help="File1: GitLab file name")

    # Slack

    parser.add_argument(
        "--slack-hookurl", type=str, default="",
        help="URL for Slack hook")
    parser.add_argument(
        "--slack-recipient", type=str, default="#devops-infra",
        help="Recipient for Slack message: #channel or @user")
    parser.add_argument(
        "--slack-icon", type=str, default=":robot:",
        help="Icon emoji, e.g. :thinking-face:, :thumbsup:")

    # Action

    parser.add_argument(
        "action", nargs="+", type=str,
        choices=["print_diff", "slack_notify"],
        help="Action")

    return parser.parse_args()


def get_file_for_branch(
        token: str, project_id: str, branch: str, filename: str
) -> Dict[str, Any]:
    if ":" in project_id:
        pid = project_id.split(":")[0]
    else:
        pid = project_id

    r = requests.get(
        API_URL + GET_FILE.format(
            id=pid, file_path=urllib.parse.quote_plus(filename)),
        headers={"PRIVATE-TOKEN": token},
        params={"ref": branch}
    )
    if r.status_code != 200:
        print("Error: Failed to get file: {}".format(r.text))
        print()
        print("cURL: curl --header \"PRIVATE-TOKEN: {}\" {}".format(
            token, r.url))
        exit(1)
    return r.json()


def diff_files(token: str, f1: GitFile, f2: GitFile) -> str:
    r1 = get_file_for_branch(token, f1.project_id, f1.branch, f1.file_path)
    r2 = get_file_for_branch(token, f2.project_id, f2.branch, f2.file_path)

    if r1["content_sha256"] == r2["content_sha256"]:
        return ""

    d = difflib.unified_diff(
        binascii.a2b_base64(r1["content"]).decode().splitlines(keepends=True),
        binascii.a2b_base64(r2["content"]).decode().splitlines(keepends=True),
        fromfile="{}:{}:{}".format(f1.project_id, f1.branch, f1.file_path),
        tofile="{}:{}:{}".format(f2.project_id, f2.branch, f2.file_path),
        fromfiledate="{}:{}".format(r1["commit_id"], r1["last_commit_id"]),
        tofiledate="{}:{}".format(r2["commit_id"], r2["last_commit_id"]),
        n=3, lineterm="\n")
    return "".join(d)


def slack_notify(hookurl: str, recipient: str, icon: str, text: str) -> None:
    req = {
        "channel": recipient,
        "icon_emoji": icon,
        "text": text,
        "username": "DiffBot"
    }
    r = requests.post(hookurl, json=req)
    if r.status_code != 200:
        print("Error: Failed to send Slack message: {}".format(r.text))
        print()
        print("cURL: curl -XPOST -d '{}' {}".format(json.dumps(req), r.url))
        exit(1)


def main() -> None:
    args = parse_args()

    if (
            args.project1 == args.project2 and
            args.branch1 == args.branch2 and
            args.file1 == args.file2
    ):
        print("No point diffing a file against itself.")
        exit(1)

    diff = diff_files(
        args.token,
        GitFile(args.project1, args.branch1, args.file1),
        GitFile(args.project2, args.branch2, args.file2))
    if diff == "":
        return

    for action in args.action:
        if action == "print_diff":
            print(diff)
        elif action == "slack_notify":
            if len(diff) > MAX_DIFF_LEN:
                d2 = diff[:MAX_DIFF_LEN]
                tr = "*diff truncated*"
            else:
                d2 = diff
                tr = ""
            t = (
                "Heads up: GraphQL schema differs between:\n\n"
                "- project `{}`, branch `{}`, file `{}`\n"
                "- project `{}`, branch `{}`, file `{}`\n\n"
                "```\n{}```\n{}"
            ).format(
                args.project1, args.branch1, args.file1,
                args.project2, args.branch2, args.file2, d2, tr)
            slack_notify(
                args.slack_hookurl, args.slack_recipient, args.slack_icon, t)


if __name__ == "__main__":
    main()
