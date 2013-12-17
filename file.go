package main

import (
	"io/ioutil"
	"os"
	"strconv"
)

type readableSysPath string
type writableSysPath string

type readableSysFile struct {
	*os.File
}
type writableSysFile struct {
	*os.File
}

type pwmLocation writableSysPath
type fanLocation readableSysPath
type tempLocation readableSysPath

func (r readableSysPath) Open() (readableSysFile, error) {
	f, err := os.Open(string(r))
	return readableSysFile{f}, err
}

func (w writableSysPath) Open() (writableSysFile, error) {
	f, err := os.OpenFile(string(w), os.O_WRONLY, 0666)
	return writableSysFile{f}, err
}

func (w writableSysFile) WriteString(val string) error {
	w.Seek(0, 0)
	_, err := w.File.WriteString(val)
	w.Sync()
	return err
}

func (r readableSysFile) ReadVal() (uint8, error) {
	// Go back to the beginning
	r.Seek(0, 0)
	// Read the file
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return 0, err
	}
	// Remove the newline
	data = data[:len(data)-1]
	
	i, err := strconv.ParseUint(string(data), 10, 32)
	i = i / 1000

	return uint8(i), err
}
