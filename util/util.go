package util

import "os"

// Try panics if err != nil
func Try(err error) {
	if err != nil {
		panic(err)
	}
}

// Try1 accepts (val, err), returns val, and panics if err != nil
func Try1[V any](val V, err error) V {
	Try(err)
	return val
}

// Try2 accepts (val1, val2,  err), returns (val1, val2),
// and panics if err != nil
func Try2[V1 any, V2 any](val1 V1, val2 V2, err error) (V1, V2) {
	Try(err)
	return val1, val2
}

// GetExePath returns the path name for the executable
// that started the current process.
func GetExePath() string {
	return Try1(os.Executable())
}
