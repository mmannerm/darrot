package tts

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// ttsProcessor handles the background processing pipeline for TTS conversion and playback
type ttsProcessor struct {
	ttsManager    TTSManager
	voiceManager  VoiceManager
	messageQueue  MessageQueue
	configService ConfigService
	userService   UserService

	// Processing control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Guild-specific processing state
	guildProcessors map[string]*guildProcessor
	mu              sync.RWMutex

	// Configuration
	processingInterval time.Duration
	inactivityTimeout  time.Duration
}

// guildProcessor manages TTS processing for a specific guild
type guildProcessor struct {
	guildID            string
	isProcessing       bool
	lastActivity       time.Time
	inactivityNotified bool
	mu                 sync.RWMutex
}

// NewTTSProcessor creates a new TTS processing pipeline
func NewTTSProcessor(ttsManager TTSManager, voiceManager VoiceManager, messageQueue MessageQueue, configService ConfigService, userService UserService) TTSProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	return &ttsProcessor{
		ttsManager:         ttsManager,
		voiceManager:       voiceManager,
		messageQueue:       messageQueue,
		configService:      configService,
		userService:        userService,
		ctx:                ctx,
		cancel:             cancel,
		guildProcessors:    make(map[string]*guildProcessor),
		processingInterval: time.Millisecond * 500, // Check for new messages every 500ms
		inactivityTimeout:  5 * time.Minute,        // Requirement 4.4
	}
}

// Start begins the background TTS processing pipeline
func (tp *ttsProcessor) Start() error {
	log.Println("Starting TTS processing pipeline")

	tp.wg.Add(1)
	go tp.processingLoop()

	return nil
}

// Stop gracefully stops the TTS processing pipeline
func (tp *ttsProcessor) Stop() error {
	log.Println("Stopping TTS processing pipeline")

	tp.cancel()
	tp.wg.Wait()

	return nil
}

// StartGuildProcessing starts TTS processing for a specific guild
func (tp *ttsProcessor) StartGuildProcessing(guildID string) error {
	if guildID == "" {
		return fmt.Errorf("guild ID cannot be empty")
	}

	tp.mu.Lock()
	defer tp.mu.Unlock()

	// Check if already processing
	if _, exists := tp.guildProcessors[guildID]; exists {
		return nil // Already processing
	}

	// Create guild processor
	tp.guildProcessors[guildID] = &guildProcessor{
		guildID:            guildID,
		isProcessing:       false,
		lastActivity:       time.Now(),
		inactivityNotified: false,
	}

	log.Printf("Started TTS processing for guild %s", guildID)
	return nil
}

// StopGuildProcessing stops TTS processing for a specific guild
func (tp *ttsProcessor) StopGuildProcessing(guildID string) error {
	if guildID == "" {
		return fmt.Errorf("guild ID cannot be empty")
	}

	tp.mu.Lock()
	defer tp.mu.Unlock()

	delete(tp.guildProcessors, guildID)

	log.Printf("Stopped TTS processing for guild %s", guildID)
	return nil
}

// processingLoop is the main processing loop that runs in the background
func (tp *ttsProcessor) processingLoop() {
	defer tp.wg.Done()

	ticker := time.NewTicker(tp.processingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-tp.ctx.Done():
			log.Println("TTS processing loop stopped")
			return
		case <-ticker.C:
			tp.processAllGuilds()
		}
	}
}

// processAllGuilds processes messages for all active guilds
func (tp *ttsProcessor) processAllGuilds() {
	tp.mu.RLock()
	guilds := make([]string, 0, len(tp.guildProcessors))
	for guildID := range tp.guildProcessors {
		guilds = append(guilds, guildID)
	}
	tp.mu.RUnlock()

	for _, guildID := range guilds {
		tp.processGuildMessages(guildID)
	}
}

// processGuildMessages processes queued messages for a specific guild
func (tp *ttsProcessor) processGuildMessages(guildID string) {
	// Get guild processor
	tp.mu.RLock()
	processor, exists := tp.guildProcessors[guildID]
	tp.mu.RUnlock()

	if !exists {
		return
	}

	// Check if voice connection exists
	if !tp.voiceManager.IsConnected(guildID) {
		return
	}

	// Check if currently processing or paused
	processor.mu.RLock()
	isProcessing := processor.isProcessing
	processor.mu.RUnlock()

	if isProcessing {
		return // Already processing a message
	}

	// Check if paused
	if tp.voiceManager.IsPaused(guildID) {
		return
	}

	// Check for messages in queue
	queueSize := tp.messageQueue.Size(guildID)
	if queueSize == 0 {
		tp.checkInactivity(guildID, processor)
		return
	}

	// Process next message
	tp.processNextMessage(guildID, processor)
}

