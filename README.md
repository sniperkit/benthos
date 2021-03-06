![Benthos](icon.png "Benthos")

[![godoc for Jeffail/benthos][godoc-badge]][godoc-url]
[![goreportcard for Jeffail/benthos][goreport-badge]][goreport-url]
[![Build Status][travis-badge]][travis-url]

Benthos is a generic and high performance streaming service, able to connect
various sources and sinks and perform arbitrary
[actions, transformations and filters][processors] on payloads. It is ready to
drop into your pipeline either as a static binary or a docker image.

A Benthos instance (stream) consists of four components; [inputs][inputs],
optional [buffer][buffers], [processor][processors] workers and
[outputs][outputs]. Inputs and outputs can be combined in a range of broker
patterns. It is possible to run multiple isolated streams within a single
Benthos instance using [`--streams` mode][streams-mode], and perform CRUD
operations on them via [REST endpoints][streams-api].

### Delivery Guarantees

Benthos is crash resilient by default. When connecting to at-least-once sources
and sinks without a buffer it guarantees at-least-once delivery without needing
to persist messages during transit.

When running a Benthos stream with a [buffer][buffers] there are various options
for choosing a level of resiliency that meets your needs.

## Supported Sources & Sinks

- [Amazon (S3, SQS)][amazons3]
- [Elasticsearch][elasticsearch]
- File
- HTTP(S)
- [Kafka][kafka]
- [MQTT][mqtt]
- [Nanomsg][nanomsg]
- [NATS][nats]
- [NATS Streaming][natsstreaming]
- [NSQ][nsq]
- [RabbitMQ (AMQP 0.91)][rabbitmq]
- [Redis][redis]
- Stdin/Stdout
- Websocket
- [ZMQ4][zmq]

## Documentation

Documentation for Benthos components, concepts and recommendations can be found
in the [docs directory.][general-docs]

For some applied examples of Benthos such as streaming and deduplicating the
Twitter firehose to Kafka [check out the cookbook section][cookbook-docs].

## Run

``` shell
benthos -c ./config.yaml
```

Or, with docker:

``` shell
# Send HTTP /POST data to Kafka:
docker run --rm \
	-e "BENTHOS_INPUT=http_server" \
	-e "BENTHOS_OUTPUT=kafka" \
	-e "KAFKA_OUTPUT_BROKER_ADDRESSES=kafka-server:9092" \
	-e "KAFKA_OUTPUT_TOPIC=benthos_topic" \
	-p 4195:4195 \
	jeffail/benthos

# Using your own config file:
docker run --rm -v /path/to/your/config.yaml:/benthos.yaml jeffail/benthos
```

## Configuration

The configuration file for a Benthos stream is made up of four main sections;
input, buffer, pipeline, output. If we were to pipe stdin directly to Kafka it
would look like this:

``` yaml
input:
  type: stdin
buffer:
  type: none
pipeline:
  threads: 1
  processors: []
output:
  type: kafka
  kafka:
    addresses:
    - localhost:9092
    topic: benthos_stream
```

There are example configs demonstrating each input, output, buffer and processor
option which [can be found here](config).

You can print a configuration file containing fields for all types with the
following command:

``` shell
benthos --print-yaml --all > config.yaml
benthos --print-json --all | jq '.' > config.json
```

There are also sections for setting logging, metrics and HTTP server options.

### Environment Variables

It is possible to select fields inside a configuration file to be set via
[environment variables][config-interp]. The docker image, for example, is built
with [a config file][env-config] where _all_ common fields can be set this way.

## Install

Build with Go:

``` shell
go get github.com/Jeffail/benthos/cmd/benthos
```

Or, pull the docker image:

``` shell
docker pull jeffail/benthos
```

Or, [grab a binary for your OS from here.][releases]

### Docker Builds

There's a multi-stage `Dockerfile` for creating a Benthos docker image which
results in a minimal image from scratch. You can build it with:

``` shell
make docker
```

Then use the image:

``` shell
docker run --rm \
	-v /path/to/your/benthos.yaml:/config.yaml \
	-v /tmp/data:/data \
	-p 4195:4195 \
	benthos -c /config.yaml
```

There are a [few examples here][compose-examples] that show you some ways of
setting up Benthos containers using `docker-compose`.

### ZMQ4 Support

Benthos supports ZMQ4 for both data input and output. To add this you need to
install libzmq4 and use the compile time flag when building Benthos:

``` shell
go install -tags "ZMQ4" ./cmd/...
```

[inputs]: docs/inputs/README.md
[buffers]: docs/buffers/README.md
[processors]: docs/processors/README.md
[outputs]: docs/outputs/README.md

[config-interp]: docs/config_interpolation.md
[compose-examples]: resources/docker/compose_examples
[streams-api]: docs/api/streams.md
[streams-mode]: docs/streams/README.md
[general-docs]: docs/README.md
[cookbook-docs]: docs/cookbook/README.md
[env-config]: config/env/default.yaml

[releases]: https://github.com/Jeffail/benthos/releases

[godoc-badge]: https://godoc.org/github.com/Jeffail/benthos?status.svg
[godoc-url]: https://godoc.org/github.com/Jeffail/benthos
[goreport-badge]: https://goreportcard.com/badge/github.com/Jeffail/benthos
[goreport-url]: https://goreportcard.com/report/Jeffail/benthos
[travis-badge]: https://travis-ci.org/Jeffail/benthos.svg?branch=master
[travis-url]: https://travis-ci.org/Jeffail/benthos

[dep]: https://github.com/golang/dep
[amazons3]: https://aws.amazon.com/s3/
[zmq]: http://zeromq.org/
[nanomsg]: http://nanomsg.org/
[rabbitmq]: https://www.rabbitmq.com/
[mqtt]: http://mqtt.org/
[nsq]: http://nsq.io/
[nats]: http://nats.io/
[natsstreaming]: https://nats.io/documentation/streaming/nats-streaming-intro/
[redis]: https://redis.io/
[kafka]: https://kafka.apache.org/
[elasticsearch]: https://www.elastic.co/
