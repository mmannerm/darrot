package tts

// This file ensures that TTS dependencies are retained in go.mod
// These imports will be used in future TTS implementation tasks

import (
	// Discord audio encoding
	_ "github.com/jonas747/dca"

	// Note: layeh.com/gopus requires CGO and will be imported when actually used

	// Google Cloud Text-to-Speech
	_ "cloud.google.com/go/texttospeech/apiv1"
)
