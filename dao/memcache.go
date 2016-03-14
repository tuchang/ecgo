//memcache操作的二次封装

package dao

import (
	"errors"
	"github.com/bradfitz/gomemcache/memcache"
	"strings"
)

//mc操作对象,对github.com/bradfitz/gomemcache/memcache的二次封装
type Mc struct {
	Err    error
	Client *memcache.Client
}

func NewMc(server string) *Mc {
	servList := strings.Split(server, ",")
	return &Mc{Client: memcache.New(servList...)}
}

//获取单条记录
func (this *Mc) Get(key string) ([]byte, error) {
	items, err := this.Client.Get(key)
	if err != nil {
		this.Err = err
		return nil, err
	} else {
		return items.Value, nil
	}
}

//获取多条记录
func (this *Mc) GetMulti(keys ...string) (map[string][]byte, error) {
	items, err := this.Client.GetMulti(keys)
	if err != nil {
		this.Err = err
		return nil, err
	} else {
		result := make(map[string][]byte)
		for k, v := range items {
			result[k] = v.Value
		}
		return result, nil
	}
}

//添加，如果已存在，报错
func (this *Mc) Add(key string, data []byte, s ...int32) error {
	expire, err := this._getExpire(s...)
	if err == nil {
		item := &memcache.Item{Key: key, Value: data, Expiration: expire}
		err = this.Client.Add(item)
	}
	this.Err = err
	return err
}

//设置，不管有没有，都直接覆盖
func (this *Mc) Set(key string, data []byte, s ...int32) error {
	expire, err := this._getExpire(s...)
	if err == nil {
		item := &memcache.Item{Key: key, Value: data, Expiration: expire}
		err = this.Client.Set(item)
	}
	this.Err = err
	return err
}

//替换，如果未存在报错
func (this *Mc) Replace(key string, data []byte, s ...int32) error {
	expire, err := this._getExpire(s...)
	if err == nil {
		item := &memcache.Item{Key: key, Value: data, Expiration: expire}
		err = this.Client.Replace(item)
	}
	this.Err = err
	return err
}

//检查expire
func (this *Mc) _getExpire(s ...int32) (int32, error) {
	switch len(s) {
	case 0:
		return 0, nil
	case 1:
		if s[0] >= 0 {
			return s[0], nil
		}
	}
	return -1, errors.New("memcache: expire error")
}
