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

"""Execute the shipped scripts with deterministic HTTP and OCI CLI boundaries."""

import http.client
import json
import os
from pathlib import Path
import subprocess
import tempfile
from unittest.mock import Mock

import pytest
import yaml


@pytest.fixture
def task():
    path = Path(__file__).parents[1] / "base/prerelease_checks_oci.yaml"
    return yaml.safe_load(path.read_text())["spec"]


@pytest.fixture
def state(monkeypatch):
    for key, value in {
        "PACKAGE": "github.com/tektoncd/tekton-kueue",
        "VERSION_TAG": "v0.4.3",
        "GITHUB_TOKEN": "test-secret-never-log",
        "IMAGE_REPOSITORY": "",
        "RELEASE_BUCKET": "tekton-releases/tekton-kueue",
        "OCI_NAMESPACE": "",
    }.items():
        monkeypatch.setenv(key, value)
    return {
        "/user": (200, {"login": "tekton-robot"}),
        "/repos/tektoncd/tekton-kueue": (
            200, {"full_name": "tektoncd/tekton-kueue", "permissions": {"push": True}},
        ),
        "/repos/tektoncd/tekton-kueue/git/ref/tags/v0.4.3": (404, {}),
        "/repos/tektoncd/tekton-kueue/releases?per_page=100&page=1": (200, []),
    }


def execute(task, name):
    script = next(step["script"] for step in task["steps"] if step["name"] == name)
    exec(compile(script, f"<{name}>", "exec"), {})


@pytest.fixture
def github(monkeypatch, state):
    calls = []

    def connect(host, timeout):
        assert timeout > 0
        connection = Mock()

        def request(method, path, headers):
            assert method == "GET"
            if host == "api.github.com":
                assert headers["Authorization"] == "Bearer test-secret-never-log"
            else:
                assert host == "ghcr.io"
                assert "test-secret-never-log" not in str(headers)
            calls.append((host, path, headers))
            result = state[path]
            if isinstance(result, OSError):
                raise result
            status, body = result
            connection.getresponse.return_value.status = status
            connection.getresponse.return_value.read.return_value = (
                body if isinstance(body, bytes) else json.dumps(body).encode()
            )

        connection.request.side_effect = request
        return connection

    monkeypatch.setattr(http.client, "HTTPSConnection", connect)
    return calls


def test_absence_and_exact_release_matching(task, state, github):
    state["/repos/tektoncd/tekton-kueue/releases?per_page=100&page=1"] = (
        200, [{"tag_name": "v0x4x3"}, {"tag_name": "prefix-v0.4.3"}],
    )
    execute(task, "check-github-state")
    assert [path for _, path, _ in github][:2] == [
        "/user", "/repos/tektoncd/tekton-kueue",
    ]
    assert all(host == "api.github.com" for host, _, _ in github)


@pytest.mark.parametrize("draft", [False, True])
def test_matching_release_on_later_page_blocks(task, state, github, draft):
    base = "/repos/tektoncd/tekton-kueue/releases?per_page=100&page="
    state[base + "1"] = (200, [{"tag_name": "v0.1.0"}] * 100)
    state[base + "2"] = (200, [{"tag_name": "v0.4.3", "draft": draft}])
    with pytest.raises(SystemExit):
        execute(task, "check-github-state")
    assert github[-1][1] == base + "2"


@pytest.mark.parametrize("body", [{"ref": "refs/tags/v0.4.3"}, None, {}, []])
def test_existing_tag_blocks_even_with_unexpected_body(task, state, github, body):
    state["/repos/tektoncd/tekton-kueue/git/ref/tags/v0.4.3"] = (200, body)
    with pytest.raises(SystemExit):
        execute(task, "check-github-state")


