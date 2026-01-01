package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/neilotoole/slogt"
	"github.com/stretchr/testify/require"
)

func TestCLI_Run(t *testing.T) {
	t.Skip("skipping while we're building out scaffolding")
	tests := []struct {
		name string
		data string
	}{
		{
			name: "one line",
			data: `|updateuser| Guest 60|0|1|{"blockChallenges":false,"blockPMs":false,"ignoreTickets":false,"hideBattlesFromTrainerCard":false,"blockInvites":false,"doNotDisturb":false,"blockFriendRequests":false,"allowFriendNotifications":false,"displayBattlesToFriends":false,"hideLogins":false,"hiddenNextBattle":false,"inviteOnlyNextBattle":false,"language":null}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(websocketTester(t, tt.data))
			defer ts.Close()
			c := &CLI{
				Address: ts.URL,
				Timeout: time.Second,
				Logger:  slogt.New(t),
			}
			require.NoError(t, c.Run(t.Context()))
		})
	}
}

func TestCLI_Login(t *testing.T) {
	doneCh := make(chan struct{})
	helper := &loginHelper{
		doneCh:  doneCh,
		loginCh: make(chan loginInfo),
	}
	ls := httptest.NewServer(helper.loginServer(t))
	t.Cleanup(ls.Close)
	ws := httptest.NewServer(helper.websocketLogin(t))
	t.Cleanup(ws.Close)

	// Use an io pipe to simulate stdin and stdout. Avoids early EOFs closing the readers early
	stdin, stdinWriter := io.Pipe()
	t.Cleanup(func() {
		if err := stdinWriter.Close(); err != nil {
			t.Log("error closing stdinWriter", err)
		}
	})
	_, stdoutWriter := io.Pipe()
	t.Cleanup(func() {
		if err := stdoutWriter.Close(); err != nil {
			t.Log("error closing stdoutWriter", err)
		}
	})
	c := &CLI{
		Address:       ws.URL,
		LoginEndpoint: ls.URL,
		Timeout:       time.Second,
		Logger:        slogt.New(t, slogt.JSON()),
		Stdin:         stdin,
		Stdout:        stdoutWriter,
	}
	go func() {
		// We don't care about the error itself as long as we've logged in successfully
		c.Run(t.Context())
	}()
	select {
	case <-doneCh:
	case <-time.After(time.Second):
		require.FailNow(t, "timed out waiting for login")
	}
}

func websocketTester(t *testing.T, data string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		require.NoError(t, err)
		require.NoError(t, c.Write(t.Context(), websocket.MessageText, []byte(data)))
	}
}

type loginInfo struct {
	username  string
	assertion string
}

type loginHelper struct {
	doneCh  chan<- struct{}
	loginCh chan loginInfo
}

func (h *loginHelper) websocketLogin(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		require.NoError(t, err)

		// Send challstr message
		const challstrMsg = `|challstr|4|a43ed9f8730defb287c1b04d91dea59ebfc8e33d22dc2d044cc4cbb4a0e39b8bb7d158a5d12414adf1025afe5f8bd08f0dda9d0bd963c296d1c473f7bf68b2dfcb5f274347dda02eced31c27153f25ad16f645804922d51314d2be5c7ebc444c605ff76902d4d75cba8fcca4a7137e98841c78d8e14f3dfdbadffd99364a195d`
		require.NoError(t, c.Write(t.Context(), websocket.MessageText, []byte(challstrMsg)))

		var info loginInfo
		select {
		case info = <-h.loginCh:
		case <-t.Context().Done():
			require.FailNow(t, "context done before websocket login")
		}

		// Make sure we get a /trn command back
		msgType, msg, err := c.Read(t.Context())
		require.NoError(t, err)
		require.Equal(t, websocket.MessageText, msgType)
		expected := fmt.Sprintf(`|/trn %s,0,%s`, info.username, info.assertion)
		require.Equal(t, expected, string(msg))
		close(h.doneCh)
	}
}

// loginServer returns the same `assertion` described in Showdown's challstr protocol documentation:
// https://github.com/smogon/pokemon-showdown/blob/master/PROTOCOL.md
func (h *loginHelper) loginServer(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Basic request validation
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var input loginInput
		require.NoError(t, json.Unmarshal(body, &input))
		require.NotEmpty(t, input.Name)
		require.NotEmpty(t, input.Pass)
		require.NotEmpty(t, input.Challstr)

		userId := strings.ReplaceAll(input.Name, "-", "")
		info := loginInfo{
			username: input.Name,
			assertion: fmt.Sprintf(
				`cfb50d5ac95c38a8cf3a1167e19013f437cf1c7025cf13d3edf694af4993154d8bd63ac1953c52e1f746d88344ec895e6d903e7fa6ce220ebdb9ccad26285f8b28f19fd03a0ad62685cbaa0a2fde9e80f2ae91ed62ceeaaea173bcc244d7e6a6c9bada1fe527ce2886e86927fd1448722e214696255e9e01f78bd8e028fae26d,%s,2,1766374653,sim3.psim.us,238568d704c1d3b21766374653,fd2349285c78743f,;7bd7c1f239641a861230f424270ec6b79f8e81780d20bbb493e8a3ed6df2fcca56d4e4f88e09b7e83460bf4a7741daca7d64155e4b691048ad3f3e06cf6eb67320a9b81aae5ba78bd495f9b062275378600a6917b6fdcd6d4e8554971ce3eea4b4e2474406d4dd3b1ec1fd367a3fa80d6c6dbdae6edcc5ffeef753252e5891032e3c2ac923bdceddbff829401360d662633d29a44db023d3c7ddb5c5e6f6ce6a0d9981f0db4b089d1ebc8e8bb2d71f05e45b39ebdd38f02b6ab8f29e6ba39a119f186acab4f461f29867e65ecc2f6324d9b64b5a45bc7cbe7459f7ed8e9389ac8bf3e6f050f42b8bfb5fca9dbeb7dd5894c8b82f94cb21d4dd700175288c7b28ec050b1d401c8547a2d578a117e0a1a39e14a3029d516685aeb20cee0df4f0b4d774973a532144dc768c934337c2b11645e4d254ef879457ad9f14076b1ee0ca3056fd7ead3776fbacbb6e6c9da6d0d57e8d1c48b857ab096a759870328c0a189e487e4a262904d5d14f1501ea9840372cb3f0b9a806858ff6321fd38c27050233a47854ff3df7c8680823f64cf3686e85a1b4bebfcdc7d06a7831d99b6285f23636f9f94297ecc12c0749b187dff2e8613f7e9ee3275b7cb79dfaa21490caa0adf6505e0d451a4668a00404523025bcd4e858e7475bf726c490e63bd0a5413bc57872dd7e684ad24e1257f299dcd36add6a8e482d58c95ec3212431f18ffac7`,
				userId,
			),
		}
		// Send the username and assertion to the websocket helper so we know what to expect
		select {
		case h.loginCh <- info:
		case <-t.Context().Done():
			require.FailNow(t, "context done before login sent")
		}

		// Send successful response back to the client
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(fmt.Sprintf(
			`]{"actionsuccess":true,"assertion":"%s","curuser":{"loggedin":true,"username":"%s","userid":"%s"}}`,
			info.assertion,
			input.Name,
			userId,
		)),
		)
		require.NoError(t, err)
	}
}
