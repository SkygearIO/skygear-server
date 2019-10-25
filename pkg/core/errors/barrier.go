package errors

type errorBarrier struct {
	inner error
	msg   string
}

func Handled(err error) error {
	return &errorBarrier{inner: err, msg: err.Error()}
}

func HandledWithMessage(err error, msg string) error {
	return &errorBarrier{inner: err, msg: msg}
}

func (e *errorBarrier) Error() string { return e.msg }