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
	"strings"
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
	orderBy, _ := strconv.Atoi(req.FormValue("order_by"))
	users, err := getUsers(limit, offset, orderBy)
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

func getUsers(limit, offset, orderBy int) ([]User, error) {
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
	users[0] = User{
		Id:     usersXml.List[0].Id,
		Name:   usersXml.List[0].FirstName + " " + usersXml.List[0].LastName,
		Age:    usersXml.List[0].Age,
		About:  usersXml.List[0].About,
		Gender: usersXml.List[0].Gender,
	}

	// Process xml users performing inserts sort on required filed.
	for i := 1; i < len(usersXml.List); i++ {
		userXml := usersXml.List[i]
		user := User{
			Id:     userXml.Id,
			Name:   userXml.FirstName + " " + userXml.LastName,
			Age:    userXml.Age,
			About:  userXml.About,
			Gender: userXml.Gender,
		}
		j := i

		for j > 0 && strings.Compare(user.Name, users[j-1].Name) == orderBy {
			users[j] = users[j-1]
			j--
		}

		users[j] = user
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
	users, _ := getUsers(3, 0, 0)
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
	users, _ := getUsers(1, 10, 0)
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

// Should handle search request with ordered data.
func TestRequestWithOrder(t *testing.T) {
	// Initialize test server instance
	testServer := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer testServer.Close()
	// Use hardcoded data instead.
	users, _ := getUsers(3, 0, 1)
	expected := &SearchResponse{Users: users, NextPage: true}
	client := SearchClient{AccessToken: "test_token", URL: testServer.URL}
	request := SearchRequest{Limit: 3, Offset: 0, OrderBy: 1}
	result, err := client.FindUsers(request)
	if err != nil {
		t.Errorf("FindUsers error: %v", err)
	} else if !reflect.DeepEqual(expected, result) {
		t.Errorf("Expected: %v\nActual: %v", expected, result)
	}
}
