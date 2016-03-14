## 简介

ecgo 是一个**易学、易用、易扩展**的web开发框架。核心功能包括：

- 自动规则路由，支持RESTful

- request的二次封装
	+ 可以直接使用格式化的Get,Post，Cookie，Session等变量来处理请求数据
	+ 方便的上传文件操作

- response二次封装
	+ 添加SetCookie,SetHeader,ShowErr,Redirect等方法
	+ 支持模板渲染Render,模板支持include子模板

- 内置基于文件和memcache的session支持，同时支持自定义sessionHandler

- 支持静态文件服务

- 提供ini配置文件读取，benchmark,log等辅助方法

- 支持mysql和memcache的dao封装，简化数据操作

- 提供pv、流量的实时统计

- 其它
	+ 配置文件和预编译模板的实时重新加载
	+ 提供validator,支持扩展规则
	+ daemon (github.com/tim1020/godaemon)


## 版权说明

Copyright 2016 ecgo Author. All Rights Reserved.
Licensed under the Apache License, Version 2.0 (the "License");
You may not use this file except in compliance with the License.
You may obtain a copy of the License at :  http://www.apache.org/licenses/LICENSE-2.0

如有任何疑问，请联系本人： tim8670@gmail.com 


## 使用说明

### 目录结构

- 开发目录 ($GOPATH/src/yourapp)
	
	+ controller/   #放置controller代码
	+ service/      #放置model相关代码
	+ conf/		    #存放配置文件
	+ logs/			#日志目录
	+ public/		#静态文件目录
	+ views/		#视图模板目录
	+ yourapp.go	#main入口文件

其中，安装时，conf,public,views三个目录会完整copy到安装目录

- 安装目录(应用开发完成后，正式部署的目录结构）

	+ conf/		 #存放配置文件
	+ logs/		 #日志目录(要求可写)
	+ public/    #静态文件目录
	+ views/     #视图模板目录
	+ yourapp    #应用执行文件

### 使用ecgo辅助工具创建应用

(首先，你要安装好你的go开发环境，并设置好$GOPATH)

- 获得ecgo

	```
	go get github.com/tim1020/ecgo
	chmod +x $GOPATH/src/github.com/tim1020/ecgo/tools/ecgo
	```

- 创建应用

	```
	cd $GOPATH/src/github.com/tim1020/ecgo/tools/
	./ecgo new appname
	```

(名为appname的应用将会被创建在$GOPATH/src/appname)

- 编写你的应用
	+ 根椐需要，修改conf.ini
	+ 编写你的service和controller

- 安装应用到指定目录

	```
	cd $GOPATH/src/github.com/tim1020/ecgo/tools
	./ecgo install appname dst
	```


### 代码示例

```
package main
import "github.com/tim1020/ecgo"
type C struct {
	*ecgo.ReqSess
}
func main() {
	ec := ecgo.New()
	//ec.SetSessHandler(myHandler) 指定自定义sessionHandler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ec.Dispatch(w, r, &C{})
	})
	log.Fatal(http.ListenAndServe(ec.ListenAddr, nil))
}
//controller
func (this *C) Action() {
	this.Render("a.tpl",data)
}
```


