package main

import (
	"fmt"
)

func createOne(arr []int, arrLen int) {
	//fmt.Println(arr, arrLen)
	var flag byte = 0
	for _, v := range arr {
		flag = flag | (1 << (7 - uint(v)))
	}
	fmt.Print(arr)
	fmt.Printf(",%d, %02X\r\n", arrLen, flag)
}

func selectNum(pre []int, preLen int, arr []int, arrLen int, cnt int) {
	//fmt.Println("selectNum", pre, preLen, arr, arrLen, cnt)
	if arrLen == cnt {
		createOne(append(pre, arr...), preLen+arrLen)
		return
	}

	if cnt == 1 {
		for _, v := range arr {
			//fmt.Println(append(pre, v))
			createOne(append(pre, v), preLen+1)
		}
		return
	}

	newPreLen, newCnt := preLen+1, cnt-1
	for idx, v := range arr {
		pre1 := make([]int, preLen)
		for idx1, v1 := range pre {
			pre1[idx1] = v1
		}

		newPre := append(pre1, v)
		arr1 := make([]int, arrLen)
		for idx1, v1 := range arr {
			arr1[idx1] = v1
		}

		newArr := append([]int{}, arr1[idx+1:]...)
		newArrLen := arrLen - idx - 1
		selectNum(newPre, newPreLen, newArr, newArrLen, newCnt)
	}
}

func main() {
	org := []int{0, 1, 2, 3, 4, 5, 6, 7}

	/*pre := []int{}
	  for i := 2; i <= 1; i++ {
	      selectNum(pre, 0, org, 8, i)
	  }*/

	newArr := make([]int, len(org))
	copy(newArr, org)
	newArr[0]++
	newArr1 := append(newArr, 100)
	fmt.Println(org, newArr, newArr1)
	//helper :=
}
