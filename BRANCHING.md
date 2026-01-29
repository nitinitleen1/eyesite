# Branching Strategy

## Main Branches

### `main` (Protected)
- **Purpose**: Production-ready, stable releases only
- **Policy**: Direct commits are disabled
- **Merges**: Only via Pull Request from `develop` after testing
- **Contains**: Features that are complete, tested, and documented

### `develop` (Default Development)
- **Purpose**: Active development happens here
- **Policy**: Frequent commits encouraged
- **Contains**: Work-in-progress features, experiments, vibe coding
- **Stability**: May be unstable, breaking changes expected

## Feature Branches

For substantial features, create feature branches:
```bash
git checkout -b feature/your-feature-name
```

Merge back to `develop` when ready.

## Workflow

```bash
# Start new work
git checkout develop
git pull origin develop
git checkout -b feature/my-feature

# Make changes, commit frequently
git add .
git commit -m "feat: add awesome feature"

# Push to GitHub
git push origin feature/my-feature

# Create PR to develop (not main!)
# After review, merge to develop

# When ready for release (maintainers only)
# Create PR from develop -> main
```

## Commit to Main
Only when a feature is:
- ✅ Fully implemented
- ✅ Tested
- ✅ Documented
- ✅ Reviewed
- ✅ Stable

## Current Status
- `main`: Foundation + Auth + Ingestion + Frontend Setup (v0.1.0-alpha)
- `develop`: Next features being built

## Why This Approach?

**Main = Confidence**: Anyone cloning `main` gets something that works (even if minimal).

**Develop = Innovation**: Rapid iteration without worrying about breaking `main`.

**Vibe Coding Friendly**: Commit freely to `develop`, clean up for `main`.
