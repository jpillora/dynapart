# dynapart

A simple command-line interface to DynamoDB using PartiQL

[![GoDev](https://img.shields.io/static/v1?label=godoc&message=reference&color=00add8)](https://pkg.go.dev/github.com/jpillora/dynapart)
[![CI](https://github.com/jpillora/dynapart/workflows/CI/badge.svg)](https://github.com/jpillora/dynapart/actions?workflow=CI)

### Demo

```
$ dynapart 'SELECT Author,ID FROM Books'
```
```json
{"Author":"Ben Horowitz","ID":"5bv2Fs1JSCiVrDyNEoRkQ"}
{"Author":"Matthew McConaughey","ID":"x66TMQ3gTf6yileYGQXvg"}
{"Author":"Ernest Cline","ID":"jxUE0ftHTD6O4ogI2Dh0g"}
{"Author":"Isaac Asimov","ID":"FVzOUf0WQujIJ1WxX2PhA"}
{"Author":"Chris Voss, Tahl Raz","ID":"4Pba0gmOStyHrF9N9yMAQ"}
{"Author":"Terry Pratchett","ID":"zx_CovWDRM6pOPdkkvfzQ"}
```

### Install

**Binaries**

[![Releases](https://img.shields.io/github/release/jpillora/dynapart.svg)](https://github.com/jpillora/dynapart/releases)
[![Releases](https://img.shields.io/github/downloads/jpillora/dynapart/total.svg)](https://github.com/jpillora/dynapart/releases)

Find [the latest pre-compiled binaries here](https://github.com/jpillora/dynapart/releases/latest)  or download and install it now with:

```
$ curl https://i.jpillora.com/dynapart! | bash
```

**Source**

```sh
$ go install -v github.com/jpillora/dynapart@latest
```

### Usage

<!--tmpl,code=plain:echo "$ dynapart --help" && go run main.go --help -->
``` plain 
$ dynapart --help

  Usage: dynapart [options] <statement> [arg] [arg] ...

  Executes the given DynamoDB PartiQL <statement>,
  which may contain ? placeholders. If placeholders are set,
  you must provide corresponding [arg]s where each [arg] is valid JSON.

  Options:
  --max-pages, -m    maximum number of result pages to return (default 1)
  --consistent-read  enable consistent reads
  --item-format, -i  json is the only format currently
  --no-colors, -c    disable json syntax highlighting (default true)
  --verbose, -v      print actions to stderr
  --endpoint, -e     dynamodb endpoint url
  --local, -l        shorthand for --endpoint=http://localhost:8000
  --version          display version
  --help, -h         display help

  For more information on using PartiQL with DynamoDB, see
  • https://partiql.org/tutorial.html
  • https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/ql-reference.html

  Each DynamoDB item is written to stdout as JSON, one object per line, which makes
  dynapart jq compatible.

  Syntax highlighting is automatically enabled when stdout is a TTY.
  This can be overridden with --no-colors=true/false.

  The default AWS SDK authorization process is performed on program start.
  Basically, AWS environment variables will be used, followed by a profile (AWS_PROFILE),
  followed by the EC2 metadata endpoint. See AWS documentation for details.

  You can use a special statement "SHOW TABLES" to list your DynamoDB tables.
  Note, This is not valid PartiQL.

  Version:
    0.0.0-src

  Read more:
    https://github.com/jpillora/dynapart

```
<!--/tmpl-->

### TODO

* Match JSON key order with SELECT
* CSV output
* Custom AWS authorization