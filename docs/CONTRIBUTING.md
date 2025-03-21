# Contributing to Panopticon Scanner

Thank you for your interest in contributing to Panopticon Scanner! This document provides guidelines and instructions for contributing to the project.

## Code of Conduct

Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md) to foster an inclusive and respectful community.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/your-username/panopticon-scanner.git
   cd panopticon-scanner
   ```
3. **Set up the development environment** following the instructions in [DEVELOPER.md](DEVELOPER.md)
4. **Create a new branch** for your changes:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Workflow

### Before You Start

1. Check the [issue tracker](https://github.com/legio314/panopticon-scanner/issues) for existing issues or feature requests related to your contribution
2. If no relevant issue exists, consider creating one to discuss your proposed changes before starting work
3. For significant changes, discuss your approach with the maintainers first

### Making Changes

1. Write clear, maintainable code that follows the project's coding standards
2. Keep your changes focused on a single objective
3. Include appropriate tests for your changes
4. Ensure your code passes all existing tests
5. Update documentation as necessary

### Coding Standards

Please follow these standards when writing code:

#### Go Code

- Follow the [Effective Go](https://golang.org/doc/effective_go) guidelines
- Use `gofmt` to format your code
- Organize imports in three blocks: standard library, third-party, and local packages
- Document all exported functions, types, and constants
- Use meaningful variable names
- Handle errors explicitly
- Use `zerolog` for structured logging

#### JavaScript/TypeScript Code

- Follow the project's ESLint configuration
- Use TypeScript for type safety
- Prefer functional components with hooks
- Document component props
- Use meaningful variable names
- Keep components small and focused

### Commit Guidelines

- Use clear, descriptive commit messages
- Reference issue numbers in commit messages when applicable
- Make small, focused commits rather than large, sweeping changes
- Keep commit history clean by using `git rebase` if necessary

Example commit message:
```
Add network port filtering feature

- Add filter component in UI
- Implement backend filtering API
- Update documentation
- Add unit tests

Fixes #123
```

## Pull Request Process

1. **Update your fork** with the latest changes from the upstream repository:
   ```bash
   git remote add upstream https://github.com/legio314/panopticon-scanner.git
   git fetch upstream
   git rebase upstream/master
   ```

2. **Push your changes** to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

3. **Create a pull request** from your branch to the original repository's `master` branch

4. **In your pull request description**:
   - Describe the changes you've made
   - Reference any related issues
   - Include screenshots for UI changes
   - Note any particular areas that need special attention during review

5. **Address review feedback** promptly and make requested changes

6. Once approved, a maintainer will merge your pull request

## Testing

- Run tests before submitting a pull request:
  ```bash
  go test ./...
  cd ui && npm test
  ```
- For backend changes, ensure integration tests pass:
  ```bash
  go test ./tests/integration
  ```
- For UI changes, verify that components render correctly in different themes and scenarios

## Documentation

- Update documentation for any new features or changes to existing functionality
- Write clear, concise documentation that's accessible to users of varying skill levels
- Include examples where appropriate
- Check for spelling and grammar errors

## Reporting Bugs

When reporting bugs, please include:

1. A clear, descriptive title
2. Steps to reproduce the bug
3. Expected behavior
4. Actual behavior
5. Screenshots if applicable
6. Environment information:
   - OS version
   - Go version
   - Node.js version
   - Browser version (for UI bugs)

## Feature Requests

When suggesting features, please include:

1. A clear, descriptive title
2. Detailed description of the proposed feature
3. Rationale for why the feature would be valuable
4. Any potential implementation approaches
5. Mockups or examples if applicable

## Code Review

All submissions require review before being merged. Reviewers will check for:

- Adherence to coding standards
- Test coverage
- Documentation completeness
- Performance implications
- Security considerations
- Compatibility with existing features

## License

By contributing to Panopticon Scanner, you agree that your contributions will be licensed under the project's [license](LICENSE).

## Questions?

If you have questions about contributing, please:
- Open a discussion on GitHub
- Ask in the project's communication channels
- Contact the maintainers directly

Thank you for contributing to Panopticon Scanner!