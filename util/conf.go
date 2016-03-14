//ini格式配置文件的读取处理

//辅助工具包，提共可独立使用的各种常用工具，包括配置读取，日志，定时器，工作池等
package util

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

var (
	bComment      = []byte{';'}
	bSectionStart = []byte{'['}
	bSectionEnd   = []byte{']'}
	bEqual        = []byte{'='}
)

type Conf struct {
	Mtime int64 //配置文件的更新时间
	data  map[string]map[string]string
}

//加载ini格式配置文件,可多个,多个文件之间有相同的key会覆盖，不属于任何section的默认为default
func LoadConf(files ...string) (*Conf, error) {
	var data = make(map[string]map[string]string)
	var mtime int64
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		stat, _ := f.Stat()
		t1 := stat.ModTime().Unix()
		if t1 > mtime {
			mtime = t1
		}
		buf := bufio.NewReader(f)
		section := "default"
		for ln := 1; ; ln++ {
			line, err := buf.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					return nil, err
				} else if len(line) == 0 {
					break
				}
			}
			line = bytes.TrimSpace(line)
			if line == nil || bytes.HasPrefix(line, bComment) {
				continue
			}
			if bytes.HasPrefix(line, bSectionStart) && bytes.HasSuffix(line, bSectionEnd) {
				section = strings.ToLower(string(line[1 : len(line)-1]))
				continue
			}
			if _, ok := data[section]; !ok { //section未初始化，先分配内存
				data[section] = make(map[string]string)
			}
			keyValue := bytes.SplitN(line, bEqual, 2)
			if len(keyValue) != 2 {
				return nil, errors.New(fmt.Sprintf("Load conf file error: file=%s,line=%d", file, ln))
			}
			key := string(bytes.TrimSpace(keyValue[0]))
			val := bytes.TrimSpace(keyValue[1])
			val = bytes.Trim(val, `"'`) //如果有，去掉引号
			data[section][key] = string(val)
		}
	}
	return &Conf{mtime, data}, nil
}

//获取指定的配置项的值
//
//第一参数key= “section,key”,缺少section时可只传"key",默认读取section=default
//
//第二参数为没有设置时的缺省值
//
//统一以字符串方式返回，未设置时使用指定的缺省值，如未指定缺省值，第二返回值为false
func (this *Conf) Get(str ...string) (string, bool) {
	defaultSet := false
	var key, defaultVal string
	switch len(str) {
	case 2:
		defaultSet = true
		defaultVal = str[1]
		fallthrough
	case 1:
		key = str[0]
	default:
		//TODO:panic conf.Get调用错误?
		return "", false
	}

	keys := strings.SplitN(key, ".", 2)
	section := "default"
	if len(keys) == 1 {
		key = keys[0]
	} else {
		section = keys[0]
		key = keys[1]
	}
	val, exists := this.data[section][key]
	if !exists && defaultSet {
		val = defaultVal
		exists = true
	}
	return val, exists
}
