#!/usr/bin/env python3
"""
Automated Feature Publisher for Multi-Agent Orchestrator.
Pushes branch to origin, creates PR via GitHub API, monitors CI checks, and merges on green.
"""

import sys
import os
import json
import time
import subprocess
import urllib.request
import urllib.error

REPO = "eeyung1/L2EStudyLink"
TOKEN_FILE = os.path.expanduser("~/.git-credentials")

def get_token():
    if os.path.exists(TOKEN_FILE):
        with open(TOKEN_FILE, "r") as f:
            for line in f:
                if "github.com" in line:
                    parts = line.strip().split("@github.com")[0].split(":")
                    if len(parts) >= 3:
                        return parts[2]
    return os.environ.get("GITHUB_TOKEN", "")

def github_api(endpoint, method="GET", data=None):
    token = get_token()
    url = f"https://api.github.com/repos/{REPO}/{endpoint.lstrip('/')}"
    headers = {
        "Authorization": f"token {token}",
        "Accept": "application/vnd.github.v3+json",
        "User-Agent": "Orchestrator-Publisher/1.0"
    }
    payload = json.dumps(data).encode("utf-8") if data else None
    req = urllib.request.Request(url, data=payload, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req) as resp:
            content = resp.read().decode("utf-8")
            return json.loads(content) if content else {}
    except urllib.error.HTTPError as e:
        err_body = e.read().decode("utf-8")
        try:
            return {"error": e.code, "detail": json.loads(err_body)}
        except Exception:
            return {"error": e.code, "detail": err_body}

def push_branch(branch, worktree_dir=None):
    cwd = worktree_dir if worktree_dir else os.getcwd()
    print(f"[*] Pushing branch '{branch}' to origin from '{cwd}'...")
    res = subprocess.run(["git", "push", "origin", branch], cwd=cwd, capture_output=True, text=True)
    if res.returncode != 0:
        print(f"[!] Push failed: {res.stderr}")
        return False
    print("[+] Branch pushed successfully.")
    return True

def create_pr(branch, title, body, base="main"):
    print(f"[*] Creating PR for '{branch}' -> '{base}'...")
    data = {
        "title": title,
        "head": branch,
        "base": base,
        "body": body
    }
    resp = github_api("pulls", method="POST", data=data)
    if "error" in resp:
        # Check if PR already exists
        if resp.get("error") == 422:
            print("[*] PR may already exist. Searching existing PRs...")
            prs = github_api(f"pulls?head=eeyung1:{branch}&state=open")
            if isinstance(prs, list) and len(prs) > 0:
                print(f"[+] Found existing open PR #{prs[0]['number']}")
                return prs[0]
        print(f"[!] Failed to create PR: {resp}")
        return None
    print(f"[+] Created PR #{resp['number']}: {resp['html_url']}")
    return resp

def wait_for_ci(commit_sha, max_wait_sec=600, poll_interval=15):
    print(f"[*] Monitoring CI runs for commit {commit_sha[:7]}...")
    start_time = time.time()
    while time.time() - start_time < max_wait_sec:
        runs = github_api(f"actions/runs?head_sha={commit_sha}")
        workflow_runs = runs.get("workflow_runs", [])
        if not workflow_runs:
            print(f"[*] Waiting for workflow to register... ({int(time.time() - start_time)}s)")
            time.sleep(poll_interval)
            continue
        
        all_completed = True
        all_success = True
        statuses = []
        for run in workflow_runs:
            status = run.get("status")
            conclusion = run.get("conclusion")
            name = run.get("name")
            statuses.append(f"{name}: {status}/{conclusion}")
            if status != "completed":
                all_completed = False
            elif conclusion != "success":
                all_success = False
        
        print(f"[*] Status: {', '.join(statuses)} ({int(time.time() - start_time)}s)")
        
        if all_completed:
            if all_success:
                print("[+] All CI checks PASSED!")
                return True
            else:
                print("[!] Some CI checks failed.")
                return False
        
        time.sleep(poll_interval)
    
    print("[!] Timed out waiting for CI.")
    return False

def merge_pr(pr_number, merge_method="merge"):
    print(f"[*] Merging PR #{pr_number}...")
    data = {
        "merge_method": merge_method
    }
    resp = github_api(f"pulls/{pr_number}/merge", method="PUT", data=data)
    if resp.get("merged"):
        print(f"[+] Successfully merged PR #{pr_number}! Merge SHA: {resp.get('sha')}")
        return resp.get("sha")
    else:
        print(f"[!] Failed to merge PR #{pr_number}: {resp}")
        return None

def main():
    if len(sys.argv) < 3:
        print("Usage: publish-feature.py <branch> <title> [body_file_or_text] [worktree_dir]")
        sys.exit(1)
    
    branch = sys.argv[1]
    title = sys.argv[2]
    body = ""
    if len(sys.argv) >= 4:
        arg = sys.argv[3]
        if os.path.exists(arg):
            with open(arg, "r") as f:
                body = f.read()
        else:
            body = arg
    worktree = sys.argv[4] if len(sys.argv) >= 5 else None

    # Step 1: Push branch
    if not push_branch(branch, worktree):
        sys.exit(1)

    # Step 2: Create PR
    pr = create_pr(branch, title, body)
    if not pr:
        sys.exit(1)

    commit_sha = pr["head"]["sha"]
    pr_number = pr["number"]

    # Step 3: Wait for CI
    if not wait_for_ci(commit_sha):
        print(f"[!] CI did not pass. PR #{pr_number} remains open for inspection.")
        sys.exit(1)

    # Step 4: Merge PR
    merge_sha = merge_pr(pr_number)
    if not merge_sha:
        sys.exit(1)

    print(f"\nSUCCESS! Feature branch {branch} merged into main at {merge_sha[:7]}.")

if __name__ == "__main__":
    main()

