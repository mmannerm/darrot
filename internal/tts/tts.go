// Package tts provides text-to-speech functionality for Discord voice channels.
// It includes interfaces and types for managing TTS operations, voice connections,
// channel pairings, user permissions, and message queuing.
package tts

import (
	"fmt"
)

// Version represents the TTS package version
const Version = "0.1.0"

// Package-level errors
var (
	ErrInvalidConfig     = fmt.Errorf("invalid TTS configuration")
	ErrVoiceNotConnected = fmt.Errorf("not connected to voice channel")
	ErrQueueFull         = fmt.Errorf("message queue is full")
	ErrUserNotOptedIn    = fmt.Errorf("user has not opted in to TTS")
	ErrInvalidPermission = fmt.Errorf("insufficient permissions")
	ErrChannelNotPaired  = fmt.Errorf("channel is not paired")
)

// Constants for TTS limits and defaults
const (
	DefaultMaxQueueSize     = 10
	DefaultMaxMessageLength = 500
	DefaultTTSSpeed         = 1.0
	DefaultTTSVolume        = 1.0
	DefaultVoice            = "en-US-Standard-A"

	MinTTSSpeed  = 0.25
	MaxTTSSpeed  = 4.0
	MinTTSVolume = 0.0
	MaxTTSVolume = 2.0

	MaxQueueSize     = 100
	MaxMessageLength = 2000
)
