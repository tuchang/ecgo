//定义session处理器接口，同时实现内置的基于file和基于memcache的处理器

package ecgo

import (
	"encoding/json"
	"fmt"
	. "github.com/tim1020/ecgo/dao"
	. "github.com/tim1020/ecgo/util"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

//开启session，若在配置中开启auto_start，由框架自动启动，否则，可在需要时手动执行
func (this *Request) SessionStart() {
	this.Bm.Set("sess_start_start")
	defer func() {
		this.Bm.Set("sess_start_end")
	}()
	if this.sessionOn { //已启动过，返回
		return
	}
	sid, exists := this.Cookie[this.Conf["session.sid"]]
	if !exists { //未存在，使用unixNano的md5值生成sid
		sid = Md5(time.Now().UnixNano())
	}
	this.Log.Write(LL_SYS, "[%s]session start,sid=%s", this.appId, sid)

	this.sessHandler.Open(sid, this.Conf)
	this.Session = this.sessHandler.Read()
	cookie := &http.Cookie{Name: this.Conf["session.sid"], Value: sid, HttpOnly: true}
	ct, _ := strconv.Atoi(this.Conf["session.cookie_lifetime"])
	if ct > 0 {
		cookie.Expires = time.Now().Add(time.Second * time.Duration(ct))
	}
	this.SetCookie(cookie)
	this.sessionOn = true
	//gc
	go func() {
		gd, _ := strconv.Atoi(this.Conf["session.gc_divisor"])
		if gd > 1 && rand.Intn(gd) == 0 {
			this.Log.Write(LL_SYS, "[%s]gc call", this.appId)
			gt, _ := strconv.ParseInt(this.Conf["session.gc_lifetime"], 10, 64)
			this.sessHandler.Gc(gt)
			return
		}
	}()
}

//设置一个session
func (this *Request) SessionSet(key string, val interface{}) {
	if !this.sessionOn {
		//pannic
		return
	}
	//todo: 限定val的类型
	this.sessHandler.Set(key, val)
	this.Session[key] = val
}

//销毁一个或多个session，如果不传参数，则销毁全部
func (this *Request) SessionUnset(keys ...interface{}) {
	if !this.sessionOn {
		//todo:pannic
		return
	}
	if keys == nil { //key为空，删除全部
		this.sessHandler.Destroy()
		for k, _ := range this.Session {
			delete(this.Session, k)
		}
	} else {
		for _, k := range keys {
			if key, ok := k.(string); ok {
				this.sessHandler.Set(key, nil)
				delete(this.Session, key)
			}
		}
	}
}

//保存SESSION(请求结束时自动调用)
func (this *Request) sessionSave() {
	if !this.sessionOn {
		return
	}
	this.Log.Write(LL_SYS, "[%s]session save", this.appId)
	this.Bm.Set("sess_save_start")
	this.sessHandler.Save()
	this.Bm.Set("sess_save_finish")
}

//内置handler,不导出
type fileSession struct {
	log    *Log
	file   string
	path   string
	change bool
	data   map[string]interface{}
}

func (this *fileSession) Open(sessId string, conf map[string]string) {
	this.path, _ = conf["session.path"]
	this.file = fmt.Sprintf("%s/%s/%s/%s", this.path, sessId[:2], sessId[2:4], sessId[4:]) //hash两层路径
	this.log.D("session open,file=%s", this.file)
}
func (this *fileSession) Set(key string, val interface{}) {
	this.change = true
	if val == nil {
		delete(this.data, key)
	} else {
		this.data[key] = val
	}
}
func (this *fileSession) Read() map[string]interface{} {
	this.data = make(map[string]interface{})
	_, err := os.Stat(this.file)
	if err == nil { //存在，读取
		os.Chtimes(this.file, time.Now(), time.Now()) //设置一下最后更新时间
		fi, err := os.Open(this.file)
		if err != nil {
			this.log.E("[filesession err]: open file fail,file=%s", this.file)
		}
		defer fi.Close()
		content, err := ioutil.ReadAll(fi)
		if err != nil {
			this.log.E("[filesession err]: file read fail,file=%s", this.file)
		}
		if err := json.Unmarshal(content, &this.data); err != nil {
			this.log.E("[filesession err]: jsondecode fail,file=%s,err=%v", this.file, err)
		}
	} else {
		this.log.E("[filesession err]: file stat fail,file=%s,err=%v", this.file, err)
	}
	this.log.D("session read,data=%v", this.data)
	return this.data
}
func (this *fileSession) Destroy() {
	os.Remove(this.file)
	for k, _ := range this.data {
		delete(this.data, k)
	}
}
func (this *fileSession) Save() {
	if this.change {
		data, err := json.Marshal(this.data)
		if err != nil {
			this.log.E("[filesession err]: json encode error while save,data=%#v,err=%#v", this.data, err)
			return
		}
		path := filepath.Dir(this.file)
		if _, err1 := os.Stat(path); err1 != nil && os.IsNotExist(err1) { //目录不存在，先创建
			os.MkdirAll(path, os.ModePerm)
		}
		fd, err2 := os.OpenFile(this.file, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, os.ModePerm)
		if err2 == nil {
			_, err2 = fd.Write(data)
		}
		if err2 != nil {
			this.log.E("[filesession err]: write file error,err=%#v", err2)
		}
	}
}
func (this *fileSession) Gc(maxLife int64) {
	//todo: session文件gc
	//遍历this.path,获取每一文件的更新时间，如果距离现在超过maxLife，则删除
}

//内置handler,不导出
type mcSession struct {
	log    *Log
	mc     *Mc
	key    string
	change bool
	data   map[string]interface{}
}

func (this *mcSession) Open(sessId string, conf map[string]string) {
	this.key = "sess_" + sessId
	mcServer, _ := conf["session.mc_server"]
	this.mc = NewMc(mcServer)
}
func (this *mcSession) Set(key string, val interface{}) {
	this.change = true
	if val == nil {
		delete(this.data, key)
	} else {
		this.data[key] = val
	}
}
func (this *mcSession) Read() map[string]interface{} {
	this.data = make(map[string]interface{})
	content, err := this.mc.Get(this.key)
	if err == nil {
		err = json.Unmarshal(content, &this.data)
	}
	if err != nil {
		this.log.E("[mcsession err]: mc get or jsondecode error,err=%v", err)
	}
	this.log.D("session read,data=%v", this.data)
	return this.data
}
func (this *mcSession) Destroy() {
	for k, _ := range this.data {
		delete(this.data, k)
	}
	this.mc.Client.Delete(this.key)
}
func (this *mcSession) Save() {
	if this.change {
		data, _ := json.Marshal(this.data)
		//TODO: 根椐生命周期设置expire
		if err := this.mc.Set(this.key, data); err != nil {
			//todo:失败处理
		}
	}
}
func (this *mcSession) Gc(maxLife int64) {
	//memcache自动过期
}
