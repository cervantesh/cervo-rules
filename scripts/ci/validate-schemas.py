#!/usr/bin/env python
"""Validate every shipped document against the JSON Schemas we publish.

The schemas are the published contract for policy authors and third-party
tooling. Go has no JSON Schema implementation in its standard library, and
adding one would be a new runtime dependency, which the dependency policy in
docs/dependencies.md does not accept for this purpose. So the check lives here,
against an external implementation, and runs in CI.

What Go covers without a dependency is in internal/policygen/schemas_test.go:
the schema documents parse, every $ref resolves, and the operator and fact-type
enums match the ones the generator accepts. That catches drift between schema
and code. This script catches the other direction: a document the generator
accepts that the published contract would reject, or the reverse.

It also refuses an orphan. Publishing a schema is publishing a promise that
something emits this shape; three of ours promised documents that nothing
produced and nothing validated. So every schema on disk has to be accounted
for here -- by documents in the repository, or by naming the Go test that holds
its producer to it. A new schema with neither fails this script until somebody
decides which it is.

Exit codes:
  0  every document validates, or the check was skipped for missing tooling
  1  at least one document failed validation, or a schema is unaccounted for
  2  usage error

Set CERVORULES_REQUIRE_SCHEMA_VALIDATION=1 to turn a missing implementation
into a failure instead of a skip. CI sets it; a developer shell does not.
"""

import glob
import json
import os
import sys

REQUIRE = os.environ.get("CERVORULES_REQUIRE_SCHEMA_VALIDATION", "") == "1"

# Directories whose *.schema.json files are published contracts.
SCHEMA_DIRS = ("schemas/v3", ".cervorules/schemas")

# Schema -> globs of the documents in this repository that must satisfy it.
CONTRACTS = {
    "schemas/v3/policy-rules.schema.json": ["examples/**/policy-rules.yaml"],
    "schemas/v3/policy-vocabulary.schema.json": ["examples/**/policy-vocabulary.yaml"],
    ".cervorules/schemas/agent-manifest.schema.json": [".cervorules/agent-manifest.json"],
    ".cervorules/schemas/task-recipe.schema.json": [".cervorules/recipes/*.json"],
}

# Schema -> the Go test that holds its producer to it. These shapes are emitted
# by code rather than committed as documents, so there is nothing here to
# validate; what matters is that the check exists and is named.
CHECKED_IN_GO = {
    "schemas/v3/generated-policy-metadata.schema.json": "TestGeneratedPolicyMetadataMatchesItsSchema",
    "schemas/v3/policy-evaluation-report.schema.json": "TestPolicyEvaluationReportMatchesItsSchema",
}


def skip(reason: str) -> int:
    stream = sys.stderr if REQUIRE else sys.stdout
    print(f"schema validation {'FAILED' if REQUIRE else 'skipped'}: {reason}", file=stream)
    if REQUIRE:
        print("install with: python -m pip install pyyaml jsonschema", file=sys.stderr)
        return 1
    return 0


def load_document(path: str, yaml_module):
    with open(path, encoding="utf-8") as handle:
        if path.endswith(".json"):
            return json.load(handle)
        return yaml_module.safe_load(handle)


def check_every_schema_is_accounted_for(repo: str) -> int:
    """Report schemas that promise a shape nothing produces."""
    failures = 0
    for directory in SCHEMA_DIRS:
        pattern = os.path.join(repo, directory, "*.schema.json")
        for path in sorted(glob.glob(pattern)):
            relative = os.path.relpath(path, repo).replace(os.sep, "/")
            if relative in CONTRACTS or relative in CHECKED_IN_GO:
                continue
            failures += 1
            print(
                f"FAIL {relative} is published but unaccounted for.\n"
                "  Publishing a schema promises that something emits this shape.\n"
                "  Either add the documents it covers to CONTRACTS, name the Go\n"
                "  test that holds its producer to it in CHECKED_IN_GO, or retire\n"
                "  the schema.",
                file=sys.stderr,
            )
    # The Go annotation rots if the test it names is renamed or deleted, which
    # would leave a schema recorded as covered by nothing.
    for relative, test_name in sorted(CHECKED_IN_GO.items()):
        if not grep_go_test(repo, test_name):
            failures += 1
            print(
                f"FAIL {relative} names {test_name}, which no Go file declares.",
                file=sys.stderr,
            )
    return failures


def grep_go_test(repo: str, test_name: str) -> bool:
    needle = f"func {test_name}("
    for root, dirs, files in os.walk(repo):
        dirs[:] = [d for d in dirs if not d.startswith(".") or d == ".cervorules"]
        for name in files:
            if not name.endswith("_test.go"):
                continue
            with open(os.path.join(root, name), encoding="utf-8", errors="replace") as handle:
                if needle in handle.read():
                    return True
    return False


def main(argv: list[str]) -> int:
    if len(argv) > 2:
        print("usage: validate-schemas.py [repo-root]", file=sys.stderr)
        return 2
    repo = os.path.abspath(argv[1] if len(argv) == 2 else os.path.join(os.path.dirname(__file__), "..", ".."))

    try:
        import yaml  # noqa: PLC0415
        from jsonschema import Draft202012Validator  # noqa: PLC0415
    except ImportError as err:
        return skip(f"{err.name} is not installed")

    failures = check_every_schema_is_accounted_for(repo)
    checked = 0

    for schema_relative, patterns in sorted(CONTRACTS.items()):
        with open(os.path.join(repo, schema_relative), encoding="utf-8") as handle:
            schema = json.load(handle)
        # A malformed schema is silently permissive in some validators, so
        # check the schema itself before checking anything against it.
        Draft202012Validator.check_schema(schema)
        validator = Draft202012Validator(schema)

        documents = sorted({
            path
            for pattern in patterns
            for path in glob.glob(os.path.join(repo, pattern), recursive=True)
        })
        if not documents:
            failures += 1
            print(f"FAIL {schema_relative} covers no documents; its globs matched nothing", file=sys.stderr)
            continue

        for path in documents:
            checked += 1
            errors = sorted(validator.iter_errors(load_document(path, yaml)), key=lambda error: list(error.path))
            if not errors:
                continue
            failures += 1
            relative = os.path.relpath(path, repo).replace(os.sep, "/")
            print(f"FAIL {relative} against {schema_relative}", file=sys.stderr)
            for error in errors:
                location = ".".join(str(part) for part in error.path) or "<root>"
                print(f"  at {location}: {error.message}", file=sys.stderr)

    print(f"schema validation: {checked} documents checked, {failures} failing")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
