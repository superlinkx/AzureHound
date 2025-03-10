// Copyright (C) 2022 Specter Ops, Inc.
//
// This file is part of AzureHound.
//
// AzureHound is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// AzureHound is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package azure

import "strings"

type Workflow struct {
	Entity

	Identity   ManagedIdentity    `json:"identity,omitempty"`
	Location   string             `json:"location,omitempty"`
	Name       string             `json:"name,omitempty"`
	Properties WorkflowProperties `json:"properties,omitempty"`
	Tags       map[string]string  `json:"tags,omitempty"`
	Type       string             `json:"type,omitempty"`
}

func (s Workflow) ResourceGroupName() string {
	parts := strings.Split(s.Id, "/")
	if len(parts) > 4 {
		return parts[4]
	} else {
		return ""
	}
}

func (s Workflow) ResourceGroupId() string {
	parts := strings.Split(s.Id, "/")
	if len(parts) > 5 {
		return strings.Join(parts[:5], "/")
	} else {
		return ""
	}
}

type WorkflowList struct {
	NextLink string     `json:"nextLink,omitempty"` // The URL to use for getting the next set of values.
	Value    []Workflow `json:"value"`              // A list of workflows.
}

type WorkflowResult struct {
	SubscriptionId string
	Error          error
	Ok             Workflow
}
