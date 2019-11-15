# Contributing to OziachBot
OziachBot thanks all of its contributors. Every contribution lends to making OziachBot a better resource for everyone!

[Environment Setup](#setup)
[Opening Issues](#issues)
[Pull Requests](#pullrequests)
[Review Process](#review)
[Coding Guidelines](#coding)

## <a name="setup"></a> Setup
This project uses Go version `1.13`. To check which version your environment is currently running, run `go version` on the command line.

To setup for contribution:

1. [Fork](https://help.github.com/articles/fork-a-repo/) this repository
2. [Clone](https://help.github.com/articles/cloning-a-repository/) your copy of this repo
3. Set a new remote [upstream](https://help.github.com/articles/configuring-a-remote-for-a-fork/) (this helps to keep your fork up to date)

OziachBot manages dependencies with Go modules as released in version 1.13. These automatically get dependencies and wire new ones in through [go.mod](go.mod) and [go.sum](go.sum). That means there is no need to manually `go get` anything! Simply run `go build` in the same directory as `main.go`, with `-v` to track a more verbose logging of modules installed and packages compiled.

## <a name="issues"></a> Opening Issues
Before you open an issue on GitHub, whether it's a question, bug, or feature request, search through the [open issues](https://github.com/mfboulos/oziachbot/issues) in the repository to make sure it's not a duplicate.

If your issue hasn't been reported, you may open a new issue. To help everyone understand the issue, be sure to include a **description** of the issue and any other generally important details. If it's a bug, the following are also recommended:

- **Specs** - operating system, Go version, and any other relevant Go dependency versions
- **Output** - logs or stacktrace help immensely
- **Reproduction Steps** - steps to reliably reproduce the bug
- **Related Stuff** - link any issues or pull requests that relate to the bug
- **Suggestions** - any suggestions toward a resolution of the bug

## <a name="pullrequests"></a> Submitting a Pull Request
If you see an opportunity to contribute to OziachBot, feel free to open a pull request. For some inspiration on opportunities to contribute, there are very likely great [open issues](https://github.com/mfboulos/oziachbot/issues), including those for [first-time contributors](https://github.com/mfboulos/oziachbot/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22).

Before you submit a pull request, search through the [open pull requests](https://github.com/mfboulos/oziachbot/pulls) to make sure a duplicate hasn't already been submitted.

If there is an open issue that your pull request addresses, first make sure no one is currently assigned to it. If all checks out, proceed with your pull request and link the issue in the PR description.

Otherwise, checkout a new branch on your cloned fork, write some awesome code based on the [coding guidelines](#coding), and direct your pull request between that remote branch and `oziachbot:master`.

## <a name="review"></a> PR Review
Each pull request must be approved and merged by a collaborator. Reviews are pretty straightforward, and generally check for the following:

- The merge builds (there is a build check on Github Actions)
- The code follows the [coding guidelines](#coding)
- Any changes to dependencies are justified within the description of the PR
- Documentation is updated with any new features

## <a name="coding"></a> Coding Guidelines
Writing code into an existing codebase can be daunting, but don't worry! Here are some guidelines to help you along:

- **Format** - do your best to follow the existing formatting/naming conventions of the project (when all else fails, defer to [Effective Go](https://golang.org/doc/effective_go.html))
- **Testing** - accompany any new features with tests
- **Documentation** - your code should be well-commented and written with other engineers in mind. All exported fields, functions, and types should be accompanied with a comment to document it in the project's [godoc](https://godoc.org/github.com/mfboulos/oziachbot)