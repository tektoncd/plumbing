#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.12"
# dependencies = ["pyyaml"]
# ///
# Copyright 2026 The Tekton Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Generate Tide context_options from config/repo-checks.yaml.

Reads the shared repo-checks.yaml and updates the context_options
section in prow/control-plane/config.yaml. Only repos with non-empty
check lists get a required-contexts entry.

Usage:
    uv run hack/generate-tide-contexts.py [--verify]

    --verify  Check that the Prow config is in sync (exit 1 if not).
"""

import argparse
import os
import re
import sys

import yaml


REPO_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
CHECKS_FILE = os.path.join(REPO_ROOT, "config", "repo-checks.yaml")
PROW_CONFIG = os.path.join(REPO_ROOT, "prow", "control-plane", "config.yaml")

# Markers in the Prow config for the generated section
BEGIN_MARKER = "  # BEGIN GENERATED context_options — do not edit manually"
END_MARKER = "  # END GENERATED context_options"


def load_repo_checks():
    """Load repo-checks.yaml and return repos with non-empty check lists."""
    with open(CHECKS_FILE) as f:
        data = yaml.safe_load(f)

    repos = {}
    for repo, checks in sorted(data.get("repos", {}).items()):
        if checks:  # skip repos with empty lists
            repos[repo] = checks
    return repos


def generate_block(repos):
    """Generate the context_options YAML block with proper indentation."""
    lines = [
        BEGIN_MARKER,
        "  # Allow conditional/skipped GitHub Actions jobs to not block Tide",
        "  # from merging PRs. Only the contexts listed in required-contexts",
        "  # are required; all others are treated as optional.",
        "  # Source: config/repo-checks.yaml",
        "  context_options:",
        "    skip-unknown-contexts: true",
        "    orgs:",
        "      tektoncd:",
        "        repos:",
    ]

    for repo, checks in repos.items():
        lines.append(f"          {repo}:")
        lines.append("            required-contexts:")
        for check in checks:
            lines.append(f'            - "{check}"')

    lines.append(END_MARKER)
    return "\n".join(lines)


def update_prow_config(block, verify=False):
    """Update or verify the context_options section in the Prow config."""
    with open(PROW_CONFIG) as f:
        content = f.read()

    # Check if markers exist
    if BEGIN_MARKER in content and END_MARKER in content:
        # Replace between markers
        pattern = re.escape(BEGIN_MARKER) + r".*?" + re.escape(END_MARKER)
        new_content = re.sub(pattern, block, content, flags=re.DOTALL)
    else:
        # First run: insert before 'queries:' under 'tide:'
        # Find the context_options section or the queries section
        pattern = r"(tide:\n  sync_period: [^\n]+\n)(  context_options:.*?(?=\n  queries:)|\s*(?=  queries:))"
        new_content = re.sub(
            pattern,
            r"\1" + block + "\n",
            content,
            flags=re.DOTALL,
        )

    if verify:
        if content == new_content:
            print("OK: Tide context_options is in sync with config/repo-checks.yaml")
            return True
        else:
            print(
                "ERROR: Tide context_options is out of sync with config/repo-checks.yaml",
                file=sys.stderr,
            )
            print("Run: python3 hack/generate-tide-contexts.py", file=sys.stderr)
            return False
    else:
        with open(PROW_CONFIG, "w") as f:
            f.write(new_content)
        print(f"Updated {PROW_CONFIG}")
        return True


def main():
    parser = argparse.ArgumentParser(
        description="Generate Tide context_options from config/repo-checks.yaml"
    )
    parser.add_argument(
        "--verify",
        action="store_true",
        help="Verify the Prow config is in sync (exit 1 if not)",
    )
    args = parser.parse_args()

    repos = load_repo_checks()
    block = generate_block(repos)

    if not update_prow_config(block, verify=args.verify):
        sys.exit(1)


if __name__ == "__main__":
    main()
