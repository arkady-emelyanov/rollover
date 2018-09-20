# Roll-it-over

Simple tool to perform rollover for a big ElasticSearch time-based indices.
Full approach described in [original article](https://www.elastic.co/blog/managing-time-based-indices-efficiently). 

Out-of-scope for now (but may change in future):
* No index shrinking
* No index reallocation to cold nodes
* No index compression

## Why?

Please read [original article](https://www.elastic.co/blog/managing-time-based-indices-efficiently).
Just few additions from my side:
* Save some HDD space
* Increase indexing speed
* Increase search speed

(sneak peek, it's all about thresholds: `max_docs` and `max_age`).

## Setting up

Let's assume we are starting from scratch.

First step is about to create settings-template for our indices:
```
curl -XPUT '127.1:9200/_template/logs-write?pretty' -d '\
{ \
    "template": "logs-write-*",\
    "settings": {\
        "refresh_interval": "10s",\
        "number_of_shards": 2,\
        "number_of_replicas": 1\
    }\
}'
```

Now, we can create a new empty index:
```
curl -XPUT '127.1:9200/logs-write-2018.09.20?pretty' -d '{}'
```

Since we don't want to write like `logstash-%Y.%m.%d`, instead, 
we want to write to alias:
```
curl -XPUT '127.1:9200/logs-write-2018.09.20/_alias/logs-write?pretty'
```

Now, we can start writing to index names `logs-write`, without any
day format layout. Due alias, all documents will be written to index
alias is currently pointing at.

Now, create `/etc/rollover/localhost.yml` configuration file, with
following content:
```yaml
elasticsearch:
  endpoints:
  - http://127.0.0.1:9200

rollover:
- alias: logs-write
  new_name: logs-write-%Y-%m-%d-%H%M%s
  conditions:
    max_docs: 10000
    max_age: 2h
  optimize:
    max_segments: 1
```


SystemD right?
Create a simple unit `/etc/systemd/system/rollover-localhost.service`
```
[Unit]
Description=Rollover ElasticSearch indices (localhost)
After=network.target

[Service]
Type=simple

Restart=always
RestartSec=5

ExecStart=/usr/bin/rollover -config /etc/rollover/localhost.yml

[Install]
WantedBy=multi-user.target
```

TBD.
