package mmap

import (
	"fmt"
	"os"
	"reflect"
	"syscall"
	"unsafe"
)

// WARNING this is not yet portable - still experimental

// MmappedFile represents an mmaped file
type MmappedFile struct {
	Data           []byte
	FileDescriptor int
	Size           int64
}

// Close munmaps a mmaped file and closes it's file descriptor
func (mf MmappedFile) Close() {
	syscall.Munmap(mf.Data)
	syscall.Close(mf.FileDescriptor)
}

// WriteAt writes the p bytes at off offset in a mmaped file's data
func (mf MmappedFile) WriteAt(p []byte, off int64) (int, error) {
	if int64(len(p))+off > mf.Size {
		// TODO
		return 0, nil
	}
	n := 0
	for i := 0; i < len(p); i++ {
		mf.Data[int64(i)+off] = p[i]
		n++
	}
	return n, nil
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

	uniAddr := uintptr(0)
	uniLength := uintptr(length)
	uniProt := uintptr(syscall.PROT_READ | syscall.PROT_WRITE)
	uniFlags := uintptr(syscall.MAP_SHARED)
	uniFd := uintptr(fd)
	uniOffset := uintptr(int64(0))
	addr, _, err := syscall.Syscall6(
		syscall.SYS_MMAP,
		uniAddr,
		uniLength,
		uniProt,
		uniFlags,
		uniFd,
		uniOffset,
	)
	if err != syscall.Errno(0) {
		return nil, err
	}

	err = syscall.Truncate(filename, length)
	if err != nil {
		return nil, err
	}

	var bytes []byte
	dh := (*reflect.SliceHeader)(unsafe.Pointer(&bytes))
	dh.Data = addr
	dh.Len = int(length)
	dh.Cap = dh.Len

	result := MmappedFile{
		Data:           bytes,
		FileDescriptor: fd,
		Size:           length,
	}

	return &result, nil
}
