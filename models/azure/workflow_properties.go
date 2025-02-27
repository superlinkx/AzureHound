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

import "github.com/bloodhoundad/azurehound/enums"

type WorkflowProperties struct {
	AccessEndpoint     string            `json:"accessEndpoint,omitempty"`
	ChangedTime        string            `json:"changedTime,omitempty"`
	CreatedTime        string            `json:"createdTime,omitempty"`
	Definition         Definition        `json:"definition,omitempty"`
	IntegrationAccount ResourceReference `json:"integrationAccount,omitempty"`
	// Note: in testing this does not get populated, instead the parameters are listed within the definition
	Parameters        map[string]WorkflowParameter    `json:"parameters,omitempty"`
	ProvisioningState enums.WorkflowProvisioningState `json:"provisioningState,omitempty"`
	Sku               WorkflowSku                     `json:"sku,omitempty"`
	State             enums.WorkflowState             `json:"state,omitempty"`
	Version           string                          `json:"version,omitempty"`

	// This does not appear in the documentation, however, it gets populated in the response
	EndpointConfiguration EndpointConfiguration `json:"endpointsConfiguration,omitempty"`
}

type EndpointConfiguration struct {
	Workflow  WorkflowEndpointConfiguration `json:"workflow,omitempty"`
	Connector WorkflowEndpointConfiguration `json:"connector,omitempty"`
}

type WorkflowEndpointConfiguration struct {
	OutgoingIpAddresses       []AddressEndpointConfiguration `json:"outgoingIpAddresses,omitempty"`
	AccessEndpointIpAddresses []AddressEndpointConfiguration `json:"accessEndpointIpAddresses,omitempty"`
}

type AddressEndpointConfiguration struct {
	Address string `json:"address,omitempty"`
}
