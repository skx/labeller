// main.go - Entry-point
//
// This script adds labels to unread messages, automatically.
//
// It is a bit of a hack largely because of the authentication-magic.
//
// Assume mail comes from "bob.smith@example.com" then I will add two
// labels:
//
//     "bob-smith"
//     "example-com"
//
// New labels will be created if they're not present, on demand.
//
//
package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/mail"
	"os"
	"strings"

	"github.com/skx/evalfilter/v2"
	"github.com/skx/evalfilter/v2/object"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

var (

	// Should we be verbose?
	verbose *bool

	// Handle to the gmail client
	srv *gmail.Service

	// Handle to our scripting engine
	eval *evalfilter.Eval

	// ID of the message the script is processing.
	//
	// This must be global, because there is no context available
	// to the scripting-engine.  That feels like a bug :)
	msgID string
)

// Message is the structure that we pass to our scripting-engine,
// which allows users to decide what they want to do with the given
// message.
type Message struct {

	// A message might have multiple recipients
	// so we have to store these as arrays.
	To       []string // steve@steve.org.uk
	ToPart   []string // steve
	ToDomain []string // steve.org.uk

	From       string // bob@example.com
	FromPart   string // bob
	FromDomain string // example.com

	Subject string
}

// parseAddress turns an email address into individual parts
func parseAddress(address string) (string, string, string) {

	// Get the raw email address from the header.
	//
	// So '"Steve Kemp" <foo@example.com>' will become 'foo@example.com'.
	//
	addr, _ := mail.ParseAddress(address)
	parts := strings.Split(addr.Address, "@")

	return addr.Address, parts[0], parts[1]
}

//
// Prepare the user-script
//
func prepareScript(path string) error {

	//
	// Load the script
	//
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	//
	// Create our script intepreter, and pass the script to it.
	//
	eval = evalfilter.New(string(data))
	err = eval.Prepare()
	if err != nil {
		return fmt.Errorf("failed to parse the script %s - %s", path, err.Error())
	}

	//
	// Extend our scripting-langague with new primitives.
	//
	//    add("String") -> Adds the given label.
	//
	eval.AddFunction("add",
		func(args []object.Object) object.Object {
			if len(args) != 1 {
				return &object.Boolean{Value: false}
			}

			// Stringify
			str := args[0].Inspect()

			if *verbose {
				fmt.Printf("\tAdding label [%s] to message %s\n", str, msgID)
			}

			// Get the label ID
			id, err := getLabelID(srv, str)
			if err != nil {
				fmt.Printf("WARNING: failed to find/create label '%s' - %s", str, err.Error())
				return &object.Boolean{Value: false}
			}

			// Create the modification of the message.
			mod := &gmail.ModifyMessageRequest{AddLabelIds: []string{id}}

			// Perform the modification
			_, err = srv.Users.Messages.Modify("me", msgID, mod).Do()
			if err != nil {
				fmt.Printf("unable to add label [%s] to message %s - %v", str, msgID, err)
				return &object.Boolean{Value: true}
			}

			return &object.Boolean{Value: true}
		})
	eval.AddFunction("remove",
		func(args []object.Object) object.Object {
			if len(args) != 1 {
				return &object.Boolean{Value: false}
			}

			// Stringify
			str := args[0].Inspect()

			if *verbose {
				fmt.Printf("\tRemoving label [%s] from message %s\n", str, msgID)
			}

			// Get the label ID
			id, err := getLabelID(srv, str)
			if err != nil {
				fmt.Printf("WARNING: failed to find/create label '%s' - %s", str, err.Error())
				return &object.Boolean{Value: false}
			}

			// Create the modification of the message.
			mod := &gmail.ModifyMessageRequest{RemoveLabelIds: []string{id}}

			// Perform the modification
			_, err = srv.Users.Messages.Modify("me", msgID, mod).Do()
			if err != nil {
				fmt.Printf("unable to remove label [%s] from message %s - %v", str, msgID, err)
				return &object.Boolean{Value: true}
			}

			return &object.Boolean{Value: true}
		})

	return nil
}

