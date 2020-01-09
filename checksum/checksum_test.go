package checksum

import (
	"fmt"
	"testing"
)

func TestCRC32CChecksum(t *testing.T) {
	bytes := []byte("sample bytes")
	expectedChecksum := uint32(1168601409)
	checksum, err := CRC32CChecksum(bytes)
	if err != nil {
		t.Errorf("Expected error to be nil; got %s", err)
	}

	if checksum != expectedChecksum {
		t.Errorf("Expected CRC32CChecksum(\"sample bytes\") to equal %d; got %d", expectedChecksum, checksum)
	}
	fmt.Println(checksum)
}
