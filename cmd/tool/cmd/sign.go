/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

 package cmd

 import (
	 "fmt"
	 "encoding/hex"

	 "github.com/seeleteam/go-seele/crypto"
	 "github.com/spf13/cobra"
 )
 
 var (
	privateKey *string
	message string
 )
 
 // signCmd represents the sign command
 var signCmd = &cobra.Command{
	 Use:   "sign",
	 Short: "sign a message",
	 Long: `For example:
			 tool.exe sign`,
	 Run: func(cmd *cobra.Command, args []string) {
		key, err := crypto.LoadECDSAFromString(*privateKey)
		if err != nil {
			fmt.Printf("failed to load the private key: %s\n", err.Error())
			return
		}
		messageBytes := []byte(message)
		messageHash := crypto.MustHash(messageBytes)
		signature := *crypto.MustSign(key, messageHash.Bytes())
		
		fmt.Println("message")
		fmt.Println(message)
		fmt.Println("message hash")
		fmt.Println(hex.EncodeToString(messageHash.Bytes()))
		fmt.Println("signature")
		fmt.Println(hex.EncodeToString(signature.Sig))
	 },
 }
 
 func init() {
	 rootCmd.AddCommand(signCmd)
 
	 privateKey = signCmd.Flags().StringP("key", "k", "", "private key")
	 signCmd.Flags().StringVarP(&message, "message", "m", "", "message to sign")
 }
 