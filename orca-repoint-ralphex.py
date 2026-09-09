#!/usr/bin/env python3
"""re-point orca's ralphex repo identity from umputun/ralphex to SvetlovA/ralphex.

WHY: orca detected C:/Dev/ralphex as a fork and pinned PR lookups to the
`upstream` remote (umputun/ralphex). PR #2 lives on the fork itself
(SvetlovA/ralphex), so orca finds no PR for branch `windows` and offers to
create one instead. this rewrites the repo/project identity to the `personal`
remote so orca queries the fork.

MUST BE RUN WITH ORCA FULLY CLOSED. orca holds this state in memory and
rewrites the file on every state change, so an edit applied while it runs is
silently clobbered. the script refuses to write if it sees orca running.

usage:
    python orca-repoint-ralphex.py --dry-run   # show changes, write nothing
    python orca-repoint-ralphex.py             # apply
    python orca-repoint-ralphex.py --verify    # report current identity only
"""

import json
import shutil
import subprocess
import sys
import time
from pathlib import Path

STORE = Path.home() / "AppData/Roaming/orca/profiles/local-default/orca-data.json"

REPO_PATH = "C:/Dev/ralphex"
OLD_PROJECT_ID = "github:umputun/ralphex"
NEW_PROJECT_ID = "github:SvetlovA/ralphex"

NEW_OWNER = "SvetlovA"
NEW_REPO = "ralphex"
NEW_HOST = "github.com"
NEW_REMOTE_NAME = "personal"
NEW_REMOTE_URL = "https://github.com/SvetlovA/ralphex.git"
NEW_CANONICAL = "github.com/SvetlovA/ralphex"
NEW_ICON_SRC = "https://github.com/SvetlovA.png?size=64"
NEW_ICON_LABEL = "SvetlovA/ralphex"

dry_run = "--dry-run" in sys.argv
verify_only = "--verify" in sys.argv


def orca_running():
    """true if any Orca process is alive; the write is unsafe while it is."""
    try:
        out = subprocess.run(
            ["tasklist", "/FI", "IMAGENAME eq Orca.exe", "/NH"],
            capture_output=True,
            text=True,
            timeout=20,
        ).stdout
    except (OSError, subprocess.SubprocessError):
        return None  # cannot tell - caller decides
    return "Orca.exe" in out


def load():
    if not STORE.exists():
        sys.exit(f"store not found: {STORE}")
    return json.loads(STORE.read_text(encoding="utf-8"))


def report(data):
    for repo in data.get("repos", []):
        if repo.get("path") == REPO_PATH:
            print(f"repo path:  {repo['path']}")
            print(f"  identity: {repo.get('gitRemoteIdentity')}")
            print(f"  upstream: {repo.get('upstream')}")
            print(f"  baseRef:  {repo.get('worktreeBaseRef')}")
            break
    else:
        print(f"no repo record for {REPO_PATH}")
    for project in data.get("projects", []):
        if project.get("id") in (OLD_PROJECT_ID, NEW_PROJECT_ID):
            print(f"project id: {project['id']}")
    for key, meta in data.get("worktreeMeta", {}).items():
        if REPO_PATH in key or "ralphex" in key:
            print(
                f"worktree {key.split('::')[-1]}: "
                f"projectId={meta.get('projectId')} linkedPR={meta.get('linkedPR')}"
            )


def remote_identity():
    return {
        "canonicalKey": NEW_CANONICAL,
        "remoteName": NEW_REMOTE_NAME,
        "remoteUrl": NEW_REMOTE_URL,
    }


def icon():
    return {
        "type": "image",
        "src": NEW_ICON_SRC,
        "source": "github",
        "label": NEW_ICON_LABEL,
    }


def main():
    data = load()

    if verify_only:
        report(data)
        return

    running = orca_running()
    if running and not dry_run:
        sys.exit(
            "Orca is still running - close it completely (quit the app, not just\n"
            "the window) and re-run. writing now would be overwritten by Orca."
        )
    if running is None and not dry_run:
        print("warning: could not determine whether Orca is running", file=sys.stderr)

    print("BEFORE:")
    report(data)
    print()

    changes = []

    # 1. repos[] entry for C:/Dev/ralphex
    for repo in data.get("repos", []):
        if repo.get("path") == REPO_PATH:
            repo["gitRemoteIdentity"] = remote_identity()
            repo["repoIcon"] = icon()
            # the fork is the tracked repo now, so it has no upstream of its own
            repo["upstream"] = None
            changes.append(f"repos[{repo['id'][:8]}]: identity -> {NEW_CANONICAL}")
            break
    else:
        sys.exit(f"no repo record with path {REPO_PATH}")

    # 2. projects[] entry - id is a derived primary key, so it gets renamed
    for project in data.get("projects", []):
        if project.get("id") == OLD_PROJECT_ID:
            project["id"] = NEW_PROJECT_ID
            project["providerIdentity"] = {
                "provider": "github",
                "owner": NEW_OWNER,
                "repo": NEW_REPO,
                "host": NEW_HOST,
            }
            project["gitRemoteIdentity"] = remote_identity()
            project["repoIcon"] = icon()
            project["updatedAt"] = int(time.time() * 1000)
            changes.append(f"projects: id -> {NEW_PROJECT_ID}")
            break
    else:
        sys.exit(f"no project record with id {OLD_PROJECT_ID} (already repointed?)")

    # 3. projectHostSetups[] foreign key
    for setup in data.get("projectHostSetups", []):
        if setup.get("projectId") == OLD_PROJECT_ID:
            setup["projectId"] = NEW_PROJECT_ID
            setup["updatedAt"] = int(time.time() * 1000)
            changes.append(f"projectHostSetups[{setup['id'][:8]}]: projectId updated")

    # 4. worktreeMeta{} foreign keys (one per worktree, keyed repoId::path)
    for key, meta in data.get("worktreeMeta", {}).items():
        if meta.get("projectId") == OLD_PROJECT_ID:
            meta["projectId"] = NEW_PROJECT_ID
            changes.append(f"worktreeMeta[...{key.split('::')[-1]}]: projectId updated")

    # 5. worktreeMetaByIdentity{} - newer schema container
    for key, meta in data.get("worktreeMetaByIdentity", {}).items():
        if isinstance(meta, dict) and meta.get("projectId") == OLD_PROJECT_ID:
            meta["projectId"] = NEW_PROJECT_ID
            changes.append(f"worktreeMetaByIdentity[{key[:36]}]: projectId updated")

    print(f"{len(changes)} change(s):")
    for change in changes:
        print(f"  {change}")

    # orca writes compact single-line json; match it so nothing else is touched
    serialized = json.dumps(data, separators=(",", ":"), ensure_ascii=False)

    leftover = serialized.count(OLD_PROJECT_ID)
    if leftover:
        sys.exit(f"aborting: {leftover} reference(s) to {OLD_PROJECT_ID} remain")

    if dry_run:
        print("\ndry run - nothing written")
        return

    backup = STORE.with_name(f"orca-data.json.bak-{time.strftime('%Y%m%d-%H%M%S')}")
    shutil.copy2(STORE, backup)

    tmp = STORE.with_suffix(".json.tmp")
    tmp.write_text(serialized, encoding="utf-8")
    tmp.replace(STORE)

    print(f"\nbackup:  {backup}")
    print(f"written: {STORE}")
    print("\nAFTER:")
    report(load())
    print("\nnow start Orca and open the Windows Support worktree.")
    print("restore with:  copy /Y \"%s\" \"%s\"" % (backup, STORE))


if __name__ == "__main__":
    main()
