//数据校验器

package util

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

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
	Check(data string) error
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
func (this *Validator) Check() (errs map[string]error) {
	errs = make(map[string]error)
	for k, v := range this.rule {
		data, exists := this.data[k]
		if !exists { //无值
			if v.required { //如果必填，报错
				errs[k] = errors.New("data error: required field miss")
			}
		} else { //有值判断规则
			if err := v.vr.Check(data); err != nil {
				errs[k] = err
			}
		}
	}
	return errs
}

//内置规则
type normalRule struct {
	key    string
	rule   string
	params string
}

//内置规则的检查实现
func (this *normalRule) Check(data string) (Err error) {
	if this.params == "" {
		Err = errors.New("rule error: params wrong of rule")
		return
	}
	p := strings.Split(this.params, ",")
	switch this.rule {
	case "string":
		min, max, err := this.parseParams4Size(p)
		if err != nil {
			Err = err
		} else {
			if l := len(data); l < min || l > max {
				Err = errors.New(fmt.Sprintf("data error: except length in range(%d,%d)", min, max))
			}
		}
	case "number":
		num, err := strconv.Atoi(data)
		if err != nil {
			Err = errors.New("data error: except a number")
		} else {
			min, max, err := this.parseParams4Size(p)
			if err != nil {
				Err = err
			} else {
				if num < min || num > max {
					Err = errors.New(fmt.Sprintf("data error: except in range(%d,%d)", min, max))
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
			Err = errors.New(fmt.Sprintf("data error: except value in list %s", p))
		}
	case "regular":
		reg := regexp.MustCompile(this.params)
		if !reg.Match([]byte(data)) {
			Err = errors.New("data error: not match regexp")
		}
	case "datetime":
		if _, err := time.Parse(this.params, data); err != nil {
			Err = errors.New("data error: not match datetime -" + err.Error())
		}
	default:
		Err = errors.New(fmt.Sprintf("rule error: not support of rule=%s", this.rule))
	}
	return
}

func (this *normalRule) parseParams4Size(p []string) (min int, max int, err error) {
	if len(p) == 2 { //格式[min,max]  包括边界
		if min, err = strconv.Atoi(p[0]); err != nil {
			err = errors.New(fmt.Sprintf("rule error: params wrong of rule %s, min not a number", this.rule))
		} else if max, err = strconv.Atoi(p[1]); err != nil {
			err = errors.New(fmt.Sprintf("rule error: params wrong of rule %s, max not a number", this.rule))
		} else if min > max {
			err = errors.New(fmt.Sprintf("rule error: params wrong of rule %s, min > max", this.rule))
		}
	} else {
		err = errors.New(fmt.Sprintf("rule error: params wrong of rule %s,except \"min,max\"", this.rule))
	}
	return
}
