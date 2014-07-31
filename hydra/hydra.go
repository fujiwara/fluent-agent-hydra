package hydra

type BulkMessage struct {
	Tag    string
	Buffer *[]byte
}

const (
	ChannelBufferLen = 1
)

// NewChannel create channel for using by Forward() and Trail().
func NewChannel() chan *BulkMessage {
	return make(chan *BulkMessage, ChannelBufferLen)
}

