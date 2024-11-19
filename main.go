package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/ddymko/go-jsonerror"
	"github.com/joho/godotenv"
	sqlite3 "github.com/mattn/go-sqlite3"
	zerolog "github.com/rs/zerolog"
	"github.com/settermjd/voicetranscriber/call"
	"github.com/settermjd/voicetranscriber/database"
	sqldblogger "github.com/simukti/sqldb-logger"
	zerologadapter "github.com/simukti/sqldb-logger/logadapter/zerologadapter"
	"github.com/twilio/twilio-go/twiml"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	app := NewApp()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /", app.recordVoiceCall)
	mux.HandleFunc("POST /transcribed", app.handleTranscribedCall)
	mux.HandleFunc("POST /recorded", app.handleCompletedCallRecording)

	log.Print("Starting server on :8080")
	err = http.ListenAndServe(":8080", mux)
	log.Fatal(err)
}

// appError provides a simple way of handling errors within the application
func appError(w http.ResponseWriter, err error) {
	var error jsonerror.ErrorJSON
	error.AddError(jsonerror.ErrorComp{
		Detail: err.Error(),
		Code:   strconv.Itoa(http.StatusBadRequest),
		Title:  "Something went wrong",
		Status: http.StatusBadRequest,
	})
	http.Error(w, error.Error(), http.StatusBadRequest)
}

type App struct {
	DB *database.Database
}

// NewApp simplifies construction of a new App object
func NewApp() App {
	dsn := fmt.Sprintf("file:%s?", os.Getenv("DATABASE_NAME"))
	loggerAdapter := zerologadapter.New(zerolog.New(os.Stdout).With().Timestamp().Logger())
	loggerOptions := []sqldblogger.Option{
		sqldblogger.WithSQLQueryFieldname("sql"),
		sqldblogger.WithWrapResult(false),
		sqldblogger.WithExecerLevel(sqldblogger.LevelDebug),
		sqldblogger.WithQueryerLevel(sqldblogger.LevelDebug),
		sqldblogger.WithPreparerLevel(sqldblogger.LevelDebug),
	}
	db := sqldblogger.OpenDriver(
		dsn,
		&sqlite3.SQLiteDriver{},
		loggerAdapter,
		loggerOptions...,
	)

	err := db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	return App{
		DB: &database.Database{DB: db},
	}
}

// handleTranscribedCall is called when a recorded voice call transcription is
// completed. The caller is then notified via email, letting them know that the
// transcription is ready to be downloaded.
func (a *App) handleTranscribedCall(w http.ResponseWriter, r *http.Request) {
	call := call.Call{
		CallSID:           r.PostFormValue("CallSid"),
		Caller:            r.PostFormValue("From"),
		RecordingSID:      r.PostFormValue("RecordingSid"),
		RecordingURL:      r.PostFormValue("RecordingUrl"),
		TranscriptionSID:  r.PostFormValue("TranscriptionSid"),
		TranscriptionText: r.PostFormValue("TranscriptionText"),
	}
	insertID, err := a.DB.StoreCallTranscription(call)
	if err != nil {
		appError(w, err)
		return
	}
	if insertID == 0 {
		appError(w, errors.New("could not store the call details in the database"))
	}
}

// handleCompletedCallRecording updates the call recording, setting the
// recording duration and that the call is completed
func (a *App) handleCompletedCallRecording(w http.ResponseWriter, r *http.Request) {
	recordingDuration := r.PostFormValue("RecordingDuration")
	callSid := r.PostFormValue("CallSid")

	if !a.DB.CallExists(callSid) {
		appError(w, fmt.Errorf("no call with call sid %s available", callSid))
		return
	}

	if duration, err := strconv.Atoi(recordingDuration); err == nil {
		affectedRows, err := a.DB.SetCallRecordingCompleted(callSid, int64(duration))
		if err != nil {
			appError(w, err)
			return
		}
		if affectedRows == 0 {
			appError(w, fmt.Errorf(
				"could not set the call recording for call: %s as being completed with the supplied recording duration. affected rows: %d",
				callSid, affectedRows,
			))

			return
		}
	} else {
		appError(w, errors.New("could not convert recording duration to an integer representation"))
	}
}

// recordVoiceCall handles calls from people, letting them record a voice
// message which is then transcribed, separately, and is later available to be
// downloaded.
func (a *App) recordVoiceCall(w http.ResponseWriter, r *http.Request) {
	say := &twiml.VoiceSay{
		Message: "Please leave a message after the tone.",
	}
	record := &twiml.VoiceRecord{
		Transcribe:                    "true",
		TranscribeCallback:            "/transcribed",
		MaxLength:                     "300",
		RecordingStatusCallback:       "/recorded",
		RecordingStatusCallbackEvent:  "completed",
		RecordingStatusCallbackMethod: "POST",
	}
	hangup := &twiml.VoiceHangup{}

	twiml, err := twiml.Voice([]twiml.Element{say, record, hangup})
	if err != nil {
		appError(w, fmt.Errorf("could not prepare TwiML. reason: %s", err))
	}

	w.Header().Add("Content-Type", "application/xml")
	_, err = w.Write([]byte(twiml))
	if err != nil {
		appError(w, fmt.Errorf("could not write response body. reason: %s", err))
	}
}
