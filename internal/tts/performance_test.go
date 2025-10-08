package tts

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// BenchmarkMessageQueueOperations benchmarks message queue performance
func BenchmarkMessageQueueOperations(b *testing.B) {
	messageQueue := NewMessageQueue()
	guildID := "benchmark-guild"

	b.Run("Enqueue", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			message := &QueuedMessage{
				ID:        fmt.Sprintf("bench-msg-%d", i),
				GuildID:   guildID,
				ChannelID: "bench-channel",
				UserID:    "bench-user",
				Username:  "BenchUser",
				Content:   fmt.Sprintf("Benchmark message %d", i),
				Timestamp: time.Now(),
			}
			messageQueue.Enqueue(message)
		}
	})

	// Fill queue for dequeue benchmark
	for i := 0; i < 1000; i++ {
		message := &QueuedMessage{
			ID:        fmt.Sprintf("setup-msg-%d", i),
			GuildID:   guildID,
			ChannelID: "setup-channel",
			UserID:    "setup-user",
			Username:  "SetupUser",
			Content:   fmt.Sprintf("Setup message %d", i),
			Timestamp: time.Now(),
		}
		messageQueue.Enqueue(message)
	}

	b.Run("Dequeue", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			messageQueue.Dequeue(guildID)
		}
	})

	b.Run("Size", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			messageQueue.Size(guildID)
		}
	})
}

// BenchmarkTTSConversion benchmarks TTS conversion performance
func BenchmarkTTSConversion(b *testing.B) {
	manager := newMockTTSManagerIntegration()
	config := TTSConfig{
		Voice:  "en-US-Standard-A",
		Speed:  1.0,
		Volume: 1.0,
		Format: AudioFormatDCA,
	}

	testTexts := []string{
		"Short message",
		"This is a medium length message for TTS conversion testing",
		"This is a much longer message that contains multiple sentences and should take more time to process. It includes various punctuation marks, numbers like 123 and 456, and different types of content that the TTS engine needs to handle properly.",
	}

	for _, text := range testTexts {
		b.Run(fmt.Sprintf("Text_Length_%d", len(text)), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := manager.ConvertToSpeech(text, config.Voice, config)
				if err != nil {
					b.Fatalf("TTS conversion failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkConcurrentOperations benchmarks concurrent TTS operations
func BenchmarkConcurrentOperations(b *testing.B) {
	_ = runtime.NumCPU() // Available for future use

	b.Run("ConcurrentMessageQueue", func(b *testing.B) {
		messageQueue := NewMessageQueue()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			guildID := "concurrent-guild"
			i := 0
			for pb.Next() {
				message := &QueuedMessage{
					ID:        fmt.Sprintf("concurrent-msg-%d", i),
					GuildID:   guildID,
					ChannelID: "concurrent-channel",
					UserID:    "concurrent-user",
					Username:  "ConcurrentUser",
					Content:   fmt.Sprintf("Concurrent message %d", i),
					Timestamp: time.Now(),
				}
				messageQueue.Enqueue(message)
				i++
			}
		})
	})

	b.Run("ConcurrentTTSConversion", func(b *testing.B) {
		manager := newMockTTSManagerIntegration()
		config := TTSConfig{
			Voice:  "en-US-Standard-A",
			Speed:  1.0,
			Volume: 1.0,
			Format: AudioFormatDCA,
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				text := fmt.Sprintf("Concurrent TTS test message %d", i)
				_, err := manager.ConvertToSpeech(text, config.Voice, config)
				if err != nil {
					b.Fatalf("Concurrent TTS conversion failed: %v", err)
				}
				i++
			}
		})
	})
}

// TestMessageQueuePerformance tests message queue performance under load
func TestMessageQueuePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	messageQueue := NewMessageQueue()
	guildID := "perf-test-guild"

	t.Run("HighVolumeMessageProcessing", func(t *testing.T) {
		numMessages := 10000
		startTime := time.Now()

		// Enqueue messages
		for i := 0; i < numMessages; i++ {
			message := &QueuedMessage{
				ID:        fmt.Sprintf("perf-msg-%d", i),
				GuildID:   guildID,
				ChannelID: "perf-channel",
				UserID:    "perf-user",
				Username:  "PerfUser",
				Content:   fmt.Sprintf("Performance test message %d", i),
				Timestamp: time.Now(),
			}
			err := messageQueue.Enqueue(message)
			require.NoError(t, err, "Should enqueue message %d", i)
		}

		enqueueTime := time.Since(startTime)
		t.Logf("Enqueued %d messages in %v (%.2f msg/sec)",
			numMessages, enqueueTime, float64(numMessages)/enqueueTime.Seconds())

		// Verify queue size
		queueSize := messageQueue.Size(guildID)
		assert.Equal(t, numMessages, queueSize, "Queue should contain all messages")

		// Dequeue messages
		dequeueStart := time.Now()
		processedCount := 0
		for {
			message, err := messageQueue.Dequeue(guildID)
			require.NoError(t, err, "Should dequeue message")
			if message == nil {
				break
			}
			processedCount++
		}

		dequeueTime := time.Since(dequeueStart)
		t.Logf("Dequeued %d messages in %v (%.2f msg/sec)",
			processedCount, dequeueTime, float64(processedCount)/dequeueTime.Seconds())

		assert.Equal(t, numMessages, processedCount, "Should process all messages")
		assert.Equal(t, 0, messageQueue.Size(guildID), "Queue should be empty")
	})

	t.Run("ConcurrentQueueAccess", func(t *testing.T) {
		numProducers := 5
		numConsumers := 3
		messagesPerProducer := 1000

		var wg sync.WaitGroup
		startTime := time.Now()

		// Start producers
		for i := 0; i < numProducers; i++ {
			wg.Add(1)
			go func(producerID int) {
				defer wg.Done()
				for j := 0; j < messagesPerProducer; j++ {
					message := &QueuedMessage{
						ID:        fmt.Sprintf("producer-%d-msg-%d", producerID, j),
						GuildID:   guildID,
						ChannelID: "concurrent-channel",
						UserID:    fmt.Sprintf("producer-%d", producerID),
						Username:  fmt.Sprintf("Producer%d", producerID),
						Content:   fmt.Sprintf("Message %d from producer %d", j, producerID),
						Timestamp: time.Now(),
					}
					err := messageQueue.Enqueue(message)
					assert.NoError(t, err, "Producer %d should enqueue message %d", producerID, j)
				}
			}(i)
		}

		// Start consumers
		processedCounts := make([]int, numConsumers)
		for i := 0; i < numConsumers; i++ {
			wg.Add(1)
			go func(consumerID int) {
				defer wg.Done()
				for {
					message, err := messageQueue.Dequeue(guildID)
					assert.NoError(t, err, "Consumer %d should dequeue message", consumerID)
					if message == nil {
						// Queue is empty, wait a bit and try again
						time.Sleep(10 * time.Millisecond)
						continue
					}
					processedCounts[consumerID]++

					// Stop when we've processed enough messages
					if processedCounts[consumerID] >= messagesPerProducer {
						break
					}
				}
			}(i)
		}

		wg.Wait()
		totalTime := time.Since(startTime)

		totalProduced := numProducers * messagesPerProducer
		totalProcessed := 0
		for _, count := range processedCounts {
			totalProcessed += count
		}

		t.Logf("Concurrent test: %d producers, %d consumers", numProducers, numConsumers)
		t.Logf("Produced %d messages, processed %d messages in %v",
			totalProduced, totalProcessed, totalTime)
		t.Logf("Throughput: %.2f msg/sec", float64(totalProcessed)/totalTime.Seconds())

		// Allow some tolerance for concurrent operations
		assert.GreaterOrEqual(t, totalProcessed, totalProduced/2,
			"Should process at least half of produced messages")
	})
}

