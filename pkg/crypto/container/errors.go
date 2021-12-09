package container

import (
	"fmt"
)

type ErrIncompatible struct {
	Subject string
	Meta    []interface{}
}

func (e ErrIncompatible) Error() string {
	return fmt.Sprintf(
		"container %s is incompatible, meta: %v",
		e.Subject, e.Meta,
	)
}

//

type ErrInvalid struct {
	Subject string
	Meta    []interface{}
}

func (e ErrInvalid) Error() string {
	return fmt.Sprintf(
		"container signature is %s, meta: %v",
		e.Subject, e.Meta,
	)
}
