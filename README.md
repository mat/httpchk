# HTTP checks (httpchk)

[![Build Status](https://travis-ci.org/mat/httpchk.svg?branch=master)](https://travis-ci.org/mat/httpchk)
[![Go Report Card](https://goreportcard.com/badge/github.com/mat/httpchk)](https://goreportcard.com/report/github.com/mat/httpchk)

This service runs multiple HTTP requests against a set of URLs and returns 200 on success, 503 otherwise.

**Example**

- Config: <https://github.com/mat/httpchk/blob/master/checks.csv>
- Result: <https://httpchk-demo.herokuapp.com/?checks=https://raw.githubusercontent.com/mat/httpchk/master/checks.csv>


## Hosting

An easy way to host this service is to use Heroku, just go to <https://heroku.com/deploy> to get started.

## Heroku / Setup

```bash
heroku labs:enable runtime-dyno-metadata
```
