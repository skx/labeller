// labels.go - Helpers for getting/creating a label.

package main

import (
	"fmt"

	"google.golang.org/api/gmail/v1"
)

// We cache the labels which are available to avoid the need to
// make too many HTTP-calls.
var labels2ID map[string]string
var id2Labels map[string]string

func LoadLabels() error {

	// Create our maps
	labels2ID = make(map[string]string)
	id2Labels = make(map[string]string)

	// Get the existing labels, if any.
	r, err := srv.Users.Labels.List("me").Do()
	if err != nil {
		return err
	}

	// Store each one in our local map/cache.
	for _, label := range r.Labels {
		labels2ID[label.Name] = label.Id
		id2Labels[label.Id] = label.Name
	}

	return nil
}

// getLabelID returns the ID of a label, creating it if it is absent.
func getLabelID(srv *gmail.Service, name string) (string, error) {

	//
	// Get the list of labels if we've not done so.
	//
	if len(labels2ID) == 0 {
		err := LoadLabels()
		if err != nil {
			return "", err
		}
	}

	// If the list of labels contains the one we want then return the ID.
	found := labels2ID[name]
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
	labels2ID[name] = created.Id
	id2Labels[created.Id] = name
	return created.Id, nil
}

// getLabelById returns the human readable label, by ID
func getLabelById(srv *gmail.Service, id string) (string, error) {
	//
	// Get the list of labels if we've not done so.
	//
	if len(labels2ID) == 0 {
		err := LoadLabels()
		if err != nil {
			return "", err
		}
	}

	// If the list of labels contains the one we want then return the ID.
	found := id2Labels[id]
	if len(found) > 0 {
		return found, nil
	}

	return "", fmt.Errorf("failed to lookup lable ID %s", id)
}
