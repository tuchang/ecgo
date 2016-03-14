//通用处理： 配置处理和模板处理（包括服务启动时读取和检查，以及请求结束时判断是否有更新并在需要时重新载入）

package ecgo

import (
	"bytes"
	"errors"
	"fmt"
	. "github.com/tim1020/ecgo/util"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	RootPath   string //应用根目录（执行文件所在目录)
	RequestSep string //Get/Post属性中，对同名参数内容使用的分隔符
	confFile   string //配置文件
	viewPath   string //模板路径
	viewMTime  int64  //模板编译时间
)

//包初始化
func init() {
	//确定运行目录
	file, _ := filepath.Abs(os.Args[0])
	RootPath = filepath.Dir(file) //将执行文件所在的路径设为应用的根路径
	confFile = RootPath + "/conf/conf.ini"
	viewPath = RootPath + "/views/"
}

//处理错误
func checkError(err error) {
	if err != nil {
		log.Fatal("server start fail\n--------------------\n", err.Error())
	}
}

//检查conf文件，并读入字典
func checkConf(conf *Conf) (m map[string]string, err error) {
	var num int
	var errs []string
	m = make(map[string]string)
	//default
	m["listen"], _ = conf.Get("listen", "localhost:8088")
	m["prefix_control"], _ = conf.Get("prefix_control", "PreControl")
	RequestSep, _ = conf.Get("request_sep", "&")

	m["static_path"], _ = conf.Get("static_path", RootPath)
	m["static_prefix"], _ = conf.Get("static_prefix", "/public/")
	m["stats_page"], _ = conf.Get("stats_page", "")
	m["RESTful"], _ = conf.Get("RESTful", "")
	m["stats_interval"], _ = conf.Get("stats_interval", "30")
	if _, err := strconv.Atoi(m["stats_interval"]); err != nil {
		errs = append(errs, fmt.Sprintf("stats_interval: %s not a number", m["stats_interval"]))
	}
	//log
	m["log.level"], _ = conf.Get("log.level", LL_ALL)
	m["log.path"], _ = conf.Get("log.path", RootPath+"/logs")
	m["log.access_log"], _ = conf.Get("log.access_log", "")
	m["log.access_log_format"], _ = conf.Get("log.access_log_format", "method path code execute_time size")
	//检查分隔符
	seps := []string{" ", "`", ",", "|", "&"}
	for _, sep := range seps {
		if strings.Contains(m["log.access_log_format"], sep) {
			m["log.access_log_sep"] = sep
			break
		}
	}
	if _, ok := m["log.access_log_sep"]; !ok {
		errs = append(errs, "log.access_log_sep: not support,expect (\"`,|&\")")
		m["log.access_log_sep"] = " "
	}
	//检查字段
	af := "method,path,code,size,execute_time,ua,ip,referer"
	files := strings.Split(m["log.access_log_format"], m["log.access_log_sep"])
	for _, f := range files {
		if !strings.Contains(af, f) {
			errs = append(errs, fmt.Sprintf("log.access_log_format: field=%s not support", f))
		}
	}
	//session
	m["session.auto_start"], _ = conf.Get("session.auto_start", "")
	m["session.handler"], _ = conf.Get("session.handler", "file")
	if m["session.handler"] != "file" && m["session.handler"] != "memcache" {
		errs = append(errs, fmt.Sprintf("session.handler: ivalid handler %s", m["session.handler"]))
	}
	m["session.sid"], _ = conf.Get("session.sid", "ECGO_SID")
	m["session.cookie_lifetime"], _ = conf.Get("session.cookie_lifetime", "0") //判断是否数字
	if _, err := strconv.Atoi(m["session.cookie_lifetime"]); err != nil {
		errs = append(errs, fmt.Sprintf("session.cookie_lifetime: %s not a number", m["session.cookie_lifetime"]))
	}
	m["session.gc_divisor"], _ = conf.Get("session.gc_divisor", "10")
	num, err = strconv.Atoi(m["session.gc_divisor"])
	if err != nil {
		errs = append(errs, fmt.Sprintf("session.gc_divisor: %s not a number", m["session.gc_divisor"]))
	} else if num < 1 || num > 100 {
		errs = append(errs, "session.gc_divisor: expect 1-100")
	}
	m["session.gc_lifetime"], _ = conf.Get("session.gc_lifetime", "36000")
	if _, err := strconv.Atoi(m["session.gc_lifetime"]); err != nil {
		errs = append(errs, fmt.Sprintf("session.gc_lifetime: %s not a number", m["session.gc_lifetime"]))
	}
	//db
	m["db.mc_server"], _ = conf.Get("db.mc_server", "")
	m["db.mysql_dsn"], _ = conf.Get("db.mysql_dsn", "")
	m["db.max_open_conns"], _ = conf.Get("db.max_open_conns", "100")
	num, err = strconv.Atoi(m["db.max_open_conns"])
	if err != nil {
		errs = append(errs, fmt.Sprintf("db.max_open_conns: %s not a number", m["db.max_open_conns"]))
	} else if num < 10 || num > 1000 {
		errs = append(errs, "db.max_open_conns: expect 10-1000")
	}
	m["db.max_idle_conns"], _ = conf.Get("db.max_idle_conns", "20")
	num, err = strconv.Atoi(m["db.max_idle_conns"])
	if err != nil {
		errs = append(errs, fmt.Sprintf("db.max_idle_conns: %s not a number", m["db.max_idle_conns"]))
	} else if num < 1 || num > 100 {
		errs = append(errs, "db.max_idle_conns error: expect 1-100")
	}

	//upload
	m["upload.path"], _ = conf.Get("upload.path", os.TempDir()+"/upload")
	m["upload.allow_mime"], _ = conf.Get("upload.allow_mime", "")
	size, _ := conf.Get("upload.max_size", "1M")
	l := len(size)
	unit := strings.ToUpper(size[l-1:])
	val := size[0 : l-1]
	num, err = strconv.Atoi(val)
	if err != nil || num < 1 || num > 1000 || (unit != "M" && unit != "K") {
		errs = append(errs, "upload.max_size: expect 1-1000(K or M)")
	}
	if unit == "M" {
		m["upload.max_size"] = strconv.Itoa(num * 1024 * 1024)
	} else if unit == "K" {
		m["upload.max_size"] = strconv.Itoa(num * 1024)
	}
	if len(errs) > 0 {
		err = errors.New("[conf error]:\n" + strings.Join(errs, "\n"))
	}
	return
}