// main is the entry-point to our application.
func main() {

	//
	// Command-line flags
	//
	filter := flag.String("filter", "is:unread -has:userlabels", "The search we perform to find messages to modify.")
	script := flag.String("script", os.Getenv("HOME")+"/.labeller.script", "The script we execute against messages.")
	updateLabels := flag.Bool("update-labels", false, "Mark all labels as 'labelShowIfUnread'.")
	verbose = flag.Bool("verbose", false, "Should we be more verbose?")
	flag.Parse()

	//
	// Read the project-credentials.
	//
	b, err := ioutil.ReadFile(os.Getenv("HOME") + "/.labeller.credentials")
	if err != nil {
		fmt.Printf("Unable to read client secret file: %v\n", err)
		fmt.Printf("These should be downloaded from the Google console:\n")
		fmt.Printf("https://console.developers.google.com/apis/credentials\n")
		return
	}

	//
	// Handle the setup.
	//
	config, err := google.ConfigFromJSON(b, gmail.GmailModifyScope)
	if err != nil {
		fmt.Printf("Unable to parse client secret file to config: %v", err)
		return
	}

	//
	// Create the client
	//
	client := getClient(config)

	//
	// At this point we have authentication handled, so we can actually
	// start processing our things.
	//
	//	srv, err = gmail.New(client)
	srv, err = gmail.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		fmt.Printf("Unable to retrieve Gmail client: %v", err)
		return
	}

	//
	// If we're reworking our labels then do so
	//
	if *updateLabels {

		//
		// Find all the labels
		//
		existing, err := srv.Users.Labels.List("me").Do()
		if err != nil {
			fmt.Printf("Error updating labels: %s\n", err.Error())
			return
		}

		//
		// For each one.
		//
		total := len(existing.Labels)

		for index, label := range existing.Labels {

			// Show progress.
			fmt.Printf("%d/%d - %0.0f%% complete\n", index, total, (float64(index) / float64(total) * 100))
			//
			// We're going to change the visibility
			//
			label.LabelListVisibility = "labelShowIfUnread"
			_, err = srv.Users.Labels.Update("me", label.Id, label).Do()

			if err != nil {
				fmt.Printf("Warning failed to change visibility of label:%s\n", err.Error())
			}
		}
		return
	}

	//
	// Setup our scripting engine, and load the user-script into it.
	//
	err = prepareScript(*script)
	if err != nil {
		fmt.Printf("Error loading user-script: %s\n", err.Error())
		return
	}

	//
	// We now search for messages.
	//
	msgs, err := srv.Users.Messages.List("me").Q(*filter).Do()
	if err != nil {
		fmt.Printf("Failed to find messages:%s\n", err.Error())
		return
	}

	//
	// Show how many we found.
	//
	if *verbose {
		fmt.Printf("The filter search returned %d messages\n", len(msgs.Messages))
	}

	//
	// Process each message - via our loaded script
	//
	for _, entry := range msgs.Messages {

		//
		// Global variable holds the message-ID being processed.
		// Gross, but ..
		//
		msgID = entry.Id

		//
		// Show the ID
		//
		if *verbose {
			fmt.Printf("\tProcessing message %s\n", entry.Id)
		}

		//
		// Get the message.
		//
		// We specify "metadata" here which means we only need to return
		// a few details from the message rather than the complete email.
		//
		msg, err := srv.Users.Messages.Get("me", entry.Id).Format("metadata").Do()
		if err != nil {
			fmt.Printf("Could not retrieve message %s %v", entry.Id, err)
			continue
		}

		//
		// Parse the details of the message into an instance of our
		// message-structure.  This object will be what we pass to
		// our embedded scripting-language.
		//
		var data Message

		//
		// Populate the structure, this is a bit horrid.
		//
		for _, h := range msg.Payload.Headers {

			// Sender
			if h.Name == "From" && strings.Contains(h.Value, "@") {
				data.From, data.FromPart, data.FromDomain = parseAddress(h.Value)
			}

			// Recipient(s)
			//
			// NOTE: here we cheat and treat Cc: as
			// synonymous with To.
			//
			// That's probably OK.
			//
			if (h.Name == "To" || h.Name == "Cc") &&
				strings.Contains(h.Value, "@") {

				addresses := strings.Split(h.Value, ",")

				for _, recipient := range addresses {
					to, part, domain := parseAddress(recipient)

					data.To = append(data.To, to)
					data.ToPart = append(data.ToPart, part)
					data.ToDomain = append(data.ToDomain, domain)
				}

			}

			// Subject
			if h.Name == "Subject" {
				data.Subject = h.Value
			}
		}

		//
		// Evaluate the user-script.
		//
		// This might add labels, etc.
		//
		_, scriptErr := eval.Run(data)
		if scriptErr != nil {
			fmt.Printf("Error executing script:%s", scriptErr.Error())
		}
	}

}
