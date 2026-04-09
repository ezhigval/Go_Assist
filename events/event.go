package events

import "time"

// Name — имя события (версионированное: v1.*, v2.*).
type Name string

const (
	V1CalendarCreated         Name = "v1.calendar.created"
	V1CalendarMeetingCreated  Name = "v1.calendar.meeting.created"
	V1TodoCreated             Name = "v1.todo.created"
	V1TodoDue                 Name = "v1.todo.due"
	V1NoteCreated             Name = "v1.note.created"
	V1ContactCreated          Name = "v1.contact.created"
	V1SchedulerTrigger        Name = "v1.scheduler.trigger"
	V1FilesUploaded           Name = "v1.files.uploaded"
	V1TransportFileRecv       Name = "v1.transport.file.received"
	V1AuthSessionCreated      Name = "v1.auth.session.created"
	V1AISuggestion            Name = "v1.ai.suggestion"
	V1SystemStartup           Name = "v1.system.startup"
	V1ReminderOnRoute         Name = "v1.reminder.on_route"
	V1EmailReceived           Name = "v1.email.received"
	V1TrackerMilestoneReached Name = "v1.tracker.milestone.reached"
)

// Event — универсальная единица шины (контракт между модулями).
type Event struct {
	ID        string
	Name      Name
	Payload   any
	Source    string
	TraceID   string
	Timestamp time.Time
	Context   map[string]any
}
