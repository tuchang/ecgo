## 简介

ecgo 是一个**易学、易用、易扩展**的web开发框架。核心功能包括：

- 自动规则路由

- 自动的RESTful服务（配置RESTful =on）：使用/:Resource/:Id格式的URI，可根椐不同的请求方法对资源进行操作
	+ 资源（Resource）对应数据库的表名
	+ :Id为数据库的主键的键(必须命名为id,且自增id)
	+ GET方法：指定Id时获取该行，不指定时返回全部（可使用参数 limit,order,field来限制），如 GET /User/1 表示获取id=1的用户
	+ POST方法：创建一条新记录（在body中指定字段及值），如 POST /User
	+ PUT方法： 更新一条记录（在body中指定字段及新的值）, 必须指定id,   如： PUT /User/1 表示修改id=1的用户信息
	+ DELETE方法： 删除指定ID的记录，如: DELETE /User/1
	
	自动化的RESTful只支持单表单一操作，如果需要复制的操作（比如删除用户时同时删除用户存在其它表中的资料），可自由定制，方法是实现一个以方法名和资源名组成的控制器方法，如：


	```
	//替换删除User的自动处理
	func(this *Controller) DeleteUser(){
		//进行处理
	}
	```



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
Licensed under LGPL ,see http://www.gnu.org/licenses/lgpl.html

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
	*ecgo.Request
}
func main() {
	log.Fatal(ecgo.Server(&C{},nil)) //第二参数可指定自定义sessionHandler
}
//controller
func (this *C) Action() {
	this.Render("a.tpl",data)
}
```


