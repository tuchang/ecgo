//解释请求内容，放到相应的成员变量中

package ecgo

import (
	"fmt"
	. "github.com/tim1020/ecgo/util"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

//上传文件时的错误代码
const (
	UPLOAD_ERR_OK            = iota
	UPLOAD_ERR_SIZE_OVERFLOW //超出设置的大小限制
	UPLOAD_ERR_TYPE_NOTALLOW //不允许的类型
	UPLOAD_ERR_TEMP          //临时目录不可用
	UPLOAD_ERR_CANT_WRITE    //文件写入失败
)

/**
 * 对http请求进行格式化处理, 并将结果存入App的成员变量 Get/Post/Cookie/Header/UpFile
 */
func (this *Request) parseReq() {
	this.Log.Write(LL_SYS, "[%s]parse request start", this.appId)
	this.Bm.Set("parse_req_start")
	defer func() {
		this.Bm.Set("parse_req_end")
		this.Log.Write(LL_SYS, "[%s]parse request finish", this.appId)
	}()
	m := false
	ct := this.Req.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		m = true
		this.Req.ParseMultipartForm(10 << 20)
	} else if ct == "application/x-www-form-urlencoded" {
		this.Req.ParseForm()
	}
	this.Header = getHeader(this.Req)
	this.Cookie = getCookie(this.Req)
	this.Get = getGet(this.Req)
	this.Post = getPost(this.Req, m)
	if m {
		this.UpFile = getFile(this.Req, this.conf)
	}
	this.Method = this.Req.Method
	RESTful := false
	if this.conf["RESTful"] == "on" {
		RESTful = true
	}
	this.ActionName, this.ActionParams = parsePath(RESTful, this.Req)

	this.Log.Write(LL_SYS, "[%s]method=%s, actionName=%s,actionParams=%s", this.appId, this.Method, this.ActionName, this.ActionParams)
	this.Log.Write(LL_SYS, "[%s]get =>%v", this.appId, this.Get)
	this.Log.Write(LL_SYS, "[%s]post =>%v", this.appId, this.Post)
	this.Log.Write(LL_SYS, "[%s]cookie =>%v", this.appId, this.Cookie)
	this.Log.Write(LL_SYS, "[%s]file =>%v", this.appId, this.UpFile)
}

//处理path
func parsePath(RESTfulOn bool, req *http.Request) (actName string, actParams []string) {
	path := strings.Split(req.URL.Path, "/")
	l := len(path)
	if path[1] != "" {
		if RESTfulOn {
			var action string
			for i := 1; i < l; i++ {
				if i%2 != 0 {
					action += strings.Title(strings.ToLower(path[i]))
				} else {
					actParams = append(actParams, path[i])
				}
			}
			actName = req.Method + string(action)
		} else {
			for i := 1; i < l; i++ {
				actName += strings.Title(strings.ToLower(path[i]))
			}
		}

	}
	return
}

//获取header
func getHeader(req *http.Request) (header map[string]string) {
	header = make(map[string]string)
	for k, v := range req.Header {
		header[k] = strings.Join(v, ";")
	}
	return
}

//获取cookie
func getCookie(req *http.Request) (cookie map[string]string) {
	cookie = make(map[string]string)
	for _, v := range req.Cookies() {
		cookie[v.Name] = v.Value
	}
	return
}

//获取GET参数，同名参数内容以req_sep串接
func getGet(req *http.Request) (get map[string]string) {
	get = make(map[string]string)
	q, _ := url.ParseQuery(req.URL.RawQuery)
	for k, v := range q {
		k = strings.TrimSuffix(k, "[]") //如果是xxx[]方式的key,只保留xx,所以 xx和xx[]会相互覆盖
		get[k] = strings.Join(v, RequestSep)
	}
	return
}

//获取post参数,m表示是否multiPart方式请求,同名参数内容以req_sep串接
func getPost(req *http.Request, m bool) (post map[string]string) {
	post = make(map[string]string)
	var vals url.Values
	if m && req.MultipartForm != nil {
		vals = req.MultipartForm.Value
	} else {
		vals = req.PostForm
	}
	for k, v := range vals {
		k = strings.TrimSuffix(k, "[]") //如果是xxx[]方式的key,只保留xx,所以 xx和xx[]会相互覆盖
		post[k] = strings.Join(v, RequestSep)
	}
	return
}

//处理上传文件
func getFile(req *http.Request, conf map[string]string) map[string][]UpFile {
	f := make(map[string][]UpFile)
	if req.MultipartForm != nil {
		for k, v := range req.MultipartForm.File { //k为上传字段
			k = strings.TrimSuffix(k, "[]") //如果是xxx[]方式的key,只保留xx,所以 xx和xx[]会相互覆盖
			for _, v1 := range v {
				mime := v1.Header.Get("Content-Type")
				uf := UpFile{Error: UPLOAD_ERR_OK, Name: v1.Filename, Type: mime}
				if conf["upload.allow_mime"] != "all" && !strings.Contains(conf["upload.allow_mime"], mime) {
					uf.Error = UPLOAD_ERR_TYPE_NOTALLOW
					f[k] = append(f[k], uf)
					continue
				}
				file, _ := v1.Open()
				defer file.Close()
				fname := Md5(time.Now().UnixNano())
				fpath := fmt.Sprintf("%s/%s/%s/", conf["path"], fname[:2], fname[2:4])
				if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
					uf.Error = UPLOAD_ERR_TEMP
					f[k] = append(f[k], uf)
					continue
				}
				tmp := fpath + fname[4:]
				f1, err := os.Create(tmp)
				if err != nil {
					uf.Error = UPLOAD_ERR_TEMP
					f[k] = append(f[k], uf)
					continue
				}
				size, err := io.Copy(f1, file)
				if err != nil {
					uf.Error = UPLOAD_ERR_CANT_WRITE
					f[k] = append(f[k], uf)
					continue
				}
				f1.Close()
				mSize, _ := strconv.ParseInt(conf["upload.max_size"], 10, 64)
				if size > mSize {
					uf.Error = UPLOAD_ERR_SIZE_OVERFLOW
					f[k] = append(f[k], uf)
					os.Remove(tmp)
					continue
				}
				uf.Size = size
				uf.Temp = tmp
				f[k] = append(f[k], uf)
			}
		}
		//fmt.Printf("%q", f)
	}
	return f
}

//TODO: XSS处理
