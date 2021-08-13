package main

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/gofiber/storage/memory"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestCookieManagerNewSession(t *testing.T) {
	cm := NewCookieManager()
	require.Nil(t, cm.Get("session1"))

	sess1 := cm.NewSession("session1")
	require.NotNil(t, sess1)

	u, err := url.Parse("https://google.com")
	require.NoError(t, err)
	expectedCookies := []*http.Cookie{{Name: "abc", Value: "def"}}
	sess1.SetCookies(u, expectedCookies)

	sess1new := cm.Get("session1")
	require.True(t, sess1 == sess1new, "Cookie Jars are not the same")
	cookies := sess1new.Cookies(u)
	require.Equal(t, expectedCookies, cookies)

	sess2 := cm.NewSession("session2")
	require.NotNil(t, sess2)
	require.True(t, sess2 != sess1, "Cookie Jars are the same")
}

func TestCookieManagerDelete(t *testing.T) {
	cm := NewCookieManager()
	sess1 := cm.NewSession("session1")

	u, err := url.Parse("https://google.com")
	require.NoError(t, err)
	cookies := []*http.Cookie{{Name: "abc", Value: "def"}}
	sess1.SetCookies(u, cookies)

	cm.Delete("session1")
	require.Nil(t, cm.Get("session1"))

	sess1new := cm.NewSession("session1")
	require.NotNil(t, sess1new)
	require.NotEqual(t, sess1new, sess1, "Cookie Jars are the same")
}

func TestNewSessionStorage(t *testing.T) {
	ctrl := gomock.NewController(t)
	cm := NewMockCookieManager(ctrl)
	s := NewSessionStorage(memory.New(), cm)

	data := []byte("def")
	err := s.Set("abc", data, 0)
	require.NoError(t, err)

	v, err := s.Get("abc")
	require.NoError(t, err)
	require.Equal(t, data, v)

	cm.EXPECT().Delete("abc")

	err = s.Delete("abc")
	require.NoError(t, err)
}
