/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package rpc

import (
	"testing"
)

type Service struct{}

type ArgsServer struct {
	S string
}

type Result struct {
	Args *ArgsServer
}

func (s *Service) Func1(args *ArgsServer, result *Result) error {
	*result = Result{args}
	return nil
}

func (s *Service) InvalidFunc2() (string, string) {
	return "", ""
}

func Test_Server_RegisterName(t *testing.T) {
	server := NewServer()
	service := new(Service)

	if err := server.RegisterName("test", service); err != nil {
		t.Fatalf("%v", err)
	}
}
