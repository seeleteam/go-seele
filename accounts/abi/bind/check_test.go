package bind

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/stretchr/testify/assert"
)

func Test_parseArg(t *testing.T) {
	value, err := parseArg("int8", "50")
	assert.NoError(t, err)
	assert.Equal(t, reflect.Int8, reflect.TypeOf(value).Kind())
	value, err = parseArg("int16", "50")
	assert.NoError(t, err)
	assert.Equal(t, reflect.Int16, reflect.TypeOf(value).Kind())
	value, err = parseArg("int32", "50")
	assert.NoError(t, err)
	assert.Equal(t, reflect.Int32, reflect.TypeOf(value).Kind())
	value, err = parseArg("int64", "50")
	assert.NoError(t, err)
	assert.Equal(t, reflect.Int64, reflect.TypeOf(value).Kind())
	value, err = parseArg("uint8", "50")
	assert.NoError(t, err)
	assert.Equal(t, reflect.Uint8, reflect.TypeOf(value).Kind())
	value, err = parseArg("uint16", "50")
	assert.NoError(t, err)
	assert.Equal(t, reflect.Uint16, reflect.TypeOf(value).Kind())
	value, err = parseArg("uint32", "50")
	assert.NoError(t, err)
	assert.Equal(t, reflect.Uint32, reflect.TypeOf(value).Kind())
	value, err = parseArg("uint64", "50")
	assert.NoError(t, err)
	assert.Equal(t, reflect.Uint64, reflect.TypeOf(value).Kind())
	value, err = parseArg("*big.Int", "50")
	assert.NoError(t, err)
	assert.Equal(t, reflect.Ptr, reflect.TypeOf(value).Kind())
	value, err = parseArg("string", "50")
	assert.NoError(t, err)
	assert.Equal(t, reflect.String, reflect.TypeOf(value).Kind())
	value, err = parseArg("bool", "true")
	assert.NoError(t, err)
	assert.Equal(t, reflect.Bool, reflect.TypeOf(value).Kind())
	value, err = parseArg("common.Address", "0x6d4fca4dc6c49ce8df30e7b2887a08cd4d5a1451")
	assert.NoError(t, err)
	addr, err := common.HexToAddress("0x6d4fca4dc6c49ce8df30e7b2887a08cd4d5a1451")
	assert.NoError(t, err)
	assert.Equal(t, addr, value.(common.Address))
	value, err = parseArg("[]byte", "0x0b573b0ab6f7db50")
	assert.NoError(t, err)
	assert.Equal(t, reflect.Slice, reflect.TypeOf(value).Kind())
	value, err = parseArg("[32]byte", "0x000001c42c79f769a00bcdde948d09279f8385b3ca5e8593e3abc13e15635b38")
	assert.NoError(t, err)
	assert.Equal(t, reflect.Array, reflect.TypeOf(value).Kind())

	// Over Flow error int/uint
	number := big.NewInt(0).Add(big.NewInt(0).SetUint64(math.MaxUint64), big.NewInt(1)).String()
	errmsg := fmt.Sprintf("strconv.ParseInt: parsing \"%s\": value out of range", number)
	_, err = parseArg("int8", number)
	assert.Error(t, err, errmsg)
	_, err = parseArg("int16", number)
	assert.Error(t, err, errmsg)
	_, err = parseArg("int32", number)
	assert.Error(t, err, errmsg)
	_, err = parseArg("int64", number)
	assert.Error(t, err, errmsg)
	_, err = parseArg("uint8", number)
	assert.Error(t, err, errmsg)
	_, err = parseArg("uint16", number)
	assert.Error(t, err, errmsg)
	_, err = parseArg("uint32", number)
	assert.Error(t, err, errmsg)
	_, err = parseArg("uint64", number)
	assert.Error(t, err, errmsg)

	// Invalid address
	_, err = parseArg("common.Address", "0x123333333333333333333333333331234")
	assert.Error(t, err)

	// Overflow bytes
	_, err = parseArg("[32]byte", "0x000001c42c79f769a00bcdde948d09279f8385b3ca5e8593e3abc13e15635b38000001c42c79f769a00bcdde948d09279f8385b3ca5e8593e3abc13e15635b38")
	assert.Error(t, err)
}
