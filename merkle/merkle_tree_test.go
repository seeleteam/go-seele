/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package merkle

import (
	"bytes"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
)

// TestContent implements the Content interface provided by merkletree and represents the content stored in the tree.
type TestContent struct {
	x string
}

// CalculateHash hashes the values of a TestContent
func (t TestContent) CalculateHash() common.Hash {
	return hash(t.x)
}

// Equals tests for equality of two Contents
func (t TestContent) Equals(other Content) bool {
	return t.x == other.(TestContent).x
}

var table = []struct {
	contents     []Content
	expectedHash []byte
}{
	{
		contents:     CreateContent([]string{"Hello", "Hi", "Hey", "Hola"}),
		expectedHash: []byte{133, 183, 57, 206, 88, 22, 237, 115, 75, 54, 242, 139, 113, 153, 155, 58, 145, 166, 191, 199, 104, 91, 72, 13, 106, 243, 58, 146, 111, 204, 252, 216},
	},
	{
		contents:     CreateContent([]string{"Hello", "Hi", "Hey"}),
		expectedHash: []byte{183, 174, 252, 165, 214, 86, 111, 189, 95, 100, 65, 114, 205, 222, 94, 157, 217, 79, 49, 175, 75, 110, 195, 121, 107, 1, 100, 208, 51, 98, 95, 51},
	},
	{
		contents:     CreateContent([]string{"Hello", "Hi", "Hey", "Greetings", "Hola"}),
		expectedHash: []byte{143, 231, 243, 158, 169, 141, 186, 240, 184, 190, 223, 105, 252, 126, 23, 209, 156, 201, 19, 85, 138, 146, 32, 182, 52, 202, 201, 99, 84, 186, 191, 227},
	},
	{
		contents:     CreateContent([]string{"123", "234", "345", "456", "1123", "2234", "3345", "4456"}),
		expectedHash: []byte{42, 124, 96, 245, 43, 157, 254, 62, 240, 122, 167, 153, 62, 112, 16, 130, 177, 143, 235, 21, 243, 204, 1, 119, 240, 62, 119, 181, 141, 42, 60, 26},
	},
	{
		contents:     CreateContent([]string{"123", "234", "345", "456", "1123", "2234", "3345", "4456", "4456"}),
		expectedHash: []byte{38, 96, 140, 67, 44, 112, 81, 66, 16, 123, 55, 76, 6, 51, 0, 47, 21, 59, 1, 32, 231, 238, 189, 192, 99, 239, 129, 146, 81, 97, 64, 2},
	},
}

func CreateContent(strs []string) []Content {
	contents := make([]Content, len(strs))
	for i, s := range strs {
		contents[i] = TestContent{
			x: s,
		}
	}

	return contents
}

func Test_NewTree(t *testing.T) {
	for i := 0; i < len(table); i++ {
		tree, err := NewTree(table[i].contents)
		if err != nil {
			t.Fatalf("error: unexpected error:  ", err)
		}

		if bytes.Compare(tree.MerkleRoot().Bytes(), table[i].expectedHash) != 0 {
			t.Errorf("error: expected hash equal to %v got %v", table[i].expectedHash, tree.MerkleRoot())
		}
	}
}

func Test_MerkleTree_MerkleRoot(t *testing.T) {
	for i := 0; i < len(table); i++ {
		tree, err := NewTree(table[i].contents)
		if err != nil {
			t.Fatalf("error: unexpected error:  ", err)
		}
		if bytes.Compare(tree.MerkleRoot().Bytes(), table[i].expectedHash) != 0 {
			t.Errorf("error: expected hash equal to %v got %v", table[i].expectedHash, tree.MerkleRoot())
		}
	}
}

func Test_MerkleTree_RebuildTree(t *testing.T) {
	for i := 0; i < len(table); i++ {
		tree, err := NewTree(table[i].contents)
		if err != nil {
			t.Fatalf("error: unexpected error:  ", err)
		}
		err = tree.RebuildTree()
		if err != nil {
			t.Fatalf("error: unexpected error:  ", err)
		}
		if bytes.Compare(tree.MerkleRoot().Bytes(), table[i].expectedHash) != 0 {
			t.Errorf("error: expected hash equal to %v got %v", table[i].expectedHash, tree.MerkleRoot())
		}
	}
}

func Test_MerkleTree_RebuildTreeWith(t *testing.T) {
	for i := 0; i < len(table)-1; i++ {
		tree, err := NewTree(table[i].contents)
		if err != nil {
			t.Fatalf("error: unexpected error:  ", err)
		}
		err = tree.RebuildTreeWith(table[i+1].contents)
		if err != nil {
			t.Fatalf("error: unexpected error:  ", err)
		}
		if bytes.Compare(tree.MerkleRoot().Bytes(), table[i+1].expectedHash) != 0 {
			t.Errorf("error: expected hash equal to %v got %v", table[i+1].expectedHash, tree.MerkleRoot())
		}
	}
}

func Test_MerkleTree_VerifyTree(t *testing.T) {
	for i := 0; i < len(table); i++ {
		tree, err := NewTree(table[i].contents)
		if err != nil {
			t.Fatalf("error: unexpected error:  ", err)
		}
		v1 := tree.VerifyTree()
		if v1 != true {
			t.Error("error: expected tree to be valid")
		}
		tree.Root.Hash = common.BytesToHash([]byte{1})
		tree.merkleRoot = common.BytesToHash([]byte{1})
		v2 := tree.VerifyTree()
		if v2 != false {
			t.Error("error: expected tree to be invalid")
		}
	}
}

func Test_MerkleTree_VerifyContent(t *testing.T) {
	for i := 0; i < len(table); i++ {
		tree, err := NewTree(table[i].contents)
		if err != nil {
			t.Fatalf("error: unexpected error:  ", err)
		}
		if len(table[i].contents) > 0 {
			v := tree.VerifyContent(tree.MerkleRoot().Bytes(), table[i].contents[0])
			if !v {
				t.Error("error: expected valid content")
			}
		}
		if len(table[i].contents) > 1 {
			v := tree.VerifyContent(tree.MerkleRoot().Bytes(), table[i].contents[1])
			if !v {
				t.Error("error: expected valid content")
			}
		}
		if len(table[i].contents) > 2 {
			v := tree.VerifyContent(tree.MerkleRoot().Bytes(), table[i].contents[2])
			if !v {
				t.Error("error: expected valid content")
			}
		}
		if len(table[i].contents) > 0 {
			tree.Root.Hash = common.BytesToHash([]byte{1})
			tree.merkleRoot = common.BytesToHash([]byte{1})
			v := tree.VerifyContent(tree.MerkleRoot().Bytes(), table[i].contents[0])
			if v {
				t.Error("error: expected invalid content")
			}
			tree.RebuildTree()
		}
		v := tree.VerifyContent(tree.MerkleRoot().Bytes(), TestContent{x: "NotInTestTable"})
		if v {
			t.Error("error: expected invalid content")
		}
	}
}

func Test_MerkleTree_String(t *testing.T) {
	for i := 0; i < len(table); i++ {
		tree, err := NewTree(table[i].contents)
		if err != nil {
			t.Fatalf("error: unexpected error:  ", err)
		}
		if tree.String() == "" {
			t.Error("error: expected not empty string")
		}
	}
}

func hash(value interface{}) common.Hash {
	return crypto.MustHash(value)
}
