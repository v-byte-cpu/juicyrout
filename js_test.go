package main

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/require"
)

func TestJavaScriptToProxy(t *testing.T) {
	vm := goja.New()
	_, err := vm.RunString(changeURLScript)
	require.NoError(t, err)

	var toProxy func(string) string
	err = vm.ExportTo(vm.Get("toProxy"), &toProxy)
	require.NoError(t, err)

	require.Equal(t, "google-com.host.juicyrout:8091", toProxy("google.com"))
	require.Equal(t, "google-com.host.juicyrout:8091", toProxy("google-com.host.juicyrout:8091"))
	require.Equal(t, "static--content-google-com.host.juicyrout:8091", toProxy("static-content.google.com"))
}