// processNextMessage processes the next message in the queue for a guild
func (tp *ttsProcessor) processNextMessage(guildID string, processor *guildProcessor) {
	// Mark as processing
	processor.mu.Lock()
	processor.isProcessing = true
	processor.lastActivity = time.Now()
	processor.inactivityNotified = false
	processor.mu.Unlock()

	defer func() {
		processor.mu.Lock()
		processor.isProcessing = false
		processor.mu.Unlock()
	}()

	// Dequeue message
	message, err := tp.messageQueue.Dequeue(guildID)
	if err != nil {
		log.Printf("Failed to dequeue message for guild %s: %v", guildID, err)
		return
	}

	if message == nil {
		return // No message to process
	}

	// Get TTS configuration for guild
	config, err := tp.getTTSConfig(guildID)
	if err != nil {
		log.Printf("Failed to get TTS config for guild %s: %v", guildID, err)
		return
	}

	// Prepare message text with author name (Requirement 2.3)
	messageText := fmt.Sprintf("%s says: %s", message.Username, message.Content)

	// Truncate message if too long (Requirement 4.2)
	if len(messageText) > MaxMessageLength {
		messageText = messageText[:MaxMessageLength-3] + "..."
		log.Printf("Truncated long message for guild %s", guildID)
	}

	// Convert to speech with error handling (Requirement 9.2)
	audioData, err := tp.convertWithRetry(messageText, config, guildID)
	if err != nil {
		log.Printf("Failed to convert message to speech for guild %s: %v", guildID, err)
		return // Skip this message and continue
	}

	// Play audio through voice connection
	err = tp.voiceManager.PlayAudio(guildID, audioData)
	if err != nil {
		log.Printf("Failed to play audio for guild %s: %v", guildID, err)

		// Attempt connection recovery (Requirement 9.1)
		if err := tp.voiceManager.RecoverConnection(guildID); err != nil {
			log.Printf("Failed to recover voice connection for guild %s: %v", guildID, err)
		}
		return
	}

	log.Printf("Successfully processed TTS message for guild %s: %d bytes audio", guildID, len(audioData))
}

// convertWithRetry attempts TTS conversion with error recovery
func (tp *ttsProcessor) convertWithRetry(text string, config TTSConfig, guildID string) ([]byte, error) {
	// First attempt
	audioData, err := tp.ttsManager.ConvertToSpeech(text, "", config)
	if err == nil {
		return audioData, nil
	}

	log.Printf("Initial TTS conversion failed for guild %s: %v", guildID, err)

	// Check if error is retryable
	if !IsRetryableError(err) && IsFatalError(err) {
		return nil, fmt.Errorf("fatal TTS error: %w", err)
	}

	// Retry with fallback mechanisms
	maxRetries := 3
	retryDelay := time.Second * 2

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("Retrying TTS conversion for guild %s, attempt %d/%d", guildID, attempt, maxRetries)
		time.Sleep(retryDelay)

		audioData, err = tp.ttsManager.ConvertToSpeech(text, "", config)
		if err == nil {
			return audioData, nil
		}

		log.Printf("TTS retry attempt %d failed for guild %s: %v", attempt, guildID, err)
	}

	// Try with fallback voice
	fallbackConfig := TTSConfig{
		Voice:  DefaultVoice,
		Speed:  DefaultTTSSpeed,
		Volume: DefaultTTSVolume,
		Format: config.Format,
	}

	log.Printf("Trying fallback configuration for guild %s", guildID)
	audioData, err = tp.ttsManager.ConvertToSpeech(text, "", fallbackConfig)
	if err == nil {
		return audioData, nil
	}

	// Try with truncated text if still failing
	if len(text) > 100 {
		truncatedText := text[:97] + "..."
		log.Printf("Trying truncated text for guild %s", guildID)
		audioData, err = tp.ttsManager.ConvertToSpeech(truncatedText, "", fallbackConfig)
		if err == nil {
			return audioData, nil
		}
	}

	return nil, fmt.Errorf("all TTS conversion attempts failed: %w", err)
}

