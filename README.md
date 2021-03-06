# BoltMQ [![Build Status](https://travis-ci.org/gunsluo/common.svg?branch=master)](https://travis-ci.org/gunsluo/common) [![Go Report Card](https://goreportcard.com/badge/github.com/sniperkit/boltmq/pkg)](https://goreportcard.com/report/github.com/sniperkit/boltmq/pkg)
BoltMQ is a distributed queue, writern on Go. it is based on apache open source project: [Apache RocketMQ](https://github.com/apache/rocketmq).

### Features

* Pub/Sub messaging
* Scheduled message
* Load balancing
* Reliable FIFO and strict ordered messaging in the same queue
* Support Master & Salve


### Get it

**Build it from source code**

Get source code from Github:
```Go
git clone https://github.com/sniperkit/boltmq/pkg.git
```


### Getting started

#### Installing

To start using BoltMQ, install Go and run:
```Go
make deps
make
```

#### Config it

first, config broker or namesrv.
```Go
cd bin/etc
vim broker.toml
vim namesrv.toml
```

#### Running

* -c set config file path 
* -p set pid file path, default .
* -f run as frontend 

E.g
run as frontend, use `bin/broker -f` `bin/namesrv -f`, usually development env & debug.


### Contributing
We always welcome new contributions, if you are interested in Go or MQ, more details see [here](https://github.com/blog/1360-introducing-contributions)

