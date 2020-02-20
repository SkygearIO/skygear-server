package loginid

import (
	"strings"

	"io/ioutil"
	"os"
)

type ReservedNameChecker struct {
	ReservedWords []string
}

func NewReservedNameCheckerWithFile(sourceFile string) (*ReservedNameChecker, error) {
	f, err := os.Open(sourceFile)
	if err != nil {
		return nil, err
	}
	content, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	reservedWords := strings.Split(string(content), "\n")

	return &ReservedNameChecker{
		ReservedWords: reservedWords,
	}, nil
}

func (c *ReservedNameChecker) IsReserved(name string) (bool, error) {
	for i := 0; i < len(c.ReservedWords); i++ {
		if c.ReservedWords[i] == name {
			return true, nil
		}
	}

	return false, nil
}
