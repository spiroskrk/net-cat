package chat

import (
	"sync"
	"testing"
	"time"
)

// fakeDestination records what the room sends to one member, so room tests
// need no session or socket.
type fakeDestination struct {
	mu sync.Mutex

	history []string // whatever Begin received
	live    []string // every Enqueue, in order
	failed  []error  // every Fail

	beginCalls int   // Begin must happen exactly once
	enqueueErr error // when set, every Enqueue fails with it
}

func (f *fakeDestination) Begin(history []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.beginCalls++
	f.history = append(f.history, history...)
	return nil
}

func (f *fakeDestination) Enqueue(text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.enqueueErr != nil {
		return f.enqueueErr
	}
	f.live = append(f.live, text)
	return nil
}

func (f *fakeDestination) Fail(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failed = append(f.failed, err)
}

// snapshot returns copies, so a test can read safely while the room still runs.
func (f *fakeDestination) snapshot() (history, live []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.history...), append([]string(nil), f.live...)
}

// fixedClock returns a clock frozen at t, so formatted output is predictable.
func fixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

// testTime is the subject's example moment, reused across tests.
var testTime = time.Date(2020, 1, 20, 16, 3, 43, 0, time.UTC)

func TestFormatMessage(t *testing.T) {
	cases := []struct {
		name string
		who  string
		body string
		want string
	}{
		{"subject example", "Yenlik", "hello", "[2020-01-20 16:03:43][Yenlik]:hello\n"},
		{"spaces preserved", "Lee", "  hi  ", "[2020-01-20 16:03:43][Lee]:  hi  \n"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := formatMessage(testTime, c.who, c.body); got != c.want {
				t.Errorf("formatMessage()\n got: %q\nwant: %q", got, c.want)
			}
		})
	}
}

func TestFormatNotice(t *testing.T) {
	if got, want := formatNotice("Lee", joinEvent), "Lee has joined our chat...\n"; got != want {
		t.Errorf("join notice\n got: %q\nwant: %q", got, want)
	}
	if got, want := formatNotice("Lee", leaveEvent), "Lee has left our chat...\n"; got != want {
		t.Errorf("leave notice\n got: %q\nwant: %q", got, want)
	}
}

func TestJoinGivesDuplicateNamesDistinctIDs(t *testing.T) {
	room := NewRoom(fixedClock(testTime))
	first, second := &fakeDestination{}, &fakeDestination{}

	id1, err := room.Join("Lee", first)
	if err != nil {
		t.Fatalf("first join: %v", err)
	}
	id2, err := room.Join("Lee", second)
	if err != nil {
		t.Fatalf("second join: %v", err)
	}

	if id1 == id2 {
		t.Fatalf("duplicate names share ID %d; names must not be membership keys", id1)
	}

	_, live := first.snapshot()
	want := "Lee has joined our chat...\n"
	if len(live) != 2 || live[0] != want || live[1] != want {
		t.Errorf("first member received %q, want two join notices", live)
	}
}

func TestJoinBeginsOnceWithEmptyHistory(t *testing.T) {
	room := NewRoom(fixedClock(testTime))
	dest := &fakeDestination{}

	if _, err := room.Join("Yenlik", dest); err != nil {
		t.Fatalf("join: %v", err)
	}

	history, live := dest.snapshot()
	if dest.beginCalls != 1 {
		t.Errorf("beginCalls = %d, want exactly 1", dest.beginCalls)
	}
	if len(history) != 0 {
		t.Errorf("history = %q, want empty", history)
	}
	if len(live) != 1 || live[0] != "Yenlik has joined our chat...\n" {
		t.Errorf("live = %q, want the newcomer's own join notice", live)
	}
}

func TestJoinRollsBackWhenBeginFails(t *testing.T) {
	room := NewRoom(fixedClock(testTime))
	existing := &fakeDestination{}
	if _, err := room.Join("Lee", existing); err != nil {
		t.Fatalf("setup join: %v", err)
	}

	broken := &failingBegin{}
	if _, err := room.Join("Ghost", broken); err == nil {
		t.Fatal("join succeeded with a failing Begin, want an error")
	}

	room.mu.Lock()
	count := len(room.members)
	room.mu.Unlock()
	if count != 1 {
		t.Errorf("members = %d, want 1; the rejected join left a ghost", count)
	}

	_, live := existing.snapshot()
	if len(live) != 1 {
		t.Errorf("existing member received %q, want only its own join notice", live)
	}
}

// failingBegin rejects the history batch, simulating a session that died
// during replay.
type failingBegin struct {
	fakeDestination
}

func (f *failingBegin) Begin([]string) error {
	return errBeginFailed
}

var errBeginFailed = errTest("begin failed")

// errTest is a tiny error type, so tests need no extra imports.
type errTest string

func (e errTest) Error() string { return string(e) }

func TestSubmitReachesEveryoneIncludingSender(t *testing.T) {
	room := NewRoom(fixedClock(testTime))
	a, b, c := &fakeDestination{}, &fakeDestination{}, &fakeDestination{}

	idA, err := room.Join("Yenlik", a)
	if err != nil {
		t.Fatalf("join Yenlik: %v", err)
	}
	if _, err := room.Join("Lee", b); err != nil {
		t.Fatalf("join Lee: %v", err)
	}
	if _, err := room.Join("Kim", c); err != nil {
		t.Fatalf("join Kim: %v", err)
	}

	if err := room.Submit(idA, "hello"); err != nil {
		t.Fatalf("submit: %v", err)
	}

	want := "[2020-01-20 16:03:43][Yenlik]:hello\n"
	for name, dest := range map[string]*fakeDestination{"sender": a, "second": b, "third": c} {
		_, live := dest.snapshot()
		if len(live) == 0 || live[len(live)-1] != want {
			t.Errorf("%s last received %q, want %q", name, live, want)
		}
	}
}