@pytest.mark.parametrize("path", [
    "/user", "/repos/tektoncd/tekton-kueue",
    "/repos/tektoncd/tekton-kueue/git/ref/tags/v0.4.3",
    "/repos/tektoncd/tekton-kueue/releases?per_page=100&page=1",
])
@pytest.mark.parametrize("failure", [401, 403, 429, 500, 302, "network", "json"])
def test_github_lookup_failures_block(task, state, github, path, failure):
    if failure == "network":
        state[path] = OSError("connection failed")
    elif failure == "json":
        state[path] = (200, b"not json")
    else:
        state[path] = (failure, {})
    with pytest.raises(SystemExit):
        execute(task, "check-github-state")


@pytest.mark.parametrize("body", [
    {"login": "waveywaves"}, {}, None,
])
def test_wrong_or_missing_robot_identity_blocks(task, state, github, body):
    state["/user"] = (200, body)
    with pytest.raises(SystemExit):
        execute(task, "check-github-state")
    assert len(github) == 1


@pytest.mark.parametrize("body", [
    {}, {"full_name": "tektoncd/tekton-kueue", "permissions": {"push": False}},
    {"full_name": "other/repo", "permissions": {"push": True}},
])
def test_missing_repository_access_blocks(task, state, github, body):
    state["/repos/tektoncd/tekton-kueue"] = (200, body)
    with pytest.raises(SystemExit):
        execute(task, "check-github-state")
    assert len(github) == 2


@pytest.mark.parametrize(("key", "value"), [
    ("GITHUB_TOKEN", ""), ("PACKAGE", "github.com/owner/repo;echo bad"),
    ("PACKAGE", "evil.example/owner/repo"), ("PACKAGE", "github.com/owner/.."),
    ("VERSION_TAG", "v0.4.3\n"), ("VERSION_TAG", "$(echo bad)"),
    ("VERSION_TAG", "v0.4.3-.."),
    ("IMAGE_REPOSITORY", "ghcr.io/owner/repo:tag"),
    ("IMAGE_REPOSITORY", "ghcr.io/owner/repo/child"),
    ("IMAGE_REPOSITORY", "other.io/owner/repo"),
])
def test_invalid_inputs_block_before_http(task, state, github, monkeypatch, key, value):
    monkeypatch.setenv(key, value)
    with pytest.raises(SystemExit):
        execute(task, "check-github-state")
    assert not github


@pytest.mark.parametrize("body", [{}, None, [{"tag_name": None}], [None]])
def test_malformed_release_listing_blocks(task, state, github, body):
    state["/repos/tektoncd/tekton-kueue/releases?per_page=100&page=1"] = (200, body)
    with pytest.raises(SystemExit):
        execute(task, "check-github-state")


@pytest.mark.parametrize("path", [
    "/user", "/repos/tektoncd/tekton-kueue",
    "/repos/tektoncd/tekton-kueue/releases?per_page=100&page=1",
])
def test_non_resource_404_blocks(task, state, github, path):
    state[path] = (404, {})
    with pytest.raises(SystemExit):
        execute(task, "check-github-state")


@pytest.mark.parametrize("status", [200, 404, 401, 403, 500])
def test_public_manifest_lookup(task, state, github, monkeypatch, status):
    monkeypatch.setenv("IMAGE_REPOSITORY", "ghcr.io/tektoncd/tekton-kueue")
    state["/token?service=ghcr.io&scope=repository:tektoncd/tekton-kueue:pull"] = (
        200, {"token": "public-pull-token"},
    )
    state["/v2/tektoncd/tekton-kueue/manifests/v0.4.3"] = (
        status, {"errors": [{"code": "MANIFEST_UNKNOWN"}]} if status == 404 else {},
    )
    if status == 404:
        execute(task, "check-github-state")
    else:
        with pytest.raises(SystemExit):
            execute(task, "check-github-state")
    assert github[-1][2]["Authorization"] == "Bearer public-pull-token"


@pytest.mark.parametrize("body", [{}, None, {"errors": []}, {"errors": [{"code": "UNAUTHORIZED"}]}])
def test_unconfirmed_manifest_404_blocks(task, state, github, monkeypatch, body):
    monkeypatch.setenv("IMAGE_REPOSITORY", "ghcr.io/tektoncd/tekton-kueue")
    state["/token?service=ghcr.io&scope=repository:tektoncd/tekton-kueue:pull"] = (
        200, {"token": "public-pull-token"},
    )
    state["/v2/tektoncd/tekton-kueue/manifests/v0.4.3"] = (404, body)
    with pytest.raises(SystemExit):
        execute(task, "check-github-state")


