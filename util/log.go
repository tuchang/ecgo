//日志相关操作

package util

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

const (
	LL_DEBUG  = "debug"
	LL_WARN   = "warn"
	LL_ERROR  = "error"
	LL_SYS    = "sys"
	LL_ACCESS = "access"
	LL_ALL    = "all"
)

type Log struct {
	lType string //日志类别
	path  string //日志目录，当target为file时有效
}

//生成Log对象
func NewLogger(lType, path string) *Log {
	//判断日志目录是否存在，不存在时先创建
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		os.MkdirAll(path, os.ModePerm)
	}
	return &Log{lType: lType, path: path}
}

//记录debug日志
func (this *Log) D(format string, vals ...interface{}) {
	this.Write(LL_DEBUG, format, vals...)
}

//记录error日志
func (this *Log) E(format string, vals ...interface{}) {
	this.Write(LL_ERROR, format, vals...)
}

//记录warnning日志
func (this *Log) W(format string, vals ...interface{}) {
	this.Write(LL_WARN, format, vals...)
}

//记录指定日志,ll为日志名称
func (this *Log) Write(lType string, format string, vals ...interface{}) {
	if !this.isNeed(lType) {
		return
	}
	t := time.Now()
	filepath := fmt.Sprintf("%s/%s_%s.log", this.path, lType, string(t.Format("20060102")))
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return
	}
	defer file.Close()
	wFile := bufio.NewWriter(file)
	wFile.WriteString(string(t.Format("2006/01/02 15:04:05 ")))
	wFile.WriteString(fmt.Sprintf(format, vals...))
	wFile.WriteString("\n")
	err = wFile.Flush()
	if err != nil {
		log.Fatalln("[err]:", err)
	}
}

//判断是否需要记录指定级别日志
func (this *Log) isNeed(lType string) bool {
	if lType != "LL_ACCESS" && !strings.Contains(this.lType, lType) && !strings.Contains(this.lType, LL_ALL) { //access在调用前控制
		return false
	}
	return true
}
