//ini格式配置文件的读取处理

package util

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

var (
	bComment      = []byte{';'}
	bSectionStart = []byte{'['}
	bSectionEnd   = []byte{']'}
	bEqual        = []byte{'='}
	confData      = make(map[string]map[string]string) //配置数据
	cMtime        int64                                //配置的最后修改时间
)

//加载ini格式配置文件,可多个,多个文件之间有相同的key会覆盖
//只读取一次文件，除非文件发生改变
func LoadConf(files ...string) (map[string]string, error) {
	for _, file := range files { //遍历要读取的配置文件
		f, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		stat, _ := f.Stat()
		if stat.ModTime().Unix() <= cMtime { //未修改过的，不需要读
			continue
		}
		//读取文件
		confData[f.Name()] = make(map[string]string)

		buf := bufio.NewReader(f)
		section := ""
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

			keyValue := bytes.SplitN(line, bEqual, 2)
			if len(keyValue) != 2 {
				return nil, errors.New(fmt.Sprintf("Load conf file error: file=%s,line=%d", file, ln))
			}
			key := string(bytes.TrimSpace(keyValue[0]))
			if section != "" {
				key = section + "." + key
			}
			val := bytes.TrimSpace(keyValue[1])
			val = bytes.Trim(val, `"'`) //如果有，去掉引号
			confData[f.Name()][key] = string(val)
		}
	}
	cMtime = time.Now().Unix()
	//获取confData的copy，因为调用者有可能会修改
	data := make(map[string]string)
	for _, d := range confData {
		for k, v := range d {
			data[k] = v
		}
	}
	return data, nil
}
