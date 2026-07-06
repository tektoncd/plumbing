package main

// EntryPayload is the stackdriver response format
type EntryPayload struct {
	Fields EntryFields `json:"fields"`
}

// EntryFields are the fields from the stackdriver response
// that we are interested in showing to the user
type EntryFields struct {
	Caller     EntryValue `json:"caller"`
	Level      EntryValue `json:"level"`
	Msg        EntryValue `json:"msg"`
	Stacktrace EntryValue `json:"stacktrace"`
}

// EntryValue is the structure of string content returned
// by stackdriver
type EntryValue struct {
	Kind struct {
		StringValue string
	}
}

// RenderableEntry is sent to the frontend for rendering
type RenderableEntry struct {
	TaskName      string `json:"task"`
	LogName       string `json:"log"`
	Message       string `json:"msg"`
	Caller        string `json:"caller"`
	ContainerName string `json:"container"`
	TimeStamp     string `json:"ts"`
}
