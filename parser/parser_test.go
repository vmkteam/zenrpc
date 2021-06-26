package parser

import (
	"testing"
)

func TestPackageInfo_fillReceivers(t *testing.T) {
	type fields struct {
		Services []*Service
	}
	tests := []struct {
		name     string
		fields   fields
		expected string
	}{
		{name: "no methods", fields: struct{ Services []*Service }{Services: []*Service{{}}}, expected: "s"},
		{name: "only forbidden", fields: struct{ Services []*Service }{Services: []*Service{{
			Methods: []*Method{
				{ReceiverName: "ctx"},
				{ReceiverName: "args"},
				{ReceiverName: "resp"},
			},
		}}}, expected: "s"},
		{name: "equal amount", fields: struct{ Services []*Service }{Services: []*Service{{
			Methods: []*Method{
				{ReceiverName: "a"},
				{ReceiverName: "a"},
				{ReceiverName: "b"},
				{ReceiverName: "b"},
			},
		}}}, expected: "a"},
		{name: "non-equal amount", fields: struct{ Services []*Service }{Services: []*Service{{
			Methods: []*Method{
				{ReceiverName: "a"},
				{ReceiverName: "b"},
				{ReceiverName: "b"},
				{ReceiverName: "a"},
				{ReceiverName: "b"},
			},
		}}}, expected: "b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pi := &PackageInfo{
				Services: tt.fields.Services,
			}

			pi.fillReceivers()

			if pi.Services[0].Receiver != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, pi.Services[0].Receiver)
			}

		})
	}
}
