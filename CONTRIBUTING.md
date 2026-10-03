# Contributing Guide

Thank you for your interest in Go-HAR! We welcome all kinds of contributions, including bug reports, feature requests, documentation improvements, and code. This guide explains how to participate in the project.

## Reporting Issues

If you find a bug or have a feature request, first check whether a related issue already exists. If not, open a new issue and include:

- A clear description of the problem
- Steps to reproduce, if applicable
- The behavior you expected
- The behavior that actually occurred
- Relevant logs or error messages
- Environment details, such as the Go version and operating system

## Development Workflow

1. Fork the repository.
2. Create a feature branch (`git checkout -b feature/amazing-feature`).
3. Commit your changes (`git commit -m 'Add some amazing feature'`).
4. Push the branch (`git push origin feature/amazing-feature`).
5. Open a pull request.

## Code Style

- Follow standard Go conventions and best practices.
- Format code with `gofmt`.
- Make sure all tests pass.
- Add appropriate documentation comments.

## Opening a Pull Request

When opening a pull request:

- Provide a clear title and description.
- Reference related issues, if applicable.
- Make sure all automated checks pass.
- Add tests and documentation for any new features.

## Testing

For code contributions, please:

- Add unit tests for new functionality.
- Make sure existing tests still pass.
- Test edge cases.

Run the tests with:

```bash
go test ./...
```

## Documentation

Documentation is an important part of the project. When adding or changing a feature, update the relevant documentation as well:

- Update `README.md`, if applicable.
- Update relevant documents in `doc/`.
- Add or update code comments, especially for public API functions.

## Code Review

All contributions go through code review. During review:

- Stay open to feedback.
- Respond to all comments.
- If changes are needed, push them to the same pull request.

## License

By contributing code, you agree that your contribution will be released under the project's MIT License.

## Code of Conduct

We expect all contributors to treat one another respectfully and maintain a professional, welcoming environment. Abusive comments, personal attacks, bullying, and harassment are not tolerated.

## Getting Help

If you have questions or need help:

- Ask in a related issue.
- Open a new issue to request help.

Thank you for contributing!