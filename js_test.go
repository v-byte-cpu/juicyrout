package main

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/require"
)

func TestJavaScriptToProxy(t *testing.T) {
	vm := goja.New()
	// mock browser objects
	_, err := vm.RunString(`
	var window = {}
	window.XMLHttpRequest = {}
	window.XMLHttpRequest.prototype = {}
	var Node = {}
	Node.prototype = {}
	var URL = function() {
		return {}
	}
	`)
	require.NoError(t, err)

	_, err = vm.RunString(jsHookScript)
	require.NoError(t, err)

	var toProxy func(string) string
	err = vm.ExportTo(vm.Get("toProxy"), &toProxy)
	require.NoError(t, err)

	require.Equal(t, "google-com.host.juicyrout:8091", toProxy("google.com"))
	require.Equal(t, "google-com.host.juicyrout:8091", toProxy("google-com.host.juicyrout:8091"))
	require.Equal(t, "static--content-google-com.host.juicyrout:8091", toProxy("static-content.google.com"))
}
