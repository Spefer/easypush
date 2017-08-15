package main

import (
	"sync"
	"github.com/Spefer/common/libs/proto"
)


// Bucket is a holder of channels 
type Bucket struct {
	sync.RWMutex			
	channels	map[string]*Channel		// userid -> channel
	Config		ConfigBucket
}


func NewBucket(config ConfigBucket) (b *Bucket) {
	b = new(Bucket)
	b.channels = make(map[string]*Channel, config.Channel.Size)
	b.Config = config

	return
}

func (b *Bucket) Put(userid string, ch *Channel) error {

	b.Lock()
	b.channels[userid] = ch
	b.Unlock()

	return nil
}

func (b *Bucket) DelSafe(userid string, och *Channel)  {
	var (
		ch *Channel
		ok bool
	)
	b.Lock()
	if ch, ok = b.channels[userid]; ok && och == ch {
		delete(b.channels, userid)
	}
	b.Unlock()
	 
}

func (b *Bucket) Channel(userid string) *Channel {
	b.RLock()
	ch, ok := b.channels[userid]
	b.RUnlock()
	if !ok {
		return nil
	}
	return ch
}

func (b *Bucket) Push(userid string, p *proto.Proto) (err error) {
	channel := b.Channel(userid)
	if channel != nil {
		err = channel.Push(p)
	}
	return
}



// ### broadcast ignore error
func (b *Bucket) Broadcast(p *proto.Proto) {
	var ch *Channel
	b.RLock()
	for _, ch = range b.channels {
		ch.Push(p)
	}
	b.RUnlock()
}

func (b *Bucket) BroadcastTopic(topic string, p *proto.Proto) {
	var ch *Channel
	b.RLock()
	for _, ch = range b.channels {
		if ch.ContainTopic(topic) {
			ch.Push(p)
		}
	}
	b.RUnlock()
}

