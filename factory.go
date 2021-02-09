package file

import (
	"github.com/paulhenri-l/gofile/contracts"
)

type managerFactory func(fileName string) (contracts.FileManager, error)
