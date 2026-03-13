package core

import "embed"

//go:embed dependencies/** scripts/**
var Assets embed.FS
