package controller

import (
	"github.com/tim1020/ecgo"
)

type Controller struct {
	*ecgo.ReqSess
}

func (this *Controller) PreControl() {

}
// Get /Hello
func(this *Controller) Hello(){
	this.Resp("hello,%s","Tim")
}