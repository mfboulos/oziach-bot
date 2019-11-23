package bot

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// HTTPMessage Converts a string message into a byte array JSON string with key "message"
func HTTPMessage(m string) []byte {
	json, _ := json.Marshal(struct {
		Message string `json:"message"`
	}{Message: m})
	return json
}

// APIGetChannel Endpoint handler function to route to GetChannel
func (bot *OziachBot) APIGetChannel(w http.ResponseWriter, r *http.Request) {
	pathParams := mux.Vars(r)
	w.Header().Set("Content-Type", "application/json")

	if name, ok := pathParams["channel"]; ok {
		channel, err := bot.ChannelDB.GetChannel(name)

		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			w.Write(HTTPMessage(err.Error()))
		} else {
			json, err := json.Marshal(channel)

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write(HTTPMessage(err.Error()))
			} else {
				w.Write(json)
			}
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(HTTPMessage("Bad request format: /channel/{channel} required"))
	}
}

// APIConnectToChannel Endpoint handler function to route to ConnectToChannel
func (bot *OziachBot) APIConnectToChannel(w http.ResponseWriter, r *http.Request) {
	pathParams := mux.Vars(r)
	w.Header().Set("Content-Type", "application/json")

	if name, ok := pathParams["channel"]; ok {
		err := bot.ConnectToChannel(name)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(HTTPMessage(err.Error()))
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(HTTPMessage("Bad request format: /channel/{channel} required"))
	}
}

// APIDisconnectFromChannel Endpoint handler function to route to DisconnectFromChannel
func (bot *OziachBot) APIDisconnectFromChannel(w http.ResponseWriter, r *http.Request) {
	pathParams := mux.Vars(r)
	w.Header().Set("Content-Type", "application/json")

	if name, ok := pathParams["channel"]; ok {
		err := bot.DisconnectFromChannel(name)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(HTTPMessage(err.Error()))
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(HTTPMessage("Bad request format: /channel/{channel} required"))
	}
}

// Heartbeat Returns "ok" to validate the health of the application
func Heartbeat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// ServeAPI Serves OziachBot's API
func (bot *OziachBot) ServeAPI() {
	router := mux.NewRouter()
	// Health check for load balancer
	router.HandleFunc("/", Heartbeat).Methods(http.MethodGet, http.MethodHead)

	channelAPI := router.PathPrefix("/oziachbot").Subrouter()

	// Configure all endpoints in the channel API
	channelAPI.HandleFunc("/channel/{channel}", bot.APIGetChannel).Methods(http.MethodGet)
	channelAPI.HandleFunc("/channel/{channel}", bot.APIConnectToChannel).Methods(http.MethodPost)
	channelAPI.HandleFunc("/channel/{channel}", bot.APIDisconnectFromChannel).Methods(http.MethodDelete)

	log.Fatal(http.ListenAndServe(":7373", router))
}
