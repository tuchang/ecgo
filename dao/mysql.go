//mysql操作的二次封装

//提供数据源的操作封装
package dao

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"strings"
)

const (
	NULL_VAL = "\\N"
)

//mysql操作对象
type MySQL struct {
	DB           *sql.DB
	Table        string //表名
	maxOpenConns int
	maxIdleConns int
	order        string //排序，“id desc”
	limit        string //分页，“offset,nums”
	field        string //查询的字段
	err          error
	tx           *sql.Tx
}

//生成mysql操作对象
func NewMySQL(dsn string, table string, openConn int, idleConn int) (mysql *MySQL, Err error) {
	db, err := sql.Open("mysql", dsn)
	if err == nil {
		db.Ping()
		mysql = &MySQL{DB: db, Table: table, maxOpenConns: openConn, maxIdleConns: idleConn, field: "*"}
	}

	return
}

//指定表名
func (this *MySQL) SetTable(table string) *MySQL {
	this.Table = table
	return this
}

//设置查询的字段
func (this *MySQL) SetField(field string) *MySQL {
	//check field
	this.field = field
	return this
}

//设置排序
func (this *MySQL) SetOrder(order string) *MySQL {
	this.order = order
	return this
}

//设置limit,SetLimit("nums") or SetLimit("offset,nums")
func (this *MySQL) SetLimit(limit string) *MySQL {
	//todo: check limit
	this.limit = limit
	return this
}

//查询单条结果集
func (this *MySQL) GetRow(wh ...map[string]interface{}) (map[string]string, error) {
	sqlStr, vals := this._buildSelect(true, wh...)
	result, err := this.Query(sqlStr, vals...)
	if err == nil && len(result) > 0 {
		return result[0], nil
	} else {
		return nil, err
	}
}

//查询记录集(多条)
func (this *MySQL) Get(wh ...map[string]interface{}) ([]map[string]string, error) {
	sqlStr, vals := this._buildSelect(false, wh...)
	return this.Query(sqlStr, vals...)
}

//插入一条数据,成功时返回自增ID(若无自增字段返回0),todo:支持批量
func (this *MySQL) Insert(data map[string]interface{}) (int64, error) {
	var fields []string
	var vals []interface{}
	for k, v := range data {
		fields = append(fields, k+"= ?")
		vals = append(vals, v)
	}
	sqlStr := fmt.Sprintf(" insert into `%s` set %s", this.Table, strings.Join(fields, ","))
	return this.Exec(sqlStr, vals...)
}

//删除指定条件的记录
func (this *MySQL) Delete(wh ...map[string]interface{}) (int64, error) {
	where, vals := this._parseWhere(wh...)
	if where == "" {
		return 0, errors.New("Can not Delete all,set where pls")
	}
	sqlStr := fmt.Sprintf("delete from `%s` where %s", this.Table, where)
	return this.Exec(sqlStr, vals...)
}

//更新指定条件的记录
func (this *MySQL) Update(data map[string]interface{}, wh ...map[string]interface{}) (int64, error) {
	var fields []string
	var vals []interface{}
	for k, v := range data {
		fields = append(fields, k+"= ?")
		vals = append(vals, v)
	}
	where, vals1 := this._parseWhere(wh...)
	for _, v := range vals1 {
		vals = append(vals, v)
	}
	if where == "" {
		return 0, errors.New("Can not Update all,set where pls")
	}
	sqlStr := fmt.Sprintf("update `%s` set %s where %s", this.Table, strings.Join(fields, ","), where)
	return this.Exec(sqlStr, vals...)
}

//执行update/delete/insert/replace语句
func (this *MySQL) Exec(sqlStr string, vals ...interface{}) (id int64, err error) {
	sqlStr = strings.Trim(sqlStr, " ")
	var stmt *sql.Stmt
	if this.tx != nil {
		stmt, err = this.tx.Prepare(sqlStr)
	} else {
		stmt, err = this.DB.Prepare(sqlStr)
	}
	if err == nil {
		defer stmt.Close()
		res, err1 := stmt.Exec(vals...)
		if err1 == nil {
			if strings.HasPrefix(strings.ToLower(sqlStr), "insert") {
				id, err = res.LastInsertId() //根椐前缀
			} else {
				id, err = res.RowsAffected()
			}
		} else {
			err = err1
		}
	}
	this.err = err
	return
}

