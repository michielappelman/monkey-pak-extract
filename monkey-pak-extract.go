package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

const (
	sizeOfFileRecord = 20
)

type PAKFileEntry struct {
	FileDataPos uint32 // + startOfData
	FileNamePos uint32 // + startOfFileNames
	DataSize    uint32
	DataSize2   uint32 // real size? (always =dataSize)
	Compressed  uint32 // compressed? (always 0)
}

type PAKHeader struct {
	Magic              [4]byte // KAPL -> "LPAK"
	Version            float32
	StartOfIndex       uint32 // 1 DWORD per file
	StartOfFileEntries uint32 // 5 DWORD per file
	StartOfFileNames   uint32 // zero-terminated string
	StartOfData        uint32
	SizeOfIndex        uint32
	SizeOfFileEntries  uint32
	SizeOfFileNames    uint32
	SizeOfData         uint32
}

type File struct {
	Index  int
	Name   string
	Offset int64
	Size   int64
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("You have to provide the filename of a Monkey Island PAK file")
	}

	PAKDataFile, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal("Error while opening file: ", err)
	}
	defer PAKDataFile.Close()

	binaryHeader := io.NewSectionReader(PAKDataFile, 0, 40)
	var header PAKHeader
	if err := binary.Read(binaryHeader, binary.LittleEndian, &header); err != nil {
		log.Fatal("Reading of header failed: ", err)
	}
	if fmt.Sprintf("%s", header.Magic) != "KAPL" {
		log.Fatal("Provide file is not a valid Monkey Island PAK file")
	}

	numFiles := int(header.SizeOfFileEntries / sizeOfFileRecord)

	var filenames []string
	var files []File

	binaryFileNames := io.NewSectionReader(PAKDataFile,
		int64(header.StartOfFileNames),
		int64(header.SizeOfFileNames))
	filenameScanner := bufio.NewScanner(binaryFileNames)
	filenameSplit := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		for i := 0; i < len(data); i++ {
			if data[i] == '\x00' {
				return i + 1, data[:i], nil
			}
		}
		if !atEOF {
			return 0, nil, nil
		}
		return 0, data, bufio.ErrFinalToken
	}
	filenameScanner.Split(filenameSplit)
	for i := 0; filenameScanner.Scan(); i++ {
		filenames = append(filenames, filenameScanner.Text())
	}

	for i := 0; i < numFiles; i++ {
		binaryFileEntry := io.NewSectionReader(PAKDataFile,
			int64(int(header.StartOfFileEntries)+sizeOfFileRecord*i),
			int64(header.SizeOfFileEntries))

		var pakje PAKFileEntry
		if err := binary.Read(binaryFileEntry, binary.LittleEndian, &pakje); err != nil {
			fmt.Println("Reading of PAK file failed:", err)
		}
		files = append(files, File{
			Index:  i,
			Name:   filenames[i],
			Offset: int64(pakje.FileDataPos + header.StartOfData),
			Size:   int64(pakje.DataSize),
		})
	}

	for _, file := range files {
		binaryData := io.NewSectionReader(PAKDataFile,
			int64(file.Offset),
			int64(file.Size))
		os.MkdirAll(filepath.Dir(file.Name), os.ModePerm)
		outputFile, err := os.Create(file.Name)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := io.Copy(outputFile, binaryData); err != nil {
			log.Fatal(err)
		}
		outputFile.Close()
	}
}
