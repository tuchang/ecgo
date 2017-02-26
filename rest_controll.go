package ecgo

import (
	"encoding/json"
	"fmt"
)

//默认restful处理器
func (this *Request) restControl() {
	var wh map[string]interface{}
	var daoErr error
	var resp = make(map[string]interface{})
	resp["code"] = 0

	//put and delete必须有Pk
	if (this.Method == "PUT" || this.Method == "DELETE") && this.Pk == 0 {
		this.ShowErr(400, fmt.Sprintf(":ResuorceId was required while %s", this.Method))
		return
	}
	if this.Method == "PUT" || this.Method == "DELETE" || this.Method == "GET" {
		wh = make(map[string]interface{})
		if this.Pk != 0 {
			wh["id"] = this.Pk
		}
	}
	ResourceDao, err := this.NewMySQLDao(this.Resource)
	if err != nil {
		this.ShowErr(500, fmt.Sprintf("%v", err))
		return
	}

	switch this.Method {
	case "GET":
		if this.Pk == 0 { //get All
			if this.Get["limit"] != "" {
				ResourceDao.SetLimit(this.Get["limit"])
			}
			if this.Get["order"] != "" {
				ResourceDao.SetOrder(this.Get["order"])
			}
			if this.Get["field"] != "" {
				ResourceDao.SetField(this.Get["field"])
			}
			if this.Get["where"] != "" { //add others where

			}
			resp["data"], daoErr = ResourceDao.Get(wh)
		} else {
			resp["data"], daoErr = ResourceDao.GetRow(wh)
		}
	case "POST":
		data := make(map[string]interface{})
		for k, v := range this.Post {
			data[k] = v
		}
		resp["insert_id"], daoErr = ResourceDao.Insert(data)
	case "PUT":
		data := make(map[string]interface{})
		for k, v := range this.Post {
			data[k] = v
		}
		if this.Get["where"] != "" { //add others where

		}
		fmt.Printf("%s", this.Post)
		resp["rows"], daoErr = ResourceDao.Update(data, wh)
	case "DELETE":
		resp["rows"], daoErr = ResourceDao.Delete(wh)
	default:
		this.ShowErr(400, fmt.Sprintf("method %s not support", this.Method))
		return
	}
	if daoErr != nil {
		this.ShowErr(500, fmt.Sprintf("%s", daoErr.Error()))
		return
	}
	str, err := json.Marshal(resp)
	if err != nil {
		this.ShowErr(500, fmt.Sprintf("%v", err))
		return
	}
	this.Resp(string(str))
}
