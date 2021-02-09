//go:generate mockgen -package contracts -destination ../mocks/contracts/file.go . RotatedFileHandler

package contracts

type RotatedFileHandler interface {
	Handle(filepath string) error
}
