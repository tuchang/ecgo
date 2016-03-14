package ecgo

import (
	. "github.com/tim1020/ecgo/dao"
	. "github.com/tim1020/ecgo/util"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func Server(c EcgoApper, sess SessionHandler) (err error) {
	confObj, err := LoadConf(confFile)
	checkError(err)
	conf, err := checkConf(confObj) //合法性检查，并读入配置项map
	checkError(err)
	logger := NewLogger(conf["log.level"], conf["log.path"])
	logger.Write(LL_SYS, "Applicatoin server start")
	app := &Application{
		Conf:          confObj,
		conf:          conf,
		Log:           logger,
		viewTemplates: make(map[string]*template.Template),
	}
	err = app.buildTemplate()
	checkError(err)
	app.newSession(sess)
	app.newStats()
	app.controller = c

	http.HandleFunc("/", app.dispatch)
	err = http.ListenAndServe(conf["listen"], nil)
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
	if this.conf["stats_page"] == "on" && strings.ToLower(r.RequestURI) == "/stats" {
		req.statsHandler()
		return
	}
	//请求结束时的处理
	defer req.finish()
	//静态文件服务
	if strings.HasPrefix(r.URL.Path, this.conf["static_prefix"]) { //静态
		req.staticHandler()
		return
	}
	//处理请求参数
	req.parseReq()
	//开启session
	if this.conf["session.auto_start"] == "on" {
		req.SessionStart()
	}
	//处理action
	req.defaultHandler(this.controller)
}

//初始化一个内置的sessionHandler
func (this *Application) newSession(s SessionHandler) {
	this.Log.Write(LL_SYS, "new session")
	if s != nil {
		this.sessHandler = s
	} else {
		switch this.conf["session.handler"] {
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
	interval, _ := strconv.ParseInt(this.conf["stats_interval"], 10, 64)
	this.stats = &stats{
		uptime:  time.Now(),
		pv:      &counter{interval: interval},
		traffic: &counter{interval: interval},
	}
}

//生成mc操作对象
func (this *Request) NewMcDao() *Mc {
	this.Log.Write(LL_SYS, "get Mc dao")
	if this.mcDao == nil {
		this.Log.Write(LL_SYS, "[%s]new mcdao,server=%s", this.appId, this.conf["db.mc_server"])
		this.mcDao = NewMc(this.conf["db.mc_server"])
	}
	return this.mcDao
}

//生成mysql操作对象
func (this *Request) NewMySQLDao(table string) (*MySQL, error) {
	this.Log.Write(LL_SYS, "[%s]get MySQL dao", this.appId)
	if this.mysqlDao == nil {
		oc, _ := strconv.Atoi(this.conf["db.max_open_conns"])
		ic, _ := strconv.Atoi(this.conf["db.max_idle_conns"])
		this.Log.Write(LL_SYS, "[%s]new mysql,dsn=%s,table=%s,openConn=%d,idleConn=%d", this.appId, this.conf["db.mysql_dsn"], table, oc, ic)
		mysql, err := NewMySQL(this.conf["db.mysql_dsn"], table, oc, ic)
		if err != nil {
			return nil, err
		}
		this.mysqlDao = mysql
	} else {
		this.Log.Write(LL_SYS, "[%s]MySQL dao already exists", this.appId)
	}
	return this.mysqlDao, nil
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
		if this.conf["log.access_log"] == "on" {
			fields := strings.Split(this.conf["log.access_log_format"], this.conf["log.access_log_sep"])
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
			this.Log.Write(LL_ACCESS, strings.Join(logs, this.conf["log.access_log_sep"]))
		}
		this.reloadConf()           //如有需要，重载配置
		err := this.buildTemplate() //如有需要，重编译模板
		if err != nil {
			eMsg := err.Error()
			for _, str := range strings.Split(eMsg, "\n") {
				this.Log.E(str)
			}
		}
		this.statsIncrease() //统计计数器增加
	}()
}
