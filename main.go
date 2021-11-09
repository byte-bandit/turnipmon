package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/integrii/flaggy"
	"github.com/madflojo/tasks"
	"github.com/pterm/pterm"
	"github.com/vartanbeno/go-reddit/v2/reddit"
)

var (
	cfgIftttEventName  string = os.Getenv("IFTTT_EVENT_NAME")
	cfgIftttKey        string = os.Getenv("IFTTT_KEY")
	cfgRedditAppID     string = os.Getenv("REDDIT_APP_ID")
	cfgRedditAppSecret string = os.Getenv("REDDIT_APP_SECRET")
	cfgRedditUsername  string = os.Getenv("REDDIT_USERNAME")
	cfgRedditPassword  string = os.Getenv("REDDIT_PASSWORD")
	lastPost           *reddit.Post
	sprintIDStyle             = pterm.NewStyle(pterm.FgGray, pterm.BgLightCyan)
	sprintTitleStyle          = pterm.NewStyle(pterm.FgGray, pterm.BgLightMagenta)
	version            string = "unknown"
)

func main() {
	mainScreen()
	flaggy.SetName("TurnipMon")
	flaggy.SetDescription("A little Animal Crossing NH turnip marketplace subreddit monitor, sending you phone notifications via IFTTT once a new turnip trade has been opened.")
	flaggy.DefaultParser.ShowHelpOnUnexpected = true
	flaggy.DefaultParser.AdditionalHelpPrepend = "http://github.com/klausklapper/turnipmon"
	flaggy.String(&cfgIftttEventName, "n", "name", "[env: IFTTT_EVENT_NAME] Required. The IFTTT web hook event name to trigger.")
	flaggy.String(&cfgIftttKey, "k", "key", "[env: IFTTT_KEY] Required. Your IFTTT web hook key (see https://ifttt.com/maker_webhooks).")
	flaggy.String(&cfgRedditAppID, "i", "id", "[env: REDDIT_APP_ID] Required. Your Reddit app API ID credential.")
	flaggy.String(&cfgRedditAppSecret, "s", "secret", "[env: REDDIT_APP_SECRET] Required. Your Reddit app API secret.")
	flaggy.String(&cfgRedditUsername, "u", "username", "[env: REDDIT_USERNAME] Required. Your Reddit username.")
	flaggy.String(&cfgRedditPassword, "p", "password", "[env: REDDIT_PASSWORD] Required. Your Reddit password.")
	flaggy.SetVersion(version)
	flaggy.Parse()

	if len(cfgIftttEventName) < 1 {
		pterm.Fatal.Println("Missing IFTTT web hook event name. Use the --help argument for more information.")
	}
	if len(cfgIftttKey) < 1 {
		pterm.Fatal.Println("Missing IFTTT web hook key. Use the --help argument for more information.")
	}
	if len(cfgRedditAppID) < 1 {
		pterm.Fatal.Println("Missing Reddit app ID. Use the --help argument for more information.")
	}
	if len(cfgRedditAppSecret) < 1 {
		pterm.Fatal.Println("Missing Reddit app secret. Use the --help argument for more information.")
	}
	if len(cfgRedditUsername) < 1 {
		pterm.Fatal.Println("Missing Reddit username. Use the --help argument for more information.")
	}
	if len(cfgRedditPassword) < 1 {
		pterm.Fatal.Println("Missing Reddit password. Use the --help argument for more information.")
	}

	pterm.Success.Printfln("Configuration parsed! Event: %s, Key: [REDACTED]", cfgIftttEventName)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(quit)

	pterm.Info.Println("Connecting to Reddit ...")
	credentials := reddit.Credentials{ID: cfgRedditAppID, Secret: cfgRedditAppSecret, Username: cfgRedditUsername, Password: cfgRedditPassword}
	client, err := reddit.NewClient(credentials)
	pterm.Fatal.PrintOnError(err)

	pterm.Info.Println("Fast forwarding to latest posts ...")
	posts, _, err := client.Subreddit.NewPosts(ctx, "acturnips+ACNHTurnips", &reddit.ListOptions{
		Limit: 5,
	})
	pterm.Fatal.PrintOnError(err)
	if len(posts) < 1 {
		pterm.Fatal.Println("No posts found. This should never happen?")
	}

	updateLastPost(posts[0])
	pterm.Info.Printfln("Fast forwarded to %v%v", sprintID(lastPost.FullID), sprintTitle(lastPost.Title))

	if time.Now().UTC().Sub(lastPost.Created.UTC()) < time.Minute*30 {
		pterm.Success.Printfln("🎉 Last post %v created within the last 30 minutes. Notifying your phone ... 🌈 🥕 ", sprintID(lastPost.FullID))
		err = sendIftttRequest(ctx, lastPost.Title, lastPost.URL)
		pterm.Fatal.PrintOnError(err)
	} else {
		pterm.Warning.Printfln("Last post %v is older than 30 minutes. Assuming it's already expired, no phone notification will be sent.", sprintID(lastPost.FullID))
	}
	pterm.Info.Println("All caught up.")
	spinner := newSpinner()

	scheduler := tasks.New()
	defer scheduler.Stop()
	_, err = scheduler.Add(&tasks.Task{
		Interval: time.Duration(90 * time.Second),
		ErrFunc: func(err error) {
			pterm.Fatal.Println(err)
		},
		TaskFunc: func() error {
			spinner.UpdateText("Checking ...")

			posts, _, err := client.Subreddit.NewPosts(ctx, "acturnips+ACNHTurnips", &reddit.ListOptions{
				Limit:  5,
				Before: lastPost.FullID,
			})
			if err != nil {
				return err
			}
			if len(posts) < 1 {
				spinner.UpdateText("No new trades found. Next check in 90 seconds")
			} else {
				spinner.Success(fmt.Sprintf("Found %d new trades!", len(posts)))
				for _, v := range posts {
					pterm.Success.Printfln("🎉 %v%v created within the last 30 minutes. Notifying your phone ... 🌈 🥕 ", sprintID(v.FullID), sprintTitle(v.Title))
					err = sendIftttRequest(ctx, v.Title, v.URL)
					if err != nil {
						return err
					}
				}
				updateLastPost(posts[0])
				spinner = newSpinner()
			}
			return nil
		},
	})
	pterm.Fatal.PrintOnError(err)

	select {
	case <-quit:
		break
	case <-ctx.Done():
		break
	}

	pterm.Info.Println("Shutdown requested ...")
	scheduler.Stop()
	cancel()

	pterm.Success.Println("Goodbye! 👋")

}

