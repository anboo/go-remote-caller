package main

import (
	"github.com/gorilla/mux"
	"github.com/twinj/uuid"
	"net/http/pprof"
	"net/http"
	"log"
	"encoding/json"
	"net/url"
	"bytes"
	"strconv"
	"fmt"
	"io/ioutil"
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
	Response Response `json:"response"`
	Status   int    `json:"status"`
}

type ResponseHttpRequestList []HttpRequest
type ResponseDelayedHttpRequestList map[string]*DelayedHttpRequest

func (this *ResponseDelayedHttpRequestList) loadResponse(guid string, resp []byte) {
	//current := &this[guid]

	for _, current := range *this {
		if current.GUID == guid {
			current.Response = Response{
				Body: string(resp),
			}
		}
	}
}

var delayedReqs = ResponseDelayedHttpRequestList{}

func main() {
	r := mux.NewRouter().StrictSlash(true)

	r.HandleFunc("/debug/pprof/", pprof.Index)
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/debug/pprof/trace", pprof.Trace)

	r.NewRoute().PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index)

	r.HandleFunc("/v1/request", IndexAction)
	r.HandleFunc("/v1/request/handle", HandleAction)
	r.HandleFunc("/v1/request/register", RegisterAction)
	r.HandleFunc("/v1/request/{guid}/response", ResponseAction)

	//go func() {
	log.Println(http.ListenAndServe(":8080", r))
}

func IndexAction(w http.ResponseWriter, r *http.Request) {
	var res = map[string]interface{} {"count": len(delayedReqs), "items": delayedReqs}

	json.NewEncoder(w).Encode(res)
	r.Body.Close()
}

func HandleAction(w http.ResponseWriter, r *http.Request) {
	newReq := HttpRequest{}

	err := json.NewDecoder(r.Body).Decode(&newReq); if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Print(err)
		return
	}

	res, err := handleRequest(newReq.Method, newReq.Uri, newReq.Params, newReq.Headers)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Print(err)
		return
	}

	w.Write(res)
	r.Body.Close()
}

func RegisterAction(w http.ResponseWriter, r *http.Request) {
	newReq := DelayedHttpRequest{}
	newReq.GUID = uuid.NewV4().String()

	err := json.NewDecoder(r.Body).Decode(&newReq); if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Print(err)
		return
	}

	delayedReqs[newReq.GUID] = &newReq

	go func (newRequest DelayedHttpRequest, delayedReq *ResponseDelayedHttpRequestList) {
		resp, err := handleRequest(newRequest.Uri, newRequest.Method, newRequest.Params, newRequest.Headers)

		if err != nil {}

		delayedReq.loadResponse(newRequest.GUID, resp)
	}(newReq, &delayedReqs)

	json.NewEncoder(w).Encode(newReq)
	r.Body.Close()
}

func ResponseAction (w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	guid := vars["guid"]

	for _, request := range delayedReqs {
		if request.GUID == guid {
			json.NewEncoder(w).Encode(request.Response)
			return
		}
	}

	http.Error(w, "Request not found", 404)
}

func handleRequest(apiUrl string, method string, params []HttpParameter, headers []HttpHeader) ([]byte, error) {
	data := url.Values{}

	for _, param := range params {
		data.Set(param.Param, param.Value)
	}

	u, _ := url.ParseRequestURI(apiUrl)
	url := u.String()

	cl := &http.Client{}
	r, err := http.NewRequest(method, url, bytes.NewBufferString(data.Encode()))

	if err != nil {
		return []byte{}, err
	}

	for _, header := range headers {
		r.Header.Add(header.Header, header.Value)
	}

	r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	resp, _ := cl.Do(r)
	fmt.Println(resp.Status)

	defer resp.Body.Close()

	res, err := ioutil.ReadAll(resp.Body); if err != nil {
		return []byte{}, err
	}

	return res, nil
}