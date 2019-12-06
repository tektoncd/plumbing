package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/nlopes/slack"
)

var (
	currentCop string
	botID      string
	token      string
	channelID  string

	vdemeest string
	cops     = map[string]struct{}{}
	rotation = map[string]string{}
)

func init() {
	// When building with "ko", templates is deployed under KO_DATA_PATH
	// If KO_DATA_PATH is not defined, the path will be a relatove one
	basePath := os.Getenv("KO_DATA_PATH")
	csvfile := path.Join(basePath, "rotation.csv")
	if r, err := readRotation(csvfile); err != nil {
		panic("Cannot read rotation.csv… bailing 😓")
	} else {
		rotation = r
	}
}

func main() {
	token = os.Getenv("SLACKTOKEN")
	if token == "" {
		fmt.Println("missing required environment variable SLACKTOKEN")
		os.Exit(1)
	}
	botID = os.Getenv("BOTID")
	if botID == "" {
		fmt.Println("missing required environment variable BOTID")
		os.Exit(1)
	}
	channelID = os.Getenv("CHANNELID")
	if channelID == "" {
		fmt.Println("missing required environment variable CHANNELID")
		os.Exit(1)
	}
	api := slack.New(token, slack.OptionLog(log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)))

	users, err := api.GetUsers()
	if err != nil {
		fmt.Println(err)
	}
	copsID := map[string]string{}
	for _, user := range users {
		if user.Name == "vdemeest" {
			vdemeest = user.ID
		}
		if _, ok := cops[user.Name]; ok {
			copsID[user.Name] = user.ID
		}
	}

	currentCop = getBuildCop(time.Now())

	rtm := api.NewRTM()
	go rtm.ManageConnection()
	go dailyPing(rtm, copsID)

	for msg := range rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.HelloEvent:
			// Ignore hello
		case *slack.ConnectedEvent:
			fmt.Println("Infos:", ev.Info)
			fmt.Println("Connection counter:", ev.ConnectionCount)
		case *slack.MessageEvent:
			if isDirectMessage(ev.Channel) || ev.Channel == channelID {
				handleMessage(rtm, ev.Text, ev.Channel, isDirectMessage(ev.Channel))
			}
		case *slack.PresenceChangeEvent:
			fmt.Printf("Presence Change: %v\n", ev)
		case *slack.LatencyReport:
			fmt.Printf("Current latency: %v\n", ev.Value)
		case *slack.RTMError:
			fmt.Printf("Error: %s\n", ev.Error())
		case *slack.InvalidAuthEvent:
			fmt.Printf("Invalid credentials")
			return
		default:
			// Ignore other events..
			// fmt.Printf("Unhandled: %v\n", msg.Data)
		}
	}
}

func handleMessage(rtm *slack.RTM, message, channel string, direct bool) {
	switch {
	case statusMessage(message, botID, direct):
		rtm.SendMessage(rtm.NewOutgoingMessage(fmt.Sprintf("<@%s> is the buildcop :cop:\nBuildcop log is here: https://docs.google.com/document/d/1kUzH8SV4coOabXLntPA1QI01lbad3Y1wP5BVyh4qzmk", currentCop), channel))
	case easterEggMessage(message, botID, direct):
		rtm.SendMessage(rtm.NewOutgoingMessage(fmt.Sprintf("<@%s> is my master :meow-wow:, and he is old :older_man:, grumpy :face_with_raised_eyebrow: but awesome :hooray: :meow-party:", vdemeest), channel))
	case directMessage(message, botID, direct):
		rtm.SendMessage(rtm.NewOutgoingMessage(":thinking_face: I ain't smart :zany_face:, I don't understand what you are telling me :robot_face: …\n Try to tell me `status` or `who is the buildcop ?` :sunglasses:", channel))
	}
}

func isDirectMessage(channel string) bool {
	return strings.HasPrefix(channel, "DR")
}

func directMessage(message, botID string, direct bool) bool {
	return direct || strings.HasPrefix(message, fmt.Sprintf("<@%s>", botID))
}

func easterEggMessage(message, botID string, direct bool) bool {
	for _, m := range getEasterEggMessages(botID, direct) {
		if m == message {
			return true
		}
	}
	return false
}

func statusMessage(message, botID string, direct bool) bool {
	for _, m := range getStatusMessages(botID, direct) {
		if m == message {
			return true
		}
	}
	return false
}

func getEasterEggMessages(botID string, direct bool) []string {
	return getMessages([]string{"who is your master ?", "master ?", "master", "where do you come from ?"}, botID, direct)
}

func getStatusMessages(botID string, direct bool) []string {
	return getMessages([]string{"who is the buildcop ?", "buildcop ?", "status"}, botID, direct)
}

func getMessages(messages []string, botID string, direct bool) []string {
	if direct {
		return messages
	}
	ms := make([]string, len(messages))
	for i := range messages {
		ms[i] = fmt.Sprintf("<@%s> %s", botID, messages[i])
	}
	return ms
}

func dailyPing(rtm *slack.RTM, cops map[string]string) {
	jt := NewJobTicker()
	for {
		<-jt.t.C
		currentCop = cops[getBuildCop(time.Now())]
		rtm.SendMessage(rtm.NewOutgoingMessage(fmt.Sprintf("Hello :wave: today's <@%s> is the buildcop :cop:\nBuildcop log is here: https://docs.google.com/document/d/1kUzH8SV4coOabXLntPA1QI01lbad3Y1wP5BVyh4qzmk", currentCop), channelID))
		jt.updateJobTicker()
	}
}

type jobTicker struct {
	t *time.Timer
}

func getNextTickDuration() time.Duration {
	now := time.Now()
	nextTick := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, time.Local)
	if nextTick.Before(now) {
		nextTick = nextTick.Add(24 * time.Hour)
	}
	return nextTick.Sub(time.Now())
}

func NewJobTicker() jobTicker {
	return jobTicker{time.NewTimer(getNextTickDuration())}
}

func (jt jobTicker) updateJobTicker() {
	jt.t.Reset(getNextTickDuration())
}

func getBuildCop(t time.Time) string {
	tf := t.Format("2006-01-02") // Mon Jan 2 15:04:05 MST 2006
	if b, ok := rotation[tf]; ok {
		fmt.Println(b)
		return b
	}
	return "nobody"
}

func readRotation(filename string) (map[string]string, error) {
	rotation := map[string]string{}
	// Open CSV file
	f, err := os.Open(filename)
	if err != nil {
		return rotation, err
	}
	defer f.Close()

	// Read File into a Variable
	lines, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return rotation, err
	}

	for i, line := range lines {
		if i == 0 {
			fmt.Println("skip header")
			continue
		}
		if _, ok := cops[line[1]]; !ok {
			cops[line[1]] = struct{}{}
		}
		rotation[line[0]] = line[1]
	}
	return rotation, nil
}
