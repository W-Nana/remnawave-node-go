//go:build tools
// +build tools

package tools

import (
	_ "github.com/gin-contrib/gzip"
	_ "github.com/gin-gonic/gin"
	_ "github.com/go-playground/validator/v10"
	_ "github.com/golang-jwt/jwt/v5"
	_ "github.com/klauspost/compress/zstd"
	_ "github.com/rs/zerolog"
	_ "github.com/stretchr/testify/assert"
	_ "github.com/xtls/xray-core/core"
)
