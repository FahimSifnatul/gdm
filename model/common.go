package model

type SubCmd struct {
	FileUrl          string `flg:"u"`
	ConcurrencyCount int    `flg:"c"`
	Location         string `flg:"l"`
	FileName         string `flg:"n"`
}