// checkInactivity checks for inactivity and announces if needed (Requirement 4.4)
func (tp *ttsProcessor) checkInactivity(guildID string, processor *guildProcessor) {
	processor.mu.RLock()
	lastActivity := processor.lastActivity
	inactivityNotified := processor.inactivityNotified
	processor.mu.RUnlock()

	if time.Since(lastActivity) > tp.inactivityTimeout && !inactivityNotified {
		// Mark as notified to prevent repeated announcements
		processor.mu.Lock()
		processor.inactivityNotified = true
		processor.mu.Unlock()

		// Create inactivity announcement
		inactivityMessage := "No new messages for 5 minutes, but I'm still here listening."

		// Get TTS configuration
		config, err := tp.getTTSConfig(guildID)
		if err != nil {
			log.Printf("Failed to get TTS config for inactivity announcement in guild %s: %v", guildID, err)
			return
		}

		// Convert announcement to speech
		audioData, err := tp.ttsManager.ConvertToSpeech(inactivityMessage, "", config)
		if err != nil {
			log.Printf("Failed to convert inactivity announcement for guild %s: %v", guildID, err)
			return
		}

		// Play inactivity announcement
		err = tp.voiceManager.PlayAudio(guildID, audioData)
		if err != nil {
			log.Printf("Failed to play inactivity announcement for guild %s: %v", guildID, err)
		} else {
			log.Printf("Announced inactivity for guild %s", guildID)
		}
	}
}

// getTTSConfig gets the TTS configuration for a guild
func (tp *ttsProcessor) getTTSConfig(guildID string) (TTSConfig, error) {
	if tp.configService != nil {
		settings, err := tp.configService.GetTTSSettings(guildID)
		if err == nil && settings != nil {
			return *settings, nil
		}
	}

	// Return default configuration
	return TTSConfig{
		Voice:  DefaultVoice,
		Speed:  DefaultTTSSpeed,
		Volume: DefaultTTSVolume,
		Format: AudioFormatDCA,
	}, nil
}

// GetProcessingStatus returns the processing status for a guild
func (tp *ttsProcessor) GetProcessingStatus(guildID string) (bool, error) {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	processor, exists := tp.guildProcessors[guildID]
	if !exists {
		return false, nil
	}

	processor.mu.RLock()
	defer processor.mu.RUnlock()

	return processor.isProcessing, nil
}

// GetActiveGuilds returns a list of guilds currently being processed
func (tp *ttsProcessor) GetActiveGuilds() []string {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	guilds := make([]string, 0, len(tp.guildProcessors))
	for guildID := range tp.guildProcessors {
		guilds = append(guilds, guildID)
	}

	return guilds
}

// SkipCurrentMessage skips the currently processing message for a guild
func (tp *ttsProcessor) SkipCurrentMessage(guildID string) error {
	// Skip in voice manager (stops current audio)
	if err := tp.voiceManager.SkipCurrentMessage(guildID); err != nil {
		return fmt.Errorf("failed to skip current message: %w", err)
	}

	// Reset processing state
	tp.mu.RLock()
	processor, exists := tp.guildProcessors[guildID]
	tp.mu.RUnlock()

	if exists {
		processor.mu.Lock()
		processor.isProcessing = false
		processor.lastActivity = time.Now()
		processor.mu.Unlock()
	}

	log.Printf("Skipped current message for guild %s", guildID)
	return nil
}

// PauseProcessing pauses TTS processing for a guild
func (tp *ttsProcessor) PauseProcessing(guildID string) error {
	return tp.voiceManager.PausePlayback(guildID)
}

// ResumeProcessing resumes TTS processing for a guild
func (tp *ttsProcessor) ResumeProcessing(guildID string) error {
	return tp.voiceManager.ResumePlayback(guildID)
}

// ClearQueue clears the message queue for a guild
func (tp *ttsProcessor) ClearQueue(guildID string) error {
	return tp.messageQueue.Clear(guildID)
}

// GetQueueSize returns the current queue size for a guild
func (tp *ttsProcessor) GetQueueSize(guildID string) int {
	return tp.messageQueue.Size(guildID)
}
