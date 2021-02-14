package gofile

import (
	"github.com/paulhenri-l/gofile/contracts"
)

type ManagerFactory func(fileName string) (contracts.FileManager, error)