//载入模板
func (this *Application) buildTemplate() (err error) {
	files, _ := ioutil.ReadDir(viewPath)
	need := false
	for _, f := range files { //遍历模板目录,
		if f.IsDir() {
			continue
		}
		file := viewPath + f.Name()
		stat, _ := os.Stat(file)
		if stat.ModTime().Unix() > viewMTime { //有新文件
			need = true
			this.Log.Write(LL_SYS, "build template")
			break
		}
	}
	var errs []string
	if need {
		for _, f := range files { //遍历编译
			file := viewPath + f.Name()
			mf, err := os.Open(file)
			if err != nil {
				this.Log.E("template fail: can not open file %s", f.Name())
				errs = append(errs, fmt.Sprintf("读取模板文件失败: file=%s", f.Name()))
				continue
			}
			defer mf.Close()
			content, _ := ioutil.ReadAll(mf)
			reg := regexp.MustCompile(`\{\{#include "(.*)"\}\}`)
			//遍历引用的子文件
			var notExistsIncFile []string
			for _, v := range reg.FindAllSubmatch(content, -1) { //遍历匹配到的内容进行替换
				incFile := fmt.Sprintf("%s/%s", viewPath, v[1]) //同一层目录
				f1, err := os.Open(incFile)
				if err != nil {
					ifile := string(v[1])
					errs = append(errs, fmt.Sprintf("读取include模板文件失败： file=%s, include=%s", f.Name(), ifile))
					notExistsIncFile = append(notExistsIncFile, ifile)
					break
				}
				defer f1.Close()
				incContent, _ := ioutil.ReadAll(f1)
				content = bytes.Replace(content, v[0], incContent, 1)
			}
			if len(notExistsIncFile) > 0 {
				this.Log.E("template fail: can not open include file, file=%s, include=(%s)", f.Name(), strings.Join(notExistsIncFile, ","))
			} else {
				this.Log.Write(LL_SYS, "template file=%s,ok", f.Name())
				this.viewTemplates[f.Name()], _ = template.New(f.Name()).Parse(string(content))
			}
		}
		viewMTime = time.Now().Unix()
	}
	if len(errs) > 0 { //有错
		err = errors.New(strings.Join(errs, "\n"))
	}
	return
}

//reload conf
func (this *Application) reloadConf() {
	//todo: 锁？
	f, err := os.Open(confFile)
	if err != nil {
		this.Log.E("reload conf error: conf file %s not found", f)
		return
	} else {
		stat, _ := f.Stat()
		if stat.ModTime().Unix() > this.Conf.Mtime { //配置文件有修改,重新载入
			confObj, err := LoadConf(confFile)
			if err != nil {
				this.Log.E("%s", err.Error())
			} else {
				tempConf, err := checkConf(confObj)
				if err != nil {
					this.Log.E("[conf error]\n%s", err.Error())
				} else { //配置文件正常，reload
					this.Log.Write(LL_SYS, "conf reload")
					for k, v := range tempConf {
						this.Log.Write(LL_SYS, "%s=%s", k, v)
					}
					this.Conf = confObj
					//如果不是自定义sessionHandler,重新设置
					_, isFileSession := this.sessHandler.(*fileSession)
					_, isMcSession := this.sessHandler.(*mcSession)
					if isFileSession || isMcSession {
						if this.conf["session.handler"] != tempConf["session.handler"] {
							this.Log.Write(LL_SYS, "reload session handler to %s", tempConf["session.handler"])
							this.newSession(nil)
						}
					}
					this.conf = tempConf
				}
			}
		}
	}
}

//计数器增加num
func (this *counter) add(num int64) {
	this.total += num
	now := time.Now().Unix()
	if now > this.time+this.interval {
		this.time = now
		this.num = num
	} else {
		this.num += num
		if this.max_num < this.num {
			this.max_num = this.num
		}
	}
}

//统计器增加计数
func (this *Request) statsIncrease() {
	this.stats.pv.add(1)
	this.stats.traffic.add(int64(this.ResWriter.Length))
}
