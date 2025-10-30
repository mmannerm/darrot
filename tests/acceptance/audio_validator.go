package acceptance

import (
	"fmt"
	"sync"
	"time"
)

// AudioCaptureSession represents an audio capture session
type AudioCaptureSession struct {
	ChannelID    string        `json:"channel_id"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      *time.Time    `json:"end_time,omitempty"`
	Duration     time.Duration `json:"duration"`
	AudioPackets []AudioPacket `json:"audio_packets"`
	PacketCount  int           `json:"packet_count"`
	TotalBytes   int64         `json:"total_bytes"`
}

// AudioPacket represents a captured audio packet
type AudioPacket struct {
	Timestamp time.Time `json:"timestamp"`
	Data      []byte    `json:"data"`
	Sequence  uint16    `json:"sequence"`
	SSRC      uint32    `json:"ssrc"`
	Format    string    `json:"format"`
	Size      int       `json:"size"`
}

// FormatValidation contains audio format validation results
type FormatValidation struct {
	ChannelID      string    `json:"channel_id"`
	ExpectedFormat string    `json:"expected_format"`
	ActualFormat   string    `json:"actual_format"`
	IsValid        bool      `json:"is_valid"`
	ValidationTime time.Time `json:"validation_time"`
	PacketsChecked int       `json:"packets_checked"`
	FormatErrors   []string  `json:"format_errors,omitempty"`
}

// QualityReport contains audio quality analysis results
type QualityReport struct {
	ChannelID     string        `json:"channel_id"`
	TotalPackets  int           `json:"total_packets"`
	TotalDuration time.Duration `json:"total_duration"`
	AvgPacketSize int           `json:"avg_packet_size"`
	PacketRate    float64       `json:"packet_rate"`
	PacketGaps    int           `json:"packet_gaps"`
	Quality       string        `json:"quality"`
	Bitrate       int           `json:"bitrate"`
	SampleRate    int           `json:"sample_rate"`
	AnalysisTime  time.Time     `json:"analysis_time"`
	QualityScore  float64       `json:"quality_score"`
	Issues        []string      `json:"issues,omitempty"`
}

// ContentValidation contains TTS content verification results
type ContentValidation struct {
	ExpectedText    string    `json:"expected_text"`
	AudioData       []byte    `json:"audio_data"`
	IsValid         bool      `json:"is_valid"`
	Confidence      float64   `json:"confidence"`
	ValidationTime  time.Time `json:"validation_time"`
	TranscribedText string    `json:"transcribed_text,omitempty"`
	Errors          []string  `json:"errors,omitempty"`
}

// audioValidator implements the AudioValidator interface
type audioValidator struct {
	config            *TestConfig
	activeSessions    map[string]*AudioCaptureSession
	completedSessions map[string]*AudioCaptureSession
	mu                sync.RWMutex
}

// NewAudioValidator creates a new audio validator
func NewAudioValidator(config *TestConfig) (AudioValidator, error) {
	return &audioValidator{
		config:            config,
		activeSessions:    make(map[string]*AudioCaptureSession),
		completedSessions: make(map[string]*AudioCaptureSession),
	}, nil
}

// StartCapture begins audio capture for a channel
func (av *audioValidator) StartCapture(channelID string) error {
	av.mu.Lock()
	defer av.mu.Unlock()

	// Check if already capturing
	if _, exists := av.activeSessions[channelID]; exists {
		return fmt.Errorf("audio capture already active for channel %s", channelID)
	}

	session := &AudioCaptureSession{
		ChannelID:    channelID,
		StartTime:    time.Now(),
		AudioPackets: make([]AudioPacket, 0),
		PacketCount:  0,
		TotalBytes:   0,
	}

	av.activeSessions[channelID] = session

	return nil
}

// StopCapture ends audio capture for a channel
func (av *audioValidator) StopCapture(channelID string) (*AudioCaptureSession, error) {
	av.mu.Lock()
	defer av.mu.Unlock()

	session, exists := av.activeSessions[channelID]
	if !exists {
		return nil, fmt.Errorf("no active audio capture for channel %s", channelID)
	}

	// Finalize session
	endTime := time.Now()
	session.EndTime = &endTime
	session.Duration = endTime.Sub(session.StartTime)

	// Move to completed sessions
	av.completedSessions[channelID] = session
	delete(av.activeSessions, channelID)

	// Return a copy
	sessionCopy := *session
	sessionCopy.AudioPackets = make([]AudioPacket, len(session.AudioPackets))
	copy(sessionCopy.AudioPackets, session.AudioPackets)

	return &sessionCopy, nil
}

// ValidateFormat validates the audio format of captured audio
func (av *audioValidator) ValidateFormat(channelID string, expectedFormat string) (*FormatValidation, error) {
	av.mu.RLock()
	defer av.mu.RUnlock()

	// Check active sessions first
	session, exists := av.activeSessions[channelID]
	if !exists {
		// Check completed sessions
		session, exists = av.completedSessions[channelID]
		if !exists {
			return nil, fmt.Errorf("no audio capture session found for channel %s", channelID)
		}
	}

	validation := &FormatValidation{
		ChannelID:      channelID,
		ExpectedFormat: expectedFormat,
		ValidationTime: time.Now(),
		PacketsChecked: len(session.AudioPackets),
		FormatErrors:   make([]string, 0),
	}

	if len(session.AudioPackets) == 0 {
		validation.IsValid = false
		validation.FormatErrors = append(validation.FormatErrors, "no audio packets captured")
		return validation, nil
	}

	// Check format of all packets
	formatMatches := 0
	actualFormats := make(map[string]int)

	for _, packet := range session.AudioPackets {
		actualFormats[packet.Format]++
		if packet.Format == expectedFormat {
			formatMatches++
		}
	}

	// Determine actual format (most common)
	maxCount := 0
	for format, count := range actualFormats {
		if count > maxCount {
			maxCount = count
			validation.ActualFormat = format
		}
	}

	// Validation passes if all packets match expected format
	validation.IsValid = formatMatches == len(session.AudioPackets)

	if !validation.IsValid {
		validation.FormatErrors = append(validation.FormatErrors,
			fmt.Sprintf("expected %s format, but found %d/%d packets with different formats",
				expectedFormat, len(session.AudioPackets)-formatMatches, len(session.AudioPackets)))
	}

	return validation, nil
}

// AnalyzeQuality performs audio quality analysis
func (av *audioValidator) AnalyzeQuality(channelID string) (*QualityReport, error) {
	av.mu.RLock()
	defer av.mu.RUnlock()

	// Check active sessions first
	session, exists := av.activeSessions[channelID]
	if !exists {
		// Check completed sessions
		session, exists = av.completedSessions[channelID]
		if !exists {
			return nil, fmt.Errorf("no audio capture session found for channel %s", channelID)
		}
	}

	report := &QualityReport{
		ChannelID:    channelID,
		TotalPackets: session.PacketCount,
		AnalysisTime: time.Now(),
		Issues:       make([]string, 0),
	}

	if len(session.AudioPackets) == 0 {
		report.Quality = "no_data"
		report.QualityScore = 0.0
		report.Issues = append(report.Issues, "no audio packets captured")
		return report, nil
	}

	// Calculate basic metrics
	totalSize := int64(0)
	for _, packet := range session.AudioPackets {
		totalSize += int64(packet.Size)
	}

	report.AvgPacketSize = int(totalSize / int64(len(session.AudioPackets)))
	report.TotalDuration = session.Duration

	if session.Duration > 0 {
		report.PacketRate = float64(session.PacketCount) / session.Duration.Seconds()
	}

	// Detect packet gaps
	report.PacketGaps = av.detectPacketGaps(session.AudioPackets)

	// Estimate bitrate (simplified calculation)
	if session.Duration > 0 {
		bitsPerSecond := (totalSize * 8) / int64(session.Duration.Seconds())
		report.Bitrate = int(bitsPerSecond)
	}

	// Assume standard sample rate for Opus (48kHz)
	report.SampleRate = 48000

	// Calculate quality score
	report.QualityScore = av.calculateQualityScore(report)

	// Determine quality rating
	if report.QualityScore >= 0.9 {
		report.Quality = "excellent"
	} else if report.QualityScore >= 0.7 {
		report.Quality = "good"
	} else if report.QualityScore >= 0.5 {
		report.Quality = "fair"
	} else {
		report.Quality = "poor"
	}

	// Add quality issues
	if report.PacketGaps > 0 {
		report.Issues = append(report.Issues, fmt.Sprintf("%d packet gaps detected", report.PacketGaps))
	}

	if report.Bitrate < av.config.Audio.QualityThresholds.MinBitrate {
		report.Issues = append(report.Issues, fmt.Sprintf("bitrate %d below minimum %d",
			report.Bitrate, av.config.Audio.QualityThresholds.MinBitrate))
	}

	if report.PacketRate < 20 { // Expect ~50 packets/second for Opus
		report.Issues = append(report.Issues, fmt.Sprintf("low packet rate: %.2f packets/second", report.PacketRate))
	}

	return report, nil
}

// VerifyContent verifies TTS content against expected text
func (av *audioValidator) VerifyContent(audioData []byte, expectedText string) (*ContentValidation, error) {
	validation := &ContentValidation{
		ExpectedText:   expectedText,
		AudioData:      audioData,
		ValidationTime: time.Now(),
		Errors:         make([]string, 0),
	}

	// For now, this is a simplified implementation
	// In a real implementation, this would use speech-to-text to verify content

	if len(audioData) == 0 {
		validation.IsValid = false
		validation.Confidence = 0.0
		validation.Errors = append(validation.Errors, "no audio data provided")
		return validation, nil
	}

	// Simplified validation based on audio length and expected text length
	expectedDuration := av.estimateTextDuration(expectedText)
	actualDuration := av.estimateAudioDuration(audioData)

	// Check if durations are reasonably close (within 50%)
	durationRatio := actualDuration / expectedDuration
	if durationRatio >= 0.5 && durationRatio <= 2.0 {
		validation.IsValid = true
		validation.Confidence = 1.0 - abs(1.0-durationRatio) // Higher confidence for closer ratios
	} else {
		validation.IsValid = false
		validation.Confidence = 0.0
		validation.Errors = append(validation.Errors,
			fmt.Sprintf("audio duration mismatch: expected ~%.1fs, got ~%.1fs",
				expectedDuration, actualDuration))
	}

	// Set transcribed text placeholder
	if validation.IsValid {
		validation.TranscribedText = fmt.Sprintf("[Simulated transcription of: %s]", expectedText)
	}

	return validation, nil
}

// detectPacketGaps detects missing packets in the audio stream
func (av *audioValidator) detectPacketGaps(packets []AudioPacket) int {
	if len(packets) < 2 {
		return 0
	}

	gaps := 0
	for i := 1; i < len(packets); i++ {
		expectedSeq := packets[i-1].Sequence + 1
		if packets[i].Sequence != expectedSeq {
			gaps++
		}
	}

	return gaps
}

// calculateQualityScore calculates an overall quality score
func (av *audioValidator) calculateQualityScore(report *QualityReport) float64 {
	score := 1.0

	// Penalize packet gaps
	if report.PacketGaps > 0 {
		gapPenalty := float64(report.PacketGaps) / float64(report.TotalPackets)
		score -= gapPenalty * 0.5 // Up to 50% penalty for gaps
	}

	// Penalize low bitrate
	if report.Bitrate > 0 && report.Bitrate < av.config.Audio.QualityThresholds.MinBitrate {
		bitratePenalty := 1.0 - (float64(report.Bitrate) / float64(av.config.Audio.QualityThresholds.MinBitrate))
		score -= bitratePenalty * 0.3 // Up to 30% penalty for low bitrate
	}

	// Penalize low packet rate
	if report.PacketRate < 20 {
		ratePenalty := (20 - report.PacketRate) / 20
		score -= ratePenalty * 0.2 // Up to 20% penalty for low packet rate
	}

	// Ensure score is between 0 and 1
	if score < 0 {
		score = 0
	}

	return score
}

// estimateTextDuration estimates how long text should take to speak
func (av *audioValidator) estimateTextDuration(text string) float64 {
	// Rough estimate: ~150 words per minute, ~5 characters per word
	wordsPerMinute := 150.0
	charactersPerWord := 5.0

	estimatedWords := float64(len(text)) / charactersPerWord
	estimatedMinutes := estimatedWords / wordsPerMinute

	return estimatedMinutes * 60.0 // Convert to seconds
}

// estimateAudioDuration estimates audio duration from data size
func (av *audioValidator) estimateAudioDuration(audioData []byte) float64 {
	// Rough estimate for Opus: ~16KB per second at 64kbps
	bytesPerSecond := 16000.0
	return float64(len(audioData)) / bytesPerSecond
}

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// CapturePacket adds an audio packet to an active session (for integration with mock voice server)
func (av *audioValidator) CapturePacket(channelID string, packet AudioPacket) error {
	av.mu.Lock()
	defer av.mu.Unlock()

	session, exists := av.activeSessions[channelID]
	if !exists {
		return fmt.Errorf("no active capture session for channel %s", channelID)
	}

	session.AudioPackets = append(session.AudioPackets, packet)
	session.PacketCount++
	session.TotalBytes += int64(packet.Size)

	return nil
}

// GetActiveSession returns an active capture session
func (av *audioValidator) GetActiveSession(channelID string) *AudioCaptureSession {
	av.mu.RLock()
	defer av.mu.RUnlock()

	if session, exists := av.activeSessions[channelID]; exists {
		// Return a copy
		sessionCopy := *session
		sessionCopy.AudioPackets = make([]AudioPacket, len(session.AudioPackets))
		copy(sessionCopy.AudioPackets, session.AudioPackets)
		return &sessionCopy
	}

	return nil
}

// GetCompletedSession returns a completed capture session
func (av *audioValidator) GetCompletedSession(channelID string) *AudioCaptureSession {
	av.mu.RLock()
	defer av.mu.RUnlock()

	if session, exists := av.completedSessions[channelID]; exists {
		// Return a copy
		sessionCopy := *session
		sessionCopy.AudioPackets = make([]AudioPacket, len(session.AudioPackets))
		copy(sessionCopy.AudioPackets, session.AudioPackets)
		return &sessionCopy
	}

	return nil
}