// TestTTSPerformance tests TTS conversion performance
func TestTTSPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	manager := newMockTTSManagerIntegration()
	config := TTSConfig{
		Voice:  "en-US-Standard-A",
		Speed:  1.0,
		Volume: 1.0,
		Format: AudioFormatDCA,
	}

	t.Run("BatchTTSConversion", func(t *testing.T) {
		numConversions := 1000
		testText := "This is a test message for TTS performance evaluation."

		startTime := time.Now()
		successCount := 0

		for i := 0; i < numConversions; i++ {
			text := fmt.Sprintf("%s Message number %d.", testText, i)
			_, err := manager.ConvertToSpeech(text, config.Voice, config)
			if err == nil {
				successCount++
			}
		}

		totalTime := time.Since(startTime)
		t.Logf("Converted %d/%d messages in %v (%.2f conversions/sec)",
			successCount, numConversions, totalTime, float64(successCount)/totalTime.Seconds())

		assert.Equal(t, numConversions, successCount, "All conversions should succeed")
	})

	t.Run("ConcurrentTTSConversion", func(t *testing.T) {
		numGoroutines := 10
		conversionsPerGoroutine := 100

		var wg sync.WaitGroup
		successCounts := make([]int, numGoroutines)
		startTime := time.Now()

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				for j := 0; j < conversionsPerGoroutine; j++ {
					text := fmt.Sprintf("Concurrent TTS test from goroutine %d, message %d", goroutineID, j)
					_, err := manager.ConvertToSpeech(text, config.Voice, config)
					if err == nil {
						successCounts[goroutineID]++
					}
				}
			}(i)
		}

		wg.Wait()
		totalTime := time.Since(startTime)

		totalConversions := numGoroutines * conversionsPerGoroutine
		totalSuccesses := 0
		for _, count := range successCounts {
			totalSuccesses += count
		}

		t.Logf("Concurrent TTS: %d goroutines, %d conversions each", numGoroutines, conversionsPerGoroutine)
		t.Logf("Completed %d/%d conversions in %v (%.2f conversions/sec)",
			totalSuccesses, totalConversions, totalTime, float64(totalSuccesses)/totalTime.Seconds())

		assert.Equal(t, totalConversions, totalSuccesses, "All concurrent conversions should succeed")
	})

	t.Run("VariableMessageLengthPerformance", func(t *testing.T) {
		messageLengths := []int{10, 50, 100, 200, 500, 1000}
		conversionsPerLength := 100

		for _, length := range messageLengths {
			t.Run(fmt.Sprintf("Length_%d", length), func(t *testing.T) {
				// Generate text of specified length
				text := generateTextOfLength(length)

				startTime := time.Now()
				successCount := 0

				for i := 0; i < conversionsPerLength; i++ {
					_, err := manager.ConvertToSpeech(text, config.Voice, config)
					if err == nil {
						successCount++
					}
				}

				totalTime := time.Since(startTime)
				avgTime := totalTime / time.Duration(successCount)

				t.Logf("Length %d: %d/%d conversions, avg time: %v",
					length, successCount, conversionsPerLength, avgTime)

				assert.Equal(t, conversionsPerLength, successCount,
					"All conversions should succeed for length %d", length)
			})
		}
	})
}

