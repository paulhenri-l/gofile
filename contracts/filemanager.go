//go:generate mockgen -package contracts -destination ../mocks/contracts/filemanager.go . FileManager

package contracts

type FileManager interface {
	Write(b []byte) (int, error)
	WrittenBytes() uint64
	Close() error
}