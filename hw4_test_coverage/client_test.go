package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"testing"
)

type UsersXml struct {
	Version string    `xml:"version,attr"`
	List    []UserXml `xml:"row"`
}

type UserXml struct {
	Id        int    `xml:"id"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Age       int    `xml:"age"`
	About     string `xml:"about"`
	Gender    string `xml:"gender"`
}

// код писать тут
func SearchServer(res http.ResponseWriter, req *http.Request) {
	limit, _ := strconv.Atoi(req.FormValue("limit"))
	offset, _ := strconv.Atoi(req.FormValue("offset"))
	users, err := getUsers(limit, offset)
	if err != nil {
		fmt.Printf("SearchServer error: %v", err)
		return
	}

	usersJson, err := json.Marshal(users)
	if err != nil {
		fmt.Printf("SearchServer error: %v", err)
		return
	}

	res.WriteHeader(http.StatusOK)
	_, err = res.Write(usersJson)
	if err != nil {
		fmt.Printf("SearchServer error: %v", err)
		return
	}
}

func getUsers(limit, offset int) ([]User, error) {
	fileReader, err := os.Open("./dataset.xml")
	if err != nil {
		return nil, err
	}
	defer fileReader.Close()
	bytes, err := ioutil.ReadAll(fileReader)
	if err != nil {
		return nil, err
	}

	usersXml := UsersXml{}
	err = xml.Unmarshal(bytes, &usersXml)
	if err != nil {
		return nil, err
	}
	users := make([]User, len(usersXml.List))

	for idx, userXml := range usersXml.List {
		users[idx] = User{
			Id:     userXml.Id,
			Name:   userXml.FirstName + " " + userXml.LastName,
			Age:    userXml.Age,
			About:  userXml.About,
			Gender: userXml.Gender,
		}
	}

	return users[offset : offset+limit], nil
}

// Should handle empty search request.
func TestEmptyRequest(t *testing.T) {
	// Initialize test server instance
	testServer := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer testServer.Close()
	// Use hardcoded data instead.
	users := []User{}
	expected := &SearchResponse{Users: users, NextPage: true}
	client := SearchClient{AccessToken: "test_token", URL: testServer.URL}
	request := SearchRequest{}
	result, err := client.FindUsers(request)
	if err != nil {
		t.Errorf("FindUsers error: %v", err)
	} else if !reflect.DeepEqual(expected, result) {
		t.Errorf("Expected: %v\nActual: %v", expected, result)
	}
}

// Should handle search request with limit.
func TestRequestWithLimit(t *testing.T) {
	// Initialize test server instance
	testServer := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer testServer.Close()
	// Use hardcoded data instead.
	users, _ := getUsers(3, 0)
	expected := &SearchResponse{Users: users, NextPage: true}
	client := SearchClient{AccessToken: "test_token", URL: testServer.URL}
	request := SearchRequest{Limit: 3, Offset: 0}
	result, err := client.FindUsers(request)
	if err != nil {
		t.Errorf("FindUsers error: %v", err)
	} else if !reflect.DeepEqual(expected, result) {
		t.Errorf("Expected: %v\nActual: %v", expected, result)
	}
}

// Should handle search request with offset.
func TestRequestWithOffset(t *testing.T) {
	// Initialize test server instance
	testServer := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer testServer.Close()
	// Use hardcoded data instead.
	users, _ := getUsers(1, 10)
	expected := &SearchResponse{Users: users, NextPage: true}
	client := SearchClient{AccessToken: "test_token", URL: testServer.URL}
	request := SearchRequest{Limit: 1, Offset: 10}
	result, err := client.FindUsers(request)
	if err != nil {
		t.Errorf("FindUsers error: %v", err)
	} else if !reflect.DeepEqual(expected, result) {
		t.Errorf("Expected: %v\nActual: %v", expected, result)
	}
}
