### About

App which will kill Safari tabs which use > 1GB of memory.


### Install

1. Install Go someway (https://golang.org/dl/).
2. Set `GOPATH` to point to some directory.
3. Install with `go get github.com/ivanzoid/safariShrink && go install github.com/ivanzoid/safariShrink`

### Usage

You may run it periodically using cron. For example:

1. `crontab -e`. Then enter:
2. `30 * * * * /Users/ivan/Go/bin/safariShrink` (edit according to your `$GOPATH`)