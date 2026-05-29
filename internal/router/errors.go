package router

import "errors"

var (
	errInvalidPort       = errors.New("invalid port")
	errInvalidTargetHost = errors.New("invalid target host")
)
