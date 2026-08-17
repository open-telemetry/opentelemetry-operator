# Auto-Instrumentation Images

This directory contains the build context for the language auto-instrumentation
images the operator SIG publishes. Each subdirectory maps to one image published to
`ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-<language>`
(and, for some languages, mirrored to `otel/autoinstrumentation-<language>` on
Docker Hub).

Go auto-instrumentation is **not** built here — the operator references the
upstream `opentelemetry-go-instrumentation` image directly. Nginx
auto-instrumentation shares the `apache-httpd` image.

## Image tagging

Image tags follow a Linux-distribution style `<sdk-version>-<revision>` scheme,
where `<sdk-version>` is the upstream language SDK/agent version and `<revision>`
is an operator-owned build revision.

Each language build publishes the following tags:

| Tag                                          | Mutability | Purpose                                                                  |
|----------------------------------------------|------------|--------------------------------------------------------------------------|
| `<sdk-version>-<revision>` (e.g. `2.30.0-1`) | Immutable  | Canonical, pinnable tag. Referenced by examples, READMEs, and e2e tests. |
| `<sdk-version>` (e.g. `2.30.0`)              | Floating   | Latest revision of a given SDK version.                                  |

Additionally, some languages publish a floating major-version tag, such as `2` for Java.

### The revision

Every language directory contains a `revision.txt` holding the current
operator-owned revision (a positive integer). The `revision` portion of the tag
is read from this file at publish time.

The revision moves according to two rules, enforced on pull requests by
[`check-autoinstrumentation-revision.sh`](../.github/workflows/scripts/check-autoinstrumentation-revision.sh):

1. **Reset to `1`** when the upstream SDK version changes. A new upstream version
   starts a fresh packaging lineage (`2.30.0-1`, then a later `2.30.1-1`).
2. **Increment** (`1` -> `2`) when the image content changes (base image,
   Dockerfile, bundled tooling) but the SDK version is unchanged.

Run the same check locally before opening a pull request:

```bash
make check-autoinstrumentation-revision
```

It diffs against the merge-base with the target branch (override with
`TARGET_BRANCH=<branch>`).

The upstream SDK version for each language is read from:

| Language     | SDK version source                                                                  |
|--------------|-------------------------------------------------------------------------------------|
| java         | `java/version.txt`                                                                  |
| dotnet       | `dotnet/version.txt`                                                                |
| php          | `php/version.txt`                                                                   |
| apache-httpd | `apache-httpd/version.txt`                                                          |
| python       | first line of `python/requirements.txt` (`opentelemetry-distro==<version>`)         |
| nodejs       | `nodejs/package.json` → `dependencies["@opentelemetry/auto-instrumentations-node"]` |
