# 🤝 Contributing Guide

Thank you for your interest in contributing to the Fintech Platform! We welcome all improvements, bug fixes, and ideas.

---

## 🏷️ Branching & PRs

- Branch from `main`.
- Use clear branch names: `feature/your-feature`, `fix/bug-description`, `chore/update-x`.
- Open a pull request (PR) with a clear description and link to any related issues.

---

## 📝 Commit Messages

- Use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) **with emoji**:
  - `feat: ✨ Add new payment provider`
  - `fix: 🐛 Fix currency rounding bug`
  - `chore: 🔧 Update dependencies`
- Start with type, add a relevant emoji, then a short summary.
- Use Commitizen with gitmoji for consistency: run `cz c` and follow the prompts.

---

## 🧑‍💻 Code Style

- Follow Go best practices and project conventions.
- Run `go fmt ./...` and `go mod tidy` before committing.
- Use property-style getters (e.g., `Name()`, not `GetName()`).
- Keep business logic in the domain layer.

---

## ✅ Pre-commit Hooks

- Install hooks once per clone: `pre-commit install`.
- Run before committing: `pre-commit run --all-files`.
- Fix any hook failures before pushing.

---

## 🧪 Testing

- Add or update tests for all new features and bug fixes.
- Run `go test -v ./...` and ensure all tests pass.
- Use `make cov_report` for coverage.

---

## 👀 Review Process

- PRs are reviewed for code quality, style, and test coverage.
- Address all review comments before merging.
- Squash commits if needed for a clean history.

---

## 🏅 Emoji & Docs Style

- Use unique, meaningful emojis in all documentation headings and navigation.
- Follow the `## :emoji: Title` style for all major doc sections.
- Add page-level icons in frontmatter where supported.

---

Thank you for making the Fintech Platform better! 🚀
