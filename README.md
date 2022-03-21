# bodash

[![Test](https://github.com/jnnl/bodash/actions/workflows/test.yml/badge.svg?branch=master)](https://github.com/jnnl/bodash/actions/workflows/test.yml)

bodash (Blue Ocean dashboard) is a CLI dashboard for displaying Jenkins job states in your terminal using the [Blue Ocean REST API](https://plugins.jenkins.io/blueocean-rest/).

By default, `bodash` queries `https://$BODASH_DOMAIN/blue/rest/users/$BODASH_USER/favorites` every 10 seconds
and displays the name, id, job state/result and either running job duration or relative finished job end time in a colorized tabular format.

Here's an example:

```
Tue, 22 Feb 2022 22:22:22 UTC
-----------------------------
App Test Deploy        9052   RUNNING  1 minute 42 seconds
App Build              12345  RUNNING  5 minutes 8 seconds
App Long Test          9125   FAILURE  1 hour 12 minutes ago
App Staging Deploy     5255   SUCCESS  4 hours 24 minutes ago
App Production Deploy  942    SUCCESS  2 days 5 hours ago
```

## Installation

Get a pre-built binary from [releases](https://github.com/jnnl/bodash/releases) or build from source:

```
$ git clone https://github.com/jnnl/bodash
$ cd bodash
$ go build
```

## Configuration

Run `bodash --help` to get available configuration options.

Required command line arguments are:
- `-domain`: domain name of the Blue Ocean API instance you want to connect to
- `-user`: Jenkins user whose favorites you want to query for job information
- `-token`: API token for supplied Jenkins user

You can also set environment variables `BODASH_DOMAIN`, `BODASH_USER`, and `BODASH_TOKEN`.
However, supplied command line arguments will always take precedence.

## Notes

bodash currently uses HTTP basic authentication to authenticate with the Blue Ocean API. This may change in the future.
