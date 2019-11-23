package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

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

func TestHTTPMessage(t *testing.T) {
	message := "This is a test message"
	expected := []byte(fmt.Sprintf(`{"message":"%s"}`, message))
	actual := HTTPMessage(message)

	if !bytes.Equal(expected, actual) {
		t.Fatalf("Expected %s, but found %s", expected, actual)
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
		expectedWrite := HTTPMessage(ChannelNotFoundError{channelName}.Error())
		bot.APIGetChannel(respWriter, req)

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
		expectedStatus := http.StatusOK
		expectedWrite := []byte{}
		wait := make(chan struct{})
		go func() {
			bot.APIConnectToChannel(respWriter, req)
			wait <- struct{}{}
		}()

		select {
		case added := <-bot.ChannelDB.(*mockChannelDB).addChan:
			if added.Name != channelName {
				t.Errorf("Expected to add %s, but added %s", channelName, added.Name)
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
		expectedWrite := HTTPMessage(ChannelNotFoundError{channelName}.Error())
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