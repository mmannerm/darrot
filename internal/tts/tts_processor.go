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

	// Error recovery
	errorRecovery *ErrorRecoveryManager

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

	processor := &ttsProcessor{
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

	// Initialize error recovery manager
	processor.errorRecovery = NewErrorRecoveryManager(voiceManager, ttsManager, messageQueue, configService)

	return processor
}

// Start begins the background TTS processing pipeline
func (tp *ttsProcessor) Start() error {
	log.Println("Starting TTS processing pipeline")

	// Start error recovery manager
	if err := tp.errorRecovery.Start(); err != nil {
		return fmt.Errorf("failed to start error recovery manager: %w", err)
	}

	tp.wg.Add(1)
	go tp.processingLoop()

	return nil
}

// Stop gracefully stops the TTS processing pipeline
func (tp *ttsProcessor) Stop() error {
	log.Println("Stopping TTS processing pipeline")

	tp.cancel()
	tp.wg.Wait()

	// Stop error recovery manager
	if err := tp.errorRecovery.Stop(); err != nil {
		log.Printf("Error stopping error recovery manager: %v", err)
	}

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
	isConnected := tp.voiceManager.IsConnected(guildID)
	if !isConnected {
		// Debug: Log why we're not processing
		queueSize := tp.messageQueue.Size(guildID)
		if queueSize > 0 {
			log.Printf("Guild %s has %d queued messages but no voice connection", guildID, queueSize)
		}
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

	log.Printf("Processing %d queued messages for guild %s", queueSize, guildID)

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

	// Message already has author name from message monitor (Requirement 2.3)
	messageText := message.Content

	// Truncate message if too long (Requirement 4.2)
	if len(messageText) > MaxMessageLength {
		messageText = messageText[:MaxMessageLength-3] + "..."
		log.Printf("Truncated long message for guild %s", guildID)
	}

	// Convert to speech with comprehensive error handling (Requirement 9.2)
	audioData, err := tp.ttsManager.ConvertToSpeech(messageText, "", config)
	if err != nil {
		log.Printf("Initial TTS conversion failed for guild %s: %v", guildID, err)

		// Use comprehensive error recovery
		audioData, err = tp.errorRecovery.HandleTTSFailure(messageText, "", config, guildID)
		if err != nil {
			log.Printf("TTS conversion failed after comprehensive recovery for guild %s: %v", guildID, err)
			return // Skip this message and continue
		}
	}

	// Play audio through voice connection with error recovery
	err = tp.voiceManager.PlayAudio(guildID, audioData)
	if err != nil {
		log.Printf("Audio playback failed for guild %s: %v", guildID, err)

		// Use comprehensive audio playback recovery (Requirement 9.1, 9.2)
		if recoveryErr := tp.errorRecovery.HandleAudioPlaybackFailure(guildID, audioData); recoveryErr != nil {
			log.Printf("Audio playback recovery failed for guild %s: %v", guildID, recoveryErr)

			// Create user-friendly error message (Requirement 9.3)
			userMessage := tp.errorRecovery.CreateUserFriendlyErrorMessage(recoveryErr, guildID)
			log.Printf("User-friendly error for guild %s: %s", guildID, userMessage)
		}
		return
	}

	log.Printf("Successfully processed TTS message for guild %s: %d bytes audio", guildID, len(audioData))
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
