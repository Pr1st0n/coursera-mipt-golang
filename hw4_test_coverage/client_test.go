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
	"time"
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
	if authorized := validateAuthToken(req.Header.Get("AccessToken")); !authorized {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}
	limit, err := strconv.Atoi(req.FormValue("limit"))
	if err != nil {
		res.Write([]byte(fmt.Sprintf("invalid limit parameter value")))
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	offset, err := strconv.Atoi(req.FormValue("offset"))
	if err != nil {
		res.Write([]byte(fmt.Sprintf("invalid offset parameter value")))
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	orderBy, err := strconv.Atoi(req.FormValue("order_by"))
	if err != nil {
		res.Write([]byte(fmt.Sprintf("invalid order_by parameter value")))
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	orderField := req.FormValue("order_field")
	if lowOrderField := strings.ToLower(orderField); len(lowOrderField) > 0 && lowOrderField != "name" &&
		lowOrderField != "age" && lowOrderField != "id" {
		res.WriteHeader(http.StatusBadRequest)
		responseWithError(res, "ErrorBadOrderField")
		return
	}
	users, err := getUsers(limit, offset, orderBy, orderField)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		responseWithError(res, "failed to obtain users data")
		return
	}
	usersJson, err := json.Marshal(users)
	if err != nil {
		responseWithError(res, "failed to process users data")
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
	res.Write(usersJson)
}

// Dummy session validation.
func validateAuthToken(token string) bool {
	if token != "test_token" {
		return false
	}
	return true
}

func getUsers(limit, offset, orderBy int, orderField string) ([]User, error) {
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
	} else if len(usersXml.List) <= offset {
		return nil, fmt.Errorf("offset is greater than actual data length")
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

		if orderBy != OrderByAsIs {
			for j > 0 && (compareUsers(user, users[j-1], orderField) == (orderBy == OrderByAsc)) {
				users[j] = users[j-1]
				j--
			}
		}

		users[j] = user
	}

	if offset+limit > len(users) {
		limit = len(users) - offset
	}

	return users[offset : offset+limit], nil
}

func compareUsers(user1, user2 User, key string) bool {
	switch key {
	case "Age":
		return user1.Age < user2.Age
	case "Id":
		return user1.Id < user2.Id
	default:
		return user1.Name < user2.Name
	}
}

func responseWithError(res http.ResponseWriter, error string) bool {
	errObj := SearchErrorResponse{Error: error}
	resBody, _ := json.Marshal(errObj)
	_, err := res.Write(resBody)
	if err != nil {
		return false
	}
	return true
}

// Should handle failed authorization.
func TestAuthFailed(t *testing.T) {
	// Initialize test server instance
	testServer := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer testServer.Close()
	client := SearchClient{AccessToken: "invalid_token", URL: testServer.URL}
	request := SearchRequest{}
	_, err := client.FindUsers(request)
	if err == nil || err.Error() != "Bad AccessToken" {
		t.Errorf("FindUsers error: should not authorize with invalid token")
	}
}

// Should handle successful authorization.
func TestAuthSuccess(t *testing.T) {
	// Initialize test server instance
	testServer := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer testServer.Close()
	client := SearchClient{AccessToken: "test_token", URL: testServer.URL}
	request := SearchRequest{}
	_, err := client.FindUsers(request)
	if err != nil {
		t.Errorf("FindUsers error: should authorize with valid token")
	}
}

// Should handle empty search request.
func TestEmptyRequest(t *testing.T) {
	// Initialize test server instance
	testServer := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer testServer.Close()
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

// Should handle search request with exceeded limit.
func TestRequestWithExceededLimit(t *testing.T) {
	// Initialize test server instance
	testServer := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer testServer.Close()
	users := []User{
		{
			Id:   15,
			Name: "Allison Valdez",
			Age:  21,
			About: "Labore excepteur voluptate velit occaecat est nisi minim. " +
				"Laborum ea et irure nostrud enim sit incididunt reprehenderit id est nostrud eu. " +
				"Ullamco sint nisi voluptate cillum nostrud aliquip et minim. " +
				"Enim duis esse do aute qui officia ipsum ut occaecat deserunt. " +
				"Pariatur pariatur nisi do ad dolore reprehenderit et et enim esse dolor qui. " +
				"Excepteur ullamco adipisicing qui adipisicing tempor minim aliquip.\n",
			Gender: "male",
		},
	}
	expected := &SearchResponse{Users: users, NextPage: false}
	client := SearchClient{AccessToken: "test_token", URL: testServer.URL}
	request := SearchRequest{Offset: 34, Limit: 30, OrderBy: OrderByDesc}
	result, err := client.FindUsers(request)
	if err != nil {
		t.Errorf("FindUsers error: %v", err)
	} else if !reflect.DeepEqual(expected, result) {
		t.Errorf("Expected: %v\nActual: %v", expected, result)
	}
}

// Should handle search request with negative limit.
func TestRequestWithNegativeLimit(t *testing.T) {
	// Initialize test server instance
	testServer := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer testServer.Close()
	client := SearchClient{AccessToken: "test_token", URL: testServer.URL}
	request := SearchRequest{Limit: -1}
	_, err := client.FindUsers(request)
	if err == nil || err.Error() != "limit must be > 0" {
		t.Errorf("FindUsers unexpected error: %v", err)
	}
}

// Should handle search request with limit.
func TestRequestWithLimit(t *testing.T) {
	// Initialize test server instance
	testServer := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer testServer.Close()
	users := []User{
		{
			Id:   0,
			Name: "Boyd Wolf",
			Age:  22,
			About: "Nulla cillum enim voluptate consequat laborum esse excepteur occaecat commodo nostrud excepteur" +
				" ut cupidatat. Occaecat minim incididunt ut proident ad sint nostrud ad laborum sint pariatur. " +
				"Ut nulla commodo dolore officia. Consequat anim eiusmod amet commodo eiusmod deserunt culpa. " +
				"Ea sit dolore nostrud cillum proident nisi mollit est Lorem pariatur. " +
				"Lorem aute officia deserunt dolor nisi aliqua consequat nulla nostrud ipsum irure id deserunt " +
				"dolore. Minim reprehenderit nulla exercitation labore ipsum.\n",
			Gender: "male",
		},
	}
	expected := &SearchResponse{Users: users, NextPage: true}
	client := SearchClient{AccessToken: "test_token", URL: testServer.URL}
	request := SearchRequest{Limit: 1}
	result, err := client.FindUsers(request)
	if err != nil {
		t.Errorf("FindUsers error: %v", err)
	} else if !reflect.DeepEqual(expected, result) {
		t.Errorf("Expected: %v\nActual: %v", expected, result)
	}
}

// Should handle search request with negative offset.
func TestRequestWithNegativeOffset(t *testing.T) {
	// Initialize test server instance
	testServer := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer testServer.Close()
	client := SearchClient{AccessToken: "test_token", URL: testServer.URL}
	request := SearchRequest{Offset: -1}
	_, err := client.FindUsers(request)
	if err == nil || err.Error() != "offset must be > 0" {
		t.Errorf("FindUsers unexpected error: %v", err)
	}
}

// Should handle search request with offset.
func TestRequestWithOffset(t *testing.T) {
	// Initialize test server instance
	testServer := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer testServer.Close()
	users := []User{
		{
			Id:   10,
			Name: "Henderson Maxwell",
			Age:  30,
			About: "Ex et excepteur anim in eiusmod. " +
				"Cupidatat sunt aliquip exercitation velit minim aliqua ad ipsum cillum dolor do sit dolore cillum. " +
				"Exercitation eu in ex qui voluptate fugiat amet.\n",
			Gender: "male",
		},
	}
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
	users := []User{
		{
			Id:   15,
			Name: "Allison Valdez",
			Age:  21,
			About: "Labore excepteur voluptate velit occaecat est nisi minim. " +
				"Laborum ea et irure nostrud enim sit incididunt reprehenderit id est nostrud eu. " +
				"Ullamco sint nisi voluptate cillum nostrud aliquip et minim. " +
				"Enim duis esse do aute qui officia ipsum ut occaecat deserunt. " +
				"Pariatur pariatur nisi do ad dolore reprehenderit et et enim esse dolor qui. " +
				"Excepteur ullamco adipisicing qui adipisicing tempor minim aliquip.\n",
			Gender: "male",
		},
		{
			Id:   16,
			Name: "Annie Osborn",
			Age:  35,
			About: "Consequat fugiat veniam commodo nisi nostrud culpa pariatur. " +
				"Aliquip velit adipisicing dolor et nostrud. " +
				"Eu nostrud officia velit eiusmod ullamco duis eiusmod ad non do quis.\n",
			Gender: "female",
		},
		{
			Id:   19,
			Name: "Bell Bauer",
			Age:  26,
			About: "Nulla voluptate nostrud nostrud do ut tempor et quis non aliqua cillum in duis. " +
				"Sit ipsum sit ut non proident exercitation. " +
				"Quis consequat laboris deserunt adipisicing eiusmod non cillum magna.\n",
			Gender: "male",
		},
	}
	expected := &SearchResponse{Users: users, NextPage: true}
	client := SearchClient{AccessToken: "test_token", URL: testServer.URL}
	request := SearchRequest{Limit: 3, OrderBy: OrderByAsc}
	result, err := client.FindUsers(request)
	if err != nil {
		t.Errorf("FindUsers error: %v", err)
	} else if !reflect.DeepEqual(expected, result) {
		t.Errorf("Expected: %v\nActual: %v", expected, result)
	}
}

// Should handle search request with ordered by specific field data.
func TestRequestWithOrderField(t *testing.T) {
	// Initialize test server instance
	testServer := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer testServer.Close()
	users := []User{
		{
			Id:   32,
			Name: "Christy Knapp",
			Age:  40,
			About: "Incididunt culpa dolore laborum cupidatat consequat. " +
				"Aliquip cupidatat pariatur sit consectetur laboris labore anim labore. " +
				"Est sint ut ipsum dolor ipsum nisi tempor in tempor aliqua. " +
				"Aliquip labore cillum est consequat anim officia non reprehenderit ex duis elit. " +
				"Amet aliqua eu ad velit incididunt ad ut magna. Culpa dolore qui anim consequat commodo aute.\n",
			Gender: "female",
		},
		{
			Id:   13,
			Name: "Whitley Davidson",
			Age:  40,
			About: "Consectetur dolore anim veniam aliqua deserunt officia eu. " +
				"Et ullamco commodo ad officia duis ex incididunt proident consequat nostrud proident quis tempor. " +
				"Sunt magna ad excepteur eu sint aliqua eiusmod deserunt proident. " +
				"Do labore est dolore voluptate ullamco est dolore excepteur magna duis quis. " +
				"Quis laborum deserunt ipsum velit occaecat est laborum enim aute. " +
				"Officia dolore sit voluptate quis mollit veniam. " +
				"Laborum nisi ullamco nisi sit nulla cillum et id nisi.\n",
			Gender: "male",
		},
		{
			Id:   26,
			Name: "Sims Cotton",
			Age:  39,
			About: "Ex cupidatat est velit consequat ad. Tempor non cillum labore non voluptate. " +
				"Et proident culpa labore deserunt ut aliquip commodo laborum nostrud. " +
				"Anim minim occaecat est est minim.\n",
			Gender: "male",
		},
	}
	expected := &SearchResponse{Users: users, NextPage: true}
	client := SearchClient{AccessToken: "test_token", URL: testServer.URL}
	request := SearchRequest{Limit: 3, OrderBy: OrderByDesc, OrderField: "Age"}
	result, err := client.FindUsers(request)
	if err != nil {
		t.Errorf("FindUsers error: %v", err)
	} else if !reflect.DeepEqual(expected, result) {
		t.Errorf("Expected: %v\nActual: %v", expected, result)
	}
}

// Should handle search request with invalid field value.
func TestOrderFieldErr(t *testing.T) {
	// Initialize test server instance
	testServer := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer testServer.Close()
	client := SearchClient{AccessToken: "test_token", URL: testServer.URL}
	request := SearchRequest{OrderField: "invalid_field"}
	_, err := client.FindUsers(request)
	if err == nil || err.Error() != "OrderFeld invalid_field invalid" {
		t.Errorf("FindUsers error: %v", err)
	}
}

// Should handle search request timeout.
func TestRequestTimeout(t *testing.T) {
	// Initialize test server instance
	timeoutServer := func(res http.ResponseWriter, req *http.Request) {
		time.Sleep(2 * time.Second)
	}
	testServer := httptest.NewServer(http.HandlerFunc(timeoutServer))
	defer testServer.Close()
	client := SearchClient{AccessToken: "test_token", URL: testServer.URL}
	request := SearchRequest{}
	_, err := client.FindUsers(request)
	if err == nil || err.Error() != "timeout for limit=1&offset=0&order_by=0&order_field=&query=" {
		t.Errorf("FindUsers error: %v", err)
	}
}

// Should handle search request JSON parse error.
func TestRequestJsonError(t *testing.T) {
	// Initialize test server instance
	emptyServer := func(res http.ResponseWriter, req *http.Request) {}
	testServer := httptest.NewServer(http.HandlerFunc(emptyServer))
	defer testServer.Close()
	client := SearchClient{AccessToken: "test_token", URL: testServer.URL}
	request := SearchRequest{}
	_, err := client.FindUsers(request)
	if err == nil || err.Error() != "cant unpack result json: unexpected end of JSON input" {
		t.Errorf("FindUsers error: %v", err)
	}
}

// Should handle bad search request.
func TestBadRequest(t *testing.T) {
	// Initialize test server instance
	testServer := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer testServer.Close()
	client := SearchClient{AccessToken: "test_token", URL: testServer.URL}
	request := SearchRequest{OrderField: "invalid"}
	_, err := client.FindUsers(request)
	if err == nil {
		t.Errorf("FindUsers should return error")
	}
}

// Should handle internal server error.
func TestInternalServerErr(t *testing.T) {
	// Initialize test server instance
	testServer := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer testServer.Close()
	client := SearchClient{AccessToken: "test_token", URL: testServer.URL}
	request := SearchRequest{Offset: 100}
	_, err := client.FindUsers(request)
	if err == nil || err.Error() != "SearchServer fatal error" {
		t.Errorf("FindUsers error: %v", err)
	}
}
