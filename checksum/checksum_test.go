package checksum

import (
	"fmt"
	"os"
	"testing"
)

func TestCRC32CChecksum(t *testing.T) {
	str := "sample bytes"
	bytes := []byte(str)
	expectedChecksum := uint32(1168601409)
	checksum, err := CRC32CChecksum(bytes)
	if err != nil {
		t.Errorf("Expected error to be nil; got %s", err)
	}

	if checksum != expectedChecksum {
		t.Errorf("Expected CRC32CChecksum(\"%s\") to equal %d; got %d", str, expectedChecksum, checksum)
	}
	fmt.Println(checksum)
}

func TestParallelCRC32CChecksum(t *testing.T) {
	filepath := "/tmp/data"
	f, err := os.Create(filepath)
	if err != nil {
		t.Errorf("Creating temporary file for parallel checksum errored with %s", err)
		t.FailNow()
	}
	defer (func() {
		err := f.Close()
		if err != nil {
			t.Errorf("Closing temporary file %s errored with %s", filepath, err)
		}
		err = os.Remove(filepath)
		if err != nil {
			t.Errorf("Removing temporary file %s errored with %s", filepath, err)
		}
	})()

	str := "sample bytes"
	for i := 0; i < 100; i++ {
		str += "sample bytes"
	}
	bytes := []byte(str)
	n, err := f.Write(bytes)
	if n != len(str) {
		t.Errorf("Didn't write all sample bytes to file wanted %d, got %d", len(str), n)
		t.FailNow()
	}
	if err != nil {
		t.Errorf("Writing sample bytes to file errored with: %s", err)
		t.FailNow()
	}

	checksum, err := CRC32CChecksum(bytes)
	if err != nil {
		t.Errorf("Computing in-memory checksum for comparison errored with: %s", err)
		t.FailNow()
	}

	opts := ParallelChecksumOptions{
		Concurrency: 10,
		PartSize:    10,
		UseMmap:     false,
	}
	parallelChecksum, err := ParallelCRC32CChecksum(filepath, opts)
	if err != nil {
		t.Errorf("ParallelCRC32CChecksum errored with %s", err)
		t.FailNow()
	}

	if parallelChecksum != checksum {
		t.Errorf("Expected parallel CRC32C Checksum to Equal %d %d", parallelChecksum, checksum)
	}
}