@pytest.mark.parametrize("response", [
    (200, b"not json"), (200, None), OSError("network failed"),
])
def test_manifest_parse_and_network_failures_block(task, state, github, monkeypatch, response):
    monkeypatch.setenv("IMAGE_REPOSITORY", "ghcr.io/tektoncd/tekton-kueue")
    state["/token?service=ghcr.io&scope=repository:tektoncd/tekton-kueue:pull"] = (
        200, {"token": "public-pull-token"},
    )
    state["/v2/tektoncd/tekton-kueue/manifests/v0.4.3"] = response
    with pytest.raises(SystemExit):
        execute(task, "check-github-state")


@pytest.mark.parametrize("response", [
    (403, {}), (500, {}), (404, {}), (200, {}), (200, {"token": ""}),
    (200, b"not json"), OSError("network failed"),
])
def test_pull_token_failures_block(task, state, github, monkeypatch, response):
    monkeypatch.setenv("IMAGE_REPOSITORY", "ghcr.io/tektoncd/tekton-kueue")
    state["/token?service=ghcr.io&scope=repository:tektoncd/tekton-kueue:pull"] = response
    with pytest.raises(SystemExit):
        execute(task, "check-github-state")
    assert "/manifests/" not in github[-1][1]


@pytest.fixture
def storage(monkeypatch, tmp_path, state):
    source = tmp_path / "credentials"
    source.mkdir()
    for name in ("fingerprint", "tenancy_ocid", "user_ocid", "region", "oci_api_key.pem"):
        (source / name).write_text("test-private-value")
    scratch = tmp_path / "scratch"
    scratch.mkdir()
    monkeypatch.setattr(tempfile, "tempdir", str(scratch))
    monkeypatch.setenv("OCI_CREDENTIALS", str(source))
    results = {
        "ns": (0, '{"data":"namespace"}', ""),
        "bucket": (0, '{"data":{"name":"tekton-releases"}}', ""),
        "object": (1, "", 'ServiceError:\n{"status":404,"code":"ObjectNotFound"}'),
    }
    calls = []

    def run(args, **kwargs):
        config = Path(args[args.index("--config-file") + 1])
        assert config.stat().st_mode & 0o777 == 0o600
        assert config.parent.stat().st_mode & 0o777 == 0o700
        assert (config.parent / "oci_api_key.pem").stat().st_mode & 0o777 == 0o600
        assert "--no-retry" in args and kwargs["timeout"] > 0
        assert kwargs["capture_output"] and kwargs["text"]
        operation = args[args.index("os") + 1]
        calls.append(args)
        if operation == "object":
            name = args[args.index("--name") + 1]
            result = results.get(name, results[operation])
        else:
            result = results[operation]
        if isinstance(result, OSError):
            raise result
        return subprocess.CompletedProcess(args, *result)

    monkeypatch.setattr(subprocess, "run", run)
    previous_umask = os.umask(0o077)
    yield results, calls
    os.umask(previous_umask)
    assert not list(scratch.iterdir())


@pytest.mark.parametrize("namespace", ["", "explicit_namespace"])
def test_oci_object_absence_and_namespace_selection(task, storage, monkeypatch, namespace):
    results, calls = storage
    monkeypatch.setenv("OCI_NAMESPACE", namespace)
    execute(task, "check-release-file")
    assert any("ns" in args for args in calls) == (not namespace)
    objects = [args for args in calls if "object" in args]
    assert {args[args.index("--name") + 1] for args in objects} == {
        f"tekton-kueue/previous/v0.4.3/{name}"
        for name in ("release.yaml", "release.notags.yaml", "SHA256SUMS")
    }
    bucket_index = next(i for i, args in enumerate(calls) if "bucket" in args)
    assert all(i > bucket_index for i, args in enumerate(calls) if "object" in args)
    assert all(args[args.index("--namespace-name") + 1] == (namespace or "namespace") for args in objects)


