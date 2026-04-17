package fuzzer

type Log = FuzzerData

func (fd *FuzzerData) GetLog() any {
	return fd
}
