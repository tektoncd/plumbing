package github

import (
	"fmt"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"google.golang.org/grpc/codes"
)

type StatusError struct {
	v1alpha1.Status
}

func (s StatusError) Error() string {
	return fmt.Sprintf("%s [code: %v]", s.Message, s.Code)

}

func Error(c codes.Code, s string) error {
	return StatusError{Status: v1alpha1.Status{
		Code:    c,
		Message: s,
	}}
}

func Errorf(c codes.Code, s string, args ...interface{}) error {
	return StatusError{Status: v1alpha1.Status{
		Code:    c,
		Message: fmt.Sprintf(s, args...),
	}}
}
