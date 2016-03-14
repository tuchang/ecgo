//对responseWriter行二次封装，提供更方便的操作header,cookie和输出响应、渲染模板等方法

package ecgo

import (
	"errors"
	"fmt"
	. "github.com/tim1020/ecgo/util"
	"net/http"
	"strconv"
	"time"
)

func (this *resWriter) Write(b []byte) (n int, err error) {
	n, err = this.ResponseWriter.Write(b)
	this.Length += n
	return
}
func (this *resWriter) WriteHeader(code int) {
	this.ResponseWriter.WriteHeader(code)
	this.Code = code
}

//在响应中添加Header,在body输出前调用
func (this *Request) SetHeader(key, val string) {
	this.ResWriter.Header().Set(key, val)
}

//在响应中添加cookie，在body输出前调用
//
//支持三种方式:
//
//1. SetCookie(name,val string) 2.SetCookie(name string,val string,expire int) 3.SetCookie(c *http.Cookie)
func (this *Request) SetCookie(c ...interface{}) (err error) {
	len := len(c)
	switch {
	case len == 1: //http.Cookie
		if cookie, ok := c[0].(*http.Cookie); ok {
			http.SetCookie(this.ResWriter, cookie)
			return
		}
	case len == 2: //name,val
		name, ok1 := c[0].(string)
		val, ok2 := c[1].(string)
		if ok1 && ok2 {
			http.SetCookie(this.ResWriter, &http.Cookie{Name: name, Value: val})
			return
		}
	case len == 3: //name,val,expire
		name, ok1 := c[0].(string)
		val, ok2 := c[1].(string)
		t, ok3 := c[2].(int)
		if ok1 && ok2 && ok3 {
			expires := time.Now().Add(time.Second * time.Duration(t))
			http.SetCookie(this.ResWriter, &http.Cookie{Name: name, Value: val, Expires: expires})
			return
		}
	}
	//if here,something wrong
	err = errors.New("params wrong")
	this.Log.W("SetCookie FAIL: params=%v", c)
	return
}

//重定向至url
func (this Request) Redirect(url string) {
	http.Redirect(this.ResWriter, this.Req, url, http.StatusFound)
}

//响应一个错误,可在view目录放置以statusCode为名称的模板,没有模板时，使用内置格式显示
func (this *Request) ShowErr(statusCode int, msg string) {
	this.ResWriter.WriteHeader(statusCode)
	code := strconv.Itoa(statusCode)
	if t, exists := this.viewTemplates[code]; exists {
		data := map[string]string{"statusCode": code, "message": msg}
		t.Execute(this.ResWriter, data)
	} else {
		html := `<!DOCTYPE html>
			<html lang="zh-CN">
			<head><title>%s</title></head>
			<body>
			<h2>%d %s</h2><li>%s
			</body>
			</html>
		`
		sText := http.StatusText(statusCode)
		fmt.Fprintf(this.ResWriter, html, sText, statusCode, sText, msg)
	}
}

//渲染模板并输出(模板存放在RootPath下的views目录)
func (this *Request) Render(tplName string, data interface{}) {
	this.Log.Write(LL_SYS, "[%s]render start,tplName=%s,data=%v", this.appId, tplName, data)
	this.Bm.Set("render_start")
	defer func() {
		this.Bm.Set("render_end")
		this.Log.Write(LL_SYS, "[%s]render finish", this.appId)
	}()
	if t, exists := this.viewTemplates[tplName]; exists {
		t.Execute(this.ResWriter, data)
	} else {
		fmt.Fprintf(this.ResWriter, "tpl not found")
	}
}

//使用格式化字串输出响应
func (this *Request) Resp(format string, data ...interface{}) {
	fmt.Fprintf(this.ResWriter, format, data...)
	return
}
