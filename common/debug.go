/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"fmt"
	"time"
)

var startTime = time.Now()
var lastTime = time.Now()
var debugMsgs []string

func ResetDebug() {
	startTime = time.Now()
	lastTime = time.Now()
	debugMsgs = nil
}

func AddDebug(module, msg string) {
	now := time.Now()
	debugMsgs = append(debugMsgs, fmt.Sprintf("[%v] | %v | %v", module, msg, now.Sub(lastTime)))
	lastTime = now
}

func PrintDebug(threshold time.Duration) {
	if d := time.Since(startTime); d > threshold {
		fmt.Println("=======================================================================")
		for _, msg := range debugMsgs {
			fmt.Println(msg)
		}
		fmt.Println("=======================================================================")
	}
}
