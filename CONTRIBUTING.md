# Contributing to go-unifi

Thank you for your interest in contributing to go-unifi!

By contributing, you agree that your contributions will be licensed under the BSD-3-Clause license. See [LICENSE](./LICENSE) for details.

## Before You Start

**Critical Requirements:**

1. **Test against real hardware or VM**: All changes must be validated against actual UniFi controllers (hardware or VM)
2. **Real-world tests**: Tests must reflect actual API behavior, not theoretical scenarios
3. **License agreement**: By contributing, you accept the BSD-3-Clause license terms

## Development Workflow

1. **Fork the repository** and create a feature branch
2. **Make small, logical commits** - avoid single commits with 30k+ lines
3. **Sign all commits**: Use `git commit --signoff` for every commit
4. **Keep history clean**: Your commit history is part of the review process

Example workflow:

```bash
git checkout -b feat/add-network-endpoint
# Make changes
git add .
git commit --signoff --message "feat(network): add support for port profiles"
# More commits as needed
git push origin feat/add-network-endpoint
```

## Commit Guidelines

All commits must follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification.

**Format:** `<type>(<scope>): <description>`

**Types:**

- `feat`: New features
- `fix`: Bug fixes
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `refactor`: Code refactoring
- `chore`: Maintenance tasks

**Examples:**

```bash
git commit --signoff --message "feat(sitemanager): add SD-WAN status endpoint"
git commit --signoff --message "fix(network): correct DELETE status code handling"
git commit --signoff --message "docs(readme): update tested hardware section"
```

**Always use `--signoff`** for Developer Certificate of Origin.

## Pull Request Requirements

1. **PR title**: Must follow Conventional Commits format
2. **Testing evidence**: Specify what hardware/VM you tested on:
   - Hardware model (e.g., UDR7, UDM-Pro)
   - UniFi OS version
   - Network Application version
3. **Real API validation**: Describe what was verified against actual API
4. **Commit history**: Keep commits logical and reviewable - we review the process, not just the final state

**Note:** PRs will be squashed when merged. Your commit history is important for review, but the final merge will be a single commit.

Example PR description:

```markdown
## Summary
Add support for port profile management in Network API.

## Testing
Tested on:
- Hardware: UniFi Dream Router (UDR7)
- UniFi OS: 4.3.9
- Network Application: 9.4.19

Validated against real API:
- Created port profiles successfully
- Updated existing profiles
- Deleted test profiles
```

## Testing

Run tests before submitting:

```bash
go test ./...
```

**Test Requirements:**

- Tests must reflect real API behavior
- Include mock responses matching actual API responses
- See [TESTING.md](./TESTING.md) for full testing guidelines

## Code Quality

- Write all code and comments in English
- Linters run automatically in CI (see `.golangci.yaml` and `.markdownlint.yaml`)
- Follow existing code style and patterns

## Questions?

- **Bug reports**: Open an [Issue](https://github.com/lexfrei/go-unifi/issues)
- **Feature requests**: Open a [Discussion](https://github.com/lexfrei/go-unifi/discussions)
- **Questions**: Use [Discussions](https://github.com/lexfrei/go-unifi/discussions)

---

Thank you for making go-unifi better!
