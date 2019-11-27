package bot

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

type stringerType struct {
	message string
}

func (s stringerType) String() string {
	return s.message
}

type MockResponseWriter struct {
	header     http.Header
	response   []byte
	statusCode int
}

func (mock *MockResponseWriter) Header() http.Header {
	return mock.header
}

func (mock *MockResponseWriter) Write(bytes []byte) (int, error) {
	mock.response = bytes
	return len(bytes), nil
}

func (mock *MockResponseWriter) WriteHeader(statusCode int) {
	mock.statusCode = statusCode
}

func NewMockResponseWriter() *MockResponseWriter {
	return &MockResponseWriter{
		header:     http.Header(make(map[string][]string)),
		statusCode: http.StatusOK,
	}
}

func TestJSONMessage(t *testing.T) {
	message := "This is a test message"
	expected := []byte(fmt.Sprintf(`{"message":"%s"}`, message))
	actual := JSONMessage(message)

	if !bytes.Equal(expected, actual) {
		t.Fatalf("Expected %s, but found %s", expected, actual)
	}
}

func TestHTTPError(t *testing.T) {
	type testCase struct {
		Name     string
		Err      interface{}
		Code     int
		Expected []byte
	}
	tcString := "This is a test message"
	tcStringer := stringerType{"This is a stringer test message"}
	tcError := errors.New("This is an error test message")
	tcOther := struct {
		Val1 string
		Val2 int
	}{
		Val1: "This is a test field",
		Val2: 123,
	}

	testCases := []testCase{
		testCase{
			Name:     "String",
			Err:      tcString,
			Code:     http.StatusInternalServerError,
			Expected: JSONMessage(tcString),
		},
		testCase{
			Name:     "Stringer",
			Err:      tcStringer,
			Code:     http.StatusNotFound,
			Expected: JSONMessage(tcStringer.String()),
		},
		testCase{
			Name:     "Error",
			Err:      tcError,
			Code:     http.StatusForbidden,
			Expected: JSONMessage(tcError.Error()),
		},
		testCase{
			Name:     "Other",
			Err:      tcOther,
			Code:     http.StatusBadRequest,
			Expected: JSONMessage(fmt.Sprintf("%+v", tcOther)),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			w := NewMockResponseWriter()
			HTTPError(w, tc.Err, tc.Code)

			if !bytes.Equal(tc.Expected, w.response) {
				t.Errorf("Expected response %s, but found %s", tc.Expected, w.response)
			}
			if tc.Code != w.statusCode {
				t.Errorf("Expected status code %v, but found %v", tc.Code, w.statusCode)
			}
		})
	}
}

func TestAPIGetChannel(t *testing.T) {
	bot := NewMockBot()
	t.Run("InvalidChannel", func(t *testing.T) {
		channelName := "not a channel"
		req, _ := http.NewRequest(http.MethodGet, "", nil)
		req = mux.SetURLVars(req, map[string]string{
			"channel": channelName,
		})
		respWriter := NewMockResponseWriter()
		expectedStatus := http.StatusNotFound
		expectedWrite := JSONMessage(ChannelNotFoundError{channelName}.Error())
		bot.APIGetChannel(respWriter, req)

		fmt.Println(respWriter.response)

		if respWriter.statusCode != expectedStatus {
			t.Errorf("Expected status code %v, but found %v", expectedStatus, respWriter.statusCode)
		}

		if !bytes.Equal(respWriter.response, expectedWrite) {
			t.Errorf("Expected response %s, but found %s", expectedWrite, respWriter.response)
		}
	})

	t.Run("ValidChannel", func(t *testing.T) {
		channelName := connectedChannel.Name
		req, _ := http.NewRequest(http.MethodGet, "", nil)
		req = mux.SetURLVars(req, map[string]string{
			"channel": channelName,
		})
		respWriter := NewMockResponseWriter()
		expectedStatus := http.StatusOK
		expectedWrite, _ := json.Marshal(connectedChannel)
		bot.APIGetChannel(respWriter, req)

		if respWriter.statusCode != expectedStatus {
			t.Errorf("Expected status code %v, but found %v", expectedStatus, respWriter.statusCode)
		}

		if !bytes.Equal(respWriter.response, expectedWrite) {
			t.Errorf("Expected response %s, but found %s", expectedWrite, respWriter.response)
		}
	})
}

