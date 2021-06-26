package model

import "github.com/vmkteam/zenrpc/v2/testdata/objects"

type Point struct {
	objects.AbstractObject     // embedded object
	X, Y                   int // coordinate
	Z                      int `json:"-"`
	ConnectedObject        objects.AbstractObject
}
