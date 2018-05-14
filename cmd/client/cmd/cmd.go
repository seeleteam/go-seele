/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/rpc/jsonrpc"
	"reflect"

	"github.com/spf13/cobra"
)

func init() {
	config, err := NewConfig(DefaultPath)
	if err != nil {
		fmt.Println("init cmd fail :\n", err)
	}

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

			client, err := jsonrpc.Dial("tcp", rpcAddr)
			if err != nil {
				fmt.Println(err)
				return
			}
			defer client.Close()

			var output interface{}
			err = client.Call(request.Method, &input, &output)
			if err != nil {
				fmt.Println(err)
				return
			}

			jsonOutput, err := json.MarshalIndent(output, "", "\t")
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println("output :\n", string(jsonOutput))
		},
	}

	for _, param := range request.Params {
		ParseCmdFlag(param, ParamFlags, cmd)
	}
	return cmd, nil
}

// ParseCmdFlag parse cmd flag
func ParseCmdFlag(param *Param, paramFlags map[string]interface{}, cmd *cobra.Command) error {
	switch param.ParamType {
	case "*string":
		if param.ShortHand == "" {
			paramFlags[param.ReflectName] = cmd.Flags().String(param.ParamName, param.DefaultValue.(string), param.Usage)
		} else {
			paramFlags[param.ReflectName] = cmd.Flags().StringP(param.ParamName, param.ShortHand, param.DefaultValue.(string), param.Usage)
		}
	case "*bool":
		if param.ShortHand == "" {
			paramFlags[param.ReflectName] = cmd.Flags().Bool(param.ParamName, param.DefaultValue.(bool), param.Usage)
		} else {
			paramFlags[param.ReflectName] = cmd.Flags().BoolP(param.ParamName, param.ShortHand, param.DefaultValue.(bool), param.Usage)
		}
	case "*int64":
		if param.ShortHand == "" {
			paramFlags[param.ReflectName] = cmd.Flags().Int64(param.ParamName, int64(param.DefaultValue.(float64)), param.Usage)
		} else {
			paramFlags[param.ReflectName] = cmd.Flags().Int64P(param.ParamName, param.ShortHand, int64(param.DefaultValue.(float64)), param.Usage)
		}
	case "*uint64":
		if param.ShortHand == "" {
			paramFlags[param.ReflectName] = cmd.Flags().Uint64(param.ParamName, uint64(param.DefaultValue.(float64)), param.Usage)
		} else {
			paramFlags[param.ReflectName] = cmd.Flags().Uint64P(param.ParamName, param.ShortHand, uint64(param.DefaultValue.(float64)), param.Usage)
		}
	case "*int":
		if param.ShortHand == "" {
			paramFlags[param.ReflectName] = cmd.Flags().Int(param.ParamName, int(param.DefaultValue.(float64)), param.Usage)
		} else {
			paramFlags[param.ReflectName] = cmd.Flags().IntP(param.ParamName, param.ShortHand, int(param.DefaultValue.(float64)), param.Usage)
		}
	default:
		return errors.New("param type match miss, check or add new match in ParseCmdFlag function")
	}

	if param.Required {
		cmd.MarkFlagRequired(param.ParamName)
	}
	return nil
}
