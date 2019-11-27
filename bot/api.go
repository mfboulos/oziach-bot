package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// HTTPError Converts err into a JSON byte array with key "message",
// then writes the marshalled JSON into the http.ResponseWriter, along with setting
// the error code given
//
// This is similar to http.Error, with 2 key differences:
//
// * HTTPError writes JSON instead of a string. Because of this, it sets the
// Content-Type of the response to application/json
//
// * HTTPError accepts an interface{} err instead of a string. This allows
// error and fmt.Stringer types to be accepted along with strings. If err
// is not a string or either of the types mentioned above, it is output
// as the format specifier `%+v`
func HTTPError(w http.ResponseWriter, err interface{}, code int) error {
	var message string
	switch err.(type) {
	case string:
		message = err.(string)
	case fmt.Stringer:
		message = err.(fmt.Stringer).String()
	case error:
		message = err.(error).Error()
	default:
		message = fmt.Sprintf("%+v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(JSONMessage(message))
	return nil
}

// JSONMessage Converts err into a JSON byte array with key "message"
func JSONMessage(message string) []byte {
	json, _ := json.Marshal(struct {
		Message string `json:"message"`
	}{Message: message})
	return json
}

// APIAddChannel Endpoint handler function to route to AddChannel
func (bot *OziachBot) APIAddChannel(w http.ResponseWriter, r *http.Request) {
	pathParams := mux.Vars(r)
	w.Header().Set("Content-Type", "application/json")

	if name, ok := pathParams["channel"]; ok {
		channel, err := bot.ChannelDB.AddChannel(name)

		if err != nil {
			HTTPError(w, err, http.StatusNotFound)
		} else {
			json, err := json.Marshal(channel)

			if err != nil {
				HTTPError(w, err, http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusCreated)
				w.Write(json)
			}
		}
	} else {
		HTTPError(w, "Bad request format: /channel/{channel} required", http.StatusBadRequest)
	}
}

// APIGetChannel Endpoint handler function to route to GetChannel
func (bot *OziachBot) APIGetChannel(w http.ResponseWriter, r *http.Request) {
	pathParams := mux.Vars(r)
	w.Header().Set("Content-Type", "application/json")

	if name, ok := pathParams["channel"]; ok {
		channel, err := bot.ChannelDB.GetChannel(name)

		if err != nil {
			HTTPError(w, err, http.StatusNotFound)
		} else {
			json, err := json.Marshal(channel)

			if err != nil {
				HTTPError(w, err, http.StatusInternalServerError)
			} else {
				w.Write(json)
			}
		}
	} else {
		HTTPError(w, "Bad request format: /channel/{channel} required", http.StatusBadRequest)
	}
}

// APIConnectToChannel Endpoint handler function to route to ConnectToChannel
func (bot *OziachBot) APIConnectToChannel(w http.ResponseWriter, r *http.Request) {
	pathParams := mux.Vars(r)
	w.Header().Set("Content-Type", "application/json")

	if name, ok := pathParams["channel"]; ok {
		err := bot.ConnectToChannel(name)

		if err != nil {
			code := http.StatusInternalServerError
			if _, ok := err.(ChannelNotFoundError); ok {
				code = http.StatusNotFound
			}
			HTTPError(w, err, code)
		}
	} else {
		HTTPError(w, "Bad request format: /channel/{channel} required", http.StatusBadRequest)
	}
}

// APIDisconnectFromChannel Endpoint handler function to route to DisconnectFromChannel
func (bot *OziachBot) APIDisconnectFromChannel(w http.ResponseWriter, r *http.Request) {
	pathParams := mux.Vars(r)
	w.Header().Set("Content-Type", "application/json")

	if name, ok := pathParams["channel"]; ok {
		err := bot.DisconnectFromChannel(name)

		if err != nil {
			code := http.StatusInternalServerError
			if _, ok := err.(ChannelNotFoundError); ok {
				code = http.StatusNotFound
			}
			HTTPError(w, err, code)
		}
	} else {
		HTTPError(w, "Bad request format: /channel/{channel} required", http.StatusBadRequest)
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

	obRouter := router.PathPrefix("/oziachbot").Subrouter()
	channelAPI := obRouter.PathPrefix("/channel").Subrouter()
	connectAPI := obRouter.PathPrefix("/connect").Subrouter()

	// Configure all endpoints in the channel API
	channelAPI.HandleFunc("/{channel}", bot.APIGetChannel).Methods(http.MethodGet)
	channelAPI.HandleFunc("/{channel}", bot.APIAddChannel).Methods(http.MethodPost)
	connectAPI.HandleFunc("/{channel}", bot.APIConnectToChannel).Methods(http.MethodPost)
	connectAPI.HandleFunc("/{channel}", bot.APIDisconnectFromChannel).Methods(http.MethodDelete)

	log.Fatal(http.ListenAndServe(":7373", router))
}
