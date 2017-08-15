easypush
=============================
`easypush` a real time push server 
supports multiple protocols(tcp or websocket)


--------------------------------

## Features
 * light weight
 * high performance
 * pure golang
 * single push, multiple push, broadcast, topic broadcast
 * heartbeat
 * authentication
 * multiple protocols（websocket，tcp）
 * scalable archiecture
 * based on message queue(kafka or nsq)
 * 

## TODO
 * comet : change protocol of beartbeat and notification to udp protocol, change protocol of message to tcp or websocket
 * protocol : MQTT

## contribution
refer to https://github.com/Terry-Mao/goim
refer to https://github.com/ikenchina/gopush

## install
### 1. dependencies
> * defaultRPC MQ to push message from admin to push module
> * redis
> * kafka[ref](http://kafka.apache.org/documentation.html#quickstart) or nsq[ref](http://nsq.io/overview/design.html) message queue

### 2. compile comet, admin, push (go build .)

### 3. deploy
> * a.start message queue and redis cluster
> * b.start flow : admin -> push -> comet

## service
### client register flow
1. client subscribe the topics to comet，then keeping heartbeat
2. comet notify admin that the user subscribed
3. comet check whether the user has offline message, then send to message queue

### push flow
1. call the restful api of admin
2. admin send message to message queue, if it is reliable message, save it to redis before send to mq(message queue)
3. push process comsume the message and send to comet process
4. comet push the message to the user. broadcast: push to all online user. reliable message: push successful then notify push to delete message from redis.
5. push delete message from redis


### comet
if admin was crashed, comet would re subscribe all online users to admin 

### admin
restful api of push 

### push
the comsumer of message queue
stateless