func newSpinner() *pterm.SpinnerPrinter {
	spinner, err := pterm.DefaultSpinner.Start()
	pterm.Fatal.PrintOnError(err)
	spinner.UpdateText("No new trades found. Next check in 90 seconds")
	return spinner
}

func updateLastPost(p *reddit.Post) {
	lastPost = p
}

func sprintID(id string) string {
	return sprintIDStyle.Sprintf("[%v]", id)
}

func sprintTitle(title string) string {
	return sprintTitleStyle.Sprint(title)
}

func mainScreen() {
	print("\033[H\033[2J")
	ptermLogo, _ := pterm.DefaultBigText.WithLetters(
		pterm.NewLettersFromStringWithStyle("Turnip", pterm.NewStyle(pterm.FgLightCyan)),
		pterm.NewLettersFromStringWithStyle("Mon", pterm.NewStyle(pterm.FgLightMagenta))).
		Srender()

	pterm.DefaultCenter.Print(ptermLogo)
	pterm.DefaultCenter.Print(pterm.DefaultHeader.WithFullWidth().WithBackgroundStyle(pterm.NewStyle(pterm.BgLightBlue)).WithMargin(10).Sprint("🥕 TurnipMon - Animal Crossing New Horizons Turnip Market Monitor"))
}

func sendIftttRequest(ctx context.Context, title string, uri string) error {
	type payload struct {
		Value1 string `json:"value1"`
		Value2 string `json:"value2"`
		Value3 string `json:"value3"`
	}

	resp, err := resty.New().R().
		SetContext(ctx).
		SetBody(payload{Value1: title, Value2: uri, Value3: "https://dodo.ac/np/images/8/86/Turnips_NH_Inv_Icon.png"}).
		Post(fmt.Sprintf("https://maker.ifttt.com/trigger/%s/with/key/%s", cfgIftttEventName, cfgIftttKey))

	if err != nil {
		return err
	}
	if !resp.IsSuccess() {
		pterm.Error.Printfln("IFTTT Request failed with [%v] %v: %v", resp.StatusCode(), resp.Status(), resp.Result())
		return errors.New("failed to send IFTTT request")
	}

	return nil
}
