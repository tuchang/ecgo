//数据校验器

package util

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	ERR_OK       = iota
	ERR_MISS     //缺少必填字段
	ERR_UNEXCEPT //非期望值
	ERR_BADRULE  //规则错误
)

type ValidErr struct {
	Code int
	Msg  string
}

func (ve *ValidErr) Error() string {
	return fmt.Sprintf("[%d]%s", ve.Code, ve.Msg)
}

type Validator struct {
	data map[string]string //要校验的数据字典
	rule map[string]*vRule
}
type vRule struct {
	vr       ValidateRuler
	required bool
}

//校验规则接口
type ValidateRuler interface {
	Check(data string) *ValidErr
}

//创建校验器对象
func NewValidator(data map[string]string) *Validator {
	v := &Validator{data: data}
	v.rule = make(map[string]*vRule)
	return v
}

//添加校验规则(同一个key只能有一条规则，重复添加会覆盖)
func (this *Validator) AddRule(key string, rule string, params string, required ...bool) {
	nr := &normalRule{key, rule, params}
	this.rule[key] = &vRule{nr, true}
	if len(required) > 0 {
		this.rule[key].required = required[0]
	}
}

//添加自定义规则
func (this *Validator) AddExtRule(key string, rule ValidateRuler, required ...bool) {
	this.rule[key] = &vRule{rule, true}
	if len(required) > 0 {
		this.rule[key].required = required[0]
	}
}

//执行检查
func (this *Validator) Check() (errs map[string]*ValidErr) {
	errs = make(map[string]*ValidErr)
	for k, v := range this.rule {
		data, exists := this.data[k]
		if !exists { //无值
			if v.required { //如果必填，报错
				errs[k] = &ValidErr{ERR_MISS, "required field miss"}
			}
		} else { //有值判断规则
			if err := v.vr.Check(data); err != nil {
				errs[k] = err
			}
		}
	}
	if len(errs) > 0 {
		return errs
	} else {
		return nil
	}
}

//内置规则
type normalRule struct {
	key    string
	rule   string
	params string
}

//内置规则的检查实现
func (this *normalRule) Check(data string) (vErr *ValidErr) {
	if this.params == "" {
		vErr = &ValidErr{ERR_BADRULE, "rule params misss"}
		return
	}
	p := strings.Split(this.params, ",")
	switch this.rule {
	case "string":
		min, max, err := this.parseParams4Size(p)
		if err != nil {
			vErr = err
		} else {
			if l := len(data); l < min || l > max {
				vErr = &ValidErr{ERR_UNEXCEPT, "length range out of except"}
			}
		}
	case "number":
		num, err := strconv.Atoi(data)
		if err != nil {
			vErr = &ValidErr{ERR_UNEXCEPT, "number except"}
		} else {
			min, max, err := this.parseParams4Size(p)
			if err != nil {
				vErr = err
			} else {
				if num < min || num > max {
					vErr = &ValidErr{ERR_UNEXCEPT, "value range out of except"}
				}
			}
		}
	case "list":
		match := false
		for _, v := range p {
			if v == data {
				match = true
			}
		}
		if !match {
			vErr = &ValidErr{ERR_UNEXCEPT, "value not in except list"}
		}
	case "regular":
		reg := regexp.MustCompile(this.params)
		if !reg.Match([]byte(data)) {
			vErr = &ValidErr{ERR_UNEXCEPT, "regexp not match"}
		}
	case "datetime":
		if _, err := time.Parse(this.params, data); err != nil {
			vErr = &ValidErr{ERR_UNEXCEPT, "wrong daatime format"}
		}
	default:
		vErr = &ValidErr{ERR_BADRULE, fmt.Sprintf("rule %s unsupport", this.rule)}
	}
	return
}

func (this *normalRule) parseParams4Size(p []string) (min int, max int, vErr *ValidErr) {
	var err error
	if len(p) == 2 { //格式[min,max]  包括边界
		if min, err = strconv.Atoi(p[0]); err != nil {
			vErr = &ValidErr{ERR_BADRULE, fmt.Sprintf("%s params wrong:  min not a number", this.rule)}
		} else if max, err = strconv.Atoi(p[1]); err != nil {
			vErr = &ValidErr{ERR_BADRULE, fmt.Sprintf("%s params wrong : max not a number", this.rule)}
		} else if min > max {
			vErr = &ValidErr{ERR_BADRULE, fmt.Sprintf("% sparams wrong: min > max", this.rule)}
		}
	} else {
		vErr = &ValidErr{ERR_BADRULE, fmt.Sprintf("%s params wrong : except \"min,max\"", this.rule)}
	}
	return
}
