// Copyright (c) 2015-2018 All rights reserved.
// 本软件源代码版权归 my.oschina.net/tantexian 所有,允许复制与学习借鉴.
// Author: tantexian, <tantexian@qq.com>
// Since: 2017/08/09
package timeutil

import "time"

func NowTimestamp() int64 {
	return time.Now().UnixNano()
}
