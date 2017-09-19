package namesrv

// PutKVConfigRequestHeader 向Namesrv追加KV配置-请求头
// Author: tianyuliang, <tianyuliang@gome.com.cn>
// Since: 2017/9/4
type PutKVConfigRequestHeader struct {
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
	Value     string `json:"value"`
}

func (header *PutKVConfigRequestHeader) CheckFields() error {
	return nil
}
