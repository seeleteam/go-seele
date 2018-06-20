/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/gorilla/websocket"
	"github.com/seeleteam/go-seele/rpc"
	"github.com/spf13/cobra"
)

func init() {
	config := NewConfig()
	for _, request := range config.request {
		cmd, err := config.InitCommand(request)
		if err != nil {
			fmt.Println("init cmd fail :\n", err)
			return
		}
		rootCmd.AddCommand(cmd)
	}
}

// InitCommand init cobra command by Request
func (c *Config) InitCommand(request *Request) (*cobra.Command, error) {
	ParamFlags := make(map[string]interface{})
	_, isBasic := c.basicMap[request.ParamReflectType]
	_, isStruct := c.structMap[request.ParamReflectType]
	if !isBasic && !isStruct {
		return nil, errors.New("request type match miss")
	}

	// cmd represents the command
	var cmd = &cobra.Command{
		Use:   request.Use,
		Short: request.Short,
		Long:  request.Long,
		Run: func(cmd *cobra.Command, args []string) {
			var input interface{}
			if isBasic && request.ParamReflectType != "nil" {
				input = ParamFlags[request.Params[0].ReflectName]
			} else if isStruct {
				t := reflect.ValueOf(c.structMap[request.ParamReflectType]).Type()
				e := reflect.New(t).Elem()
				for k, v := range ParamFlags {
					s := reflect.ValueOf(ParsePointInterface(v))
					e.FieldByName(k).Set(s)
				}
				input = e.Interface()
			}

			var output interface{}
			var client *rpc.Client
			var err error

			if request.UseWebsocket {
				ws, _, err := websocket.DefaultDialer.Dial(wsAddr, nil)
				if err != nil {
					fmt.Println(err)
					return
				}
				defer ws.Close()

				client = rpc.NewClient(ws.UnderlyingConn())
				defer client.Close()
			} else {
				client, err = rpc.Dial("tcp", rpcAddr)
				if err != nil {
					fmt.Println(err)
					return
				}
				defer client.Close()
			}

			err = client.Call(request.Method, &input, &output)
			if err != nil {
				fmt.Println(err)
				return
			}

			if output != nil {
				jsonOutput, err := json.MarshalIndent(output, "", "\t")
				if err != nil {
					fmt.Println(err)
					return
				}

				fmt.Println("output :\n", string(jsonOutput))
			}

			if request.Handler != nil {
				request.Handler(output)
			}
		},
	}

	for _, param := range request.Params {
		ParseCmdFlag(param, ParamFlags, cmd)
	}
	return cmd, nil
}

// ParseCmdFlag parse cmd flag
func ParseCmdFlag(param *Param, paramFlags map[string]interface{}, cmd *cobra.Command) {
	switch param.ParamType {
	case "*string":
		paramFlags[param.ReflectName] = cmd.Flags().StringP(param.FlagName, param.ShortFlag, param.DefaultValue.(string), param.Usage)
	case "*bool":
		paramFlags[param.ReflectName] = cmd.Flags().BoolP(param.FlagName, param.ShortFlag, param.DefaultValue.(bool), param.Usage)
	case "*int64":
		paramFlags[param.ReflectName] = cmd.Flags().Int64P(param.FlagName, param.ShortFlag, int64(param.DefaultValue.(int)), param.Usage)
	case "*uint64":
		paramFlags[param.ReflectName] = cmd.Flags().Uint64P(param.FlagName, param.ShortFlag, uint64(param.DefaultValue.(int)), param.Usage)
	case "*int":
		paramFlags[param.ReflectName] = cmd.Flags().IntP(param.FlagName, param.ShortFlag, param.DefaultValue.(int), param.Usage)
	case "*uint":
		paramFlags[param.ReflectName] = cmd.Flags().UintP(param.FlagName, param.ShortFlag, uint(param.DefaultValue.(int)), param.Usage)
	default:
		panic(fmt.Sprintf("unsupported param type [%v].", param.ParamType))
	}

	if param.Required {
		cmd.MarkFlagRequired(param.FlagName)
	}
}
