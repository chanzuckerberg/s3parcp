package checksum

import (
	"hash/crc32"
	"io"
	"os"

	"github.com/vimeo/go-util/crc32combine"
	"golang.org/x/exp/mmap"
)

// CRC32CChecksum computes the crc32c checksum of a file
func CRC32CChecksum(data []byte) (uint32, error) {
	// This ensures that we use the crc32c system command if available
	//   I stepped though the code to verify
	crc32q := crc32.MakeTable(crc32.Castagnoli)
	return crc32.Checksum(data, crc32q), nil
}

type chunkRange struct {
	Chunk int64
	Start int64
	End   int64
}

type chunkChecksum struct {
	Chunk    int64
	Checksum uint32
	Error    error
}

func checksumWorker(readerAt *io.ReaderAt, chunkRanges <-chan chunkRange, checksums chan<- chunkChecksum) {
	for chunkRange := range chunkRanges {
		data := make([]byte, chunkRange.End-chunkRange.Start)
		(*readerAt).ReadAt(data, chunkRange.Start)
		checksum, err := CRC32CChecksum(data)
		checksums <- chunkChecksum{
			Chunk:    chunkRange.Chunk,
			Checksum: checksum,
			Error:    err,
		}
	}
}

// ParallelCRC32CChecksum a
func ParallelCRC32CChecksum(filename string, partSize int64, concurrency int, useMmap bool) (uint32, error) {
	stats, err := os.Stat(filename)
	if err != nil {
		return 0, err
	}
	length := stats.Size()

	var readerAt io.ReaderAt
	if useMmap {
		readerAt, err = mmap.Open(filename)
	} else {
		readerAt, err = os.Open(filename)
	}
	if err != nil {
		return 0, err
	}

	chunks := length / partSize
	lastChunkSize := length % partSize
	if lastChunkSize > 0 {
		chunks++
	} else {
		lastChunkSize = partSize
	}

	chunkRanges := make(chan chunkRange, chunks)
	chunkChecksums := make(chan chunkChecksum, chunks)
	checksums := make([]uint32, chunks)

	for w := 0; w < concurrency; w++ {
		go checksumWorker(&readerAt, chunkRanges, chunkChecksums)
	}

	for i := int64(0); i < chunks-1; i++ {
		chunkRanges <- chunkRange{
			Chunk: i,
			Start: i * partSize,
			End:   (i + 1) * partSize,
		}
	}

	chunkRanges <- chunkRange{
		Chunk: chunks - 1,
		Start: (chunks - 1) * partSize,
		End:   length,
	}

	close(chunkRanges)

	for i := int64(0); i < chunks; i++ {
		chunkChecksum := <-chunkChecksums
		if chunkChecksum.Error != nil {
			return 0, chunkChecksum.Error
		}
		checksums[chunkChecksum.Chunk] = chunkChecksum.Checksum
	}

	checksum := checksums[0]

	for i := int64(1); i < chunks-1; i++ {
		checksum = crc32combine.CRC32Combine(crc32.Castagnoli, checksum, checksums[i], partSize)
	}

	checksum = crc32combine.CRC32Combine(crc32.Castagnoli, checksum, checksums[chunks-1], lastChunkSize)

	return checksum, nil
}
