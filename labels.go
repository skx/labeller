// labels.go - Helpers for getting/creating a label.

package main

import (
	"google.golang.org/api/gmail/v1"
)

// We cache the labels which are available to avoid the need to
// make too many HTTP-calls.
var labels map[string]string

// getLabelID returns the ID of a label, creating it if it is absent.
func getLabelID(srv *gmail.Service, name string) (string, error) {

	//
	// Get the list of labels if we've not done so.
	//
	if len(labels) == 0 {

		// Create our map
		labels = make(map[string]string)

		// Get the existing labels, if any.
		r, err := srv.Users.Labels.List("me").Do()
		if err != nil {
			return "", err
		}

		// Store each one in our local map/cache.
		for _, label := range r.Labels {
			labels[label.Name] = label.Id
		}
	}

	// If the list of labels contains the one we want then return the ID.
	found := labels[name]
	if len(found) > 0 {
		return found, nil
	}

	// Otherwise we need to create the label.
	req := &gmail.Label{Name: name}
	created, err := srv.Users.Labels.Create("me", req).Do()
	if err != nil {
		return "", err
	}

	// Store the ID for next time, and return it.
	labels[name] = created.Id
	return created.Id, nil
}
