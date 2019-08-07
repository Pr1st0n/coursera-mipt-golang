package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

//easyjson:json
type User struct {
	Browsers []string `json:"browsers"`
	Company  string   `json:"-"`
	Country  string   `json:"-"`
	Email    string   `json:"email"`
	Job      string   `json:"-"`
	Name     string   `json:"name"`
	Phone    string   `json:"-"`
}

// вам надо написать более быструю оптимальную этой функции
func FastSearch(out io.Writer) {
	fileReader, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}

	defer func() {
		err := fileReader.Close()
		if err != nil {
			fmt.Printf("error: %v", err)
			return
		}
	}()

	scanner := bufio.NewScanner(fileReader)
	browsers := map[string]bool{}
	users := []string{}
	counter := -1

	for scanner.Scan() {
		bytes := scanner.Bytes()
		user := &User{}
		isAndroid := false
		isMSIE := false
		counter++

		err := user.UnmarshalJSON(bytes)
		if err != nil {
			fmt.Printf("error: %v", err)
			return
		}

		for _, browser := range user.Browsers {
			if strings.Contains(browser, "Android") {
				isAndroid = true
			} else if strings.Contains(browser, "MSIE") {
				isMSIE = true
			} else {
				continue
			}

			if _, exists := browsers[browser]; !exists {
				browsers[browser] = true
			}
		}

		if !(isAndroid && isMSIE) {
			continue
		}

		email := strings.Replace(user.Email, "@", " [at] ", 1)
		users = append(users, fmt.Sprintf("[%d] %s <%s>\n", counter, user.Name, email))
	}

	_, err = fmt.Fprintln(out, "found users:\n"+strings.Join(users, ""))
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}

	_, err = fmt.Fprintln(out, "Total unique browsers", len(browsers))
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}
}
