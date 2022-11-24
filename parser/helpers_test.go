package parser

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestLoadPackage(t *testing.T) {
	Convey("Should load package with syntax and imports", t, func() {
		_, err := loadPackage("../testdata/subservice/subarithservice.go")
		So(err, ShouldBeNil)
	})
}
