# git-gb

`git-gb` is a better way to list git branches in your terminal. Inspired by the GitHub branches view, the output

- is sorted by timestamp of the last commit for each branch
- shows how many commits a branch is ahead/behind of master
- whether a branch is merged or not

Sample output:

```
~/c/gb:master$ git gb
2014-11-22 20:54PM | foobar                   | behind:   15 | ahead:    2
2014-11-24 21:18PM | readme                   | behind:    0 | ahead:    1
```

## Usage

See `git gb -help` for available options.

## Default branch

By default, `git gb` will run the comparison against these in order of first found:

* The CLI argument: `git gb some-base-branch`
* The `init.defaultBranch` value found in git's configuration (global or per repository)
* Fallback to `main` if not configured above

## Installation

### Mac

*Install dependencies:*

```
brew install go libgit2
```

*Configure dependencies:*

Make sure that `$GOPATH` is set. For more details see: `go help gopath`.

Also make sure that `$GOPATH/bin` is in your `$PATH`

*Install git-gb:*

```
go get github.com/vroy/git-gb
```
