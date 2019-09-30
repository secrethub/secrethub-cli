package app

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
)

type Server struct {
	host string
	port int

	appUsername string
	appPassword string
}

func NewServer(host string, port int) *Server {
	username := os.Getenv("APP_USERNAME")
	password := os.Getenv("APP_PASSWORD")

	return &Server{
		host:        host,
		port:        port,
		appUsername: username,
		appPassword: password,
	}
}

func (s *Server) Serve() error {
	http.HandleFunc("/", s.ServeIndex)
	http.HandleFunc("/api", s.ServeAPI)

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	return http.ListenAndServe(addr, nil)
}

func (s *Server) ServeIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("test").Parse(page)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(w, s.port)
	if err != nil {
		panic(err)
	}
}

func (s *Server) ServeAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Origin", "*")

	if s.appUsername == "" {
		writeError(w, http.StatusInternalServerError, "APP_USERNAME not set")
		return
	}

	if s.appPassword == "" {
		writeError(w, http.StatusInternalServerError, "APP_PASSWORD not set")
		return
	}

	req, err := http.NewRequest("GET", "http://httpbin.org/basic-auth/demo/very-secret-password", nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	req.SetBasicAuth(s.appUsername, s.appPassword)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(resp.StatusCode)
	if resp.StatusCode == http.StatusUnauthorized {
		fmt.Fprint(w, "unauthorized")
	} else {
		io.Copy(w, resp.Body)
	}
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	fmt.Fprint(w, message)
}
