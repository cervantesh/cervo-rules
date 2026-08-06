#!/usr/bin/env python
"""Validate every shipped policy YAML against the v3 JSON Schemas.

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

Exit codes:
  0  every document validates, or the check was skipped for missing tooling
  1  at least one document failed validation
  2  usage error

Set CERVORULES_REQUIRE_SCHEMA_VALIDATION=1 to turn a missing implementation
into a failure instead of a skip. CI sets it; a developer shell does not.
"""

import glob
import json
import os
import sys

REQUIRE = os.environ.get("CERVORULES_REQUIRE_SCHEMA_VALIDATION", "") == "1"


def skip(reason: str) -> int:
    stream = sys.stderr if REQUIRE else sys.stdout
    print(f"schema validation {'FAILED' if REQUIRE else 'skipped'}: {reason}", file=stream)
    if REQUIRE:
        print("install with: python -m pip install pyyaml jsonschema", file=sys.stderr)
        return 1
    return 0


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

    schema_dir = os.path.join(repo, "schemas", "v3")
    schemas = {}
    for base in ("policy-rules", "policy-vocabulary"):
        path = os.path.join(schema_dir, f"{base}.schema.json")
        with open(path, encoding="utf-8") as handle:
            schema = json.load(handle)
        # A malformed schema is silently permissive in some validators, so
        # check the schema itself before checking anything against it.
        Draft202012Validator.check_schema(schema)
        schemas[f"{base}.yaml"] = schema

    patterns = [
        os.path.join(repo, "examples", "**", "policy-rules.yaml"),
        os.path.join(repo, "examples", "**", "policy-vocabulary.yaml"),
    ]
    documents = sorted({path for pattern in patterns for path in glob.glob(pattern, recursive=True)})
    if not documents:
        print("no policy documents found", file=sys.stderr)
        return 1

    failures = 0
    for path in documents:
        schema = schemas[os.path.basename(path)]
        with open(path, encoding="utf-8") as handle:
            document = yaml.safe_load(handle)
        errors = sorted(Draft202012Validator(schema).iter_errors(document), key=lambda error: list(error.path))
        relative = os.path.relpath(path, repo).replace(os.sep, "/")
        if not errors:
            continue
        failures += 1
        print(f"FAIL {relative}", file=sys.stderr)
        for error in errors:
            location = ".".join(str(part) for part in error.path) or "<root>"
            print(f"  at {location}: {error.message}", file=sys.stderr)

    print(f"schema validation: {len(documents)} documents checked, {failures} failing")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
