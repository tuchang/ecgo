//提供记时点功能

package util

import (
	"time"
)

type Bm struct {
	TsLst map[string]int64
}

//生成benchmark对象
func NewBenchMark() *Bm {
	start := int64(time.Now().UnixNano() / 1e6)
	lst := make(map[string]int64)
	lst["start"] = start
	return &Bm{lst}
}

//todo:开启记录当前计时点的资源使用情况: runtime

//设置一个记时点
func (this *Bm) Set(key string) {
	this.TsLst[key] = int64(time.Now().UnixNano() / 1e6)
}

//获取两个记时点之间的时间差(毫秒数),如果任一记时点不存在，返回0
func (this *Bm) Get(start, end string) int64 {
	ts1, exists1 := this.TsLst[start]
	ts2, exists2 := this.TsLst[end]
	if !exists1 || !exists2 {
		return 0
	} else {
		return ts2 - ts1
	}
}

//获取所有记时点及其时间戳(毫秒)
func (this *Bm) GetAll() map[string]int64 {
	return this.TsLst
}
