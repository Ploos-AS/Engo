package botai

import "sync"

type Conversations struct {
	mu sync.Mutex
	maxMessages int
	items map[string][]Message
}

func NewConversations(maxMessages int) *Conversations {
	if maxMessages < 0 { maxMessages = 0 }
	if maxMessages > 20 { maxMessages = 20 }
	return &Conversations{maxMessages:maxMessages, items:make(map[string][]Message)}
}

func (c *Conversations) History(key string) []Message {
	c.mu.Lock(); defer c.mu.Unlock()
	return append([]Message(nil), c.items[key]...)
}

func (c *Conversations) AddExchange(key, user, assistant string) {
	if c.maxMessages == 0 { return }
	c.mu.Lock(); defer c.mu.Unlock()
	h:=append(c.items[key], Message{Role:"user",Content:user}, Message{Role:"assistant",Content:assistant})
	if len(h)>c.maxMessages { h=append([]Message(nil),h[len(h)-c.maxMessages:]...) }
	c.items[key]=h
}

func (c *Conversations) Reset(key string) {
	c.mu.Lock(); defer c.mu.Unlock()
	delete(c.items,key)
}
