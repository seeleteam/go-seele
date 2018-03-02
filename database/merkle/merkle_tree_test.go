// Copyright 2017 Cameron Bergoon
// Licensed under the MIT License, see LICENCE file for details.

package merkle

import (
	"bytes"
	_ "fmt"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
)

//TestContent implements the Content interface provided by merkletree and represents the content stored in the tree.
type TestContent struct {
	x string
}

//CalculateHash hashes the values of a TestContent
func (t TestContent) CalculateHash() []byte {
	return hash(t.x)
}

//Equals tests for equality of two Contents
func (t TestContent) Equals(other Content) bool {
	return t.x == other.(TestContent).x
}

var table = []struct {
	contents     []Content
	expectedHash []byte
}{
	{
		contents: []Content{
			TestContent{
				x: "Hello",
			},
			TestContent{
				x: "Hi",
			},
			TestContent{
				x: "Hey",
			},
			TestContent{
				x: "Hola",
			},
		},
		expectedHash: []byte{133, 183, 57, 206, 88, 22, 237, 115, 75, 54, 242, 139, 113, 153, 155, 58, 145, 166, 191, 199, 104, 91, 72, 13, 106, 243, 58, 146, 111, 204, 252, 216},
	},
	{
		contents: []Content{
			TestContent{
				x: "Hello",
			},
			TestContent{
				x: "Hi",
			},
			TestContent{
				x: "Hey",
			},
		},
		expectedHash: []byte{183, 174, 252, 165, 214, 86, 111, 189, 95, 100, 65, 114, 205, 222, 94, 157, 217, 79, 49, 175, 75, 110, 195, 121, 107, 1, 100, 208, 51, 98, 95, 51},
	},
	{
		contents: []Content{
			TestContent{
				x: "Hello",
			},
			TestContent{
				x: "Hi",
			},
			TestContent{
				x: "Hey",
			},
			TestContent{
				x: "Greetings",
			},
			TestContent{
				x: "Hola",
			},
		},
		expectedHash: []byte{143, 231, 243, 158, 169, 141, 186, 240, 184, 190, 223, 105, 252, 126, 23, 209, 156, 201, 19, 85, 138, 146, 32, 182, 52, 202, 201, 99, 84, 186, 191, 227},
	},
	{
		contents: []Content{
			TestContent{
				x: "123",
			},
			TestContent{
				x: "234",
			},
			TestContent{
				x: "345",
			},
			TestContent{
				x: "456",
			},
			TestContent{
				x: "1123",
			},
			TestContent{
				x: "2234",
			},
			TestContent{
				x: "3345",
			},
			TestContent{
				x: "4456",
			},
		},
		expectedHash: []byte{42, 124, 96, 245, 43, 157, 254, 62, 240, 122, 167, 153, 62, 112, 16, 130, 177, 143, 235, 21, 243, 204, 1, 119, 240, 62, 119, 181, 141, 42, 60, 26},
	},
	{
		contents: []Content{
			TestContent{
				x: "123",
			},
			TestContent{
				x: "234",
			},
			TestContent{
				x: "345",
			},
			TestContent{
				x: "456",
			},
			TestContent{
				x: "1123",
			},
			TestContent{
				x: "2234",
			},
			TestContent{
				x: "3345",
			},
			TestContent{
				x: "4456",
			},
			TestContent{
				x: "4456",
			},
		},
		expectedHash: []byte{38, 96, 140, 67, 44, 112, 81, 66, 16, 123, 55, 76, 6, 51, 0, 47, 21, 59, 1, 32, 231, 238, 189, 192, 99, 239, 129, 146, 81, 97, 64, 2},
	},
}

func TestNewTree(t *testing.T) {
	for i := 0; i < len(table); i++ {
		tree, err := NewTree(table[i].contents)
		if err != nil {
			t.Error("error: unexpected error:  ", err)
		}

		//for _, v := range tree.MerkleRoot() {
		//	fmt.Print(v)
		//	fmt.Print(", ")
		//}
		//fmt.Println("")

		if bytes.Compare(tree.MerkleRoot(), table[i].expectedHash) != 0 {
			t.Errorf("error: expected hash equal to %v got %v", table[i].expectedHash, tree.MerkleRoot())
		}
	}
}