//执行查询语句select,返回结果集
func (this *MySQL) Query(sqlStr string, vals ...interface{}) (result []map[string]string, err error) {
	var rows *sql.Rows
	if this.tx != nil {
		rows, err = this.tx.Query(sqlStr, vals...)
	} else {
		rows, err = this.DB.Query(sqlStr, vals...)
	}

	if err == nil { //处理结果
		defer rows.Close()
		cols, _ := rows.Columns()
		l := len(cols)
		rawResult := make([][]byte, l)

		dest := make([]interface{}, l) // A temporary interface{} slice
		for i, _ := range rawResult {
			dest[i] = &rawResult[i] // Put pointers to each string in the interface slice
		}
		for rows.Next() {
			rowResult := make(map[string]string)
			err = rows.Scan(dest...)
			if err == nil {
				for i, raw := range rawResult {
					key := cols[i]
					if raw == nil {
						rowResult[key] = NULL_VAL
					} else {
						rowResult[key] = string(raw)
					}
				}
				result = append(result, rowResult)
			}
		}
	}
	this.err = err
	return
}

//开启事务
func (this *MySQL) TransStart() error {
	tx, err := this.DB.Begin()
	if err != nil {
		return err
	}
	this.err = nil
	this.tx = tx
	return nil
}

//提交事务，如果事务中有错误发生，则自动回滚，并返回错误
func (this *MySQL) TransCommit() (err error) {
	if this.err != nil {
		err = this.err
		this.tx.Rollback()
	} else {
		err = this.tx.Commit()
	}
	this.tx = nil
	return
}

//手工回滚事务
func (this *MySQL) TransRollback() (err error) {
	err = this.tx.Rollback()
	this.tx = nil
	return
}

//返回最后发生的错误
func (this *MySQL) LastError() error {
	return this.err
}

//释放连接
func (this *MySQL) Close() {
	if this.DB != nil {
		this.DB.Close()
	}
}

//组装select语句
func (this *MySQL) _buildSelect(one bool, wh ...map[string]interface{}) (string, []interface{}) {
	sqlStr := fmt.Sprintf("select %s from `%s`", this.field, this.Table)
	where, vals := this._parseWhere(wh...)
	if where != "" {
		sqlStr = sqlStr + " where " + where
	}
	if this.order != "" {
		sqlStr += " order by " + this.order
	}
	if one {
		sqlStr += " limit 1"
	} else if this.limit != "" {
		sqlStr += " limit " + this.limit
	}
	//重置order,limit选项
	this.order = ""
	this.limit = ""
	return sqlStr, vals
}

//处理where参数,返回拼接后的where字串以及对应的占位值
func (this *MySQL) _parseWhere(wh ...map[string]interface{}) (string, []interface{}) {
	var cond []string
	var vals []interface{}
	for _, w := range wh {
		var c1 []string
		for k, v := range w {
			k = strings.TrimSpace(k)
			if strings.HasSuffix(strings.ToLower(k), "in") {
				val, ok := v.(string)
				if !ok {
					panic("where in must be string separate with \",\"")
				}
				inVals := strings.Split(val, ",")
				c1 = append(c1, k+" (?"+strings.Repeat(",?", len(inVals)-1)+")")
				for _, val := range inVals {
					vals = append(vals, val)
				}
			} else {
				r := []rune(k)
				last := string(r[len(r)-1:])
				if last == "<" || last == ">" || last == "=" {
					c1 = append(c1, k+" ?")
				} else {
					c1 = append(c1, k+" = ?")
				}
				vals = append(vals, v)
			}
		}
		cStr := strings.Join(c1, " and ")
		if cStr != "" {
			cond = append(cond, "("+cStr+")")
		}
	}
	return strings.Join(cond, " or "), vals
}
