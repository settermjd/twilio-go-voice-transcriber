package database

import (
	"database/sql"
	"log"

	"github.com/settermjd/voicetranscriber/call"
)

type Database struct {
	DB *sql.DB
}

// StoreCallTranscription inserts a record into the calls table that stores the
// core details of a voice call transcription. It does not store that a voice
// call recording has completed, as this might not be know at the time of
// calling this method.
func (d *Database) StoreCallTranscription(call call.Call) (int, error) {
	res, err := d.DB.Exec(
		`INSERT INTO calls (
			call_sid, 
			caller, 
			recording_sid, 
			recording_url, 
			transcription_sid, 
			transcription_text
		) VALUES(?,?,?,?,?,?);
		`,
		call.CallSID,
		call.Caller,
		call.RecordingSID,
		call.RecordingURL,
		call.TranscriptionSID,
		call.TranscriptionText,
	)
	if err != nil {
		return 0, err
	}

	var id int64
	if id, err = res.LastInsertId(); err != nil {
		return 0, err
	}
	return int(id), nil
}

// CallExists checks in the calls table to see if a call with the supplied call
// SID exists.  It returns a boolean to indicated if it does or not, or an error
// if something went wrong querying the database.
func (d *Database) CallExists(callSID string) bool {
	row := d.DB.QueryRow("SELECT COUNT(*) AS RESULTS FROM calls WHERE call_sid = ?;", callSID)
	var rowCount int
	if err := row.Scan(&rowCount); err == sql.ErrNoRows {
		return false
	}

	return true
}

// SetCallRecordingCompleted updates the relevant record in the calls table
// confirming that the voice recording has been completed and is therefore
// available for download.
func (d *Database) SetCallRecordingCompleted(callSID string, recordingDuration int64) (int, error) {
	log.Printf("Updating calls with recording duration: %d seconds and call sid: %s", recordingDuration, callSID)
	res, err := d.DB.Exec(
		`UPDATE calls SET recording_duration = ?, recording_status = "confirmed" WHERE call_sid = ?`,
		recordingDuration,
		callSID,
	)
	if err != nil {
		return 0, err
	}

	var affectedRows int64
	if affectedRows, err = res.RowsAffected(); err == nil {
		log.Printf("Affected %d rows updating the calls table to set the recording duration", affectedRows)
		return int(affectedRows), err
	}
	return 0, nil
}
