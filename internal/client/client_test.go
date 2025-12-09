package client

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/require"
)

func TestCLI_Run(t *testing.T) {
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
			}
			require.NoError(t, c.Run())
		})
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