func TestMerkleTree_MerkleRoot(t *testing.T) {
	for i := 0; i < len(table); i++ {
		tree, err := NewTree(table[i].contents)
		if err != nil {
			t.Error("error: unexpected error:  ", err)
		}
		if bytes.Compare(tree.MerkleRoot(), table[i].expectedHash) != 0 {
			t.Errorf("error: expected hash equal to %v got %v", table[i].expectedHash, tree.MerkleRoot())
		}
	}
}

func TestMerkleTree_RebuildTree(t *testing.T) {
	for i := 0; i < len(table); i++ {
		tree, err := NewTree(table[i].contents)
		if err != nil {
			t.Error("error: unexpected error:  ", err)
		}
		err = tree.RebuildTree()
		if err != nil {
			t.Error("error: unexpected error:  ", err)
		}
		if bytes.Compare(tree.MerkleRoot(), table[i].expectedHash) != 0 {
			t.Errorf("error: expected hash equal to %v got %v", table[i].expectedHash, tree.MerkleRoot())
		}
	}
}

func TestMerkleTree_RebuildTreeWith(t *testing.T) {
	for i := 0; i < len(table)-1; i++ {
		tree, err := NewTree(table[i].contents)
		if err != nil {
			t.Error("error: unexpected error:  ", err)
		}
		err = tree.RebuildTreeWith(table[i+1].contents)
		if err != nil {
			t.Error("error: unexpected error:  ", err)
		}
		if bytes.Compare(tree.MerkleRoot(), table[i+1].expectedHash) != 0 {
			t.Errorf("error: expected hash equal to %v got %v", table[i+1].expectedHash, tree.MerkleRoot())
		}
	}
}

func TestMerkleTree_VerifyTree(t *testing.T) {
	for i := 0; i < len(table); i++ {
		tree, err := NewTree(table[i].contents)
		if err != nil {
			t.Error("error: unexpected error:  ", err)
		}
		v1 := tree.VerifyTree()
		if v1 != true {
			t.Error("error: expected tree to be valid")
		}
		tree.Root.Hash = []byte{1}
		tree.merkleRoot = []byte{1}
		v2 := tree.VerifyTree()
		if v2 != false {
			t.Error("error: expected tree to be invalid")
		}
	}
}

func TestMerkleTree_VerifyContent(t *testing.T) {
	for i := 0; i < len(table); i++ {
		tree, err := NewTree(table[i].contents)
		if err != nil {
			t.Error("error: unexpected error:  ", err)
		}
		if len(table[i].contents) > 0 {
			v := tree.VerifyContent(tree.MerkleRoot(), table[i].contents[0])
			if !v {
				t.Error("error: expected valid content")
			}
		}
		if len(table[i].contents) > 1 {
			v := tree.VerifyContent(tree.MerkleRoot(), table[i].contents[1])
			if !v {
				t.Error("error: expected valid content")
			}
		}
		if len(table[i].contents) > 2 {
			v := tree.VerifyContent(tree.MerkleRoot(), table[i].contents[2])
			if !v {
				t.Error("error: expected valid content")
			}
		}
		if len(table[i].contents) > 0 {
			tree.Root.Hash = []byte{1}
			tree.merkleRoot = []byte{1}
			v := tree.VerifyContent(tree.MerkleRoot(), table[i].contents[0])
			if v {
				t.Error("error: expected invalid content")
			}
			tree.RebuildTree()
		}
		v := tree.VerifyContent(tree.MerkleRoot(), TestContent{x: "NotInTestTable"})
		if v {
			t.Error("error: expected invalid content")
		}
	}
}

func TestMerkleTree_String(t *testing.T) {
	for i := 0; i < len(table); i++ {
		tree, err := NewTree(table[i].contents)
		if err != nil {
			t.Error("error: unexpected error:  ", err)
		}
		if tree.String() == "" {
			t.Error("error: expected not empty string")
		}
	}
}

func hash(value interface{}) []byte {
	buff := common.SerializePanic(value)

	return crypto.Keccak256Hash(buff)
}
