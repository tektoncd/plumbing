#!/usr/bin/env python3
import json
import os
import sys

# Directories to exclude from Go project discovery.
# tools/ subdirectories contain tool dependency modules (e.g. golangci-lint),
# not buildable projects.
EXCLUDE_DIRS = {"vendor", "tools"}


def find_go_projects(root):
    """
    Discovers Go projects by finding directories containing a go.mod file,
    excluding the root module and directories in EXCLUDE_DIRS.

    Args:
        root (str): The root directory to search from.

    Returns:
        list: Sorted list of project paths relative to root.
    """
    projects = []

    for dirpath, dirnames, filenames in os.walk(root):
        # Prune excluded directories
        dirnames[:] = [d for d in dirnames if d not in EXCLUDE_DIRS]

        if "go.mod" not in filenames:
            continue

        rel = os.path.relpath(dirpath, root)
        if rel == ".":
            continue

        projects.append(rel)

    return sorted(projects)


def generate_github_matrix(root):
    """
    Generates a GitHub Actions workflow matrix JSON from discovered Go projects.

    Args:
        root (str): The root directory to search from.

    Returns:
        str: A JSON string representing the GitHub Actions matrix.
    """
    projects = find_go_projects(root)
    return json.dumps({"project": projects}, indent=2)


if __name__ == "__main__":
    root = sys.argv[1] if len(sys.argv) > 1 else "."
    print(generate_github_matrix(root))