@pytest.mark.parametrize("filename", ["release.notags.yaml", "SHA256SUMS"])
def test_partial_versioned_artifact_blocks(task, storage, filename):
    results, _ = storage
    results[f"tekton-kueue/previous/v0.4.3/{filename}"] = (0, "", "")
    with pytest.raises(SystemExit):
        execute(task, "check-release-file")


@pytest.mark.parametrize(("status", "code"), [
    (404, "ObjectNotFound"), (404, "NotFound"), (404, "NotAuthorizedOrNotFound"),
    (401, "NotAuthenticated"), (403, "NotAuthorized"), (429, "TooManyRequests"),
    (500, "InternalError"), (403, "ObjectNotFound"),
])
def test_only_known_object_absence_passes(task, storage, status, code):
    results, _ = storage
    results["object"] = (1, "", "ServiceError:\n" + json.dumps({"status": status, "code": code}))
    if status == 404 and code in ("ObjectNotFound", "NotFound"):
        execute(task, "check-release-file")
    else:
        with pytest.raises(SystemExit):
            execute(task, "check-release-file")


@pytest.mark.parametrize("operation", ["ns", "bucket", "object"])
@pytest.mark.parametrize("failure", ["network", "malformed", "http", "unstructured"])
def test_oci_failures_block_and_cleanup(task, storage, operation, failure):
    results, _ = storage
    if failure == "network":
        results[operation] = OSError("network failure")
    elif failure == "malformed":
        results[operation] = (
            (1, "", "ServiceError: not json") if operation == "object" else (0, "not json", "")
        )
    elif failure == "http":
        results[operation] = (1, "", 'ServiceError:\n{"status":403,"code":"NotAuthorized"}')
    else:
        results[operation] = (1, "", "lookup failed")
    with pytest.raises((SystemExit, OSError, json.JSONDecodeError)):
        execute(task, "check-release-file")


def test_existing_object_blocks_and_cleanup(task, storage):
    results, _ = storage
    results["object"] = (0, "", "")
    with pytest.raises(SystemExit):
        execute(task, "check-release-file")


@pytest.mark.parametrize("operation", ["ns", "bucket"])
def test_missing_bucket_or_namespace_does_not_mean_object_absence(task, storage, operation):
    results, calls = storage
    results[operation] = (1, "", 'ServiceError:\n{"status":404,"code":"NotFound"}')
    with pytest.raises(SystemExit):
        execute(task, "check-release-file")
    assert all("object" not in args for args in calls)


@pytest.mark.parametrize("filename", ["fingerprint", "oci_api_key.pem"])
def test_missing_credentials_cleanup(task, storage, monkeypatch, filename):
    (Path(os.environ["OCI_CREDENTIALS"]) / filename).unlink()
    with pytest.raises(FileNotFoundError):
        execute(task, "check-release-file")

def test_transport_error_cannot_log_credentials(task, state, github):
    state["/user"] = OSError("test-secret-never-log")
    with pytest.raises(SystemExit) as error:
        execute(task, "check-github-state")
    assert "test-secret-never-log" not in str(error.value)
    assert error.value.__suppress_context__


def test_credential_and_contract_preservation(task, state, capsys, github, storage):
    execute(task, "check-github-state")
    execute(task, "check-release-file")
    captured = capsys.readouterr()
    assert "test-secret-never-log" not in captured.out + captured.err
    assert "test-private-value" not in captured.out + captured.err
    defaults = {param["name"]: param.get("default") for param in task["params"]}
    assert defaults == {
        "package": "github.com/tektoncd/pipeline", "versionTag": None,
        "releaseBucket": None, "ociNamespace": "", "imageRepository": "",
    }
    assert {workspace["name"] for workspace in task["workspaces"]} == {
        "source-to-release", "oci-credentials",
    }
    secret = task["steps"][0]["env"][0]["valueFrom"]["secretKeyRef"]
    assert secret == {"name": "github-token", "key": "GITHUB_TOKEN", "optional": True}