func TestSubmitIgnoresWhitespaceOnly(t *testing.T) {
	room := NewRoom(fixedClock(testTime))
	dest := &fakeDestination{}

	id, err := room.Join("Yenlik", dest)
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	_, before := dest.snapshot()

	for _, blank := range []string{"", "   ", "\t", " \t "} {
		if err := room.Submit(id, blank); err != nil {
			t.Errorf("Submit(%q) = %v, want nil", blank, err)
		}
	}

	_, after := dest.snapshot()
	if len(after) != len(before) {
		t.Errorf("blank input was delivered: %q", after[len(before):])
	}

	room.mu.Lock()
	n := len(room.history)
	room.mu.Unlock()
	if n != 0 {
		t.Errorf("history has %d entries, want 0", n)
	}
}

func TestSubmitRejectsUnknownID(t *testing.T) {
	room := NewRoom(fixedClock(testTime))
	dest := &fakeDestination{}
	if _, err := room.Join("Yenlik", dest); err != nil {
		t.Fatalf("join: %v", err)
	}
	_, before := dest.snapshot()

	if err := room.Submit(ClientID(999), "hello"); err == nil {
		t.Fatal("Submit with unknown ID returned nil, want an error")
	}

	_, after := dest.snapshot()
	if len(after) != len(before) {
		t.Errorf("rejected message still reached members: %q", after[len(before):])
	}
}

func TestNewMemberReceivesHistoryThenLiveEvents(t *testing.T) {
	room := NewRoom(fixedClock(testTime))
	first := &fakeDestination{}

	id1, err := room.Join("Yenlik", first)
	if err != nil {
		t.Fatalf("join Yenlik: %v", err)
	}
	if err := room.Submit(id1, "hello"); err != nil {
		t.Fatalf("submit hello: %v", err)
	}
	if err := room.Submit(id1, "How are you?"); err != nil {
		t.Fatalf("submit second: %v", err)
	}

	newcomer := &fakeDestination{}
	if _, err := room.Join("Lee", newcomer); err != nil {
		t.Fatalf("join Lee: %v", err)
	}
	if err := room.Submit(id1, "welcome"); err != nil {
		t.Fatalf("submit welcome: %v", err)
	}

	history, live := newcomer.snapshot()

	wantHistory := []string{
		"[2020-01-20 16:03:43][Yenlik]:hello\n",
		"[2020-01-20 16:03:43][Yenlik]:How are you?\n",
	}
	if len(history) != len(wantHistory) {
		t.Fatalf("history = %q, want %q", history, wantHistory)
	}
	for i := range wantHistory {
		if history[i] != wantHistory[i] {
			t.Errorf("history[%d] = %q, want %q", i, history[i], wantHistory[i])
		}
	}

	wantLive := []string{
		"Lee has joined our chat...\n",
		"[2020-01-20 16:03:43][Yenlik]:welcome\n",
	}
	if len(live) != len(wantLive) {
		t.Fatalf("live = %q, want %q", live, wantLive)
	}
	for i := range wantLive {
		if live[i] != wantLive[i] {
			t.Errorf("live[%d] = %q, want %q", i, live[i], wantLive[i])
		}
	}
}

func TestLeaveAnnouncesOnceAndIsIdempotent(t *testing.T) {
	room := NewRoom(fixedClock(testTime))
	staying, leaving := &fakeDestination{}, &fakeDestination{}

	if _, err := room.Join("Yenlik", staying); err != nil {
		t.Fatalf("join Yenlik: %v", err)
	}
	id, err := room.Join("Lee", leaving)
	if err != nil {
		t.Fatalf("join Lee: %v", err)
	}
	_, before := staying.snapshot()

	if err := room.Leave(id); err != nil {
		t.Fatalf("first leave: %v", err)
	}
	if err := room.Leave(id); err != nil {
		t.Errorf("repeated leave = %v, want nil", err)
	}

	_, after := staying.snapshot()
	got := after[len(before):]
	want := "Lee has left our chat...\n"
	if len(got) != 1 || got[0] != want {
		t.Errorf("remaining member received %q, want exactly one %q", got, want)
	}

	if err := room.Submit(id, "ghost"); err == nil {
		t.Error("Submit after Leave returned nil, want an error")
	}
}

func TestBrokenDestinationDoesNotStopHealthyOnes(t *testing.T) {
	room := NewRoom(fixedClock(testTime))
	healthy := &fakeDestination{}
	broken := &fakeDestination{enqueueErr: errTest("write failed")}

	id, err := room.Join("Yenlik", healthy)
	if err != nil {
		t.Fatalf("join Yenlik: %v", err)
	}
	if _, err := room.Join("Lee", broken); err != nil {
		t.Fatalf("join Lee: %v", err)
	}
	_, before := healthy.snapshot()

	if err := room.Submit(id, "hello"); err != nil {
		t.Fatalf("submit: %v", err)
	}

	_, after := healthy.snapshot()
	got := after[len(before):]
	want := "[2020-01-20 16:03:43][Yenlik]:hello\n"
	if len(got) != 1 || got[0] != want {
		t.Errorf("healthy member received %q, want %q", got, want)
	}

	broken.mu.Lock()
	failures := len(broken.failed)
	broken.mu.Unlock()
	if failures == 0 {
		t.Error("broken destination was never told to clean up")
	}
}
