# HTTP checks (httpchk)

[![Go Test](https://github.com/mat/httpchk/actions/workflows/tests.yml/badge.svg)](https://github.com/mat/httpchk/actions/workflows/tests.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/mat/httpchk)](https://goreportcard.com/report/github.com/mat/httpchk)

This service runs multiple HTTP requests against a set of URLs and returns 200 on success, 503 otherwise.

**Example**

- Config: <https://github.com/mat/httpchk/blob/master/checks.csv>
- Result: <https://httpchk.onrender.com/check.txt?checks=https://raw.githubusercontent.com/mat/httpchk/master/checks.csv>


## Hosting

Simple options to host this service are, for example:

- Render: <https://render.com/deploy?repo=https://github.com/mat/httpchk>
- Heroku: <https://heroku.com/deploy>

## Heroku / Setup

```bash
heroku labs:enable runtime-dyno-metadata
```

## Docker

Build the Docker image using

```bash
make docker_build
```

and run it with

```bash
make docker_run
```