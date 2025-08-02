//go:build embed

package main

import (
	_ "embed"
)

//go:embed can.dbc
var DbcContentRaw []byte

func init() {
	DbcContent = DbcContentRaw
}
