#!/usr/bin/python3

import argparse
import binascii
import datetime
import difflib
import json
import requests
import urllib.parse
from collections import namedtuple
from typing import Any, Dict, Tuple

API_URL = "https://gitlab.com/api/v4"

MAX_SLACK_DIFF_LEN = 1500

GitFile = namedtuple("GitFile", "project_id project_name branch file_path")


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
        help="File1: GitLab numeric project ID")
    parser.add_argument(
        "--project2", type=str, required=True,
        help="File2: GitLab numeric project ID")

    parser.add_argument(
        "--projectname1", type=str, required=False,
        help="File1: GitLab project name")
    parser.add_argument(
        "--projectname2", type=str, required=True,
        help="File2: GitLab project name")

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
        choices=["print_diff", "slack_notify", "create_mr"],
        help="Action")

    return parser.parse_args()


def headers(token: str) -> Dict[str, Any]:
    return {
        "PRIVATE-TOKEN": token
    }


def get_file_for_branch(
    token: str, project_id: str, branch: str, filename: str
) -> Dict[str, Any]:

    # Check for branch existence
    if branch not in ["master", "develop"]:
        r = requests.get(
            API_URL + "/projects/{id}/repository/branches".format(
                id=project_id),
            headers=headers(token),
            params={"search": branch}
        )
        if r.status_code != 200:
            print(
                "Error: Failed to get list of branches for project {}: "
                "{}".format(project_id, r.text))
            print("cURL: curl --header \"PRIVATE-TOKEN: {}\" {}".format(
                token, r.url))
            exit(1)

        if branch not in [b["name"] for b in r.json()]:
            print(
                "Error: Branch does not exist in project {}: {}".format(
                    project_id, branch))
            exit(1)

    # Get file
    r = requests.get(
        API_URL + "/projects/{id}/repository/files/{file_path}".format(
            id=project_id, file_path=urllib.parse.quote_plus(filename)),
        headers=headers(token),
        params={"ref": branch}
    )
    if r.status_code != 200:
        print("Error: Failed to get file: {}".format(r.text))
        print("cURL: curl --header \"PRIVATE-TOKEN: {}\" {}".format(
            token, r.url))
        exit(1)
    return r.json()


def diff_files(token: str, f1: GitFile, f2: GitFile) -> Tuple[str, str, str]:
    r1 = get_file_for_branch(token, f1.project_id, f1.branch, f1.file_path)
    r2 = get_file_for_branch(token, f2.project_id, f2.branch, f2.file_path)

    if r1["content_sha256"] == r2["content_sha256"]:
        return ("", "-", "-")

    d = difflib.unified_diff(
        binascii.a2b_base64(r1["content"]).decode().splitlines(keepends=True),
        binascii.a2b_base64(r2["content"]).decode().splitlines(keepends=True),
        fromfile="{}:{}:{}:{}".format(
            f1.project_id, f1.project_name, f1.branch, f1.file_path),
        tofile="{}:{}:{}:{}".format(
            f2.project_id, f2.project_name, f2.branch, f2.file_path),
        fromfiledate="{}:{}".format(r1["commit_id"], r1["last_commit_id"]),
        tofiledate="{}:{}".format(r2["commit_id"], r2["last_commit_id"]),
        n=3, lineterm="\n")
    return ("".join(d), r1["content"], r2["content"])


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
        print("cURL: curl -XPOST -d '{}' {}".format(json.dumps(req), r.url))
        exit(1)


def basename(fn: str) -> str:
    return fn.split("/")[-1] if "/" in fn else fn


def update_file_on_new_branch(
    token: str, f: GitFile, newbranch: str, content64: str
) -> Dict[str, Any]:

    req = {
        "id": f.project_id,
        "branch": newbranch,
        "commit_message": "Update file",
        "start_branch": f.branch,
        "actions": [
            {
                "action": "update",
                "file_path": f.file_path,
                "content": content64,
                "encoding": "base64"
            }
        ]
    }
    r = requests.post(
        API_URL + "/projects/{id}/repository/commits".format(id=f.project_id),
        headers=headers(token),
        json=req
    )
    if r.status_code >= 400:
        print("Error: Failed to update file: {}".format(r.text))
        exit(1)
    return r.json()


def create_mr(
    token: str, f1: GitFile, f2: GitFile, newcontent64: str, diff: str
) -> Dict[str, Any]:

    dt = datetime.datetime.utcnow().strftime("%Y-%m-%d-%H-%M-%S")
    newbranch = "diffbot/{}/{}".format(basename(f1.file_path), dt)

    update_file_on_new_branch(token, f1, newbranch, newcontent64)

    req = {
        "id": f1.project_id,
        "source_branch": newbranch,
        "target_branch": f1.branch,
        "title": "Update {} from {} at {}".format(
            basename(f1.file_path),
            f2.project_name, dt),
        "description": "Update `{}`.\n".format(f1.file_path),
        "labels": "diffbot",
        "remove_source_branch": True,
        "allow_collaboration": True
    }
    r = requests.post(
        API_URL + "/projects/{id}/merge_requests".format(id=f1.project_id),
        headers=headers(token),
        json=req
    )
    if r.status_code >= 400:
        print("Error: Failed to create merge request: {}".format(r.text))
        exit(1)

    rr = r.json()
    print("Created {}".format(rr["web_url"]))
    return rr


def main() -> None:
    args = parse_args()

    if (
        args.project1 == args.project2 and
        args.branch1 == args.branch2 and
        args.file1 == args.file2
    ):
        print("No point diffing a file against itself.")
        exit(1)

    f1 = GitFile(args.project1, args.projectname1, args.branch1, args.file1)
    f2 = GitFile(args.project2, args.projectname2, args.branch2, args.file2)
    (diff, content1, content2) = diff_files(args.token, f1, f2)
    if diff == "":
        return

    for action in args.action:
        if action == "print_diff":
            print(diff)
        elif action == "slack_notify":
            if len(diff) > MAX_SLACK_DIFF_LEN:
                d2 = diff[:MAX_SLACK_DIFF_LEN]
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
        elif action == "create_mr":
            create_mr(args.token, f1, f2, content2, diff)


if __name__ == "__main__":
    main()
