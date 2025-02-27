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

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/bloodhoundad/azurehound/client"
	"github.com/bloodhoundad/azurehound/enums"
	"github.com/bloodhoundad/azurehound/models"
	"github.com/bloodhoundad/azurehound/pipeline"
	"github.com/spf13/cobra"
)

func init() {
	listRootCmd.AddCommand(listVirtualMachineRoleAssignmentsCmd)
}

var listVirtualMachineRoleAssignmentsCmd = &cobra.Command{
	Use:          "virtual-machine-role-assignments",
	Long:         "Lists Virtual Machine Role Assignments",
	Run:          listVirtualMachineRoleAssignmentsCmdImpl,
	SilenceUsage: true,
}

func listVirtualMachineRoleAssignmentsCmdImpl(cmd *cobra.Command, args []string) {
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, os.Kill)
	defer gracefulShutdown(stop)

	log.V(1).Info("testing connections")
	if err := testConnections(); err != nil {
		exit(err)
	} else if azClient, err := newAzureClient(); err != nil {
		exit(err)
	} else {
		log.Info("collecting azure virtual machine role assignments...")
		start := time.Now()
		subscriptions := listSubscriptions(ctx, azClient)
		stream := listVirtualMachineRoleAssignments(ctx, azClient, listVirtualMachines(ctx, azClient, subscriptions))
		outputStream(ctx, stream)
		duration := time.Since(start)
		log.Info("collection completed", "duration", duration.String())
	}
}

func listVirtualMachineRoleAssignments(ctx context.Context, client client.AzureClient, virtualMachines <-chan interface{}) <-chan azureWrapper[models.VirtualMachineRoleAssignments] {
	var (
		out     = make(chan azureWrapper[models.VirtualMachineRoleAssignments])
		ids     = make(chan string)
		streams = pipeline.Demux(ctx.Done(), ids, 25)
		wg      sync.WaitGroup
	)

	go func() {
		defer close(ids)

		for result := range pipeline.OrDone(ctx.Done(), virtualMachines) {
			if virtualMachine, ok := result.(AzureWrapper).Data.(models.VirtualMachine); !ok {
				log.Error(fmt.Errorf("failed type assertion"), "unable to continue enumerating virtual machine role assignments", "result", result)
				return
			} else {
				ids <- virtualMachine.Id
			}
		}
	}()

	wg.Add(len(streams))
	for i := range streams {
		stream := streams[i]
		go func() {
			defer wg.Done()
			for id := range stream {
				var (
					virtualMachineRoleAssignments = models.VirtualMachineRoleAssignments{
						VirtualMachineId: id,
					}
					count = 0
				)
				for item := range client.ListRoleAssignmentsForResource(ctx, id, "") {
					if item.Error != nil {
						log.Error(item.Error, "unable to continue processing role assignments for this virtual machine", "virtualMachineId", id)
					} else {
						virtualMachineRoleAssignment := models.VirtualMachineRoleAssignment{
							VirtualMachineId: item.ParentId,
							RoleAssignment:   item.Ok,
						}
						log.V(2).Info("found virtual machine role assignment", "virtualMachineRoleAssignment", virtualMachineRoleAssignment)
						count++
						virtualMachineRoleAssignments.RoleAssignments = append(virtualMachineRoleAssignments.RoleAssignments, virtualMachineRoleAssignment)
					}
				}
				out <- NewAzureWrapper(enums.KindAZVMRoleAssignment, virtualMachineRoleAssignments)
				log.V(1).Info("finished listing virtual machine role assignments", "virtualMachineId", id, "count", count)
			}
		}()
	}

	go func() {
		wg.Wait()
		close(out)
		log.Info("finished listing all virtual machine role assignments")
	}()

	return out
}
