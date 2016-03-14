// Copyright 2016 ecgo Author. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at :  http://www.apache.org/licenses/LICENSE-2.0

// 一个易学、易用、易扩展的web开发框架。核心功能包括：
//
// 1. 自动规则路由
//
// 2. request的二次封装，可以直接使用格式化的Get,Post，Cookie，Session等变量来处理请求数据
//
// 3. response二次封装，添加SetCookie,Render等常用方法
//
// 4. 内置基于文件和memcache的session支持，同时支持自定义sessionHandler
//
// 5. 支持静态文件服务
//
// 6. 提供ini配置文件读取，benchmark,log等辅助方法
//
// 7. mysql和mc的dao
//
// 更多内容请参考： http://github.com/tim1020/ecgo
//
// 基本使用方法：
//
//	package main
//	import "github.com/tim1020/ecgo"
//	type C struct {
//		*ecgo.Request
//	}
//	func main() {
//		log.Fatal(ecgo.Serve())
//	}
//	//controller
//	func (this *C) Action() {
//		//this.Render("a.tpl",data)
//	}
//
package ecgo

import (
	. "github.com/tim1020/ecgo/dao"
	. "github.com/tim1020/ecgo/util"
	"html/template"
	"net/http"
	"time"
)

//框架方法接口，用来判断dispatch时传递的对象是否组合了框架的核心方法
type EcgoApper interface {
	Render(tpl string, data interface{})
	SetHeader(key, val string)
	SetCookie(c ...interface{}) error
	Redirect(url string)
	ShowErr(statusCode int, msg string)
	SessionStart()
	SessionSet(key string, val interface{})
	SessionUnset(keys ...interface{})
	sessionSave()
	newSession(s SessionHandler)
	newStats()
	parseReq()
	dispatch(w http.ResponseWriter, r *http.Request)
}

//Session处理接口
type SessionHandler interface {
	Open(sessId string, config *Conf) //开启session时调用
	Set(key string, val interface{})  //写入session，给Session赋值时调用,val设为nil时，表示删除
	Read() map[string]interface{}     //读取session，返回反序列后的格式数据，用来设置到Session的map
	Destroy()                         //销毁一个session（SessionDestory时调用）
	Save()                            //将session的值序列化后持久化保存(请求完成时或SessionWrite时)
	Gc(maxLife int64)                 //过期数据清理,系统按特定机率触发
}

//服务对象，生命周期为整个程序运行时,服务启动时创建
type Application struct {
	Conf          *Conf                         //配置文件操作对象
	Log           *Log                          //日志操作对象
	stats         *stats                        //统计器对象
	sessHandler   SessionHandler                //session处理器
	viewTemplates map[string]*template.Template //编译过的模板字典
	conf          map[string]string             //配置内容项
	controller    EcgoApper
}

//请求会话对象，生命周期为一次请求，请求到达时创建
type Request struct {
	*Application     //继承Conf,Bm,Log和SessHandler
	Bm           *Bm //benchMark操作
	appId        string
	sessionOn    bool

	ResWriter *resWriter
	Req       *http.Request

	UpFile       map[string][]UpFile    //存放上传的文件信息
	Get          map[string]string      //存放Get参数
	Post         map[string]string      //存放Post/put参数
	Cookie       map[string]string      //存放cookie
	Header       map[string]string      //存放header
	Session      map[string]interface{} //存放session
	Method       string                 //请求的方法 GET/POST...
	ActionName   string                 //path中的资源名称
	ActionParams []string               //path中资源的id列表

	mcDao    *Mc
	mysqlDao *MySQL
}

//上传文件信息结构
type UpFile struct {
	Error int    //错误码，没有错误时为0
	Name  string //上传原始的文件名
	Size  int64  //文件大小
	Type  string //文件content-type
	Temp  string //上传后保存在服务器的临时文件路径
}

//自定义responseWriter,增加Length和Code
type resWriter struct {
	http.ResponseWriter
	Length int
	Code   int
}

//计数器
type counter struct {
	interval int64 //切换的时间长度(s)
	time     int64 //上次切换的时间点
	total    int64 //总数
	num      int64 //当前时间段计数
	max_num  int64 //时间段计数的峰值
}

//统计对象
type stats struct {
	uptime  time.Time //启动的时间
	pv      *counter
	traffic *counter
}

//view对象
type tpl struct {
	mtime     int64                         //上次编译时间
	path      string                        //模板目录
	templates map[string]*template.Template //模板
}