func TestAPIConnectToChannel(t *testing.T) {
	bot := NewMockBot()
	t.Run("NewChannel", func(t *testing.T) {
		channelName := "not a channel"
		req, _ := http.NewRequest(http.MethodPost, "", nil)
		req = mux.SetURLVars(req, map[string]string{
			"channel": channelName,
		})
		respWriter := NewMockResponseWriter()
		expectedStatus := http.StatusInternalServerError
		expectedWrite := JSONMessage(ChannelNotFoundError{channelName}.Error())
		wait := make(chan struct{})
		go func() {
			bot.APIConnectToChannel(respWriter, req)
			wait <- struct{}{}
		}()

		select {
		case <-bot.ChannelDB.(*mockChannelDB).addChan:
			t.Errorf("Unexpected add to channel database")
		case <-bot.TwitchClient.(*mockIRC).joinChan:
			t.Errorf("Unexpected join to channel")
		case <-time.After(3 * time.Second):
			t.Error("API connect call failed due to timeout")
		case <-wait:
			if respWriter.statusCode != expectedStatus {
				t.Errorf("Expected status code %v, but found %v", expectedStatus, respWriter.statusCode)
			}

			if !bytes.Equal(respWriter.response, expectedWrite) {
				t.Errorf("Expected response %s, but found %s", expectedWrite, respWriter.response)
			}
		}
	})

	t.Run("ValidChannel", func(t *testing.T) {
		channelName := connectedChannel.Name
		req, _ := http.NewRequest(http.MethodPost, "", nil)
		req = mux.SetURLVars(req, map[string]string{
			"channel": channelName,
		})
		respWriter := NewMockResponseWriter()
		expectedStatus := http.StatusOK
		expectedWrite := []byte{}

		wait := make(chan struct{})
		go func() {
			bot.APIConnectToChannel(respWriter, req)
			wait <- struct{}{}
		}()

		select {
		case updated := <-bot.ChannelDB.(*mockChannelDB).updateChan:
			if updated != channelName {
				t.Errorf("Expected to update %s, but updated %s", channelName, updated)
			}
		case <-time.After(3 * time.Second):
			t.Error("API connect call failed due to timeout")
		}

		select {
		case joined := <-bot.TwitchClient.(*mockIRC).joinChan:
			if joined != channelName {
				t.Errorf("Expected to join %s, but joined %s", channelName, joined)
			}
		case <-time.After(3 * time.Second):
			t.Error("API connect call failed due to timeout")
		}

		select {
		case <-time.After(3 * time.Second):
			t.Error("API connect call failed due to timeout")
		case <-wait:
			if respWriter.statusCode != expectedStatus {
				t.Errorf("Expected status code %v, but found %v", expectedStatus, respWriter.statusCode)
			}

			if !bytes.Equal(respWriter.response, expectedWrite) {
				t.Errorf("Expected response %s, but found %s", expectedWrite, respWriter.response)
			}
		}
	})
}

func TestAPIDisconnectFromChannel(t *testing.T) {
	bot := NewMockBot()
	t.Run("InvalidChannel", func(t *testing.T) {
		channelName := "not a channel"
		req, _ := http.NewRequest(http.MethodDelete, "", nil)
		req = mux.SetURLVars(req, map[string]string{
			"channel": channelName,
		})
		respWriter := NewMockResponseWriter()
		expectedStatus := http.StatusInternalServerError
		expectedWrite := JSONMessage(ChannelNotFoundError{channelName}.Error())
		wait := make(chan struct{})
		go func() {
			bot.APIDisconnectFromChannel(respWriter, req)
			wait <- struct{}{}
		}()

		select {
		case <-bot.ChannelDB.(*mockChannelDB).addChan:
			t.Error("Unexpected add to channel database")
		case <-bot.ChannelDB.(*mockChannelDB).updateChan:
			t.Error("Unexpected update to channel database")
		case <-time.After(3 * time.Second):
			t.Error("API disconnect call failed due to timeout")
		case <-wait:
		}

		if respWriter.statusCode != expectedStatus {
			t.Errorf("Expected status code %v, but found %v", expectedStatus, respWriter.statusCode)
		}

		if !bytes.Equal(respWriter.response, expectedWrite) {
			t.Errorf("Expected response %s, but found %s", expectedWrite, respWriter.response)
		}
	})

	t.Run("ValidChannel", func(t *testing.T) {
		channelName := connectedChannel.Name
		req, _ := http.NewRequest(http.MethodDelete, "", nil)
		req = mux.SetURLVars(req, map[string]string{
			"channel": channelName,
		})
		respWriter := NewMockResponseWriter()
		expectedStatus := http.StatusOK
		expectedWrite := []byte{}

		wait := make(chan struct{})
		go func() {
			bot.APIDisconnectFromChannel(respWriter, req)
			wait <- struct{}{}
		}()

		select {
		case updated := <-bot.ChannelDB.(*mockChannelDB).updateChan:
			if updated != channelName {
				t.Errorf("Expected to update %s, but updated %s", channelName, updated)
			}
		case <-time.After(3 * time.Second):
			t.Error("API disconnect call failed due to timeout")
		}

		select {
		case departed := <-bot.TwitchClient.(*mockIRC).departChan:
			if departed != channelName {
				t.Errorf("Expected to join %s, but joined %s", channelName, departed)
			}
		case <-time.After(3 * time.Second):
			t.Error("API disconnect call failed due to timeout")
		}

		select {
		case <-time.After(3 * time.Second):
			t.Error("API disconnect call failed due to timeout")
		case <-wait:
			if respWriter.statusCode != expectedStatus {
				t.Errorf("Expected status code %v, but found %v", expectedStatus, respWriter.statusCode)
			}

			if !bytes.Equal(respWriter.response, expectedWrite) {
				t.Errorf("Expected response %s, but found %s", expectedWrite, respWriter.response)
			}
		}
	})
}

func TestHeartbeat(t *testing.T) {
	respWriter := NewMockResponseWriter()
	req, _ := http.NewRequest(http.MethodGet, "", nil)

	Heartbeat(respWriter, req)

	if !bytes.Equal(respWriter.response, []byte("ok")) {
		t.Fatalf("Expected response ok, but found %s", respWriter.response)
	}

	if respWriter.statusCode != http.StatusOK {
		t.Fatalf("Expected status code %v, but found %v", http.StatusOK, respWriter.statusCode)
	}
}
