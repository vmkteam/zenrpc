package subarithservice

import (
	"time"

	"github.com/vmkteam/zenrpc/v2/testdata/objects"
)

type Point struct {
	objects.AbstractObject
	A, B int        // coordinate
	C    int        `json:"-"`
	When *time.Time `json:"when"` // when it happened
}
