package testdata

import (
	"github.com/vmkteam/zenrpc/v2"
)

//go:generate zenrpc

type Groups []Group

type Group struct {
	Id       int      `json:"id"`
	Title    string   `json:"title"`
	Nodes    []Group  `json:"nodes"`
	Groups   []Group  `json:"groups"`
	ChildOpt *Group   `json:"child"`
	Sub      SubGroup `json:"sub"`
}

type SubGroup struct {
	Id    int     `json:"id"`
	Title string  `json:"title"`
	Nodes []Group `json:"nodes"`
}

type Campaign struct {
	Id     int    `json:"id"`
	Groups Groups `json:"groups"`
}

type CatalogueService struct{ zenrpc.Service }

func (s CatalogueService) First(groups Groups) (bool, error) {
	return true, nil
}

func (s CatalogueService) Second(campaigns []Campaign) (bool, error) {
	return true, nil
}

func (s CatalogueService) Third() (Campaign, error) {
	return Campaign{}, nil
}
