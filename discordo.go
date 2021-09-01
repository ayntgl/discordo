package main

import (
	"sort"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rigormorrtiss/discordgo"
	"github.com/rigormorrtiss/discordo/ui"
	"github.com/rigormorrtiss/discordo/util"
	"github.com/rivo/tview"
	"github.com/zalando/go-keyring"
)

var (
	app               *tview.Application
	loginForm         *tview.Form
	guildsTreeView    *tview.TreeView
	messagesTextView  *tview.TextView
	messageInputField *tview.InputField
	mainFlex          *tview.Flex

	config          *util.Config
	session         *discordgo.Session
	selectedChannel *discordgo.Channel
	selectedMessage *discordgo.Message
)

func main() {
	config = util.NewConfig()

	if config.Theme != nil {
		tview.Styles = *config.Theme
	}

	app = tview.NewApplication().
		EnableMouse(config.Mouse).
		SetInputCapture(onAppInputCapture)
	guildsTreeView = ui.NewGuildsTreeView(onGuildsTreeViewSelected)
	messagesTextView = ui.NewMessagesTextView(
		app,
		onMessagesTextViewInputCapture,
	)
	messageInputField = ui.NewMessageInputField(onMessageInputFieldInputCapture)
	mainFlex = ui.NewMainFlex(
		guildsTreeView,
		messagesTextView,
		messageInputField,
	)

	token := config.Token
	if t, _ := keyring.Get("discordo", "token"); t != "" {
		token = t
	}

	if token != "" {
		app.
			SetRoot(mainFlex, true).
			SetFocus(guildsTreeView)

		session = newSession()
		session.Token = token
		session.Identify.Token = token
		if err := session.Open(); err != nil {
			panic(err)
		}
	} else {
		loginForm = ui.NewLoginForm(onLoginFormLoginButtonSelected)
		app.SetRoot(loginForm, true)
	}

	if err := app.Run(); err != nil {
		panic(err)
	}
}

func onAppInputCapture(e *tcell.EventKey) *tcell.EventKey {
	if e.Modifiers() == tcell.ModAlt {
		switch e.Rune() {
		case '1':
			app.SetFocus(guildsTreeView)
		case '2':
			app.SetFocus(messagesTextView)
		case '3':
			app.SetFocus(messageInputField)
		}
	}

	return e
}

func findIndexByMessageID(ms []*discordgo.Message, mID string) int {
	for i, m := range ms {
		if mID == m.ID {
			return i
		}
	}

	return -1
}

