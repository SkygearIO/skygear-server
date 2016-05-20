Skygear Server is a cloud backend for your app.

[![Build Status](https://travis-ci.org/SkygearIO/skygear-server.svg?branch=master)](https://travis-ci.org/SkygearIO/skygear-server)

## Getting Started

To get started, you need to install Skygear Server and include one of
our SDKs into your app. For more information on how to do this, check
out the [Skygear Server Documentation](http://docs.skygear.io/).

### Configuration

Check out `development.ini` for example configuration.

You need to specify the configuration file when running Skygear Server:

```shell
$ ./skygear-server development.ini
```

Alternatively,
```shell
$ `SKY_CONFIG=development.ini ./skygear`
```

## How to contribute

### Dependencies

* Golang 1.5
* PostgreSQL 9.4 with PostGIS extension
* Redis
* libsodium, zeromq and czmq if using ZeroMQ as a plugin transport

If using Mac OS X, you can get ZeroMQ dependencies using Homebrew:

```shell
$ brew install libsodium zeromq czmq
```

### Building from source

```shell
$ go get github.com/tools/godep
$ godep restore
$ go build  # or `go build --tags zmq` for ZeroMQ support
```

### Testing

1. Create a PostgreSQL database called `skygear_test` with PostGIS enabled:

```shell
psql -h db -c 'CREATE DATABASE skygear_test;' -U postgres
psql -h db -c 'CREATE EXTENSION postgis;' -U postgres -d skygear_test
```

2. Test case assume the 127.0.0.1 have access to `skygear_test`, add the
following to `pg_hba.conf`:

```
host    all             all             127.0.0.1/32            trust
```

3. Install golang packages required for testing (check `.travis.yml` for the
   list).

4. Run `go test github.com/skygeario/skygear-server/...`.
