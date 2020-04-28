// +build tools

package tools

import (
	_ "github.com/golang/mock/mockgen"
	_ "github.com/mitchellh/gox"
	_ "github.com/tinylib/msgp"
	_ "golang.org/x/tools/cmd/cover"
	_ "golang.org/x/tools/cmd/stringer"
)
