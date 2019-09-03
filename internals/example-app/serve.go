package example_app

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
)

type ServeCommand struct {
	io ui.IO

	host string
	port int

	appUsername string
	appPassword string
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *ServeCommand) Register(r command.Registerer) {
	clause := r.Command("serve", "Runs the secrethub example by serving a web page.")

	clause.Flag("host", "The host to serve the webpage on").Short('h').Default("127.0.0.1").StringVar(&cmd.host)
	clause.Flag("port", "The port to serve the webpage on").Default("8080").IntVar(&cmd.port)

	command.BindAction(clause, cmd.Run)
}

func NewServeCommand(io ui.IO) *ServeCommand {
	username := os.Getenv("APP_USERNAME")
	password := os.Getenv("APP_PASSWORD")

	return &ServeCommand{
		io:          io,
		appUsername: username,
		appPassword: password,
	}
}

// Run handles the command with the options as specified in the command.
func (cmd *ServeCommand) Run() error {
	http.HandleFunc("/", cmd.ServeIndex)
	http.HandleFunc("/api", cmd.ServeAPI)

	addr := fmt.Sprintf("%s:%d", cmd.host, cmd.port)
	fmt.Fprintf(cmd.io.Stdout(), "Serving example app on http://%s\n", addr)
	return http.ListenAndServe(addr, nil)
}

func (cmd *ServeCommand) ServeIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("test").Parse(page)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(w, cmd.port)
	if err != nil {
		panic(err)
	}
}

func (cmd *ServeCommand) ServeAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Origin", "*")

	if cmd.appUsername == "" {
		writeError(w, http.StatusInternalServerError, "APP_USERNAME not set")
		return
	}

	if cmd.appPassword == "" {
		writeError(w, http.StatusInternalServerError, "APP_PASSWORD not set")
		return
	}

	req, err := http.NewRequest("GET", "http://httpbin.org/basic-auth/demo/very-secret-password", nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	req.SetBasicAuth(cmd.appUsername, cmd.appPassword)
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