func onMessagesTextViewInputCapture(e *tcell.EventKey) *tcell.EventKey {
	if selectedChannel == nil {
		return nil
	}

	switch {
	case e.Key() == tcell.KeyUp || e.Rune() == 'k': // Up
		ms := selectedChannel.Messages
		hs := messagesTextView.GetHighlights()
		// If there are no currently highlighted message, highlight the last
		// message in the TextView.
		if len(hs) == 0 {
			messagesTextView.
				Highlight(ms[len(ms)-1].ID).
				ScrollToHighlight()
		} else {
			// Find the index of the currently highlighted message in the
			// *discordgo.Channel.Messages slice.
			idx := findIndexByMessageID(ms, hs[0])
			// If the index of the currently highlighted message is equal to
			// zero
			// (first message in the TextView), do not handle the event.
			if idx == 0 {
				return nil
			}
			// Highlight the message just before the currently highlighted
			// message.
			messagesTextView.
				Highlight(ms[idx-1].ID).
				ScrollToHighlight()
		}

		return nil
	case e.Key() == tcell.KeyDown || e.Rune() == 'j': // Down
		ms := selectedChannel.Messages
		hs := messagesTextView.GetHighlights()
		// If there are no currently highlighted message, highlight the last
		// message in the TextView.
		if len(hs) == 0 {
			messagesTextView.
				Highlight(ms[len(ms)-1].ID).
				ScrollToHighlight()
		} else {
			// Find the index of the highlighted message in the
			// *discordgo.Channel.Messages slice.
			idx := findIndexByMessageID(ms, hs[0])
			// If the index of the currently highlighted message is equal to the
			// total number of elements in the *discordgo.Channel.Messages
			// slice, do not handle the event.
			if idx == len(ms)-1 {
				return nil
			}
			// Highlight the message just after the currently highlighted
			// message.
			messagesTextView.
				Highlight(ms[idx+1].ID).
				ScrollToHighlight()
		}

		return nil
	case e.Key() == tcell.KeyHome || e.Rune() == 'g': // Top
		ms := selectedChannel.Messages
		// Highlight the last message in the selectedChannel.Messages slice
		// (the first message rendered in the TextView).
		messagesTextView.
			Highlight(ms[0].ID).
			ScrollToHighlight()
	case e.Key() == tcell.KeyEnd || e.Rune() == 'G': // Bottom
		ms := selectedChannel.Messages
		// Highlight the first message in the selectedChannel.Messages slice
		// (the last message rendered in the TextView).
		messagesTextView.
			Highlight(ms[len(ms)-1].ID).
			ScrollToHighlight()
	case e.Rune() == 'r': // Reply
		hs := messagesTextView.GetHighlights()
		if len(hs) == 0 {
			return nil
		}

		for _, m := range selectedChannel.Messages {
			if m.ID == hs[0] {
				selectedMessage = m
				break
			}
		}

		messageInputField.SetTitle(
			"Replying to " + selectedMessage.Author.Username,
		)
		app.SetFocus(messageInputField)
	}

	return e
}

func onMessageInputFieldInputCapture(e *tcell.EventKey) *tcell.EventKey {
	// If the "Alt" modifier key is pressed, do not handle the event.
	if e.Modifiers() == tcell.ModAlt {
		return nil
	}

	switch e.Key() {
	case tcell.KeyEnter:
		if selectedChannel == nil {
			return nil
		}

		t := strings.TrimSpace(messageInputField.GetText())
		if t == "" {
			return nil
		}

		if selectedMessage != nil {
			messageInputField.SetTitle("")
			go session.ChannelMessageSendReply(
				selectedMessage.ChannelID,
				t,
				selectedMessage.Reference(),
			)

			selectedMessage = nil
		} else {
			go session.ChannelMessageSend(selectedChannel.ID, t)
		}

		messageInputField.SetText("")
	case tcell.KeyCtrlV:
		text, _ := clipboard.ReadAll()
		text = messageInputField.GetText() + text
		messageInputField.SetText(text)
	case tcell.KeyEscape: // Cancel
		messageInputField.SetTitle("")
		selectedMessage = nil
	}

	return e
}

func newSession() *discordgo.Session {
	s, err := discordgo.New()
	if err != nil {
		panic(err)
	}

	s.UserAgent = "" +
		"Mozilla/5.0 (X11; Linux x86_64) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) " +
		"Chrome/92.0.4515.131 Safari/537.36"
	s.Identify.Compress = false
	s.Identify.Intents = 0
	s.Identify.LargeThreshold = 0
	s.Identify.Properties.Device = ""
	s.Identify.Properties.Browser = "Chrome"
	s.Identify.Properties.OS = "Linux"

	s.AddHandlerOnce(onSessionReady)
	s.AddHandler(onSessionMessageCreate)

	return s
}

func onSessionReady(_ *discordgo.Session, r *discordgo.Ready) {
	sort.Slice(r.Guilds, func(a, b int) bool {
		found := false
		for _, gID := range r.Settings.GuildPositions {
			if found {
				if gID == r.Guilds[b].ID {
					return true
				}
			} else {
				if gID == r.Guilds[a].ID {
					found = true
				}
			}
		}

		return false
	})

	n := guildsTreeView.GetRoot()
	for _, g := range r.Guilds {
		gn := tview.NewTreeNode(g.Name).
			SetReference(g.ID)
		n.AddChild(gn)
	}

	guildsTreeView.SetCurrentNode(n)
}

