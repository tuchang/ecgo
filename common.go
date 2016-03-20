//通用处理： 配置处理和模板处理（包括服务启动时读取和检查，以及请求结束时判断是否有更新并在需要时重新载入）

package ecgo

import (
	"errors"
	"fmt"
	. "github.com/tim1020/ecgo/dao"
	. "github.com/tim1020/ecgo/util"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	RootPath   string   //应用根目录（执行文件所在目录)
	RequestSep string   //Get/Post属性中，对同名参数内容使用的分隔符
	confFile   []string //配置文件
	viewPath   string   //模板路径
	viewMTime  int64    //模板编译时间
)

//包初始化
func init() {
	//确定运行目录
	file, _ := filepath.Abs(os.Args[0])
	RootPath = filepath.Dir(file) //将执行文件所在的路径设为应用的根路径
	//遍历获取所有配置文件
	cPath := RootPath + "/conf/"
	files, _ := ioutil.ReadDir(cPath)
	for _, f := range files { //遍历模板目录,
		if f.IsDir() {
			continue
		}
		if filepath.Ext(f.Name()) == ".ini" {
			confFile = append(confFile, cPath+f.Name())
		}
	}
	//模板目录
	viewPath = RootPath + "/views/"
}

//处理错误
func checkError(err error) {
	if err != nil {
		log.Fatal("server start fail\n--------------------\n", err.Error())
	}
}

//检查conf
func checkConf(conf map[string]string) (err error) {
	var num int
	var errs []string
	//default
	setConfDefault(conf, "listen", ":8080")
	setConfDefault(conf, "prefix_control", "PreControl")
	setConfDefault(conf, "request_sep", "&")
	setConfDefault(conf, "static_path", RootPath+"/public/")
	setConfDefault(conf, "static_prefix", "/public/")
	setConfDefault(conf, "stats_page", "off")
	setConfDefault(conf, "RESTful", "off")
	setConfDefault(conf, "stats_interval", "30")
	if _, err := strconv.Atoi(conf["stats_interval"]); err != nil {
		errs = append(errs, fmt.Sprintf("stats_interval: %s not a number", conf["stats_interval"]))
	}
	//log
	setConfDefault(conf, "log.level", LL_ALL)
	setConfDefault(conf, "log.path", RootPath+"/logs")
	setConfDefault(conf, "log.access_log", "off")
	setConfDefault(conf, "log.access_log_format", "method path code execute_time size")
	//检查分隔符
	seps := []string{" ", "`", ",", "|", "&"}
	for _, sep := range seps {
		if strings.Contains(conf["log.access_log_format"], sep) {
			conf["log.access_log_sep"] = sep
			break
		}
	}
	setConfDefault(conf, "log.access_log_sep", " ")
	//检查字段
	af := "method,path,code,size,execute_time,ua,ip,referer"
	files := strings.Split(conf["log.access_log_format"], conf["log.access_log_sep"])
	for _, f := range files {
		if !strings.Contains(af, f) {
			errs = append(errs, fmt.Sprintf("log.access_log_format: field=%s not support", f))
		}
	}
	//session
	setConfDefault(conf, "session.auto_start", "off")
	setConfDefault(conf, "session.handler", "file")
	if conf["session.handler"] != "file" && conf["session.handler"] != "memcache" {
		errs = append(errs, fmt.Sprintf("session.handler: ivalid handler %s", conf["session.handler"]))
	}
	setConfDefault(conf, "session.path", os.TempDir()+"/sess")
	setConfDefault(conf, "session.sid", "ECGO_SID")
	setConfDefault(conf, "session.cookie_lifetime", "0")
	if _, err := strconv.Atoi(conf["session.cookie_lifetime"]); err != nil {
		errs = append(errs, fmt.Sprintf("session.cookie_lifetime: %s not a number", conf["session.cookie_lifetime"]))
	}
	setConfDefault(conf, "session.gc_divisor", "10")
	num, err = strconv.Atoi(conf["session.gc_divisor"])
	if err != nil {
		errs = append(errs, fmt.Sprintf("session.gc_divisor: %s not a number", conf["session.gc_divisor"]))
	} else if num < 1 || num > 100 {
		errs = append(errs, "session.gc_divisor: expect 1-100")
	}
	setConfDefault(conf, "session.gc_lifetime", "36000")
	if _, err := strconv.Atoi(conf["session.gc_lifetime"]); err != nil {
		errs = append(errs, fmt.Sprintf("session.gc_lifetime: %s not a number", conf["session.gc_lifetime"]))
	}
	//db
	setConfDefault(conf, "db.mc_server", "")
	setConfDefault(conf, "db.mysql_dsn", "")
	setConfDefault(conf, "db.max_open_conns", "100")
	num, err = strconv.Atoi(conf["db.max_open_conns"])
	if err != nil {
		errs = append(errs, fmt.Sprintf("db.max_open_conns: %s not a number", conf["db.max_open_conns"]))
	} else if num < 10 || num > 1000 {
		errs = append(errs, "db.max_open_conns: expect 10-1000")
	}
	setConfDefault(conf, "db.max_idle_conns", "20")
	num, err = strconv.Atoi(conf["db.max_idle_conns"])
	if err != nil {
		errs = append(errs, fmt.Sprintf("db.max_idle_conns: %s not a number", conf["db.max_idle_conns"]))
	} else if num < 1 || num > 100 {
		errs = append(errs, "db.max_idle_conns error: expect 1-100")
	}
	//upload
	setConfDefault(conf, "upload.path", os.TempDir()+"/upload")
	setConfDefault(conf, "upload.allow_mime", "")
	size, _ := conf["upload.max_size"]
	if size == "" {
		size = "1M"
	}
	l := len(size)
	unit := strings.ToUpper(size[l-1:])
	val := size[0 : l-1]
	num, err = strconv.Atoi(val)
	if err != nil || num < 1 || num > 1000 || (unit != "M" && unit != "K") {
		errs = append(errs, "upload.max_size: expect 1-1000(K or M)")
	}
	if unit == "M" {
		conf["upload.max_size"] = strconv.Itoa(num * 1024 * 1024)
	} else if unit == "K" {
		conf["upload.max_size"] = strconv.Itoa(num * 1024)
	}
	//处理错误
	if len(errs) > 0 {
		err = errors.New(strings.Join(errs, "; "))
	}
	return
}

