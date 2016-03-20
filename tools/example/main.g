package main

import (
	"{app_name}/controller"
	"github.com/tim1020/ecgo"
	"log"
)

func main() {
	log.Fatal(ecgo.Server(&controller.Controller{},nil)) //第二参数可指定sessionHandler
}