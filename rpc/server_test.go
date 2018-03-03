/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package rpc

import (
	"testing"
)

type Service struct{}

type Args struct {
	S string
}

type Result struct {
	Args *Args
}

func (s *Service) Func1(args *Args, result *Result) error {
	*result = Result{args}
	return nil
}

func (s *Service) InvalidFunc2() (string, string) {
	return "", ""
}

func Test_ServerRegisterName(t *testing.T) {
	server := NewServer()
	service := new(Service)

	if err := server.RegisterName("test", service); err != nil {
		t.Fatalf("%v", err)
	}
}
