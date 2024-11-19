BEGIN;

-- Define the calls table with just enough information 
-- to make the project meaningful
CREATE TABLE IF NOT EXISTS calls (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    call_sid VARCHAR(34) NOT NULL 
        CHECK(length(recording_sid) == 34),
    -- The phone number of the caller in E.164 format
    caller VARCHAR(15) NOT NULL,
    recording_sid VARCHAR(34) 
        CHECK(length(recording_sid) == 34),
    recording_url TEXT,
    recording_status TEXT 
        CHECK(recording_status == "confirmed"),
    recording_duration INTEGER DEFAULT 0,
    transcription_sid VARCHAR(34) 
        CHECK(length(recording_sid) == 34),
    transcription_text TEXT
);

-- Create an index on call_sid as that column is used to update the calls table
CREATE UNIQUE INDEX IF NOT EXISTS idx_callsid ON calls (call_sid ASC);

COMMIT;
