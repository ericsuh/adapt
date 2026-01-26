# Release Process

This document describes the steps necessary to create a new release of adapt.

## Prerequisites

- You must have write access to the repository
- All changes intended for the release should be merged to the `main` branch
- All CI checks should be passing on the `main` branch

## Release Steps

### 1. Determine the Version Number

Follow [Semantic Versioning](https://semver.org/) guidelines:
- **MAJOR** version for incompatible API changes
- **MINOR** version for new functionality in a backward compatible manner
- **PATCH** version for backward compatible bug fixes

Example: `v0.2.0`, `v1.0.0`, `v1.1.0`, `v1.1.1`

### 2. Create and Push a Tag

From your local repository with the latest `main` branch:

```bash
# Ensure you're on the main branch and up to date
git checkout main
git pull origin main

# Create an annotated tag
git tag -a v0.2.0 -m "Release v0.2.0"

# Push the tag to GitHub
git push origin v0.2.0
```

### 3. Automatic Release Process

Once the tag is pushed, GitHub Actions will automatically:
1. Trigger the release workflow (`.github/workflows/release.yml`)
2. Build binaries for all configured platforms (linux/amd64, linux/arm64)
3. Create a draft GitHub release with the tag
4. Upload the built artifacts to the draft release
5. Generate and attach checksums
6. Automatically publish the draft release

### 4. Verify the Release

1. Navigate to the [Releases page](https://github.com/ericsuh/adapt/releases)
2. Verify the new release appears with the correct version
3. Check that all expected artifacts are attached:
   - `adapt_<version>_linux_x86_64.tar.gz`
   - `adapt_<version>_linux_arm64.tar.gz`
   - `adapt_<version>-checksums.txt`
4. Verify the release notes are formatted correctly

### 5. Update Release Notes (Optional)

If needed, you can edit the release notes on GitHub:
1. Go to the release page
2. Click "Edit release"
3. Add additional details, highlights, or breaking changes
4. Click "Update release"

## Troubleshooting

### Release Workflow Failed

If the release workflow fails:
1. Check the workflow logs in the Actions tab
2. Fix any issues in the code or configuration
3. Delete the failed release from GitHub (if created)
4. Delete the tag locally and remotely:
   ```bash
   git tag -d v0.2.0
   git push origin :refs/tags/v0.2.0
   ```
5. Once fixed, repeat the release process from step 2

### Need to Re-release

If you need to re-run a failed release:
1. The GoReleaser configuration creates releases as **drafts** first, which are mutable and allow artifact uploads
2. Delete the failed draft release from GitHub (if created)
3. Re-run the workflow:
   - Either push the tag again (after deleting it first)
   - Or manually trigger the release workflow from GitHub Actions
4. The workflow will create a new draft, upload artifacts, then automatically publish it

## Testing Releases

Before creating a production release, you can test the release process:

1. Create a test tag with a pre-release suffix:
   ```bash
   git tag -a v0.2.0-rc1 -m "Release candidate v0.2.0-rc1"
   git push origin v0.2.0-rc1
   ```

2. This will create a pre-release on GitHub (automatically detected by GoReleaser)

3. Test the artifacts and process

4. When ready, create the final release without the suffix

## Notes

- The GoReleaser configuration (`.goreleaser.yaml`) creates releases as **drafts** first to avoid conflicts with GitHub's immutable release system
- After all artifacts are uploaded to the draft, the GitHub Actions workflow automatically publishes it
- Tags should always follow the format `vX.Y.Z` (with the `v` prefix)
- The workflow requires the `GITHUB_TOKEN` which is automatically provided by GitHub Actions
- Build metadata (version, commit, date) is automatically injected into the binary during the build process
- Draft releases are mutable and can receive artifact uploads; published releases are immutable
