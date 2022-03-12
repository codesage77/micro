package server

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChiImplementsHandlerInterface(t *testing.T) {
	var sh interface{} = NewChiHandler()
	_, ok := sh.(http.Handler)
	assert.True(t, ok)
}
