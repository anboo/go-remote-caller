package main

import (
	"github.com/gorilla/mux"
	"github.com/twinj/uuid"
	"net/http"
	"log"
	"encoding/json"
	"net/url"
	"bytes"
	"strconv"
	"fmt"
)

type HttpParameter struct {
	Param string	`json:"param"`
	Value string	`json:"value"`
}

type HttpHeader struct {
	Header string	`json:"header"`
	Value  string	`json:"value"`
}

type HttpRequest struct {
	Method  string	`json:"method"`
	Uri     string	`json:"uri"`
	Params  [] HttpParameter `json:"params"`
	Body    string `json:"body"`
	Headers [] HttpHeader `json:"headers"`
}

type Response struct {
	Headers [] HttpHeader `json:"headers"`
	Body	string `json:"body"`
}

type DelayedHttpRequest struct {
	GUID     string  `json:"guid"`
	Method   string	`json:"method"`
	Uri      string	`json:"uri"`
	Body     string `json:"body"`
	Params   [] HttpParameter `json:"params"`
	Headers  [] HttpHeader `json:"headers"`
	Response Response
	TTL 	 int64	`json:"ttl"`
	Status   int    `json:"status"`
}

type ResponseHttpRequestList []HttpRequest
type ResponseDelayedHttpRequestList map[string]*DelayedHttpRequest

func (this *ResponseDelayedHttpRequestList) loadResponse(guid string, resp string) {
	//current := &this[guid]

	for _, current := range *this {
		if current.GUID == guid {
			current.Response = Response{
				Body: resp,
			}
		}
	}
}

var requests = ResponseHttpRequestList{}
var delayedRequests = ResponseDelayedHttpRequestList{}

func main() {
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/v1/request", IndexAction)
	router.HandleFunc("/v1/request/handle", HandleAction)
	router.HandleFunc("/v1/request/register", RegisterAction)
	router.HandleFunc("/v1/request/{guid}/response", ResponseAction)

	//go func() {
	log.Println(http.ListenAndServe(":8080", router))
}

func IndexAction(w http.ResponseWriter, r *http.Request) {
	var response = map[string]interface{} {"count": len(delayedRequests), "items": delayedRequests}

	json.NewEncoder(w).Encode(response)
}

func HandleAction(w http.ResponseWriter, r *http.Request) {
	newRequest := HttpRequest{}
	json.NewDecoder(r.Body).Decode(&newRequest)

	response, err := handleRequest(newRequest.Method, newRequest.Uri, newRequest.Params, newRequest.Headers)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Write(bytes.NewBufferString(response).Bytes())
}

func RegisterAction(w http.ResponseWriter, r *http.Request) {
	newRequest := DelayedHttpRequest{}
	newRequest.GUID = uuid.NewV4().String()

	json.NewDecoder(r.Body).Decode(&newRequest)

	delayedRequests[newRequest.GUID] = &newRequest

	go func (newRequest DelayedHttpRequest, delayedRequests *ResponseDelayedHttpRequestList) {
		resp, err := handleRequest(newRequest.Uri, newRequest.Method, newRequest.Params, newRequest.Headers)

		if err != nil {}

		delayedRequests.loadResponse(newRequest.GUID, resp)
	}(newRequest, &delayedRequests)

	json.NewEncoder(w).Encode(newRequest)
}

func ResponseAction (w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	guid := vars["guid"]

	for _, request := range delayedRequests {
		if request.GUID == guid {
			json.NewEncoder(w).Encode(request.Response)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
}

func handleRequest(apiUrl string, method string, params []HttpParameter, headers []HttpHeader) (string, error) {
	data := url.Values{}

	for _, param := range params {
		data.Set(param.Param, param.Value)
	}

	u, _ := url.ParseRequestURI(apiUrl)
	urlStr := u.String()

	client := &http.Client{}
	r, err := http.NewRequest(method, urlStr, bytes.NewBufferString(data.Encode()))

	if err != nil {
		return "", err
	}

	for _, header := range headers {
		r.Header.Add(header.Header, header.Value)
	}

	r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	resp, _ := client.Do(r)
	fmt.Println(resp.Status)

	buf := bytes.Buffer{}
	buf.ReadFrom(resp.Body)

	return buf.String(), nil
}