package workspace

import (
	"fmt"
	"reflect"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

type Empty struct{}

func TestXxx(t *testing.T) {
	pc, _, _, ok := runtime.Caller(0)
	require.True(t, ok)
	f := runtime.FuncForPC(pc)
	file, _ := f.FileLine(pc)
	fmt.Println(file)

	fmt.Println(reflect.TypeOf(Empty{}).PkgPath())
}
