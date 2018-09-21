package bind

import (
	"fmt"
	"reflect"
	"testing"
)

func Test_bindUnnestedTypeGo(t *testing.T) {
	ty := reflect.Uint.String()
	i, s := bindUnnestedTypeGo(ty)
	fmt.Println("ty:", ty)
	fmt.Println("i:", i)
	fmt.Println("s:", s)

	ty = reflect.Ptr.String()
	i, s = bindUnnestedTypeGo(ty)
	fmt.Println("ty:", ty)
	fmt.Println("i:", i)
	fmt.Println("s:", s)

	ty = reflect.Bool.String()
	i, s = bindUnnestedTypeGo(ty)
	fmt.Println("ty:", ty)
	fmt.Println("i:", i)
	fmt.Println("s:", s)

	ty = reflect.Array.String()
	i, s = bindUnnestedTypeGo(ty)
	fmt.Println("ty:", ty)
	fmt.Println("i:", i)
	fmt.Println("s:", s)

	ty = reflect.String.String()
	i, s = bindUnnestedTypeGo(ty)
	fmt.Println("ty:", ty)
	fmt.Println("i:", i)
	fmt.Println("s:", s)

	ty = reflect.Slice.String()
	i, s = bindUnnestedTypeGo(ty)
	fmt.Println("ty:", ty)
	fmt.Println("i:", i)
	fmt.Println("s:", s)

}