func onSessionMessageCreate(_ *discordgo.Session, m *discordgo.MessageCreate) {
	if selectedChannel == nil || selectedChannel.ID != m.ChannelID {
		return
	}

	selectedChannel.Messages = append(selectedChannel.Messages, m.Message)
	util.WriteMessage(
		messagesTextView,
		m.Message,
		session.State.Ready.User.ID,
	)
}

func onGuildsTreeViewSelected(n *tview.TreeNode) {
	selectedChannel = nil
	selectedMessage = nil
	messagesTextView.
		Clear().
		SetTitle("")

	switch n.GetLevel() {
	case 1:
		if len(n.GetChildren()) != 0 {
			n.SetExpanded(!n.IsExpanded())
			return
		}

		n.ClearChildren()

		gID := n.GetReference().(string)
		g, _ := session.State.Guild(gID)

		cs := g.Channels
		sort.Slice(cs, func(i, j int) bool {
			return cs[i].Position < cs[j].Position
		})

		// Top-level channels
		ui.CreateTopLevelChannelsTreeNodes(session.State, n, cs)
		// Category channels
		ui.CreateCategoryChannelsTreeNodes(session.State, n, cs)
		// Second-level channels
		ui.CreateSecondLevelChannelsTreeNodes(session.State, guildsTreeView, cs)
	default:
		cID := n.GetReference().(string)
		c, _ := session.State.Channel(cID)

		if c.Type == discordgo.ChannelTypeGuildCategory {
			n.SetExpanded(!n.IsExpanded())
		} else if c.Type == discordgo.ChannelTypeGuildNews || c.Type == discordgo.ChannelTypeGuildText {
			selectedChannel = c
			app.SetFocus(messageInputField)

			title := "#" + c.Name
			if c.Topic != "" {
				title += " - " + c.Topic
			}
			messagesTextView.
				Clear().
				SetTitle(title)

			go writeMessages(c.ID)
		}
	}
}

func writeMessages(cID string) {
	msgs, _ := session.ChannelMessages(cID, config.GetMessagesLimit, "", "", "")
	for i := len(msgs) - 1; i >= 0; i-- {
		selectedChannel.Messages = append(selectedChannel.Messages, msgs[i])

		util.WriteMessage(
			messagesTextView,
			msgs[i],
			session.State.Ready.User.ID,
		)
	}
}

func onLoginFormLoginButtonSelected() {
	email := loginForm.GetFormItem(0).(*tview.InputField).GetText()
	password := loginForm.GetFormItem(1).(*tview.InputField).GetText()
	if email == "" || password == "" {
		return
	}

	session = newSession()
	// Try to login without TOTP
	lr, err := util.Login(session, email, password)
	if err != nil {
		panic(err)
	}

	if lr.Token != "" && !lr.MFA {
		app.
			SetRoot(mainFlex, true).
			SetFocus(guildsTreeView)

		session.Token = lr.Token
		session.Identify.Token = lr.Token
		if err = session.Open(); err != nil {
			panic(err)
		}

		go keyring.Set("discordo", "token", lr.Token)
	} else if lr.MFA {
		loginForm = ui.NewMfaLoginForm(func() {
			code := loginForm.GetFormItem(0).(*tview.InputField).GetText()
			if code == "" {
				return
			}

			lr, err = util.TOTP(session, code, lr.Ticket)
			if err != nil {
				panic(err)
			}

			app.
				SetRoot(mainFlex, true).
				SetFocus(guildsTreeView)

			session.Token = lr.Token
			session.Identify.Token = lr.Token
			if err = session.Open(); err != nil {
				panic(err)
			}

			go keyring.Set("discordo", "token", lr.Token)
		})

		app.SetRoot(loginForm, true)
	}
}
