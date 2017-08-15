package main

import (
	"errors"
	"fmt"
	log "github.com/ikenchina/golog"
	"github.com/Spefer/common/libs/define"
	"github.com/Spefer/common/libs/proto"
	"net/rpc"
	"time"
	"strings"
)

const (
	InternalMqName     = "defaultMQ"
	TCP_PROTOCOL       = "tcp"
	InternalMqNotReady = "defaultMQ is not ready, please make sure it ready for used"
	RPC_METHOD         = "A2PMsg.RECEIVE"
)

type PushMsgClient struct {
	Client  *rpc.Client
	Quit    chan struct{}
	Network string
	Addr    string
}

type InternalMq struct {
	Clients          []*PushMsgClient
	hasPushConnected bool
	addrs            string
}

func (mq *InternalMq) Init(Addrs []string) error {
	mq.Clients = make([]*PushMsgClient, 0)
	if len(Addrs) != 1 {
		panic("push server address should be only one record.")
	}
	mq.addrs = Addrs[0]
	go mq.dail2Server()
	return nil
}

func (mq *InternalMq) dail2Server() {
	c := &PushMsgClient{}
	for !mq.hasPushConnected {
		log.Infof("dial push server rpc (%s)......", mq.addrs)
		client, err := rpc.Dial(TCP_PROTOCOL, mq.addrs)
		if err != nil {
			// dial later
			msg := fmt.Sprintf("Error on dail to PushServer[%s] for:%s. would try again in 5 seconds", mq.addrs, err)
			log.Infof(msg)
			time.Sleep(5 * time.Second)
			continue
		}
		c.Addr = mq.addrs
		c.Client = client
		c.Network = TCP_PROTOCOL
		c.Quit = make(chan struct{}, 1)
		mq.Clients = append(mq.Clients, c)
		log.Infof("connect to push [%s] successful.", mq.addrs)
		mq.hasPushConnected = true
	}
}

func (mq *InternalMq) UnInit() {
	if !mq.hasPushConnected {
		log.Warnf(InternalMqNotReady)
		return
	}

	defer func() {
		mq.hasPushConnected = false
	}()

	for _, c := range mq.Clients {
		c.Client.Close()
	}
}

func (mq *InternalMq) push(msg *proto.MQMsg) error {
	if !mq.hasPushConnected {
		log.Warnf(InternalMqNotReady)
		return errors.New(InternalMqNotReady)
	}

	var reply = proto.NoRES{}
	log.Debugf("client num:%d",len(mq.Clients))
	for _, c := range mq.Clients {
		go func(m *proto.MQMsg) {
			log.Debugf("userids:%s; userid=%s, body=%s", strings.Join(m.UserId, ","), m.Msg.UserId, m.Msg.Body)
			if err := c.Client.Call(RPC_METHOD, m, &reply); err != nil {
				log.Errorf("c.Call(%s, %v, reply) failed : %v", RPC_METHOD, msg, err)
			}
		}(msg)
	}
	return nil
}

func (mq *InternalMq) Push(serverId int32, msg *proto.PushMsg) error {
	v := &proto.MQMsg{OP: define.MQ_MESSAGE, ServerId: []int32{serverId}, Msg: msg}
	return mq.push(v)
}

// push multi msg to one user
func (mq *InternalMq) MPush(serverId int32, userIds []string, msg *proto.PushMsg) error {
	v := &proto.MQMsg{OP: define.MQ_MESSAGE_MULTI, ServerId: []int32{serverId}, UserId: userIds, Msg: msg}
	return mq.push(v)
}

func (mq *InternalMq) Broadcast(msg *proto.PushMsg) error {
	v := &proto.MQMsg{OP: define.MQ_MESSAGE_BROADCAST, Msg: msg}
	return mq.push(v)
}

func (mq *InternalMq) BroadcastTopic(serverId []int32, topic string, msg *proto.PushMsg) error {
	v := &proto.MQMsg{OP: define.MQ_MESSAGE_BROADCAST_TOPIC, ServerId: serverId, Topic: topic, Msg: msg}
	return mq.push(v)
}
