package consumer

import "git.oschina.net/cloudzone/smartgo/stgcommon/message"
// ConsumeConcurrentlyContext: 普通消息消费上下文
// Author: yintongqiang
// Since:  2017/8/10

type ConsumeConcurrentlyContext struct {
	MessageQueue              message.MessageQueue
	DelayLevelWhenNextConsume int
	AckIndex                  int
}