//如果conf中不存在key，则把key设置为val
func setConfDefault(conf map[string]string, key string, val string) {
	if _, exists := conf[key]; !exists {
		conf[key] = val
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

//生成mc操作对象
func (this *Request) NewMcDao() *Mc {
	this.Log.Write(LL_SYS, "get Mc dao")
	if this.mcDao == nil {
		this.Log.Write(LL_SYS, "[%s]new mcdao,server=%s", this.appId, this.Conf["db.mc_server"])
		this.mcDao = NewMc(this.Conf["db.mc_server"])
	}
	return this.mcDao
}

//生成mysql操作对象
func (this *Request) NewMySQLDao(table string) (*MySQL, error) {
	this.Log.Write(LL_SYS, "[%s]get MySQL dao", this.appId)
	if this.mysqlDao == nil {
		oc, _ := strconv.Atoi(this.Conf["db.max_open_conns"])
		ic, _ := strconv.Atoi(this.Conf["db.max_idle_conns"])
		this.Log.Write(LL_SYS, "[%s]new mysql,dsn=%s,table=%s,openConn=%d,idleConn=%d", this.appId, this.Conf["db.mysql_dsn"], table, oc, ic)
		mysql, err := NewMySQL(this.Conf["db.mysql_dsn"], table, oc, ic)
		if err != nil {
			return nil, err
		}
		this.mysqlDao = mysql
	} else {
		this.Log.Write(LL_SYS, "[%s]MySQL dao already exists", this.appId)
	}
	return this.mysqlDao, nil
}