// TestMemoryUsage tests memory usage under load
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory usage test in short mode")
	}

	t.Run("MessageQueueMemoryUsage", func(t *testing.T) {
		messageQueue := NewMessageQueue()
		guildID := "memory-test-guild"

		// Get initial memory stats
		var m1 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		// Add many messages
		numMessages := 50000
		for i := 0; i < numMessages; i++ {
			message := &QueuedMessage{
				ID:        fmt.Sprintf("memory-msg-%d", i),
				GuildID:   guildID,
				ChannelID: "memory-channel",
				UserID:    "memory-user",
				Username:  "MemoryUser",
				Content:   fmt.Sprintf("Memory test message %d with some additional content to make it longer", i),
				Timestamp: time.Now(),
			}
			err := messageQueue.Enqueue(message)
			require.NoError(t, err, "Should enqueue message %d", i)
		}

		// Get memory stats after adding messages
		var m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m2)

		memoryUsed := m2.Alloc - m1.Alloc
		memoryPerMessage := float64(memoryUsed) / float64(numMessages)

		t.Logf("Memory usage: %d bytes for %d messages (%.2f bytes/message)",
			memoryUsed, numMessages, memoryPerMessage)

		// Clear queue
		err := messageQueue.Clear(guildID)
		require.NoError(t, err, "Should clear queue")

		// Get memory stats after clearing
		var m3 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m3)

		t.Logf("Memory after clear: %d bytes (freed: %d bytes)",
			m3.Alloc, m2.Alloc-m3.Alloc)

		// Verify queue is empty
		assert.Equal(t, 0, messageQueue.Size(guildID), "Queue should be empty after clear")
	})
}

// TestQueueLimitsPerformance tests queue behavior at limits
func TestQueueLimitsPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping queue limits test in short mode")
	}

	messageQueue := NewMessageQueue()
	guildID := "limits-test-guild"
	maxSize := 100

	t.Run("QueueOverflowBehavior", func(t *testing.T) {
		// Set queue limit
		err := messageQueue.SetMaxSize(guildID, maxSize)
		require.NoError(t, err, "Should set max queue size")

		// Add messages beyond the limit
		numMessages := maxSize * 2
		startTime := time.Now()

		for i := 0; i < numMessages; i++ {
			message := &QueuedMessage{
				ID:        fmt.Sprintf("overflow-msg-%d", i),
				GuildID:   guildID,
				ChannelID: "overflow-channel",
				UserID:    "overflow-user",
				Username:  "OverflowUser",
				Content:   fmt.Sprintf("Overflow test message %d", i),
				Timestamp: time.Now(),
			}
			err := messageQueue.Enqueue(message)
			assert.NoError(t, err, "Should handle overflow gracefully for message %d", i)
		}

		enqueueTime := time.Since(startTime)
		queueSize := messageQueue.Size(guildID)

		t.Logf("Added %d messages to queue with limit %d in %v",
			numMessages, maxSize, enqueueTime)
		t.Logf("Final queue size: %d (should be <= %d)", queueSize, maxSize)

		assert.LessOrEqual(t, queueSize, maxSize, "Queue size should not exceed limit")

		// Verify we can still dequeue messages
		dequeueCount := 0
		for {
			message, err := messageQueue.Dequeue(guildID)
			require.NoError(t, err, "Should dequeue message")
			if message == nil {
				break
			}
			dequeueCount++
		}

		assert.Equal(t, queueSize, dequeueCount, "Should dequeue all remaining messages")
	})
}

// generateTextOfLength generates text of approximately the specified length
func generateTextOfLength(length int) string {
	baseText := "This is a sample text for testing purposes. "
	result := ""

	for len(result) < length {
		result += baseText
	}

	// Trim to exact length
	if len(result) > length {
		result = result[:length]
	}

	return result
}
