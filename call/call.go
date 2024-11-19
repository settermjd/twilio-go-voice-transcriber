package call

// Calls stores the relevant details of a call, recording, and transcription
type Call struct {
	ID, RecordingDuration                                                            int64
	CallSID, Caller, RecordingSID, RecordingURL, TranscriptionSID, TranscriptionText string
}
