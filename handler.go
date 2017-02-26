//http请求的处理器: 由dispatch根椐相应的path调用

package ecgo

import (
	"fmt"
	. "github.com/tim1020/ecgo/util"
	"net/http"
	"os"
	"reflect"
)

//默认处理器
func (this *Request) defaultHandler(c EcgoApper) {
	this.Log.Write(LL_SYS, "[%s]defaultHandler start,action=%s,method=%s,id=%d", this.appId, this.Action, this.Method, this.Pk)
	rValue := reflect.ValueOf(c)
	rType := reflect.TypeOf(c)
	reciver := rValue.Elem().FieldByName("Request")
	reciver.Set(reflect.ValueOf(this))

	method, resExist := rType.MethodByName(this.Action)
	if !resExist && this.Conf["RESTful"] != "on" { //方法不存在且没有开启RESTful
		this.Log.Write(LL_SYS, "[%s]controller(action=%s) not found", this.appId, this.Action)
		this.ShowErr(404, fmt.Sprintf("Action Not Found(%s)!", this.Action))
	} else {
		args := []reflect.Value{rValue}
		//前置控制器
		onBefore, exist := rType.MethodByName(this.Conf["prefix_control"])
		if exist {
			this.Log.Write(LL_SYS, "[%s]prefix_control start: %s", this.appId, this.Conf["prefix_control"])
			this.Bm.Set("pre_control_start")
			onBefore.Func.Call(args)
			this.Bm.Set("pre_control_end")
			this.Log.Write(LL_SYS, "[%s]prefix_control finish", this.appId)
		}
		this.Log.Write(LL_SYS, "[%s]control %s start", this.appId, this.Action)
		this.Bm.Set("control_start")
		if resExist {
			method.Func.Call(args)
		} else {
			this.restControl()
		}
		this.Bm.Set("control_end")
	}
	this.Log.Write(LL_SYS, "[%s]control %s finish", this.appId, this.Action)
}

//静态文件服务
func (this *Request) staticHandler() {
	path := this.Req.URL.Path
	this.Log.Write(LL_SYS, "[%s]match static , path=%s", this.appId, path)
	file := this.Conf["static_path"] + path
	//todo: 缓存，content-type白名单
	f, err := os.Open(file)
	if err != nil && os.IsNotExist(err) { //自定义404
		this.ShowErr(404, fmt.Sprintf("File %s Not Found!", path))
		return
	}
	defer f.Close()
	staticHandler := http.FileServer(http.Dir(this.Conf["static_path"]))
	staticHandler.ServeHTTP(this.ResWriter, this.Req)
}

//显示运行状态
func (this *Request) statsHandler() {
	this.SetHeader("content-type", "text/html;chartset=utf8")
	fmt.Fprintf(this.ResWriter, "<h2>server status:</h2>")
	fmt.Fprintf(this.ResWriter, "<div>uptime:%s</div>", this.stats.uptime.Format("2006-01-02 15:03:04"))
	fmt.Fprintf(this.ResWriter, "<div>pv:</div>")
	fmt.Fprintf(this.ResWriter, "<li>Total:%d</li><li>最近5分钟:%d</li><li>5分钟峰值:%d</li>", this.stats.pv.total, this.stats.pv.num, this.stats.pv.max_num)
	fmt.Fprintf(this.ResWriter, "<div>traffic:</div>")
	fmt.Fprintf(this.ResWriter, "<li>Total:%d</li><li>最近5分钟:%d</li><li>5分钟峰值:%d</li>", this.stats.traffic.total, this.stats.traffic.num, this.stats.traffic.max_num)
	// var mem runtime.MemStats
	// runtime.ReadMemStats(&mem)
	// fmt.Fprintf(this.ResWriter, "<br/><div>mem:%d</div>", mem.Alloc)
}
