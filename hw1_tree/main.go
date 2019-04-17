package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
)

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}

func dirTree(out io.Writer, path string, printFiles bool) error {
	offset := [2]int{-1, -1}
	str, err := scanDir("", path, 1, offset, printFiles)

	if err != nil {
		fmt.Println("Error: " + err.Error())
	}

	_, _ = fmt.Fprint(out, str)

	return nil
}

func scanDir(str string, path string, lvl int, offset [2]int, printFiles bool) (string, error) {
	dir, err := ioutil.ReadDir(path)
	subLvl := lvl + 1

	if err != nil {
		return "", err
	}

	if !printFiles {
		dir = filterFiles(dir)
	}

	for idx, file := range dir {
		isLast := idx == len(dir)-1

		for step := 0; step < lvl-1; step++ {
			if step >= offset[0] && step <= offset[1] {
				str += "	"
			} else {
				str += "│	"
			}
		}

		if isLast {
			str += "└───"

			if file.IsDir() {
				if offset[0] == -1 {
					offset[0] = lvl - 1
				}
				if offset[1] < lvl {
					offset[1] = lvl - 1
				}
			}
		} else {
			str += "├───"
		}

		if file.IsDir() {
			str += file.Name() + "\n"
			str, err = scanDir(str, path+"/"+file.Name(), subLvl, offset, printFiles)

			if err != nil {
				return "", err
			}
		} else if printFiles {
			size := "empty"

			if file.Size() > 0 {
				size = strconv.FormatInt(file.Size(), 10) + "b"
			}

			str += file.Name() + " (" + size + ")" + "\n"
		}
	}

	return str, nil
}

func filterFiles(dir []os.FileInfo) []os.FileInfo {
	tmpDir := make([]os.FileInfo, 0)

	for _, file := range dir {
		if file.IsDir() {
			tmpDir = append(tmpDir, file)
		}
	}

	return tmpDir
}
