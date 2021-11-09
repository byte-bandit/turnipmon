# TurnipMon - Animal Crossing Turnip Market Monitor

This hastily slapped together project can easily be run on a free t2micro instance or any other cheap server infrastructure you might have available. It will constantly monitor two popular turnip markets on reddit for new available trades and send an update to your phone within 90 seconds using IFTTT.

The idea for this project initially stemmed from finding the official reddit app insufficient for my needs of quick-ish notifications for certain subreddits only.

This was written up and tested over the duration of a coffee infused early morning, there are no tests and code quality is lacking severely. Still, if you're interested in giving it a try, please go ahead. I might come back and do all the cleanup and housekeeping work needed to make this solution suitable for a broader audience in the future, given the interest.

# How to run

- Setup your IFTTT project, using a web hook (without JSON payload) as trigger and a notification as action. The IFTTT app will need to be installed on your phone for the notification to be received on there
- Log in to reddit and follow the steps outlined [here](https://github.com/reddit-archive/reddit/wiki/OAuth2-Quick-Start-Example#first-steps) in order to create your own app. This step is needed to due to inconsistent rate limiting on the public API access.
- Run `make build` in order to compile for x64 Linux or adapt the file as suited for your needs
- Take a look at the provided `.env.local` or call `turnipmon --help` to see how to supply the necessary configuration values
- Run the binary
- ???
- Profit!