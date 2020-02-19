package mmap

import (
	"fmt"
	"os"
	"syscall"
)

// MmappedFile represents an mmaped file
type MmappedFile struct {
	Data           []byte
	FileDescriptor int
	FileName       string
	Size           int64
}

// Close munmaps a mmaped file and closes it's file descriptor
func (mf MmappedFile) Close() error {
	err := syscall.Munmap(mf.Data)
	if err != nil {
		return err
	}
	err = syscall.Close(mf.FileDescriptor)
	if err != nil {
		return err
	}
	return nil
}

// WriteAt writes the bytes from p starting at off offset in a mmaped file's data
func (mf MmappedFile) WriteAt(p []byte, off int64) (int, error) {
	if int64(len(p))+off > mf.Size {
		return 0, nil
	}
	n := 0
	for i := 0; i < len(p); i++ {
		mf.Data[int64(i)+off] = p[i]
		n++
	}
	return n, nil
}

// ReadAt reads bytes into p starting at off offset in a mmaped file's data
func (mf MmappedFile) ReadAt(p []byte, off int64) (int, error) {
	if int64(len(p))+off > mf.Size {
		return 0, nil
	}
	n := 0
	for i := 0; i < len(p); i++ {
		p[i] = mf.Data[int64(i)+off]
		n++
	}
	return n, nil
}

// Read reads bytes into p starting at offset 0 in a mmaped file's data
func (mf MmappedFile) Read(p []byte) (int, error) {
	return mf.ReadAt(p, 0)
}

// CreateFile creates a new mmaped file
func CreateFile(filename string, length int64) (*MmappedFile, error) {
	_, err := os.Stat(filename)
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("File %s already exists", filename)
	}

	fd, err := syscall.Open(filename, syscall.O_CREAT|syscall.O_RDWR, 0664)
	if err != nil {
		panic(err)
	}

	data, err := syscall.Mmap(fd, 0, int(length), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return nil, err
	}

	err = syscall.Truncate(filename, length)
	if err != nil {
		return nil, err
	}

	result := MmappedFile{
		Data:           data,
		FileDescriptor: fd,
		FileName:       filename,
		Size:           length,
	}

	return &result, nil
}

// OpenFile opens an mmaped file
func OpenFile(filename string) (*MmappedFile, error) {
	stats, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("File %s does not exist", filename)
	}

	fd, err := syscall.Open(filename, syscall.O_CREAT|syscall.O_RDWR, 0664)
	if err != nil {
		panic(err)
	}

	data, err := syscall.Mmap(fd, 0, int(stats.Size()), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return nil, err
	}

	result := MmappedFile{
		Data:           data,
		FileDescriptor: fd,
		FileName:       filename,
		Size:           stats.Size(),
	}

	return &result, nil
}
