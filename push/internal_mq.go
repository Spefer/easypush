package main

import (
	"github.com/Spefer/common/libs/proto"
	log "github.com/ikenchina/golog"
	"net/rpc"
)

const NETWORK_PROTOCOL = "tcp"

type A2PMsg struct {
}

func (m *A2PMsg) RECEIVE(msg *proto.MQMsg, reply *proto.NoRES) error {
	//log.Debugf("received:")
	if cometRPC == nil {
		return ErrComet
	}
	cometRPC.push(msg)
	return nil
}

type InternalMq struct {
	apush *A2PMsg
	addrs []string
}

func (mq *InternalMq) Init(Addrs []string) error {
	mq.addrs = Addrs
	mq.apush = &A2PMsg{}
	err := mq.Start()
	if err == nil {
		log.Infoln("start message queue successful\n")
	} else {
		log.Errorln("start message queue failed\n")
	}
	return nil
}

func (mq *InternalMq) Start() error {
	rpc.Register(mq.apush)
	for _, addr := range mq.addrs {
		go rpcListen(NETWORK_PROTOCOL, addr)
	}
	return nil
}

func (mq *InternalMq) UnInit() {
}
