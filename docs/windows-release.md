# Windows Fork Release Guide

The `windows` branch is the default development and release branch for `SvetlovA/ralphex`. The `master` branch is reserved for synchronizing the fork with `umputun/ralphex`.

## Versioning

Tags use this format:

```text
v<upstream-version>-windows.<increment>
```

For example, `v1.7.0-windows.1` is the first Windows-fork release based on upstream `v1.7.0`. Use `v1.7.0-windows.2` for the next fork-only release. After synchronizing with a newer upstream release, reset the fork increment, such as `v1.8.0-windows.1`.

The release workflow rejects tags that do not match this format, skip an increment, point outside the `windows` branch, or do not contain the corresponding upstream base tag. It passes the resolved base or previous fork tag to GoReleaser so generated notes cover only the relevant Windows changes.

## GitHub Actions secrets

No custom repository secrets are required.

The workflows use GitHub's automatic `GITHUB_TOKEN` for these operations:

- creating GitHub releases and uploading Windows artifacts;
- publishing the `svetlova/ralphex` and `svetlova/ralphex-go` container packages to GHCR;
- submitting CI coverage data.

Do not create `GORELEASER_GITHUB_TOKEN` or `PKG_TOKEN`; the fork workflows do not consume them.

In **Settings → Actions → General → Workflow permissions**, select **Read and write permissions**. GitHub creates `GITHUB_TOKEN` for every workflow run, so it must not be added manually as a secret.

If GHCR already contains packages with restricted Actions access, open each package's settings and grant this repository write access.

## Publish a release

From an up-to-date, clean `windows` branch:

```powershell
git tag -a v1.7.0-windows.1 -m "Windows 1.7.0-windows.1"
git push origin v1.7.0-windows.1
```

The tag starts the release workflow. GoReleaser builds `amd64` and `arm64` ZIP archives. Each archive contains `ralphex.windows.exe` and the release page is created in `SvetlovA/ralphex`.
