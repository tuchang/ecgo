package ecgo

import (
	"bytes"
	"errors"
	"fmt"
	. "github.com/tim1020/ecgo/util"
	"github.com/tim1020/godaemon"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func Server(c EcgoApper, sess SessionHandler) (err error) {
	conf, err := LoadConf(confFile...)
	checkError(err)
	err = checkConf(conf)
	checkError(err)
	logger := NewLogger(conf["log.level"], conf["log.path"])
	logger.Write(LL_SYS, "Applicatoin server start")
	logger.Write(LL_SYS, "LoadConf:")
	logger.Write(LL_SYS, "====>")
	for k, v := range conf {
		logger.Write(LL_SYS, "%s=%v", k, v)
	}
	logger.Write(LL_SYS, "<====")

	app := &Application{
		Conf:          conf,
		Log:           logger,
		viewTemplates: make(map[string]*template.Template),
	}
	err = app.buildTemplate()
	checkError(err)
	app.newSession(sess)
	app.newStats()
	app.controller = c
	//接入godaemon
	mux1 := http.NewServeMux()
	mux1.HandleFunc("/", app.dispatch)
	log.Fatalln(godaemon.GracefulServe(app.Conf["listen"], mux1))
	return
}

//自动路由分派，在http.HandleFunc中调用
func (this *Application) dispatch(w http.ResponseWriter, r *http.Request) {
	if strings.ToLower(r.RequestURI) == "/favicon.ico" {
		return
	}
	req := &Request{
		appId:       Md5(time.Now().UnixNano(), 8),
		Bm:          NewBenchMark(),
		Application: this,
		ResWriter:   &resWriter{w, 0, 200},
		Req:         r,
	}
	this.Log.Write(LL_SYS, "[%s]request reach,dispatch start, path=%s", req.appId, r.URL.Path)

	//统计服务
	if this.Conf["stats_page"] == "on" && strings.ToLower(r.RequestURI) == "/stats" {
		req.statsHandler()
		return
	}
	//请求结束时的处理
	defer req.finish()
	//静态文件服务
	if strings.HasPrefix(r.URL.Path, this.Conf["static_prefix"]) { //静态
		req.staticHandler()
		return
	}
	//处理请求参数
	req.parseReq()
	//开启session
	if this.Conf["session.auto_start"] == "on" {
		req.SessionStart()
	}
	//处理action
	req.defaultHandler(this.controller)
}

//获取conf的值
func (this *Application) GetConf(key string, defaultVal ...string) (val string, exists bool) {
	if len(defaultVal) != 0 {
		setConfDefault(this.Conf, key, defaultVal[0])
	}
	val, exists = this.Conf[key]
	return
}

//重载配置
func (this *Application) reloadConf() {
	conf, err := LoadConf(confFile...)
	if err == nil {
		err = checkConf(conf)
	}
	if err != nil {
		this.Log.E("conf reload fail: %s", err.Error())
		return
	}
	this.Log.Write(LL_SYS, "conf reload ===>")
	for k, v := range conf {
		this.Log.Write(LL_SYS, "%s=%s", k, v)
	}
	this.Log.Write(LL_SYS, "===>")
	this.Conf = conf
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

//初始化一个内置的sessionHandler
func (this *Application) newSession(s SessionHandler) {
	this.Log.Write(LL_SYS, "new session")
	if s != nil {
		this.sessHandler = s
	} else {
		switch this.Conf["session.handler"] {
		case "file":
			this.sessHandler = &fileSession{}
		case "memcache":
			this.sessHandler = &mcSession{}
		}
	}
}

//初始化状态统计
func (this *Application) newStats() {
	this.Log.Write(LL_SYS, "new stats")
	interval, _ := strconv.ParseInt(this.Conf["stats_interval"], 10, 64)
	this.stats = &stats{
		uptime:  time.Now(),
		pv:      &counter{interval: interval},
		traffic: &counter{interval: interval},
	}
}

//请求结束时的处理
func (this *Request) finish() {
	this.sessionSave()
	go func() {
		//耗时统计
		this.Bm.Set("dispatch_end")
		tTotal := this.Bm.Get("start", "dispatch_end")
		tParse := this.Bm.Get("parse_req_start", "parse_req_end")
		tPre := this.Bm.Get("pre_control_start", "pre_control_end")
		tControl := this.Bm.Get("control_start", "control_end")
		tRender := this.Bm.Get("render_start", "render_end")
		tSessStart := this.Bm.Get("sess_start_start", "sess_start_finish")
		tSessSave := this.Bm.Get("sess_save_start", "sess_save_finish")
		//todo:如果是静态，简化输出内容
		this.Log.Write(LL_SYS, "[%s]request finish,[bench_time(ms):total=%d,parseReq=%d,sessStart=%d,sessSave=%d,preControl=%d,control=%d,render=%d]", this.appId, tTotal, tParse, tSessStart, tSessSave, tPre, tControl, tRender)
		//access_log
		if this.Conf["log.access_log"] == "on" {
			fields := strings.Split(this.Conf["log.access_log_format"], this.Conf["log.access_log_sep"])
			var logs []string
			for _, field := range fields {
				switch field {
				case "method":
					logs = append(logs, this.Req.Method)
				case "path":
					logs = append(logs, this.Req.URL.Path)
				case "code":
					logs = append(logs, strconv.Itoa(this.ResWriter.Code))
				case "size":
					logs = append(logs, strconv.Itoa(this.ResWriter.Length))
				case "execute_time":
					logs = append(logs, strconv.FormatInt(tTotal, 10))
				case "ua":
					logs = append(logs, this.Req.UserAgent())
				case "ip":
					logs = append(logs, this.Req.RemoteAddr)
				case "referer":
					logs = append(logs, this.Req.Referer())
				default:
					logs = append(logs, "-")
				}
			}
			this.Log.Write(LL_ACCESS, strings.Join(logs, this.Conf["log.access_log_sep"]))
		}

		if !this.mutex {
			this.mutex = true
			this.reloadConf()           //如有需要，重载配置
			err := this.buildTemplate() //如有需要，重编译模板
			this.mutex = false
			if err != nil {
				eMsg := err.Error()
				for _, str := range strings.Split(eMsg, "\n") {
					this.Log.E(str)
				}
			}
		}
		this.statsIncrease() //统计计数器增加
	}()
}
