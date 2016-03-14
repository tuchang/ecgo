package models

import (
	. "github.com/tim1020/ecgo/dao"
)

type Service struct{
	*MySQL
}

func NewService(*MySQL) *Service{
	return &Service{mysql}
}

func (this *Service) XX(){

}