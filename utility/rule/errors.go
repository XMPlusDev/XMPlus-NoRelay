package rule

import "github.com/xmplusdev/xray-core/common/errors"

func newError(values ...interface{}) *errors.Error {
	return errors.New(values...)
}
