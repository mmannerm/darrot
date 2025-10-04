# 🔍 Audio Format Debugging Applied

## 🎯 **Root Cause Investigation**

The pitch issue persists, so we're now **debugging the actual Google TTS response** to understand what format is really being returned.

## 🔧 **New Debugging Features Added**

### **1. Google TTS Response Inspection**
```go
// Check if response has WAV header
if string(header[0:4]) == "RIFF" && string(header[8:12]) == "WAVE" {
    // Extract actual sample rate and channels from WAV header
    actualSampleRate := // bytes 24-27
    actualChannels := // bytes 22-23
}
```

### **2. Proper Audio Processing Pipeline**
```
Google TTS Response
    ↓
Check for WAV header (skip if present)
    ↓
Detect actual format (mono/stereo, sample rate)
    ↓
Convert mono → stereo (if needed)
    ↓
Resample to 48kHz (if needed)
    ↓
Opus encode at true 48kHz stereo
```

### **3. Enhanced Logging**
- **WAV header detection**: Check if Google returns WAV or raw PCM
- **Actual format detection**: Real sample rate and channel count
- **Processing steps**: Mono→stereo, resampling details
- **Frame calculations**: Verify Opus frame math

## 🔍 **What We're Testing**

### **Hypothesis 1: Mono Audio**
- **Problem**: Google TTS returns mono, we treat as stereo
- **Result**: Audio plays 2x too fast (half the samples)
- **Fix**: Detect mono, duplicate samples for stereo

### **Hypothesis 2: WAV Header**
- **Problem**: Google returns WAV file, we process header as audio
- **Result**: First 44 bytes are header, not audio data
- **Fix**: Skip WAV header, process only audio data

### **Hypothesis 3: Wrong Sample Rate**
- **Problem**: Google returns different rate than requested
- **Result**: Opus encoder gets wrong rate assumption
- **Fix**: Detect actual rate, resample correctly

## 📊 **Expected Debug Output**

### **If WAV Format**
```
[DEBUG] Response contains WAV header
[DEBUG] WAV header indicates: 24000 Hz, 1 channels
[DEBUG] Skipping WAV header (44 bytes)
[DEBUG] Converted mono to stereo: 12000 -> 24000 samples
[DEBUG] Resampled: 24000 samples (24000Hz) -> 48000 samples (48000Hz)
```

### **If Raw PCM**
```
[DEBUG] Response is raw PCM (no WAV header)
[DEBUG] First 16 bytes: [...]
[DEBUG] Processing audio: 24000Hz 1ch -> 48000Hz 2ch
[DEBUG] Converted mono to stereo: 12000 -> 24000 samples
```

## 🎵 **Audio Processing Improvements**

### **Mono to Stereo Conversion**
```go
// For each mono sample, create stereo pair
stereoSamples[i*2] = sample     // Left channel
stereoSamples[i*2+1] = sample   // Right channel (duplicate)
```

### **Stereo-Aware Resampling**
```go
// Process samples in stereo pairs (left, right)
inputFrames := len(stereoSamples) / 2
// Interpolate both channels independently
```

### **Format Detection**
```go
// Check WAV header signature
if string(header[0:4]) == "RIFF" && string(header[8:12]) == "WAVE"
// Extract format from header bytes
```

## 🚀 **Testing Strategy**

1. **Run the bot** and send a test message
2. **Check logs** for format detection:
   - WAV vs raw PCM
   - Actual sample rate and channels
   - Processing steps applied
3. **Listen to audio** - should be natural pitch
4. **Verify timing** - correct duration

## 🎯 **Expected Results**

### **If Mono Was the Issue**
- ✅ Natural pitch (not 2x fast)
- ✅ Correct duration
- ✅ Proper stereo output

### **If WAV Header Was the Issue**
- ✅ Clean audio (no header noise)
- ✅ Correct format processing
- ✅ Proper sample count

### **If Sample Rate Was Wrong**
- ✅ True 48kHz processing
- ✅ Accurate resampling
- ✅ Correct Opus encoding

## 🔧 **Debug Information to Watch**

Look for these log patterns:
- `WAV header indicates: X Hz, Y channels`
- `Converted mono to stereo: X -> Y samples`
- `Processing audio: XHz Ych -> 48000Hz 2ch`
- `Resampled: X samples (XHz) -> Y samples (48000Hz)`

This will tell us exactly what Google TTS is returning and how we're processing it! 🎵✨