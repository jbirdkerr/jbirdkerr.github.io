#!/usr/bin/env python3
"""
scripts/detect_changed_units.py

Reads deploy/units.toml and determines which units are affected by git changes.
Supports GitHub Actions output variables and topological dependency ordering.
"""

import argparse
import fnmatch
import json
import os
import subprocess
import tomllib


def load_manifest(manifest_path="deploy/units.toml"):
    with open(manifest_path, "rb") as f:
        data = tomllib.load(f)
    return data.get("units", {})


def get_changed_files(base_ref=None, head_ref=None):
    if not base_ref:
        files = set()
        try:
            diff = subprocess.check_output(
                ["git", "diff", "--name-only", "HEAD"], text=True
            ).strip()
            if diff:
                files.update(diff.splitlines())
            untracked = subprocess.check_output(
                ["git", "ls-files", "--others", "--exclude-standard"], text=True
            ).strip()
            if untracked:
                files.update(untracked.splitlines())
            if files:
                return sorted(files)
            # If nothing dirty, compare against HEAD~1 if git history exists
            diff_head = subprocess.check_output(
                ["git", "diff", "--name-only", "HEAD~1", "HEAD"], text=True
            ).strip()
            return sorted(diff_head.splitlines()) if diff_head else []
        except subprocess.CalledProcessError:
            return sorted(files)

    cmd = ["git", "diff", "--name-only", base_ref]
    if head_ref:
        cmd.append(head_ref)
    try:
        output = subprocess.check_output(cmd, text=True).strip()
        return sorted(output.splitlines()) if output else []
    except subprocess.CalledProcessError:
        return []


def matches_patterns(filepath, patterns):
    for pattern in patterns:
        if fnmatch.fnmatch(filepath, pattern):
            return True
        if pattern.endswith("/**"):
            prefix = pattern[:-3]
            if filepath.startswith(prefix + "/") or filepath == prefix:
                return True
    return False


def detect_affected_units(units, changed_files):
    directly_affected = set()
    for unit_name, config in units.items():
        patterns = config.get("paths", [])
        for f in changed_files:
            if matches_patterns(f, patterns):
                directly_affected.add(unit_name)
                break

    # Transitive dependents: if a dependency changed, pull in dependents
    affected = set(directly_affected)
    prev_len = -1
    while len(affected) != prev_len:
        prev_len = len(affected)
        for unit_name, config in units.items():
            for dep in config.get("depends_on", []):
                if dep in affected:
                    affected.add(unit_name)

    return directly_affected, affected


def get_topological_order(units, affected_units):
    order = []
    visited = set()
    visiting = set()

    def visit(unit):
        if unit in visiting:
            raise ValueError(f"Cyclic dependency detected: {unit}")
        if unit not in visited:
            visiting.add(unit)
            for dep in units.get(unit, {}).get("depends_on", []):
                if dep in affected_units:
                    visit(dep)
            visiting.remove(unit)
            visited.add(unit)
            order.append(unit)

    for u in sorted(affected_units):
        if u not in visited:
            visit(u)
    return order


def main():
    parser = argparse.ArgumentParser(
        description="Detect changed monorepo units from deploy/units.toml."
    )
    parser.add_argument("--base", default=None, help="Base git commit/ref")
    parser.add_argument("--head", default=None, help="Head git commit/ref")
    parser.add_argument("--manifest", default="deploy/units.toml", help="Path to units manifest")
    parser.add_argument("--all", action="store_true", help="Force treat all units as affected")
    parser.add_argument(
        "--github-output", action="store_true", help="Write step outputs to GITHUB_OUTPUT"
    )
    parser.add_argument("--json", action="store_true", help="Print result as JSON")
    args = parser.parse_args()

    units = load_manifest(args.manifest)

    if args.all:
        directly_affected = set(units.keys())
        affected = set(units.keys())
        changed_files = []
    else:
        changed_files = get_changed_files(args.base, args.head)
        directly_affected, affected = detect_affected_units(units, changed_files)

    deploy_order = get_topological_order(units, affected)

    result = {
        "changed_files_count": len(changed_files),
        "directly_affected": sorted(directly_affected),
        "affected": sorted(affected),
        "deploy_order": deploy_order,
        "units": {u: (u in affected) for u in units.keys()},
    }

    if args.json:
        print(json.dumps(result, indent=2))
    else:
        print(f"Changed files: {len(changed_files)}")
        print(f"Directly affected units: {', '.join(sorted(directly_affected)) or 'none'}")
        print(f"Total affected units (with dependents): {', '.join(sorted(affected)) or 'none'}")
        print(f"Deployment order: {' -> '.join(deploy_order) or 'none'}")

    if args.github_output and "GITHUB_OUTPUT" in os.environ:
        output_file = os.environ["GITHUB_OUTPUT"]
        with open(output_file, "a") as f:
            f.write(f"any_changed={'true' if affected else 'false'}\n")
            f.write(f"deploy_order={json.dumps(deploy_order)}\n")
            for u in units.keys():
                key = u.replace("-", "_")
                val = "true" if u in affected else "false"
                f.write(f"{key}={val}\n")


if __name__ == "__main__":
    main()
