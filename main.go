package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

var (
	arc                                                           *os.File
	NT_buf                                                        bytes.Buffer
	folders_entry_massive                                         []FolderEntry
	files_entry_massive                                           []FileEntry
	path_level, folders, files, folders_count, baseoffset, fcount int32
	filepath_massive                                              = make([]string, 50)
	foldercount_massive                                           = make([]int32, 50)
	filecount_massive                                             = make([]int32, 50)
	path                                                          string
)

type FolderEntry struct {
	FN_offset        int32 // offset of the folder name in the Name Table
	subfolder_count  int32 // number of the subfolders(only the 1st level subfolders, nested folders are not included)
	branch_start_num int32 // Not sure about this      (-1 when subfolder_count is null)
	branch_end_num   int32 // part with 100% certainty (-1 when it's the last subfolder for its parent folder)
	filecount        int32 // number of files in the folder
	start_file_num   int32 // number of the file from which start to read filecount number (-1 when filecount is null)
}

type FileEntry struct {
	FiN_offset int32 // offset of the file name in the Name Table
	offset     int32 // relative offset of the file
	size       int32 // size of the file
}

func FolderEntryRead(f *os.File) FolderEntry {
	return FolderEntry{
		FN_offset:        ReadInt32(f),
		subfolder_count:  ReadInt32(f),
		branch_start_num: ReadInt32(f),
		branch_end_num:   ReadInt32(f),
		filecount:        ReadInt32(f),
		start_file_num:   ReadInt32(f),
	}

}

func FileEntryRead(f *os.File) FileEntry {
	return FileEntry{
		FiN_offset: ReadInt32(f),
		offset:     ReadInt32(f),
		size:       ReadInt32(f),
	}
}

func ReadInt32(r io.Reader) int32 {
	var buf bytes.Buffer
	io.CopyN(&buf, r, 4)
	return int32(binary.LittleEndian.Uint32(buf.Bytes()))
}

func ReadCString(r *bytes.Reader) string {
	var str string
	b := make([]byte, 1)
	b[0] = 1
	for b[0] != 0 {
		r.Read(b)
		if b[0] != 0 {
			str = str + string(b[0])
		}
	}
	return str
}

func unpack() {

	NT_reader := bytes.NewReader(NT_buf.Bytes())
	for j := 0; j <= int(path_level); j++ {
		path = path + filepath_massive[j] + "\\"
	}
	path = strings.Replace(path, "\\", "", 1)
	current_path := path
	os.MkdirAll(path, 0700)
	path = ""
	for i := 0; i < int(foldercount_massive[path_level]); i++ {
		NT_reader.Seek(int64(folders_entry_massive[folders_count].FN_offset), 0)
		Fname := ReadCString(NT_reader)
		path_level++
		foldercount_massive[path_level] = folders_entry_massive[folders_count].subfolder_count
		filecount_massive[path_level] = folders_entry_massive[folders_count].filecount
		filepath_massive[path_level] = Fname
		folders_count++
		unpack()
	}
	for j := 0; j < int(filecount_massive[path_level]); j++ {
		NT_reader.Seek(int64(files_entry_massive[folders_entry_massive[folders_count-1].start_file_num+int32(j)].FiN_offset), 0)
		FiName := ReadCString(NT_reader)
		f, _ := os.Create(current_path + FiName)
		defer f.Close()
		arc.Seek(int64(files_entry_massive[folders_entry_massive[folders_count-1].start_file_num+int32(j)].offset+baseoffset), 0)
		io.CopyN(f, arc, int64(files_entry_massive[folders_entry_massive[folders_count-1].start_file_num+int32(j)].size))
		fcount++
		fmt.Printf("0x%X       %v        %s\n", files_entry_massive[folders_entry_massive[folders_count-1].start_file_num+int32(j)].offset+baseoffset,
			files_entry_massive[folders_entry_massive[folders_count-1].start_file_num+int32(j)].size, current_path+FiName)
	}
	path_level--
}

func main() {

	arc, _ = os.Open("DATA.WAD")
	NT_size := ReadInt32(arc)
	arc.Seek(int64(NT_size), 1)
	folders = ReadInt32(arc)
	for i := 0; i < int(folders); i++ {
		FE := FolderEntryRead(arc)
		folders_entry_massive = append(folders_entry_massive, FE)
	}
	files = ReadInt32(arc)
	for i := 0; i < int(files); i++ {
		FiE := FileEntryRead(arc)
		files_entry_massive = append(files_entry_massive, FiE)
	}
	arc.Seek(4, 0)
	io.CopyN(&NT_buf, arc, int64(NT_size))
	baseoffset = (NT_size + 4 + (folders * 0x18) + 4 + (files * 0x0C)) + (0x800 - (NT_size+4+(folders*0x18)+4+(files*0x0C))%0x800)

	//Root Folder Init
	foldercount_massive[path_level] = folders_entry_massive[path_level].subfolder_count
	filecount_massive[path_level] = folders_entry_massive[path_level].filecount
	folders_count++
	fmt.Println("Offset       Size                     Name   ")
	unpack()
	fmt.Printf("\nSuccesfully extracted %v files\n", fcount)

}
